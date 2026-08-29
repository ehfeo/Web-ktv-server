package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/skip2/go-qrcode"
)

// QR客户端配置
var (
	qrServerAddr  string // 外接二维码服务器地址，如 "123.45.67.89:8352"
	qrPassword    string // 手机端验证密码
	qrEnabled     bool   // 是否启用QR功能
	qrMode        string // "internal"=内置二维码服务器 / "external"=外接，默认外接
	qrCtrlEnabled bool   // 是否允许手机端远程控制播放（切歌/播放暂停/重唱/音量）
	qrTrackMode   string // 当前播放曲目的音轨模式：track(原唱/伴奏) / channel(立体声/左右声道)，默认track
	qrChannelCnt  int    // 当前播放曲目的音频声道数（channel模式用于显示立体声/左/右按钮）
)

// qrReplyHook 分发KTV侧处理结果：
// 外接模式=发送到外接中继服务器（中继按requestId投递给对应手机）
// 内置模式=进程内直接路由到对应手机WebSocket连接（见 qr_internal.go）
var qrReplyHook func(resp qrMessage)

// QR客户端运行时状态
var (
	qrConnMutex    sync.Mutex
	qrConn         *websocket.Conn // 当前WebSocket连接
	qrConnected    bool            // 连接状态

	qrPendingMutex sync.Mutex
	qrPendingSongs map[string][]PendingSong // 按sessionId分组的待处理点歌请求

	qrQueueMutex sync.Mutex
	qrQueueData  map[string]json.RawMessage // 按sessionId分组的队列数据
	qrQueueIndex map[string]int             // 按sessionId分组的当前播放索引

	qrControlMutex    sync.Mutex
	qrPendingControls map[string][]qrControlCmd // 按sessionId分组的遥控指令（主控端轮询取走执行）
)

// qrControlCmd 手机端下发的遥控指令
type qrControlCmd struct {
	Action string `json:"action"` // next / restart / togglePause / volume
	Value  int    `json:"value"`  // 音量等附加数值
}

func init() {
	qrPendingSongs = make(map[string][]PendingSong)
	qrQueueData = make(map[string]json.RawMessage)
	qrQueueIndex = make(map[string]int)
	qrPendingControls = make(map[string][]qrControlCmd)
}

// PendingSong 手机端发来的点歌请求
type PendingSong struct {
	Path string `json:"path"`
	Name string `json:"name"`
	Type string `json:"type"`
}

// QR消息类型
type qrMessage struct {
	Type                string          `json:"type"`
	SessionID           string          `json:"sessionId,omitempty"`
	Password            string          `json:"password,omitempty"`
	RequestID           string          `json:"requestId,omitempty"`
	Keyword             string          `json:"keyword,omitempty"`
	Page                int             `json:"page,omitempty"`
	PageSize            int             `json:"pageSize,omitempty"`
	Path                string          `json:"path,omitempty"`
	Name                string          `json:"name,omitempty"`
	SongType            string          `json:"songType,omitempty"`
	Success             bool            `json:"success,omitempty"`
	Message             string          `json:"message,omitempty"`
	Results             []qrSearchItem  `json:"results,omitempty"`
	Total               int             `json:"total,omitempty"`
	Queue               json.RawMessage `json:"queue,omitempty"`
	CurrentPlayingIndex int             `json:"currentPlayingIndex,omitempty"`
	Singer              string          `json:"singer,omitempty"`
	Language            string          `json:"language,omitempty"`
	Category            string          `json:"category,omitempty"`
	Data                json.RawMessage `json:"data,omitempty"`
	Action              string          `json:"action,omitempty"`  // 遥控指令：next/restart/togglePause/volume
	Value               int             `json:"value,omitempty"`   // 遥控附加数值（如音量0-100）
	CtrlEnabled         bool            `json:"ctrlEnabled,omitempty"`  // 手机遥控权限是否开启
	PasswordNeeded      bool            `json:"passwordNeeded,omitempty"` // 是否需要输入密码
	TrackMode           string          `json:"trackMode,omitempty"`  // 当前音轨模式：track=多音轨(原唱/伴奏) / channel=单音轨(立体声/左右声道)
	ChannelCount        int             `json:"channelCount,omitempty"` // 当前曲目音频声道数（用于channel模式）
}

type qrSearchItem struct {
	Name   string `json:"name"`
	Path   string `json:"path"`
	Type   string `json:"type"`
	Singer string `json:"singer"`
}

// generateSessionID 生成随机会话ID（8字符hex）
func generateSessionID() string {
	b := make([]byte, 4)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// generateRequestID 生成随机请求ID（8字符hex），用于内置模式下回包路由
func generateRequestID() string {
	b := make([]byte, 4)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// initQRConfig 初始化QR客户端配置，从配置文件加载
func initQRConfig() {
	data, err := ioutil.ReadFile(configFile)
	if err != nil {
		return
	}

	var cfg struct {
		QRServerAddr   string `json:"qrServerAddr"`
		QRPassword     string `json:"qrPassword"`
		QREnabled      bool   `json:"qrEnabled"`
		QRMode         string `json:"qrMode"`
		QRCtrlEnabled  bool   `json:"qrCtrlEnabled"`
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return
	}

	qrServerAddr = cfg.QRServerAddr
	qrPassword = cfg.QRPassword
	qrEnabled = cfg.QREnabled
	qrMode = cfg.QRMode
	qrCtrlEnabled = cfg.QRCtrlEnabled
	if qrMode == "" {
		qrMode = "external" // 默认外接
	}

	if qrEnabled {
		fmt.Printf("[QR客户端] 配置加载: mode=%s server=%s\n", qrMode, qrServerAddr)
	}
}

// saveQRConfig 保存QR配置到配置文件
func saveQRConfig() {
	data, err := ioutil.ReadFile(configFile)
	if err != nil {
		return
	}

	var cfg map[string]interface{}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return
	}

	cfg["qrServerAddr"] = qrServerAddr
	cfg["qrPassword"] = qrPassword
	cfg["qrEnabled"] = qrEnabled
	cfg["qrMode"] = qrMode
	cfg["qrCtrlEnabled"] = qrCtrlEnabled

	out, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return
	}
	ioutil.WriteFile(configFile, out, 0644)
}

// isQRExternal 判断当前是否为外接模式（启用QR且模式为外接）
func isQRExternal() bool {
	return qrEnabled && qrMode == "external"
}

// qrCtrlAllowed 是否允许手机端远程控制播放
func qrCtrlAllowed() bool {
	return qrEnabled && qrCtrlEnabled
}

// handleQRControl 处理手机端遥控指令：切歌/重唱/播放暂停/音量
func handleQRControl(msg qrMessage) {
	if !qrCtrlAllowed() {
		qrReply(qrMessage{
			Type:      "controlDenied",
			RequestID: msg.RequestID,
			Message:   "主控端未开启手机遥控权限",
		})
		return
	}

	sid := msg.SessionID
	if sid == "" {
		sid = "default"
	}
	qrControlMutex.Lock()
	qrPendingControls[sid] = append(qrPendingControls[sid], qrControlCmd{Action: msg.Action, Value: msg.Value})
	qrControlMutex.Unlock()

	qrReply(qrMessage{Type: "controlOk", RequestID: msg.RequestID, Message: "ok"})
}

// qrControlPollHandler 主控端轮询取走手机端下发的遥控指令（取走后清空）
func qrControlPollHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	sid := r.URL.Query().Get("sessionId")

	var cmds []qrControlCmd
	qrControlMutex.Lock()
	if sid == "" {
		cmds = qrPendingControls[""]
		qrPendingControls[""] = nil
	} else {
		cmds = qrPendingControls[sid]
		qrPendingControls[sid] = nil
	}
	qrControlMutex.Unlock()

	if cmds == nil {
		cmds = []qrControlCmd{}
	}
	json.NewEncoder(w).Encode(struct {
		Controls []qrControlCmd `json:"controls"`
	}{Controls: cmds})
}

// qrBuildControlState 构造遥控权限状态消息
func qrBuildControlState(sid string) qrMessage {
	if qrTrackMode == "" {
		qrTrackMode = "track"
	}
	if qrChannelCnt <= 0 {
		qrChannelCnt = 2
	}
	return qrMessage{
		Type:           "controlState",
		SessionID:      sid,
		CtrlEnabled:    qrCtrlAllowed(),
		PasswordNeeded: qrPassword != "",
		TrackMode:      qrTrackMode,
		ChannelCount:   qrChannelCnt,
	}
}

// qrTrackStateHandler 主控端上报当前曲目的音轨模式（track/channel）及声道数，并推送给所有手机
func qrTrackStateHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	q := r.URL.Query()
	mode := q.Get("mode")
	if mode == "track" || mode == "channel" {
		qrTrackMode = mode
	}
	if n, err := strconv.Atoi(q.Get("channels")); err == nil && n > 0 {
		qrChannelCnt = n
	}
	broadcastQRControlState()
	w.Write([]byte(`{"ok":true}`))
}

// broadcastQRControlState 主控端修改遥控权限后，向所有已连接手机推送最新状态
func broadcastQRControlState() {
	// 内置模式：遍历内置服务器会话，向每个已连接手机推送
	if qrMode == "internal" {
		qrInternalMu.Lock()
		conns := []*websocket.Conn{}
		for _, s := range qrInternalSessions {
			for c := range s.mobiles {
				conns = append(conns, c)
			}
		}
		qrInternalMu.Unlock()
		msg := qrBuildControlState("")
		for _, c := range conns {
			data, _ := json.Marshal(msg)
			if err := c.WriteMessage(websocket.TextMessage, data); err != nil {
				// 忽略单连接写入错误
			}
		}
	}
	// 外接模式：通知中继服务器，让它重新向各手机询问最新权限状态
	if isQRExternal() {
		qrWriteJSON(qrMessage{Type: "broadcastControlState"})
	}
}

// qrReply 统一回复出口：内置模式走进程内路由，外接模式发送到中继服务器
func qrReply(resp qrMessage) {
	if qrReplyHook != nil {
		qrReplyHook(resp)
		return
	}
	// 默认(外接)：发送到外接中继服务器
	qrWriteJSON(resp)
}

// startQRClient 启动QR WebSocket客户端（应在goroutine中调用，仅外接模式）
func startQRClient() {
	if !isQRExternal() {
		return
	}
	if qrServerAddr == "" {
		return
	}

	fmt.Printf("[QR客户端] 启动，服务器: %s\n", qrServerAddr)

	for {
		err := connectQRServer()
		if err != nil {
			fmt.Printf("[QR客户端] 连接断开: %v，5秒后重连...\n", err)
		} else {
			fmt.Println("[QR客户端] 连接正常关闭，5秒后重连...")
		}

		setQRConnected(false)
		time.Sleep(5 * time.Second)
	}
}

// connectQRServer 连接到QR中继服务器并处理消息
func connectQRServer() error {
	scheme := "ws"
	if strings.HasPrefix(strings.ToLower(qrServerAddr), "https") {
		scheme = "wss"
	}

	// 如果地址已经包含协议前缀，直接使用
	url := fmt.Sprintf("%s://%s/ws/ktv", scheme, qrServerAddr)
	if strings.HasPrefix(strings.ToLower(qrServerAddr), "ws://") || strings.HasPrefix(strings.ToLower(qrServerAddr), "wss://") {
		url = qrServerAddr + "/ws/ktv"
	}

	fmt.Printf("[QR客户端] 正在连接: %s\n", url)

	dialer := websocket.DefaultDialer
	conn, _, err := dialer.Dial(url, nil)
	if err != nil {
		return fmt.Errorf("连接失败: %v", err)
	}

	qrConnMutex.Lock()
	qrConn = conn
	qrConnMutex.Unlock()

	setQRConnected(true)
	fmt.Printf("[QR客户端] 已连接到 %s\n", url)

	// 不再在连接时发送注册消息，改为前端注册会话时按需发送
	fmt.Println("[QR客户端] 已连接，等待前端注册会话...")

	// 消息循环
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			return err
		}

		go handleQRMessage(message)
	}
}

// handleQRMessage 处理从QR服务器收到的消息
func handleQRMessage(data []byte) {
	var msg qrMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		fmt.Printf("[QR客户端] 解析消息失败: %v\n", err)
		return
	}

	switch msg.Type {
	case "search":
		handleQRSearch(msg)
	case "addSong":
		handleQRAddSong(msg)
	case "verifyPassword":
		handleQRVerifyPassword(msg)
	case "queueUpdate":
		handleQRQueueUpdate(msg)
	case "requestQueue":
		handleQRRequestQueue(msg)
	case "browse":
		handleQRBrowse(msg)
	case "control":
		handleQRControl(msg)
	case "requestControlState":
		// 外接模式：中继服务器收到手机连接后询问遥控权限，回复后由中继广播给对应手机
		qrReply(qrBuildControlState(msg.SessionID))
	default:
		fmt.Printf("[QR客户端] 未知消息类型: %s\n", msg.Type)
	}
}

// handleQRSearch 处理搜索请求
func handleQRSearch(msg qrMessage) {
	page := msg.Page
	if page < 1 {
		page = 1
	}
	pageSize := msg.PageSize
	if pageSize < 1 {
		pageSize = 20
	}

	songs, total := getMediaListPaged(page, pageSize, msg.Keyword)

	if total == 0 && msg.Keyword != "" {
		logZeroResultKeyword(msg.Keyword)
	}

	results := make([]qrSearchItem, 0, len(songs))
	for _, s := range songs {
		results = append(results, qrSearchItem{
			Name:   s.Name,
			Path:   s.Path,
			Type:   s.Type,
			Singer: s.Singer,
		})
	}

	resp := qrMessage{
		Type:      "searchResult",
		RequestID: msg.RequestID,
		Results:   results,
		Total:     total,
		Page:      page,
		PageSize:  pageSize,
	}

	qrReply(resp)
}

// handleQRAddSong 处理点歌请求，存入待处理列表供前端拉取
func handleQRAddSong(msg qrMessage) {
	song := PendingSong{
		Path: msg.Path,
		Name: msg.Name,
		Type: msg.SongType,
	}

	qrPendingMutex.Lock()
	sid := msg.SessionID
	if sid == "" {
		sid = "default"
	}
	qrPendingSongs[sid] = append(qrPendingSongs[sid], song)
	qrPendingMutex.Unlock()

	// 回复确认
	resp := qrMessage{
		Type:      "addSongResult",
		RequestID: msg.RequestID,
		Success:   true,
		Message:   "已加入待处理列表",
	}

	qrReply(resp)
	fmt.Printf("[QR客户端] 收到点歌请求: %s [session=%s]\n", msg.Name, sid)
}

// handleQRVerifyPassword 处理密码验证请求
func handleQRVerifyPassword(msg qrMessage) {
	success := msg.Password == qrPassword

	resp := qrMessage{
		Type:      "passwordResult",
		RequestID: msg.RequestID,
		Success:   success,
		Message:   func() string { if success { return "验证成功" }; return "密码错误" }(),
	}

	qrReply(resp)
}

// handleQRQueueUpdate 处理队列更新（从手机端发来的队列查询等）
func handleQRQueueUpdate(msg qrMessage) {
	qrQueueMutex.Lock()
	sid := msg.SessionID
	if sid == "" {
		sid = "default"
	}
	qrQueueData[sid] = msg.Queue
	qrQueueIndex[sid] = msg.CurrentPlayingIndex
	qrQueueMutex.Unlock()

	// 队列数据已缓存，无需额外日志
}

// handleQRRequestQueue 手机端连接时，QR服务器请求当前队列
func handleQRRequestQueue(msg qrMessage) {
	sid := msg.SessionID
	if sid == "" {
		sid = "default"
	}
	qrQueueMutex.Lock()
	cachedQueue := qrQueueData[sid]
	cachedIndex := qrQueueIndex[sid]
	qrQueueMutex.Unlock()

	if cachedQueue == nil {
		return
	}

	resp := qrMessage{
		Type:                "queueUpdate",
		SessionID:           sid,
		Queue:               cachedQueue,
		CurrentPlayingIndex: cachedIndex,
	}
	qrReply(resp)
}

// handleQRBrowse 处理手机端浏览请求（歌手/语种/曲种/热播）
func handleQRBrowse(msg qrMessage) {
	page := msg.Page
	if page < 1 {
		page = 1
	}
	pageSize := msg.PageSize
	if pageSize < 1 {
		pageSize = 20
	}

	browseType := msg.Keyword
	list := getCachedMediaList()

	switch browseType {
	case "singerIndex":
		// 歌手列表（按首字母分组，组内按歌曲数降序）
		singerCount := make(map[string]int)
		for _, item := range list {
			if item.Singer != "" && item.Singer != "未知歌手" {
				singerCount[item.Singer]++
			}
		}
		letterMap := make(map[string][]map[string]interface{})
		for singer, count := range singerCount {
			letter := singerFirstChar(singer)
			letterMap[letter] = append(letterMap[letter], map[string]interface{}{
				"name":  singer,
				"count": count,
			})
		}
		for letter := range letterMap {
			sort.Slice(letterMap[letter], func(i, j int) bool {
				ci, _ := letterMap[letter][i]["count"].(int)
				cj, _ := letterMap[letter][j]["count"].(int)
				if ci != cj {
					return ci > cj
				}
				ni, _ := letterMap[letter][i]["name"].(string)
				nj, _ := letterMap[letter][j]["name"].(string)
				return ni < nj
			})
		}
		data, _ := json.Marshal(letterMap)
		resp := qrMessage{
			Type:      "browseResult",
			RequestID: msg.RequestID,
			Data:      data,
		}
		qrReply(resp)

	case "singerSongs":
		// 某歌手的歌曲列表（分页）
		var songs []qrSearchItem
		for _, item := range list {
			if item.Singer == msg.Singer {
				songs = append(songs, qrSearchItem{
					Name:   item.Name,
					Path:   item.Path,
					Type:   item.Type,
					Singer: item.Singer,
				})
			}
		}
		total := len(songs)
		start := (page - 1) * pageSize
		if start > total {
			start = total
		}
		end := start + pageSize
		if end > total {
			end = total
		}
		paged := songs[start:end]
		resp := qrMessage{
			Type:      "browseResult",
			RequestID: msg.RequestID,
			Results:   paged,
			Total:     total,
			Page:      page,
		PageSize:  pageSize,
	}
	qrReply(resp)

	case "languageIndex":
		// 语种列表（按歌曲数降序）
		langCount := make(map[string]int)
		for _, item := range list {
			lang := item.Language
			if lang == "" {
				lang = "未知"
			}
			langCount[lang]++
		}
		var result []map[string]interface{}
		for lang, count := range langCount {
			result = append(result, map[string]interface{}{
				"name":  lang,
				"count": count,
			})
		}
		sort.Slice(result, func(i, j int) bool {
			ci, _ := result[i]["count"].(int)
			cj, _ := result[j]["count"].(int)
			return ci > cj
		})
		data, _ := json.Marshal(result)
		resp := qrMessage{
			Type:      "browseResult",
			RequestID: msg.RequestID,
			Data:      data,
		}
		qrReply(resp)

	case "languageSongs":
		// 某语种的歌曲列表（分页）
		var songs []qrSearchItem
		for _, item := range list {
			lang := item.Language
			if lang == "" {
				lang = "未知"
			}
			if lang == msg.Language {
				songs = append(songs, qrSearchItem{
					Name:   item.Name,
					Path:   item.Path,
					Type:   item.Type,
					Singer: item.Singer,
				})
			}
		}
		total := len(songs)
		start := (page - 1) * pageSize
		if start > total {
			start = total
		}
		end := start + pageSize
		if end > total {
			end = total
		}
		paged := songs[start:end]
		resp := qrMessage{
			Type:      "browseResult",
			RequestID: msg.RequestID,
			Results:   paged,
			Total:     total,
			Page:      page,
			PageSize:  pageSize,
		}
		qrReply(resp)

	case "categoryIndex":
		// 曲种列表（按歌曲数降序）
		catCount := make(map[string]int)
		for _, item := range list {
			cat := item.Category
			if cat == "" {
				cat = "未知"
			}
			catCount[cat]++
		}
		var result []map[string]interface{}
		for cat, count := range catCount {
			result = append(result, map[string]interface{}{
				"name":  cat,
				"count": count,
			})
		}
		sort.Slice(result, func(i, j int) bool {
			ci, _ := result[i]["count"].(int)
			cj, _ := result[j]["count"].(int)
			return ci > cj
		})
		data, _ := json.Marshal(result)
		resp := qrMessage{
			Type:      "browseResult",
			RequestID: msg.RequestID,
			Data:      data,
		}
		qrReply(resp)

	case "categorySongs":
		// 某曲种的歌曲列表（分页）
		var songs []qrSearchItem
		for _, item := range list {
			cat := item.Category
			if cat == "" {
				cat = "未知"
			}
			if cat == msg.Category {
				songs = append(songs, qrSearchItem{
					Name:   item.Name,
					Path:   item.Path,
					Type:   item.Type,
					Singer: item.Singer,
				})
			}
		}
		total := len(songs)
		start := (page - 1) * pageSize
		if start > total {
			start = total
		}
		end := start + pageSize
		if end > total {
			end = total
		}
		paged := songs[start:end]
		resp := qrMessage{
			Type:      "browseResult",
			RequestID: msg.RequestID,
			Results:   paged,
			Total:     total,
			Page:      page,
			PageSize:  pageSize,
		}
		qrWriteJSON(resp)

	case "hotSongs":
		// 热播歌曲（分页）
		hotList := GetHotSongs(100)
		total := len(hotList)
		start := (page - 1) * pageSize
		if start > total {
			start = total
		}
		end := start + pageSize
		if end > total {
			end = total
		}
		paged := hotList[start:end]
		var results []qrSearchItem
		for _, h := range paged {
			// 从媒体列表中查找类型
			songType := "video"
			for _, m := range list {
				if m.Path == h.Path {
					songType = m.Type
					break
				}
			}
			results = append(results, qrSearchItem{
				Name:   h.Name,
				Path:   h.Path,
				Type:   songType,
				Singer: fmt.Sprintf("%d次播放", h.Count),
			})
		}
		resp := qrMessage{
			Type:      "browseResult",
			RequestID: msg.RequestID,
			Results:   results,
			Total:     total,
			Page:      page,
			PageSize:  pageSize,
		}
		qrReply(resp)
	}
}

// qrWriteJSON 线程安全地发送JSON消息
func qrWriteJSON(v interface{}) error {
	qrConnMutex.Lock()
	defer qrConnMutex.Unlock()

	if qrConn == nil {
		return fmt.Errorf("未连接")
	}
	return qrConn.WriteJSON(v)
}

// setQRConnected 线程安全地设置连接状态
func setQRConnected(connected bool) {
	qrConnMutex.Lock()
	qrConnected = connected
	qrConnMutex.Unlock()
}

// registerQRHandlers 注册QR相关的HTTP处理函数
func registerQRHandlers() {
	http.HandleFunc("/api/qr/status", qrStatusHandler)
	http.HandleFunc("/api/qr/pending-songs", qrPendingSongsHandler)
	http.HandleFunc("/api/qr/queue-update", qrQueueUpdateHandler)
	http.HandleFunc("/api/qr/image", qrImageHandler)
	http.HandleFunc("/api/qr/register-session", qrRegisterSessionHandler)
	http.HandleFunc("/api/qr/control", qrControlPollHandler)
	http.HandleFunc("/api/qr/track-state", qrTrackStateHandler)
}

// qrStatusHandler 返回QR连接状态
func qrStatusHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	connected := qrConnected
	// 内置模式下，本进程即二维码服务器，视为已连接
	if qrEnabled && qrMode == "internal" {
		connected = true
	}

	// 二维码页面访问前缀：
	// 内置=当前请求的主机（与主服务器同IP同端口），外接=外接二维码服务器地址
	qrBase := "http://" + qrServerAddr
	if qrMode == "internal" {
		scheme := "http"
		if r.TLS != nil {
			scheme = "https"
		}
		qrBase = scheme + "://" + r.Host
	}

	result := struct {
		Connected       bool   `json:"connected"`
		QRServerAddr    string `json:"qrServerAddr"`
		QrUrlBase       string `json:"qrUrlBase"`
		Mode            string `json:"mode"`
		Enabled         bool   `json:"enabled"`
		CtrlEnabled     bool   `json:"ctrlEnabled"`
	}{
		Connected:    connected,
		QRServerAddr: qrServerAddr,
		QrUrlBase:    qrBase,
		Mode:         qrMode,
		Enabled:      qrEnabled,
		CtrlEnabled:  qrCtrlEnabled,
	}

	json.NewEncoder(w).Encode(result)
}

// qrRegisterSessionHandler 前端注册会话，返回sessionId
func qrRegisterSessionHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	sessionID := generateSessionID()

	// 初始化该session的数据
	qrPendingMutex.Lock()
	qrPendingSongs[sessionID] = []PendingSong{}
	qrPendingMutex.Unlock()

	qrQueueMutex.Lock()
	qrQueueData[sessionID] = nil
	qrQueueIndex[sessionID] = 0
	qrQueueMutex.Unlock()

	// 向QR中继服务器发送注册消息（仅外接模式）
	if isQRExternal() {
		regMsg := qrMessage{
			Type:      "register",
			SessionID: sessionID,
			Password:  qrPassword,
		}
		if err := qrWriteJSON(regMsg); err != nil {
			fmt.Printf("[QR客户端] 向中继服务器注册会话失败: %v [session=%s]\n", err, sessionID)
		} else {
			fmt.Printf("[QR客户端] 新会话注册: %s (已通知中继服务器)\n", sessionID)
		}
	}

	json.NewEncoder(w).Encode(struct {
		SessionID string `json:"sessionId"`
	}{
		SessionID: sessionID,
	})
}

// qrPendingSongsHandler 返回待处理的点歌请求，返回后清空列表
func qrPendingSongsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	sessionID := r.URL.Query().Get("sessionId")
	if sessionID == "" {
		sessionID = "default"
	}

	qrPendingMutex.Lock()
	songs := qrPendingSongs[sessionID]
	qrPendingSongs[sessionID] = []PendingSong{}
	qrPendingMutex.Unlock()

	result := struct {
		Songs []PendingSong `json:"songs"`
	}{
		Songs: songs,
	}

	if result.Songs == nil {
		result.Songs = []PendingSong{}
	}

	json.NewEncoder(w).Encode(result)
}

// qrQueueUpdateHandler 接收前端推送的队列更新，转发给QR服务器
func qrQueueUpdateHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "方法不允许", http.StatusMethodNotAllowed)
		return
	}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "读取数据失败", 400)
		return
	}

	// 从请求体中提取queue、currentPlayingIndex和sessionId
	var payload struct {
		Queue               json.RawMessage `json:"queue"`
		CurrentPlayingIndex int             `json:"currentPlayingIndex"`
		SessionID           string          `json:"sessionId"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		http.Error(w, "数据格式错误", 400)
		return
	}

	sid := payload.SessionID
	if sid == "" {
		sid = "default"
	}

	// 缓存队列数据
	qrQueueMutex.Lock()
	qrQueueData[sid] = payload.Queue
	qrQueueIndex[sid] = payload.CurrentPlayingIndex
	qrQueueMutex.Unlock()

	// 转发给QR服务器（仅外接模式）
	if isQRExternal() {
		msg := qrMessage{
			Type:                "queueUpdate",
			SessionID:           sid,
			Queue:               payload.Queue,
			CurrentPlayingIndex: payload.CurrentPlayingIndex,
		}

		if err := qrWriteJSON(msg); err != nil {
			fmt.Printf("[QR客户端] 转发队列更新失败: %v\n", err)
		}
	} else if qrEnabled && qrMode == "internal" {
		// 内置模式：无需经过外接中继，直接广播队列更新到该会话的所有手机端
		qrInternalBroadcastQueue(sid, payload.Queue, payload.CurrentPlayingIndex)
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Write([]byte(`{"success":true}`))
}

// qrImageHandler 本地生成二维码图片
func qrImageHandler(w http.ResponseWriter, r *http.Request) {
	url := r.URL.Query().Get("url")
	if url == "" {
		http.Error(w, "缺少url参数", http.StatusBadRequest)
		return
	}

	png, err := qrcode.Encode(url, qrcode.Medium, 256)
	if err != nil {
		http.Error(w, "二维码生成失败", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "no-cache")
	w.Write(png)
}
