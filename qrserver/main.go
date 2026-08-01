package main

import (
	"bufio"
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

//go:embed mobile.html
var mobileHTML embed.FS

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// Session 表示一个KTV会话，包含KTV服务端连接和所有移动端连接
type Session struct {
	sessionID  string
	password   string
	ktvConn    *websocket.Conn
	mobiles    map[*websocket.Conn]bool
	mobileAuth map[*websocket.Conn]bool // 已通过密码验证的移动端
	mu         sync.Mutex
}

// KTVConn 表示一个KTV客户端连接（一个KTV实例可有多个session）
type KTVConn struct {
	conn     *websocket.Conn
	sessions map[string]*Session // 该KTV实例管理的所有session
	mu       sync.Mutex
}

// Server 是中继服务器
type Server struct {
	sessions map[string]*Session
	ktvConns map[*websocket.Conn]*KTVConn // KTV连接到KTVConn的映射
	mu       sync.RWMutex
	// 用于匹配请求ID到移动端连接
	pendingSearch   map[string]*websocket.Conn
	pendingPassword map[string]*websocket.Conn
	pendingMu       sync.Mutex
}

func NewServer() *Server {
	return &Server{
		sessions:        make(map[string]*Session),
		ktvConns:        make(map[*websocket.Conn]*KTVConn),
		pendingSearch:   make(map[string]*websocket.Conn),
		pendingPassword: make(map[string]*websocket.Conn),
	}
}

// ==================== 消息类型定义 ====================

// KTV → QR Server 消息
type KTVMessage struct {
	Type                string          `json:"type"`
	SessionID           string          `json:"sessionId,omitempty"`
	Password            string          `json:"password,omitempty"`
	RequestID           string          `json:"requestId,omitempty"`
	Songs               json.RawMessage `json:"songs,omitempty"`
	Results             json.RawMessage `json:"results,omitempty"`
	Queue               json.RawMessage `json:"queue,omitempty"`
	CurrentPlayingIndex int             `json:"currentPlayingIndex,omitempty"`
	Valid               bool            `json:"valid,omitempty"`
	Total               int             `json:"total,omitempty"`
	Page                int             `json:"page,omitempty"`
	PageSize            int             `json:"pageSize,omitempty"`
}

// QR Server → KTV 消息
type KTVOutMessage struct {
	Type      string `json:"type"`
	RequestID string `json:"requestId,omitempty"`
	SessionID string `json:"sessionId,omitempty"`
	Keyword   string `json:"keyword,omitempty"`
	Path      string `json:"path,omitempty"`
	Name      string `json:"name,omitempty"`
	SongType  string `json:"songType,omitempty"`
	Password  string `json:"password,omitempty"`
	Page      int    `json:"page,omitempty"`
	PageSize  int    `json:"pageSize,omitempty"`
}

// Mobile → QR Server 消息
type MobileMessage struct {
	Type     string `json:"type"`
	Keyword  string `json:"keyword,omitempty"`
	Path     string `json:"path,omitempty"`
	Name     string `json:"name,omitempty"`
	SongType string `json:"songType,omitempty"`
	Password string `json:"password,omitempty"`
	Page     int    `json:"page,omitempty"`
	PageSize int    `json:"pageSize,omitempty"`
}

// QR Server → Mobile 消息
type MobileOutMessage struct {
	Type                string          `json:"type"`
	Songs               json.RawMessage `json:"songs,omitempty"`
	Queue               json.RawMessage `json:"queue,omitempty"`
	CurrentPlayingIndex int             `json:"currentPlayingIndex,omitempty"`
	Valid               bool            `json:"valid,omitempty"`
	Message             string          `json:"message,omitempty"`
	Total               int             `json:"total,omitempty"`
	Page                int             `json:"page,omitempty"`
	PageSize            int             `json:"pageSize,omitempty"`
}

// ==================== 会话管理 ====================

func (s *Server) getOrCreateSession(sessionID string) *Session {
	s.mu.Lock()
	defer s.mu.Unlock()
	if sess, ok := s.sessions[sessionID]; ok {
		return sess
	}
	sess := &Session{
		sessionID:  sessionID,
		mobiles:    make(map[*websocket.Conn]bool),
		mobileAuth: make(map[*websocket.Conn]bool),
	}
	s.sessions[sessionID] = sess
	return sess
}

func (s *Server) getSession(sessionID string) *Session {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.sessions[sessionID]
}

func (s *Server) removeSession(sessionID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, sessionID)
}

// ==================== 请求ID匹配 ====================

func (s *Server) registerPendingSearch(requestID string, conn *websocket.Conn) {
	s.pendingMu.Lock()
	s.pendingSearch[requestID] = conn
	s.pendingMu.Unlock()
}

func (s *Server) consumePendingSearch(requestID string) *websocket.Conn {
	s.pendingMu.Lock()
	defer s.pendingMu.Unlock()
	conn, ok := s.pendingSearch[requestID]
	if ok {
		delete(s.pendingSearch, requestID)
	}
	return conn
}

func (s *Server) registerPendingPassword(requestID string, conn *websocket.Conn) {
	s.pendingMu.Lock()
	s.pendingPassword[requestID] = conn
	s.pendingMu.Unlock()
}

func (s *Server) consumePendingPassword(requestID string) *websocket.Conn {
	s.pendingMu.Lock()
	defer s.pendingMu.Unlock()
	conn, ok := s.pendingPassword[requestID]
	if ok {
		delete(s.pendingPassword, requestID)
	}
	return conn
}

// ==================== 生成唯一请求ID ====================

func generateRequestID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

// ==================== KTV WebSocket 处理 ====================

func (s *Server) handleKTV(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("KTV WebSocket升级失败: %v", err)
		return
	}

	ktv := &KTVConn{
		conn:     conn,
		sessions: make(map[string]*Session),
	}

	// 注册KTV连接
	s.mu.Lock()
	s.ktvConns[conn] = ktv
	s.mu.Unlock()

	defer func() {
		conn.Close()
		// 清理该KTV连接的所有session
		s.mu.Lock()
		delete(s.ktvConns, conn)
		s.mu.Unlock()
		ktv.mu.Lock()
		for sid, sess := range ktv.sessions {
			sess.mu.Lock()
			sess.ktvConn = nil
			for mobileConn := range sess.mobiles {
				mobileConn.Close()
			}
			sess.mobiles = make(map[*websocket.Conn]bool)
			sess.mobileAuth = make(map[*websocket.Conn]bool)
			sess.mu.Unlock()
			s.removeSession(sid)
		}
		ktv.mu.Unlock()
		log.Printf("KTV断开连接")
	}()

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			log.Printf("KTV读取消息失败: %v", err)
			return
		}

		var km KTVMessage
		if err := json.Unmarshal(msg, &km); err != nil {
			log.Printf("KTV消息解析失败: %v", err)
			continue
		}

		switch km.Type {
		case "register":
			// 注册新session
			sessionID := km.SessionID
			if sessionID == "" {
				sessionID = generateRequestID() // 生成一个随机ID
			}
			sess := s.getOrCreateSession(sessionID)
			sess.mu.Lock()
			sess.ktvConn = conn
			sess.password = km.Password
			sess.mu.Unlock()
			ktv.mu.Lock()
			ktv.sessions[sessionID] = sess
			ktv.mu.Unlock()
			log.Printf("KTV注册会话: session=%s, 有密码=%v", sessionID, km.Password != "")

		case "searchResult":
			// 将搜索结果转发给对应的移动端
			mobileConn := s.consumePendingSearch(km.RequestID)
			if mobileConn != nil {
				// KTV客户端可能用results或songs字段发送搜索结果
				songData := km.Results
				if songData == nil {
					songData = km.Songs
				}
				outMsg := MobileOutMessage{
					Type:     "searchResult",
					Songs:    songData,
					Total:    km.Total,
					Page:     km.Page,
					PageSize: km.PageSize,
				}
				data, _ := json.Marshal(outMsg)
				if err := mobileConn.WriteMessage(websocket.TextMessage, data); err != nil {
					log.Printf("转发搜索结果到移动端失败: %v", err)
				}
			} else {
				log.Printf("搜索结果无匹配请求: requestId=%s", km.RequestID)
			}

		case "queueUpdate":
			// 广播队列更新到该会话的所有移动端
			sid := km.SessionID
			if sid == "" {
				continue
			}
			sess := s.getSession(sid)
			if sess != nil {
				outMsg := MobileOutMessage{
					Type:                "queueUpdate",
					Queue:               km.Queue,
					CurrentPlayingIndex: km.CurrentPlayingIndex,
				}
				data, _ := json.Marshal(outMsg)
				sess.mu.Lock()
				for mobileConn := range sess.mobiles {
					if err := mobileConn.WriteMessage(websocket.TextMessage, data); err != nil {
						log.Printf("广播队列更新到移动端失败: %v", err)
					}
				}
				sess.mu.Unlock()
				log.Printf("广播队列更新: session=%s, 移动端数=%d", sid, len(sess.mobiles))
			}

		case "passwordResult":
			// 将密码验证结果转发给对应的移动端
			mobileConn := s.consumePendingPassword(km.RequestID)
			if mobileConn != nil {
				outMsg := MobileOutMessage{
					Type:  "passwordResult",
					Valid: km.Valid,
				}
				data, _ := json.Marshal(outMsg)
				if err := mobileConn.WriteMessage(websocket.TextMessage, data); err != nil {
					log.Printf("转发密码验证结果到移动端失败: %v", err)
				}
				// 如果验证通过，标记该移动端为已验证
				if km.Valid {
					// 找到该移动端所属的session
					s.mu.RLock()
					for _, sess := range s.sessions {
						sess.mu.Lock()
						if sess.mobileAuth[mobileConn] == false {
							// 检查该连接是否属于此session
							if _, ok := sess.mobiles[mobileConn]; ok {
								sess.mobileAuth[mobileConn] = true
							}
						}
						sess.mu.Unlock()
					}
					s.mu.RUnlock()
				}
			}

		default:
			log.Printf("KTV未知消息类型: %s", km.Type)
		}
	}
}

// ==================== Mobile WebSocket 处理 ====================

func (s *Server) handleMobile(w http.ResponseWriter, r *http.Request) {
	sessionID := r.URL.Query().Get("session")
	if sessionID == "" {
		http.Error(w, "缺少session参数", http.StatusBadRequest)
		return
	}

	sess := s.getSession(sessionID)
	if sess == nil {
		http.Error(w, "会话不存在", http.StatusNotFound)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("移动端WebSocket升级失败: %v", err)
		return
	}

	sess.mu.Lock()
	sess.mobiles[conn] = true
	sess.mobileAuth[conn] = false // 初始未验证
	needPassword := sess.password != ""
	sess.mu.Unlock()

	log.Printf("移动端连接: session=%s", sessionID)

	defer func() {
		conn.Close()
		sess.mu.Lock()
		delete(sess.mobiles, conn)
		delete(sess.mobileAuth, conn)
		sess.mu.Unlock()
		log.Printf("移动端断开: session=%s", sessionID)
	}()

	// 如果需要密码，发送needPassword
	if needPassword {
		outMsg := MobileOutMessage{Type: "needPassword"}
		data, _ := json.Marshal(outMsg)
		conn.WriteMessage(websocket.TextMessage, data)
	}

	// 通知KTV客户端发送当前队列
	sess.mu.Lock()
	ktvConn := sess.ktvConn
	sess.mu.Unlock()
	if ktvConn != nil {
		reqMsg := KTVOutMessage{Type: "requestQueue"}
		data, _ := json.Marshal(reqMsg)
		if err := ktvConn.WriteMessage(websocket.TextMessage, data); err != nil {
			log.Printf("请求KTV队列失败: %v", err)
		}
	}

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			log.Printf("移动端读取消息失败: %v", err)
			return
		}

		var mm MobileMessage
		if err := json.Unmarshal(msg, &mm); err != nil {
			log.Printf("移动端消息解析失败: %v", err)
			continue
		}

		// 检查是否需要密码验证（verifyPassword和searchResult除外）
		sess.mu.Lock()
		isAuthed := sess.mobileAuth[conn]
		hasPassword := sess.password != ""
		ktvConn := sess.ktvConn
		sess.mu.Unlock()

		switch mm.Type {
		case "verifyPassword":
			if ktvConn == nil {
				sendMobileError(conn, "KTV服务未连接")
				continue
			}
			requestID := generateRequestID()
			s.registerPendingPassword(requestID, conn)
			outMsg := KTVOutMessage{
				Type:      "verifyPassword",
				RequestID: requestID,
				Password:  mm.Password,
			}
			data, _ := json.Marshal(outMsg)
			if err := ktvConn.WriteMessage(websocket.TextMessage, data); err != nil {
				log.Printf("转发密码验证到KTV失败: %v", err)
				s.consumePendingPassword(requestID)
				sendMobileError(conn, "KTV服务连接异常")
			}

		case "search":
			if hasPassword && !isAuthed {
				sendMobileError(conn, "请先输入密码")
				continue
			}
			if ktvConn == nil {
				sendMobileError(conn, "KTV服务未连接")
				continue
			}
			requestID := generateRequestID()
			s.registerPendingSearch(requestID, conn)
			outMsg := KTVOutMessage{
				Type:      "search",
				RequestID: requestID,
				Keyword:   mm.Keyword,
				Page:      mm.Page,
				PageSize:  mm.PageSize,
			}
			data, _ := json.Marshal(outMsg)
			if err := ktvConn.WriteMessage(websocket.TextMessage, data); err != nil {
				log.Printf("转发搜索请求到KTV失败: %v", err)
				s.consumePendingSearch(requestID)
				sendMobileError(conn, "KTV服务连接异常")
			}

		case "addSong":
			if hasPassword && !isAuthed {
				sendMobileError(conn, "请先输入密码")
				continue
			}
			if ktvConn == nil {
				sendMobileError(conn, "KTV服务未连接")
				continue
			}
			outMsg := KTVOutMessage{
				Type:      "addSong",
				SessionID: sessionID,
				Path:      mm.Path,
				Name:      mm.Name,
				SongType:  mm.SongType,
			}
			data, _ := json.Marshal(outMsg)
			if err := ktvConn.WriteMessage(websocket.TextMessage, data); err != nil {
				log.Printf("转发点歌请求到KTV失败: %v", err)
				sendMobileError(conn, "KTV服务连接异常")
			}

		default:
			log.Printf("移动端未知消息类型: %s", mm.Type)
		}
	}
}

func sendMobileError(conn *websocket.Conn, message string) {
	outMsg := MobileOutMessage{Type: "error", Message: message}
	data, _ := json.Marshal(outMsg)
	conn.WriteMessage(websocket.TextMessage, data)
}

// ==================== 移动端页面 ====================

func (s *Server) handleMobilePage(w http.ResponseWriter, r *http.Request) {
	sessionID := r.URL.Path[len("/m/"):]
	if sessionID == "" {
		http.Error(w, "缺少会话ID", http.StatusBadRequest)
		return
	}

	tmpl, err := template.ParseFS(mobileHTML, "mobile.html")
	if err != nil {
		http.Error(w, "页面加载失败", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	tmpl.Execute(w, map[string]string{
		"SessionID": sessionID,
	})
}

// ==================== 主函数 ====================

const qrConfigFile = "qr_config.json"

type QRConfig struct {
	Port int `json:"port"`
}

func loadQRConfig() QRConfig {
	cfg := QRConfig{Port: 8352}
	data, err := os.ReadFile(qrConfigFile)
	if err != nil {
		return cfg
	}
	json.Unmarshal(data, &cfg)
	if cfg.Port <= 0 {
		cfg.Port = 8352
	}
	return cfg
}

func saveQRConfig(cfg QRConfig) {
	data, _ := json.MarshalIndent(cfg, "", "  ")
	os.WriteFile(qrConfigFile, data, 0644)
}

func main() {
	portFlag := flag.Int("port", 0, "服务端口(0则使用配置文件)")
	flag.Parse()

	cfg := loadQRConfig()

	// 如果命令行指定了端口，直接使用
	if *portFlag > 0 {
		cfg.Port = *portFlag
		saveQRConfig(cfg)
		startServer(cfg.Port)
		return
	}

	// 交互式菜单
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Println()
		fmt.Println("╔══════════════════════════════════════╗")
		fmt.Println("║       QR 中继服务器 (qrserver)       ║")
		fmt.Println("╚══════════════════════════════════════╝")
		fmt.Printf("  当前服务器端口设置：%d\n", cfg.Port)
		fmt.Println()
		fmt.Println("  1. 启动服务器")
		fmt.Println("  2. 修改端口")
		fmt.Println("  3. 退出")
		fmt.Println()
		fmt.Print("请选择 [1-3]: ")

		input, _ := reader.ReadString('\n')
		choice := strings.TrimSpace(input)

		switch choice {
		case "1":
			startServer(cfg.Port)
			return
		case "2":
			fmt.Print("请输入新端口 (1-65535): ")
			portInput, _ := reader.ReadString('\n')
			portStr := strings.TrimSpace(portInput)
			newPort, err := strconv.Atoi(portStr)
			if err != nil || newPort < 1 || newPort > 65535 {
				fmt.Println("  端口无效，请输入1-65535之间的数字")
				continue
			}
			cfg.Port = newPort
			saveQRConfig(cfg)
			fmt.Printf("  端口已修改为 %d 并保存\n", newPort)
		case "3":
			fmt.Println("  再见！")
			return
		default:
			fmt.Println("  无效选择，请输入 1-3")
		}
	}
}

func startServer(port int) {
	server := NewServer()

	http.HandleFunc("/ws/ktv", server.handleKTV)
	http.HandleFunc("/ws/mobile", server.handleMobile)
	http.HandleFunc("/m/", server.handleMobilePage)

	addr := fmt.Sprintf(":%d", port)
	fmt.Printf("\n  QR中继服务器启动，监听端口 %d\n", port)
	fmt.Printf("  KTV连接地址: ws://localhost:%d/ws/ktv\n", port)
	fmt.Printf("  移动端连接地址: ws://localhost:%d/ws/mobile\n", port)
	fmt.Println()
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("服务器启动失败: %v", err)
	}
}
