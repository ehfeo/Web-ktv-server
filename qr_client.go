package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/skip2/go-qrcode"
)

// QR客户端配置
var (
	qrServerAddr string // QR中继服务器地址，如 "123.45.67.89:8352"
	qrPassword   string // 手机端验证密码
	qrEnabled    bool   // 是否启用QR功能
)

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
)

func init() {
	qrPendingSongs = make(map[string][]PendingSong)
	qrQueueData = make(map[string]json.RawMessage)
	qrQueueIndex = make(map[string]int)
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

// initQRConfig 初始化QR客户端配置，从配置文件加载
func initQRConfig() {
	data, err := ioutil.ReadFile(configFile)
	if err != nil {
		return
	}

	var cfg struct {
		QRServerAddr string `json:"qrServerAddr"`
		QRPassword   string `json:"qrPassword"`
		QREnabled    bool   `json:"qrEnabled"`
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return
	}

	qrServerAddr = cfg.QRServerAddr
	qrPassword = cfg.QRPassword
	qrEnabled = cfg.QREnabled

	if qrEnabled {
		fmt.Printf("[QR客户端] 配置加载: server=%s\n", qrServerAddr)
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

	out, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return
	}
	ioutil.WriteFile(configFile, out, 0644)
}

// startQRClient 启动QR WebSocket客户端（应在goroutine中调用）
func startQRClient() {
	if !qrEnabled || qrServerAddr == "" {
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
		handleQRRequestQueue()
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

	if err := qrWriteJSON(resp); err != nil {
		fmt.Printf("[QR客户端] 发送搜索结果失败: %v\n", err)
	}
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

	if err := qrWriteJSON(resp); err != nil {
		fmt.Printf("[QR客户端] 发送点歌确认失败: %v\n", err)
	}
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

	if err := qrWriteJSON(resp); err != nil {
		fmt.Printf("[QR客户端] 发送密码验证结果失败: %v\n", err)
	}
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
func handleQRRequestQueue() {
	// 不再全局推送，由前端按sessionId推送
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
}

// qrStatusHandler 返回QR连接状态
func qrStatusHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	qrConnMutex.Lock()
	connected := qrConnected
	qrConnMutex.Unlock()

	result := struct {
		Connected    bool   `json:"connected"`
		QRServerAddr string `json:"qrServerAddr"`
		Enabled      bool   `json:"enabled"`
	}{
		Connected:    connected,
		QRServerAddr: qrServerAddr,
		Enabled:      qrEnabled,
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

	// 向QR中继服务器发送注册消息（仅在QR启用时）
	if qrEnabled && qrServerAddr != "" {
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

	// 转发给QR服务器（仅在QR启用时）
	if qrEnabled && qrServerAddr != "" {
		msg := qrMessage{
			Type:                "queueUpdate",
			SessionID:           sid,
			Queue:               payload.Queue,
			CurrentPlayingIndex: payload.CurrentPlayingIndex,
		}

		if err := qrWriteJSON(msg); err != nil {
			fmt.Printf("[QR客户端] 转发队列更新失败: %v\n", err)
		}
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
