package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"text/template"
	"time"
)

const missingSongsFile = "缺少歌曲.txt"

func MissingPageHandler(w http.ResponseWriter, r *http.Request) {
	tpl := `<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="UTF-8">
<title>缺歌登记</title>
<link rel="icon" href="/favicon.ico">
<style>
*{margin:0;padding:0;box-sizing:border-box;font-family:Microsoft YaHei,sans-serif}
body{background:#1a252f;color:#ecf0f1;padding:20px;min-height:100vh}
h2{color:#337ab9;margin-bottom:18px;font-size:22px;letter-spacing:1px;font-weight:600}
.input-row{display:flex;gap:10px;margin-bottom:20px}
.input-row input{flex:1;padding:11px 16px;background:#2c3e50;border:1px solid #34495e;border-radius:6px;color:#ecf0f1;font-size:15px;outline:none;transition:border-color 0.2s}
.input-row input::placeholder{color:#95a5a6}
.input-row input:focus{border-color:#337ab9}
.input-row button{padding:11px 28px;background:#428bca;border:none;border-radius:6px;color:#fff;font-size:15px;font-weight:bold;cursor:pointer;white-space:nowrap;box-shadow:0 1px 3px rgba(0,0,0,0.2);transition:background 0.15s}
.input-row button:hover{background:#3071a9}
.input-row button:active{background:#286090;box-shadow:0 1px 1px rgba(0,0,0,0.2)}
.list-title{color:#95a5a6;font-size:14px;margin-bottom:10px}
.song-list{max-height:380px;overflow-y:auto;scrollbar-width:thin;scrollbar-color:#34495e #1a252f}
.song-list::-webkit-scrollbar{width:6px}
.song-list::-webkit-scrollbar-track{background:#1a252f}
.song-list::-webkit-scrollbar-thumb{background:#34495e;border-radius:3px}
.song-item{padding:10px 14px;background:#2c3e50;margin-bottom:7px;border-radius:6px;display:flex;justify-content:space-between;align-items:center;font-size:14px;border-left:3px solid #337ab9;box-shadow:0 1px 3px rgba(0,0,0,0.15)}
.song-time{color:#95a5a6;font-size:12px}
.empty-hint{text-align:center;color:#95a5a6;padding:30px;font-size:15px}
.toast{position:fixed;top:20px;left:50%;transform:translateX(-50%);background:#428bca;color:#fff;padding:10px 28px;border-radius:6px;font-size:14px;font-weight:bold;opacity:0;transition:opacity 0.3s,transform 0.3s;pointer-events:none;z-index:100;box-shadow:0 2px 8px rgba(0,0,0,0.2)}
.toast.show{opacity:1;transform:translateX(-50%) translateY(0)}
.toast.error{background:#d9534f;color:#fff;box-shadow:0 2px 8px rgba(0,0,0,0.2)}
</style>
</head>
<body>
<h2>缺歌登记</h2>
<div class="input-row">
    <input type="text" id="songInput" placeholder="输入缺少的歌曲名称..." />
    <button onclick="submitSong()">提交</button>
</div>
<div class="list-title" id="listTitle">已登记的缺歌列表：</div>
<div class="song-list" id="songList"></div>
<div class="toast" id="toast"></div>
<script>
document.getElementById('songInput').addEventListener('keydown', function(e) {
    if (e.key === 'Enter') submitSong();
});

function submitSong() {
    var input = document.getElementById('songInput');
    var name = input.value.trim();
    if (!name) return;
    fetch('/api/missing/submit', {
        method: 'POST',
        headers: {'Content-Type': 'application/json'},
        body: JSON.stringify({name: name})
    }).then(function(r) { return r.json(); }).then(function(data) {
        if (data.success) {
            showToast('已登记: ' + name, '');
            input.value = '';
            loadList();
        } else {
            showToast(data.message || '提交失败', 'error');
        }
    }).catch(function() {
        showToast('提交失败', 'error');
    });
}

function loadList() {
    fetch('/api/missing/list').then(function(r) { return r.json(); }).then(function(data) {
        var container = document.getElementById('songList');
        var songs = data.songs || [];
        document.getElementById('listTitle').textContent = '已登记的缺歌列表（共' + songs.length + '首）：';
        if (!songs.length) {
            container.innerHTML = '<div class="empty-hint">暂无缺歌登记</div>';
            return;
        }
        var html = '';
        for (var i = 0; i < songs.length; i++) {
            html += '<div class="song-item">'
                + '<span>' + escHtml(songs[i].name) + '</span>'
                + '<span class="song-time">' + escHtml(songs[i].time) + '</span>'
                + '</div>';
        }
        container.innerHTML = html;
    });
}

function showToast(msg, type) {
    var el = document.getElementById('toast');
    el.textContent = msg;
    el.className = 'toast show' + (type ? ' ' + type : '');
    setTimeout(function() { el.className = 'toast'; }, 2000);
}

function escHtml(s) {
    if (!s) return '';
    return s.replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;').replace(/"/g,'&quot;');
}

loadList();
</script>
</body>
</html>`
	template.Must(template.New("missing").Parse(tpl)).Execute(w, nil)
}

func MissingSubmitHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "方法不允许", http.StatusMethodNotAllowed)
		return
	}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "读取数据失败", 400)
		return
	}

	var req struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(body, &req); err != nil || req.Name == "" {
		http.Error(w, "参数错误", 400)
		return
	}

	// 检查是否已存在
	existing, _ := loadMissingSongs()
	for _, s := range existing {
		if s.Name == req.Name {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.Write([]byte(`{"success":false,"message":"该歌曲已登记过"}`))
			return
		}
	}

	// 追加写入
	f, err := os.OpenFile(missingSongsFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		http.Error(w, "写入失败", 500)
		return
	}
	defer f.Close()

	line := req.Name + "\t" + time.Now().Format("2006-01-02 15:04") + "\n"
	f.WriteString(line)

	fmt.Printf("[缺歌登记] 新增: %s\n", req.Name)

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Write([]byte(`{"success":true}`))
}

func MissingListHandler(w http.ResponseWriter, r *http.Request) {
	songs, _ := loadMissingSongs()

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	result := struct {
		Songs []missingSong `json:"songs"`
	}{Songs: songs}

	if result.Songs == nil {
		result.Songs = []missingSong{}
	}

	data, _ := json.Marshal(result)
	w.Write(data)
}

type missingSong struct {
	Name string `json:"name"`
	Time string `json:"time"`
}

func loadMissingSongs() ([]missingSong, error) {
	data, err := ioutil.ReadFile(missingSongsFile)
	if err != nil {
		return []missingSong{}, nil
	}

	var songs []missingSong
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 2)
		s := missingSong{Name: parts[0]}
		if len(parts) > 1 {
			s.Time = parts[1]
		}
		songs = append(songs, s)
	}

	// 倒序，最新的在前面
	for i, j := 0, len(songs)-1; i < j; i, j = i+1, j-1 {
		songs[i], songs[j] = songs[j], songs[i]
	}

	return songs, nil
}

const zeroResultFile = "0结果搜索词.txt"

func logZeroResultKeyword(keyword string) {
	// 检查是否已存在
	data, _ := ioutil.ReadFile(zeroResultFile)
	if data != nil {
		scanner := bufio.NewScanner(strings.NewReader(string(data)))
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" {
				continue
			}
			parts := strings.SplitN(line, "\t", 2)
			if parts[0] == keyword {
				return // 已记录过
			}
		}
	}

	f, err := os.OpenFile(zeroResultFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()

	f.WriteString(keyword + "\t" + time.Now().Format("2006-01-02 15:04") + "\n")
	fmt.Printf("[0结果] 搜索词已记录: %s\n", keyword)
}
