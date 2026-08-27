package main

import (
	"embed"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"

	"github.com/gorilla/websocket"
)

//go:embed qr_mobile.html
var qrMobilePage embed.FS

// 内置二维码服务器：手机端直接连接到主程序（与主服务器同IP同端口），
// 不再经过外接中继，KTV侧处理逻辑复用 qr_client.go 中的处理器。
// 处理器回复经由 qrReplyHook -> qrInternalDeliver 按 requestId 路由到对应手机。

var qrInternalUpgrader = websocket.Upgrader{
	CheckOrigin:      func(r *http.Request) bool { return true },
	ReadBufferSize:   1024,
	WriteBufferSize:  1024,
}

// qrInternalSession 一个会话下的移动端连接集合
type qrInternalSession struct {
	mu      sync.Mutex
	mobiles map[*websocket.Conn]bool
	authed  map[*websocket.Conn]bool
}

var (
	qrInternalMu       sync.Mutex
	qrInternalSessions = make(map[string]*qrInternalSession)
	qrInternalPending  = make(map[string]*websocket.Conn) // requestId -> 移动端连接
)

// qrInternalGetSession 获取或创建会话
func qrInternalGetSession(sid string) *qrInternalSession {
	qrInternalMu.Lock()
	defer qrInternalMu.Unlock()
	if s, ok := qrInternalSessions[sid]; ok {
		return s
	}
	s := &qrInternalSession{
		mobiles: make(map[*websocket.Conn]bool),
		authed:  make(map[*websocket.Conn]bool),
	}
	qrInternalSessions[sid] = s
	return s
}

// qrSetupMode 按模式配置扫码点歌：内置模式注册移动端路由，外接模式保持原逻辑
func qrSetupMode() {
	if qrMode == "internal" {
		registerQRInternalRoutes()
	} else {
		qrReplyHook = nil // 默认外接：回复走外接中继连接
	}
}

// registerQRInternalRoutes 注册内置模式专用的移动端路由
func registerQRInternalRoutes() {
	http.HandleFunc("/ws/mobile", qrInternalMobileWSHandler)
	http.HandleFunc("/m/", qrInternalMobilePageHandler)
	// 将KTV侧处理器回复分发给对应手机
	qrReplyHook = qrInternalDeliver
}

// qrInternalDeliver 将处理器回复按 requestId 路由到对应手机
func qrInternalDeliver(resp qrMessage) {
	if resp.RequestID == "" {
		return // 无requestId（如多余的addSong确认），忽略
	}
	qrInternalMu.Lock()
	conn := qrInternalPending[resp.RequestID]
	delete(qrInternalPending, resp.RequestID)
	qrInternalMu.Unlock()
	if conn == nil {
		return
	}

	var out interface{}
	switch resp.Type {
	case "searchResult":
		out = qrInternalOut{Type: "searchResult", Songs: resp.Results, Total: resp.Total, Page: resp.Page, PageSize: resp.PageSize}
	case "browseResult":
		if resp.Data != nil {
			out = qrInternalOut{Type: "browseResult", Data: resp.Data}
		} else {
			out = qrInternalOut{Type: "browseResult", Songs: resp.Results, Total: resp.Total, Page: resp.Page, PageSize: resp.PageSize}
		}
	default:
		out = resp
	}
	data, _ := json.Marshal(out)
	conn.WriteMessage(websocket.TextMessage, data)
}

// qrInternalOut 内置模式发送给手机端的消息（字段与手机端期望一致）
type qrInternalOut struct {
	Type                string          `json:"type"`
	Songs               []qrSearchItem  `json:"songs,omitempty"`
	Total               int             `json:"total,omitempty"`
	Page                int             `json:"page,omitempty"`
	PageSize            int             `json:"pageSize,omitempty"`
	Data                json.RawMessage `json:"data,omitempty"`
	Queue               json.RawMessage `json:"queue,omitempty"`
	CurrentPlayingIndex int             `json:"currentPlayingIndex,omitempty"`
	Valid               bool            `json:"valid,omitempty"`
	Message             string          `json:"message,omitempty"`
}

// qrInternalMobileWSHandler 处理手机端WebSocket连接（内置模式）
func qrInternalMobileWSHandler(w http.ResponseWriter, r *http.Request) {
	sid := r.URL.Query().Get("session")
	if sid == "" {
		http.Error(w, "缺少session参数", http.StatusBadRequest)
		return
	}

	sess := qrInternalGetSession(sid)

	conn, err := qrInternalUpgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	sess.mu.Lock()
	sess.mobiles[conn] = true
	sess.authed[conn] = false
	sess.mu.Unlock()

	defer func() {
		conn.Close()
		sess.mu.Lock()
		delete(sess.mobiles, conn)
		delete(sess.authed, conn)
		sess.mu.Unlock()
	}()

	// 需要密码时提示
	if qrPassword != "" {
		conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"needPassword"}`))
	}

	// 立即推送当前缓存的队列
	qrInternalSendQueue(conn, sid)

	for {
		_, data, err := conn.ReadMessage()
		if err != nil {
			return
		}

		var msg qrMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			continue
		}
		// 手机端消息未携带sessionId，外接模式由中继注入；内置模式此处从ws连接注入
		msg.SessionID = sid
		switch msg.Type {
		case "search", "browse", "addSong", "verifyPassword":
		default:
			continue
		}

		sess.mu.Lock()
		isAuthed := sess.authed[conn]
		sess.mu.Unlock()

		// 分配 requestId 用于回包路由（手机端搜索/验证时不携带）
		msg.RequestID = generateRequestID()
		qrInternalMu.Lock()
		qrInternalPending[msg.RequestID] = conn
		qrInternalMu.Unlock()

		// 已设置密码时，除验证外的操作需先通过验证
		if qrPassword != "" && !isAuthed && msg.Type != "verifyPassword" {
			qrInternalDeliver(qrMessage{Type: "error", RequestID: msg.RequestID, Message: "请先输入密码"})
			continue
		}

		switch msg.Type {
		case "verifyPassword":
			qrInternalVerifyPassword(conn, sess, msg)
		case "search":
			handleQRSearch(msg)
		case "browse":
			handleQRBrowse(msg)
		case "addSong":
			handleQRAddSong(msg)
		}
	}
}

// qrInternalVerifyPassword 内置模式下的密码验证（手机端期望 valid 字段）
func qrInternalVerifyPassword(conn *websocket.Conn, sess *qrInternalSession, msg qrMessage) {
	ok := msg.Password == qrPassword
	if ok {
		sess.mu.Lock()
		sess.authed[conn] = true
		sess.mu.Unlock()
	}
	// 直接发往当前连接（手机端未携带requestId，也无须跨连接路由）
	out := qrInternalOut{Type: "passwordResult", Valid: ok}
	if !ok {
		out.Message = "密码错误"
	}
	data, _ := json.Marshal(out)
	conn.WriteMessage(websocket.TextMessage, data)
}

// qrInternalSendQueue 向手机端推送当前队列
func qrInternalSendQueue(conn *websocket.Conn, sid string) {
	qrQueueMutex.Lock()
	q := qrQueueData[sid]
	idx := qrQueueIndex[sid]
	qrQueueMutex.Unlock()
	if q == nil {
		q = json.RawMessage("[]")
	}
	out := qrInternalOut{Type: "queueUpdate", Queue: q, CurrentPlayingIndex: idx}
	data, _ := json.Marshal(out)
	conn.WriteMessage(websocket.TextMessage, data)
}

// qrInternalBroadcastQueue 内置模式下将队列更新广播给该会话的所有手机端
func qrInternalBroadcastQueue(sid string, queue json.RawMessage, index int) {
	qrInternalMu.Lock()
	sess := qrInternalSessions[sid]
	qrInternalMu.Unlock()
	if sess == nil {
		return
	}
	out := qrInternalOut{Type: "queueUpdate", Queue: queue, CurrentPlayingIndex: index}
	data, _ := json.Marshal(out)
	sess.mu.Lock()
	defer sess.mu.Unlock()
	for conn := range sess.mobiles {
		conn.WriteMessage(websocket.TextMessage, data)
	}
}

// qrInternalMobilePageHandler 内置模式下的移动端点歌页面
// 页面内容已通过 go:embed 内置进主程序(qr_mobile.html)，保证单个exe即可运行；
// 若内置缺失则回退读取开发目录的 qrserver/mobile.html。
func qrInternalMobilePageHandler(w http.ResponseWriter, r *http.Request) {
	sid := strings.TrimPrefix(r.URL.Path, "/m/")
	if sid == "" {
		http.Error(w, "缺少会话ID", http.StatusBadRequest)
		return
	}

	var b []byte
	var err error
	b, err = qrMobilePage.ReadFile("qr_mobile.html")
	if err != nil {
		// 回退：从开发目录读取（与源码目录联跑时）
		b, err = ioutil.ReadFile("qrserver/mobile.html")
		if err != nil {
			http.Error(w, "内置扫码页面未找到: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}
	html := strings.Replace(string(b), "{{.SessionID}}", sid, 1)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(html))
}