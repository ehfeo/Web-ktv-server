package main

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// =========================================================
// 上传内容审核：上传的歌曲/资源未经审核能搜索到，但不能点播。
// 采用"已审核名单持久化"方案：不移动文件，审核通过即把文件名加入本地名单。
// 密码 + 已审核名单持久化到 ktv_review.json。
// =========================================================

const reviewConfigFile = "ktv_review.json"
const reviewCookieName = "ktv_review_token"
const defaultReviewPassword = "admin123"

// ReviewConfig 审核配置（密码哈希 + 已审核文件名单）
type ReviewConfig struct {
	PasswordHash string   `json:"passwordHash"`
	Approved     []string `json:"approved"` // 已审核通过的 uploads 相对路径（正斜杠）
}

type reviewManager struct {
	mu   sync.RWMutex
	cfg  ReviewConfig
	sess map[string]time.Time // token -> 过期时间
}

var reviewMgr = &reviewManager{sess: make(map[string]time.Time)}

// ensureReviewLoaded 加载审核配置（首次调用时初始化默认密码）
func ensureReviewLoaded() {
	reviewMgr.mu.Lock()
	defer reviewMgr.mu.Unlock()
	if len(reviewMgr.cfg.Approved) == 0 && reviewMgr.cfg.PasswordHash == "" {
		if data, err := ioutil.ReadFile(reviewConfigFile); err == nil {
			json.Unmarshal(data, &reviewMgr.cfg)
		}
	}
	// 无密码时设置默认密码
	if reviewMgr.cfg.PasswordHash == "" {
		reviewMgr.cfg.PasswordHash = hashPassword(defaultReviewPassword)
		saveReviewConfigLocked()
		fmt.Println("[审核] 已初始化默认审核密码: admin123（可通过审核页面修改）")
	}
	// 兼容旧版"名单式审核"：迁移已审核文件到 Approved 目录（仅执行一次）
	if !reviewMigrated {
		reviewMigrated = true
		migrateApprovedFilesLocked()
	}
}

func saveReviewConfigLocked() {
	data, _ := json.MarshalIndent(&reviewMgr.cfg, "", "  ")
	if err := ioutil.WriteFile(reviewConfigFile, data, 0644); err != nil {
		fmt.Printf("[审核] 保存配置失败: %v\n", err)
	}
}

func hashPassword(pw string) string {
	h := sha256.Sum256([]byte(pw))
	return hex.EncodeToString(h[:])
}

func newSessionToken() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// ---- 登录防暴力破解 ----
// 按客户端 IP 限制：指定时间窗内失败次数达到上限即锁定该 IP，锁定期间拒绝登录。
const (
	loginMaxFails       = 5               // 时间窗内允许的最大失败次数
	loginWindow         = 15 * time.Minute // 失败计数时间窗
	loginLockDuration   = 15 * time.Minute // 锁定持续时间
	loginLockedIPMsg    = "登录尝试过于频繁，请15分钟后再试。"
)

type loginGuardState struct {
	mu          sync.Mutex
	fails       map[string][]time.Time // IP -> 失败时间戳（窗口内）
	lockedUntil map[string]time.Time   // IP -> 锁定到期时间
}

var loginGuard = &loginGuardState{
	fails:       make(map[string][]time.Time),
	lockedUntil: make(map[string]time.Time),
}

// reviewClientIP 取客户端真实 IP（优先取 X-Forwarded-For，回退 RemoteAddr）
func reviewClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if i := strings.IndexByte(xff, ','); i >= 0 {
			xff = xff[:i]
		}
		xff = strings.TrimSpace(xff)
		if xff != "" {
			return xff
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// reviewLoginLocked 该 IP 当前是否处于锁定状态
func reviewLoginLocked(ip string) bool {
	loginGuard.mu.Lock()
	defer loginGuard.mu.Unlock()
	if until, ok := loginGuard.lockedUntil[ip]; ok {
		if time.Now().Before(until) {
			return true
		}
		delete(loginGuard.lockedUntil, ip) // 锁定已到期，自动解除
	}
	return false
}

// reviewRecordFail 记录一次失败；返回是否已触发锁定
func reviewRecordFail(ip string) bool {
	loginGuard.mu.Lock()
	defer loginGuard.mu.Unlock()
	now := time.Now()
	// 清理窗口外旧记录
	list := loginGuard.fails[ip]
	fresh := list[:0]
	for _, t := range list {
		if now.Sub(t) <= loginWindow {
			fresh = append(fresh, t)
		}
	}
	fresh = append(fresh, now)
	loginGuard.fails[ip] = fresh
	if len(fresh) >= loginMaxFails {
		loginGuard.lockedUntil[ip] = now.Add(loginLockDuration)
		return true
	}
	return false
}

// reviewLoginSucceed 登录成功后清除该 IP 的失败记录与锁定
func reviewLoginSucceed(ip string) {
	loginGuard.mu.Lock()
	defer loginGuard.mu.Unlock()
	delete(loginGuard.fails, ip)
	delete(loginGuard.lockedUntil, ip)
}

// reviewAuthed 验证请求是否已通过审核登录
func reviewAuthed(r *http.Request) bool {
	c, err := r.Cookie(reviewCookieName)
	if err != nil {
		return false
	}
	reviewMgr.mu.RLock()
	defer reviewMgr.mu.RUnlock()
	exp, ok := reviewMgr.sess[c.Value]
	return ok && time.Now().Before(exp)
}

// uploadPathToRel 将 uploads 目录下的绝对路径转为相对 uploads 的正斜杠相对路径；不属于 uploads 返回 ""
func uploadPathToRel(absPath string) string {
	if uploadDirPath == "" {
		return ""
	}
	cleanAbs, err := filepath.Rel(uploadDirPath, absPath)
	if err != nil {
		return ""
	}
	rel := filepath.ToSlash(cleanAbs)
	// 必须严格位于 uploads 内，且不是 uploads 自身
	if rel == "." || strings.HasPrefix(rel, "../") {
		return ""
	}
	return rel
}

var reviewMigrated bool // 旧版"名单式审核"是否已完成迁移

// reviewFileAllowed 判断该文件是否允许点播。
// 规则：uploads 目录下的文件都是待审核，一律拒绝；其余（正常曲库、Approved 已审核目录）放行。
func reviewFileAllowed(absPath string) bool {
	ensureReviewLoaded()
	if uploadPathToRel(absPath) == "" {
		return true
	}
	return false
}

// migrateApprovedFilesLocked 兼容旧版"名单式审核"：把名单中仍留在 uploads 的文件移动
// 到 Approved 目录，然后清空名单。调用前必须已持有 reviewMgr.mu.Lock。
func migrateApprovedFilesLocked() {
	if len(reviewMgr.cfg.Approved) == 0 {
		return
	}
	if approvedDirPath == "" {
		// 极端情况：Approved 目录未初始化，保留名单不动，避免丢失可播放状态
		return
	}
	var remaining []string
	moved := 0
	for _, rel := range reviewMgr.cfg.Approved {
		src := filepath.Join(uploadDirPath, filepath.FromSlash(rel))
		// 校验确实在 uploads 下，且文件存在
		if uploadPathToRel(src) == "" {
			continue
		}
		if _, err := os.Stat(src); err != nil {
			continue
		}
		dst := filepath.Join(approvedDirPath, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
			remaining = append(remaining, rel)
			continue
		}
		if err := os.Rename(src, dst); err != nil {
			remaining = append(remaining, rel)
			continue
		}
		moved++
	}
	if moved > 0 {
		reviewMgr.cfg.Approved = remaining
		saveReviewConfigLocked()
		// 移动文件后刷新曲库缓存，让 Approved 立即可搜索、可播放
		rescanUploadDir()
	}
}

// ReviewPageHandler /review 审核页面
func ReviewPageHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(`<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="UTF-8">
<title>上传内容审核</title>
<link rel="icon" href="/favicon.ico">
<meta name="viewport" content="width=device-width,initial-scale=1">
<style>
*{margin:0;padding:0;box-sizing:border-box;font-family:Microsoft YaHei,sans-serif}
body{background:#1a252f;color:#ecf0f1;padding:20px;min-height:100vh}
h2{color:#337ab9;margin-bottom:16px;letter-spacing:1px}
.card{background:#243342;border:1px solid #2f4254;border-radius:8px;padding:16px;margin-bottom:16px}
.login-row{display:flex;gap:10px;align-items:center;flex-wrap:wrap}
.login-row input{padding:9px 12px;background:#2c3e50;border:1px solid #34495e;border-radius:6px;color:#ecf0f1;font-size:15px;outline:none;flex:1;min-width:200px}
button{padding:9px 20px;border:none;border-radius:6px;color:#fff;font-size:14px;font-weight:bold;cursor:pointer;background:#428bca;transition:background .15s}
button:hover{background:#337ab9}
button.primary{background:#5cb85c}
button.primary:hover{background:#449d44}
button.danger{background:#d9534f}
button.danger:hover{background:#c9302c}
button[disabled]{opacity:.5;cursor:not-allowed}
.toolbar{display:flex;gap:10px;align-items:center;margin-bottom:12px;flex-wrap:wrap}
.file-row{display:flex;align-items:center;gap:12px;padding:10px 8px;border-bottom:1px solid #1f2f3d;font-size:14px}
.file-row input[type=checkbox]{width:18px;height:18px;cursor:pointer}
.file-row .name{flex:1;word-break:break-all}
.file-row .approved{color:#5cb85c;font-weight:bold}
.file-row .pending{color:#f0ad4e;font-weight:bold}
.badge{padding:2px 8px;border-radius:4px;font-size:12px}
.file-row .row-actions{display:flex;gap:6px}
.file-row .row-actions button{padding:5px 14px;font-size:13px;white-space:nowrap}
.tip{color:#95a5a6;font-size:13px;margin-top:8px}
.toast{position:fixed;top:20px;left:50%;transform:translateX(-50%);background:#428bca;color:#fff;padding:10px 26px;border-radius:6px;font-size:14px;font-weight:bold;opacity:0;transition:opacity .3s;pointer-events:none;z-index:100;box-shadow:0 2px 8px rgba(0,0,0,.2)}
.toast.show{opacity:1}
#pwdForm{display:none}
#mainPanel{display:none}
</style>
</head>
<body>
<h2>上传内容审核</h2>
<div class="toast" id="toast"></div>

<div class="card" id="pwdForm">
  <div style="margin-bottom:10px;font-weight:bold">输入审核密码：</div>
  <div class="login-row">
    <input type="password" id="pwd" placeholder="审核密码" />
    <button onclick="doLogin()">进入审核</button>
  </div>
  <div class="tip">密码可在审核页面内修改。初始密码：admin123</div>
</div>

<div class="card" id="mainPanel">
  <div class="toolbar">
    <button class="danger" onclick="logout()">退出</button>
    <span style="flex:1"></span>
    <input type="password" id="newPwd" placeholder="新密码" style="padding:8px 10px;background:#2c3e50;border:1px solid #34495e;border-radius:6px;color:#ecf0f1;outline:none" />
    <button onclick="changePwd()">修改密码</button>
  </div>
  <div class="tip" style="margin-bottom:8px">操作说明：点击文件右侧“通过”后，文件将移入 Approved 文件夹并纳入正常曲库（可搜索可播放）；点击“删除”将删除该文件。</div>
  <div id="fileList" style="max-height:60vh;overflow-y:auto"></div>
</div>

<script>
function toast(msg){var t=document.getElementById('toast');t.textContent=msg;t.className='toast show';setTimeout(function(){t.className='toast';},2000);}
function api(path,method,body){
  var opts={method:method||'GET',headers:{'Content-Type':'application/json'}};
  if(body) opts.body=JSON.stringify(body);
  return fetch(path,opts).then(function(r){return r.json().then(function(j){return {ok:r.ok,data:j};}).catch(function(){return {ok:r.ok,data:null};});});
}
function doLogin(){
  var pw=document.getElementById('pwd').value;
  api('/api/review/login','POST',{password:pw}).then(function(res){
    if(res.ok && res.data && res.data.success){toast('登录成功');loadPanel();}
    else {
      var m=(res.data&&res.data.message)?res.data.message:'密码错误';
      toast(m);
    }
  });
}
function logout(){
  api('/api/review/logout','POST').then(function(){location.reload();});
}
function changePwd(){
  var np=document.getElementById('newPwd').value;
  if(!np || np.length<3){toast('新密码至少3位');return;}
  api('/api/review/password','POST',{newPassword:np}).then(function(res){
    if(res.ok && res.data && res.data.success){toast('密码已修改');document.getElementById('newPwd').value='';}
    else {toast('修改失败');}
  });
}
function loadPanel(){
  document.getElementById('pwdForm').style.display='none';
  document.getElementById('mainPanel').style.display='block';
  api('/api/review/files').then(function(res){
    var list=res.data&&res.data.files?res.data.files:[];
    var html='';
    for(var i=0;i<list.length;i++){
      var f=list[i];
      var relEsc=f.rel.replace(/'/g,"\\'").replace(/\\/g,'/');
      html+='<div class="file-row">';
      html+='<span class="pending badge">待审核</span>';
      html+='<span class="name">'+f.rel+'</span>';
      html+='<span class="row-actions">';
      html+='<button class="primary" onclick="approveFile(\''+relEsc+'\',this)">通过</button>';
      html+='<button class="danger" onclick="delFile(\''+relEsc+'\',this)">删除</button>';
      html+='</span>';
      html+='</div>';
    }
    if(!list.length) html='<div class="tip" style="padding:10px">上传目录为空，暂无需审核</div>';
    document.getElementById('fileList').innerHTML=html;
  });
}
function approveFile(rel,btn){
  if(!confirm('确认通过该文件的审核并移入 Approved 目录？\n'+rel)) return;
  btn.disabled=true;
  api('/api/review/approve','POST',{paths:[rel]}).then(function(r){
    if(r.ok && r.data && r.data.success){toast('已通过：'+rel);loadPanel();}
    else {toast('通过失败');btn.disabled=false;}
  });
}
function delFile(rel,btn){
  if(!confirm('确认删除该待审核文件？此操作不可恢复！\n'+rel)) return;
  btn.disabled=true;
  api('/api/review/delete','POST',{paths:[rel]}).then(function(r){
    if(r.ok && r.data && r.data.success){toast('已删除：'+rel);loadPanel();}
    else {toast('删除失败');btn.disabled=false;}
  });
}
function init(){
  api('/api/review/status').then(function(res){
    if(res.data && res.data.authed){loadPanel();}
    else {document.getElementById('pwdForm').style.display='block';}
  });
}
init();
</script>
</body>
</html>
`))
}

// ReviewLoginHandler POST /api/review/login
func ReviewLoginHandler(w http.ResponseWriter, r *http.Request) {
	ensureReviewLoaded()
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", 405)
		return
	}
	var req struct {
		Password string `json:"password"`
	}
	body, _ := ioutil.ReadAll(r.Body)
	json.Unmarshal(body, &req)

	// 防暴力破解：锁定中的 IP 直接拒绝
	ip := reviewClientIP(r)
	if reviewLoginLocked(ip) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`{"success":false,"message":"` + loginLockedIPMsg + `"}`))
		return
	}

	reviewMgr.mu.RLock()
	stored := reviewMgr.cfg.PasswordHash
	reviewMgr.mu.RUnlock()

	if stored == "" || hashPassword(req.Password) != stored {
		// 记录失败，达到阈值即锁定该 IP
		locked := reviewRecordFail(ip)
		msg := "密码错误"
		if locked {
			msg = "密码错误次数过多，已锁定15分钟。"
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"success":false,"message":"` + msg + `"}`))
		return
	}

	// 登录成功：清除该 IP 的失败记录与锁定
	reviewLoginSucceed(ip)

	token := newSessionToken()
	reviewMgr.mu.Lock()
	reviewMgr.sess[token] = time.Now().Add(30 * time.Minute)
	reviewMgr.mu.Unlock()
	http.SetCookie(w, &http.Cookie{Name: reviewCookieName, Value: token, Path: "/", HttpOnly: true, MaxAge: 1800})

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"success":true}`))
}

// ReviewLogoutHandler POST /api/review/logout
func ReviewLogoutHandler(w http.ResponseWriter, r *http.Request) {
	c, err := r.Cookie(reviewCookieName)
	if err == nil {
		reviewMgr.mu.Lock()
		delete(reviewMgr.sess, c.Value)
		reviewMgr.mu.Unlock()
		http.SetCookie(w, &http.Cookie{Name: reviewCookieName, Value: "", Path: "/", HttpOnly: true, MaxAge: -1})
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"success":true}`))
}

// ReviewStatusHandler GET /api/review/status 返回登录状态
func ReviewStatusHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	authed := reviewAuthed(r)
	w.Write([]byte(`{"authed":`))
	if authed {
		w.Write([]byte(`true}`))
	} else {
		w.Write([]byte(`false}`))
	}
}

// ReviewFilesHandler GET /api/review/files 列出 uploads 目录下所有待审核文件
func ReviewFilesHandler(w http.ResponseWriter, r *http.Request) {
	if !reviewAuthed(r) {
		http.Error(w, "未授权", 401)
		return
	}
	ensureReviewLoaded()
	var result []map[string]interface{}
	if uploadDirPath != "" {
		filepath.Walk(uploadDirPath, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return nil
			}
			rel := uploadPathToRel(path)
			if rel == "" {
				return nil
			}
			result = append(result, map[string]interface{}{"rel": rel, "approved": false})
			return nil
		})
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i]["rel"].(string) < result[j]["rel"].(string)
	})
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"files": result})
}

// reviewMoveToApproved 将 uploads 下的某个文件移动到 Approved 目录并刷新曲库缓存。
// 返回是否成功。
func reviewMoveToApproved(rel string) bool {
	rel = strings.TrimSpace(rel)
	if rel == "" || approvedDirPath == "" {
		return false
	}
	src := filepath.Join(uploadDirPath, filepath.FromSlash(rel))
	// 校验确实在 uploads 下，防止任意路径被移动
	if uploadPathToRel(src) == "" {
		return false
	}
	if _, err := os.Stat(src); err != nil {
		return false
	}
	dst := filepath.Join(approvedDirPath, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return false
	}
	if err := os.Rename(src, dst); err != nil {
		return false
	}
	return true
}

// reviewRemoveApprovedRel 从已审核名单中剔除一个相对路径（移动后名单不再需要该条）
func reviewRemoveApprovedRel(rel string) {
	reviewMgr.mu.Lock()
	defer reviewMgr.mu.Unlock()
	removed := false
	n := 0
	for _, a := range reviewMgr.cfg.Approved {
		if a == rel {
			removed = true
			continue
		}
		reviewMgr.cfg.Approved[n] = a
		n++
	}
	reviewMgr.cfg.Approved = reviewMgr.cfg.Approved[:n]
	if removed {
		saveReviewConfigLocked()
	}
}

// ReviewApproveHandler POST /api/review/approve 通过审核：把 uploads 下的文件移动到 Approved 目录
func ReviewApproveHandler(w http.ResponseWriter, r *http.Request) {
	if !reviewAuthed(r) {
		http.Error(w, "未授权", 401)
		return
	}
	ensureReviewLoaded()
	var req struct {
		Paths []string `json:"paths"`
	}
	body, _ := ioutil.ReadAll(r.Body)
	json.Unmarshal(body, &req)

	passed := 0
	for _, p := range req.Paths {
		if reviewMoveToApproved(p) {
			reviewRemoveApprovedRel(strings.TrimSpace(p))
			passed++
		}
	}
	if passed > 0 {
		// 刷新曲库缓存，使 Approved 中新文件立即可搜索、可播放
		rescanUploadDir()
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "added": passed})
}

// ReviewDeleteHandler POST /api/review/delete 删除 uploads 下的待审核文件
func ReviewDeleteHandler(w http.ResponseWriter, r *http.Request) {
	if !reviewAuthed(r) {
		http.Error(w, "未授权", 401)
		return
	}
	ensureReviewLoaded()
	var req struct {
		Paths []string `json:"paths"`
	}
	body, _ := ioutil.ReadAll(r.Body)
	json.Unmarshal(body, &req)

	deleted := 0
	for _, p := range req.Paths {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		abs := filepath.Join(uploadDirPath, filepath.FromSlash(p))
		if uploadPathToRel(abs) == "" {
			continue
		}
		if err := os.Remove(abs); err != nil {
			continue
		}
		reviewRemoveApprovedRel(p)
		deleted++
	}
	if deleted > 0 {
		rescanUploadDir()
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "deleted": deleted})
}

// ReviewPasswordHandler POST /api/review/password 修改审核密码
func ReviewPasswordHandler(w http.ResponseWriter, r *http.Request) {
	if !reviewAuthed(r) {
		http.Error(w, "未授权", 401)
		return
	}
	ensureReviewLoaded()
	var req struct {
		NewPassword string `json:"newPassword"`
	}
	body, _ := ioutil.ReadAll(r.Body)
	json.Unmarshal(body, &req)

	if len(req.NewPassword) < 3 {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"success":false,"message":"密码至少3位"}`))
		return
	}
	reviewMgr.mu.Lock()
	reviewMgr.cfg.PasswordHash = hashPassword(req.NewPassword)
	saveReviewConfigLocked()
	reviewMgr.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"success":true}`))
}