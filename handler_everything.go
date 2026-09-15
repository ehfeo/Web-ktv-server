package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// everythingSupportedExts 与 ffmpeg/曲库支持一致的媒体扩展名集合（小写，不含点）。
// 曲库音频扩展见 handler_index/handler_player 的 audioExtensions/videoExtensions。
var everythingSupportedExts = map[string]bool{
	// 音频
	"mp3": true, "wav": true, "flac": true, "aac": true, "m4a": true, "m4r": true,
	"alac": true, "ogg": true, "oga": true, "opus": true, "wma": true, "ape": true,
	"aiff": true, "aif": true, "amr": true, "dvf": true, "msv": true, "dts": true,
	"dff": true, "dsf": true, "sacd": true, "tak": true, "tta": true, "wv": true,
	"mka": true,
	// 视频
	"mp4": true, "avi": true, "mov": true, "wmv": true, "flv": true, "mkv": true,
	"mpg": true, "mpeg": true, "rm": true, "rmvb": true, "ts": true, "m2ts": true,
	"mts": true, "m2t": true, "3gp": true, "3g2": true, "m4v": true, "vob": true,
	"ogv": true, "asf": true, "divx": true, "f4v": true, "mxf": true, "wtv": true,
	"webm": true,
}

func mediaExtSupported(p string) bool {
	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(p), "."))
	return everythingSupportedExts[ext]
}

// locateEverythingIni 定位真实的 Everything.ini。
// 注意：安装版 Everything 的真实配置（含 http_server_enabled/port/username/password）存放在
// %APPDATA%\Everything\Everything.ini；安装目录下通常只有一行 allow_http_server=1 的便携标记。
// 因此优先返回含 "http_server_enabled=" 的 ini；无则返回第一个存在的。
func locateEverythingIni(exePath string) string {
	var cands []string
	cands = append(cands, filepath.Join(os.Getenv("APPDATA"), "Everything", "Everything.ini"))
	cands = append(cands, filepath.Join(os.Getenv("LOCALAPPDATA"), "Everything", "Everything.ini"))
	if exePath != "" {
		cands = append(cands, filepath.Join(filepath.Dir(exePath), "Everything.ini"))
	}
	cands = append(cands, "Everything.ini")
	best := ""
	for _, cand := range cands {
		if cand == "" {
			continue
		}
		fi, err := os.Stat(cand)
		if err != nil || fi.IsDir() {
			continue
		}
		if best == "" {
			best = cand
		}
		if data, e := ioutil.ReadFile(cand); e == nil && strings.Contains(string(data), "http_server_enabled=") {
			return cand
		}
	}
	return best
}

// everythingHTTPConfig 自动读取 Everything.ini 获取 HTTP 服务器端口与认证信息。
// 端口优先用户填写的 everythingPort，否则取 ini 的 http_server_port（缺省 80）；
// 用户名/密码始终从 ini 的 http_server_username / http_server_password 读取。
// 若 Everything HTTP 服务未启用，返回明确的中文错误提示。
func everythingHTTPConfig() (port int, user, pass string, err error) {
	ini := locateEverythingIni(everythingPath)
	iniEnabled := ""
	iniPort := "80"
	iniUser := ""
	iniPass := ""
	if ini != "" {
		if data, e := ioutil.ReadFile(ini); e == nil {
			for _, ln := range strings.Split(string(data), "\n") {
				ln = strings.TrimSpace(strings.TrimSuffix(ln, "\r"))
				lower := strings.ToLower(ln)
				switch {
				case strings.HasPrefix(lower, "http_server_enabled="):
					iniEnabled = strings.TrimSpace(ln[len("http_server_enabled="):])
				case strings.HasPrefix(lower, "allow_http_server="):
					if iniEnabled == "" {
						iniEnabled = strings.TrimSpace(ln[len("allow_http_server="):])
					}
				case strings.HasPrefix(lower, "http_server_port="):
					iniPort = strings.TrimSpace(ln[len("http_server_port="):])
				case strings.HasPrefix(lower, "http_server_username="):
					iniUser = strings.TrimSpace(ln[len("http_server_username="):])
				case strings.HasPrefix(lower, "http_server_password="):
					iniPass = strings.TrimSpace(ln[len("http_server_password="):])
				}
			}
		}
	}

	// 只要读得到 ini 且其 HTTP 服务未启用，就明确提示（即便用户填了端口）
	if ini != "" && iniEnabled != "1" {
		return 0, "", "", fmt.Errorf("Everything 的 HTTP 服务器未启用（读取 %s，http_server_enabled=%q）。请在 Everything 菜单 工具→选项→HTTP服务器 中勾选启用该服务，并记下端口号。", ini, iniEnabled)
	}

	userPort := strings.TrimSpace(everythingPort)
	if userPort != "" {
		n, e := strconv.Atoi(userPort)
		if e != nil || n <= 0 || n > 65535 {
			return 0, "", "", fmt.Errorf("Everything 端口无效: %s", userPort)
		}
		port = n
	} else {
		if ini == "" {
			return 0, "", "", fmt.Errorf("未找到 Everything.ini，无法确定 HTTP 服务器端口/是否启用。请在设置填写 Everything HTTP 服务器端口，或在 Everything 中启用 HTTP 服务器。")
		}
		if n, e := strconv.Atoi(iniPort); e == nil && n > 0 && n <= 65535 {
			port = n
		} else {
			port = 80
		}
	}
	return port, iniUser, iniPass, nil
}

// everythingSearch 调用本机 Everything HTTP 服务搜索关键字，返回命中的完整路径列表。
// 每次搜索独立查询，避免缓存路径影响。
func everythingSearch(keyword string) ([]string, error) {
	port, user, pass, err := everythingHTTPConfig()
	if err != nil {
		return nil, err
	}
	// Everything HTTP JSON API：?search=<kw>&json=1&path_column=1 会返回包含完整路径的 JSON
	u := fmt.Sprintf("http://127.0.0.1:%d/?search=%s&json=1&path_column=1", port, url.QueryEscape(keyword))
	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return nil, fmt.Errorf("构造 Everything 请求失败：%v", err)
	}
	// Everything HTTP 服务器若配置了用户名/密码，需要 Basic Auth
	if user != "" {
		req.SetBasicAuth(user, pass)
	}
	client := &http.Client{Timeout: 8 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("无法连接 Everything HTTP 服务(端口 %d)%s：%v", port, authHint(user), err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusUnauthorized {
		return nil, fmt.Errorf("Everything HTTP 服务拒绝了认证请求(401)。请在 Everything 菜单 工具→选项→HTTP服务器 中核对用户名/密码，或取消勾选“启用 HTTP 服务器用户名和密码”。")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Everything HTTP 服务返回状态码 %d", resp.StatusCode)
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取 Everything 响应失败：%v", err)
	}

	var paths []string
	trim := func(s string) string { return strings.TrimSpace(strings.Trim(s, "\"")) }

	// 容错解析：可能是顶层数组，也可能是 { "results": [...] }
	var raw interface{}
	if err := json.Unmarshal(body, &raw); err == nil {
		arr := toArray(raw)
		for _, item := range arr {
			var p string
			if s, ok := item.(string); ok {
				p = s
			} else if m, ok := item.(map[string]interface{}); ok {
				// Everything JSON 对象：type/name/path。其中 path 是文件所在目录（不含文件名），
				// name 是文件名。fullpath 若存在则已含完整路径，优先使用；
				// 否则用 filepath.Join(path, name) 拼出完整路径，避免只取到目录而被后续 IsDir 过滤。
				if v, has := m["fullpath"]; has && v != nil {
					p, _ = v.(string)
				} else {
					dir, _ := m["path"].(string)
					name, _ := m["name"].(string)
					if dir != "" && name != "" {
						p = filepath.Join(dir, name)
					} else if dir != "" {
						p = dir
					} else {
						p = name
					}
				}
			}
			p = trim(p)
			if p != "" {
				paths = append(paths, p)
			}
		}
	} else {
		// 退而求其次：按行解析（非 JSON 情况）
		for _, ln := range strings.Split(string(body), "\n") {
			ln = trim(ln)
			if ln != "" {
				paths = append(paths, ln)
			}
		}
	}

	// 扩展名过滤：只保留媒体文件，并按文件存在过滤
	result := make([]string, 0, len(paths))
	seen := make(map[string]bool)
	for _, p := range paths {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		key := strings.ToLower(p)
		if seen[key] {
			continue
		}
		seen[key] = true
		if !mediaExtSupported(p) {
			continue
		}
		if fi, err := os.Stat(p); err == nil && !fi.IsDir() {
			result = append(result, p)
		}
	}
	return result, nil
}

func toArray(v interface{}) []interface{} {
	switch t := v.(type) {
	case []interface{}:
		return t
	case map[string]interface{}:
		if r, ok := t["results"].([]interface{}); ok {
			return r
		}
	}
	return nil
}

// EverythingSearchHandler /api/everything?q=<keyword>
// 返回 Everything 命中的媒体文件列表（路径+显示名+类型），供前端追加到搜索结果。
func EverythingSearchHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if q == "" {
		json.NewEncoder(w).Encode(MResult{Ok: false, Error: "缺少搜索关键字"})
		return
	}
	paths, err := everythingSearch(q)
	if err != nil {
		json.NewEncoder(w).Encode(MResult{Ok: false, Error: err.Error()})
		return
	}
	type item struct {
		Path string `json:"path"`
		Name string `json:"name"`
		Type string `json:"type"`
		From string `json:"from"` // 标记来源：everything
	}
	var items []item
	for _, p := range paths {
		base := filepath.Base(p)
		items = append(items, item{Path: p, Name: base, Type: mediaTypeOf(p), From: "everything"})
	}
	json.NewEncoder(w).Encode(map[string]interface{}{
		"ok":    true,
		"count": len(items),
		"items": items,
		"warn":  "Everything 搜索结果不受审核控制，可直接点播！",
	})
}

type MResult struct {
	Ok    bool   `json:"ok"`
	Error string `json:"error"`
}

// authHint 当 Everything 配了认证时，在连接错误信息里附带提示。
func authHint(user string) string {
	if user != "" {
		return "(已启用 HTTP 认证，请核对用户名/密码)"
	}
	return ""
}

func mediaTypeOf(p string) string {
	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(p), "."))
	if _, isAudio := map[string]bool{"mp3": true, "wav": true, "flac": true, "aac": true, "m4a": true, "ogg": true, "oga": true, "opus": true, "wma": true, "ape": true, "aiff": true, "amr": true, "dts": true, "dff": true, "dsf": true, "wv": true, "mka": true, "alac": true, "m4r": true, "dvf": true, "msv": true, "sacd": true, "tak": true, "tta": true}[ext]; isAudio {
		return "audio"
	}
	return "video"
}