package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"text/template"
)

func IndexHandler(w http.ResponseWriter, r *http.Request) {
	if isMobileBrowser(r.UserAgent()) {
		http.Redirect(w, r, "/m", http.StatusFound)
		return
	}
	tpl := `
<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <title>KTV 双屏点歌机 - 控制台</title>
    <style>
        *{margin:0;padding:0;box-sizing:border-box;font-family:Microsoft YaHei}
        body{background:#1a252f;color:#ecf0f1}
        .wrap{display:flex;height:100vh}
        .ktv-left{width:66.67%;background:#243447;border-right:1px solid #3a5068;display:flex;flex-direction:column}
        .ktv-top{display:flex;align-items:center;gap:10px;padding:10px 15px;background:#2c3e50;border-bottom:1px solid #3a5068;min-height:56px}
        #search{flex:1;padding:8px 14px;background:#1a252f;border:1px solid #3a5068;border-radius:4px;color:#ecf0f1;font-size:14px;max-width:300px;transition:border-color 0.2s,box-shadow 0.2s}
        #search:focus{border-color:#428bca;box-shadow:0 0 0 2px rgba(66,139,202,0.25);outline:none}
        #search::placeholder{color:#7f8c8d}
        #searchBtn{padding:8px 20px;background:linear-gradient(180deg,#428bca,#337ab9);border:1px solid #2a6496;border-radius:4px;color:#fff;font-size:14px;font-weight:bold;cursor:pointer;box-shadow:0 2px 4px rgba(0,0,0,0.2);transition:background 0.15s,box-shadow 0.15s}
        #searchBtn:hover{background:linear-gradient(180deg,#4e97d1,#3578b5);box-shadow:0 3px 6px rgba(0,0,0,0.25)}
        #searchBtn:active{background:#2a6496;box-shadow:0 1px 2px rgba(0,0,0,0.2)}
        .top-btn{padding:8px 14px;background:linear-gradient(180deg,#5a6b7d,#4a5568);border:1px solid #3a5068;border-radius:4px;color:#ecf0f1;font-size:13px;font-weight:bold;cursor:pointer;white-space:nowrap;box-shadow:0 2px 4px rgba(0,0,0,0.2);transition:background 0.15s,box-shadow 0.15s}
        .top-btn:hover{background:linear-gradient(180deg,#428bca,#337ab9);color:#fff;border-color:#2a6496;box-shadow:0 3px 6px rgba(0,0,0,0.25)}
        .top-btn:active{background:#2a6496;box-shadow:0 1px 2px rgba(0,0,0,0.2)}
        .status-info{display:flex;gap:15px;font-size:14px;color:#5bc0de;white-space:nowrap;align-items:center}
        .toggle-switch{display:inline-flex;align-items:center;gap:5px;cursor:pointer;font-size:13px;color:#95a5a6}
        .toggle-switch input{display:none}
        .toggle-slider{width:36px;height:20px;background:#3a5068;border-radius:10px;position:relative;transition:background 0.3s}
        .toggle-slider::after{content:'';position:absolute;top:2px;left:2px;width:16px;height:16px;background:#7f8c8d;border-radius:50%;transition:all 0.3s}
        .toggle-switch input:checked+.toggle-slider{background:#337ab9}
        .toggle-switch input:checked+.toggle-slider::after{left:18px;background:#fff}
        .song-list{flex:1;overflow-y:auto;padding:6px;display:grid;grid-template-columns:repeat(auto-fill, minmax(200px, 1fr));grid-template-rows:repeat(auto-fill, 60px);gap:8px}
        .song-item{padding:6px 4px 6px 7px;background:#243447;border:1px solid #3a5068;border-left:3px solid #428bca;border-radius:4px;cursor:pointer;display:flex;flex-direction:column;align-items:center;text-align:center;transition:background 0.15s,border-color 0.15s,box-shadow 0.15s;overflow:hidden;min-height:60px;max-height:60px;position:relative;box-shadow:0 1px 3px rgba(0,0,0,0.1)}
        .song-item:hover{background:#2c3e50;border-color:#428bca;box-shadow:0 2px 6px rgba(0,0,0,0.2)}
        .song-item:active{background:#1e2d3d;box-shadow:0 1px 2px rgba(0,0,0,0.15)}
        .song-item .song-name{color:#bdc3c7;transition:color 0.15s}
        .song-item:hover .song-name{color:#ecf0f1}
        .ktv-right{width:33.33%;background:#1e2d3d;display:flex;flex-direction:column;box-shadow:-2px 0 8px rgba(0,0,0,0.15)}
        .control-bar{padding:10px 15px;background:#2c3e50;border-bottom:1px solid #3a5068;display:flex;justify-content:center;gap:15px;flex-wrap:wrap;min-height:56px;align-items:center}
        .control-btn{width:80px;height:40px;border-radius:4px;border:1px solid transparent;background:linear-gradient(180deg,#5a6b7d,#4a5568);color:#ecf0f1;font-size:16px;font-weight:bold;cursor:pointer;box-shadow:0 2px 4px rgba(0,0,0,0.2);transition:background 0.15s,box-shadow 0.15s}
        .control-btn:hover{background:linear-gradient(180deg,#428bca,#337ab9);color:#fff;box-shadow:0 3px 6px rgba(0,0,0,0.25)}
        .control-btn:active{background:#2a6496;box-shadow:0 1px 2px rgba(0,0,0,0.2)}
        .control-btn.active{background:linear-gradient(180deg,#5cb85c,#4cae4c);border-color:#3d8b3d;box-shadow:0 2px 4px rgba(0,0,0,0.2);color:#fff}
        .control-btn.active:hover{background:linear-gradient(180deg,#6ec96e,#5cb85c);box-shadow:0 3px 6px rgba(0,0,0,0.25)}
        .queue-area{flex:1;background:#1e2d3d;padding:15px;overflow-y:auto}
        .queue-title{color:#428bca;margin-bottom:8px;font-size:16px;font-weight:bold}
        .queue-item{padding:8px 12px;background:#243447;margin:4px 0;border-radius:4px;display:flex;justify-content:space-between;align-items:center;box-shadow:0 1px 3px rgba(0,0,0,0.1);border:1px solid #3a5068;transition:background 0.15s,border-color 0.15s}
        .queue-item:hover{background:#2c3e50;border-color:#4a6078}
        .transcode-progress{font-size:12px;color:#f0ad4e;margin-left:8px}
        .transcode-progress-bar{width:60px;height:6px;background:#3a5068;border-radius:3px;overflow:hidden;margin:4px 0}
        .transcode-progress-fill{height:100%;background:#f0ad4e;border-radius:3px;transition:width 0.3s}
        .top-btn{background:linear-gradient(180deg,#5a6b7d,#4a5568);border:1px solid #3a5068;color:#ecf0f1;border-radius:4px;padding:2px 8px;cursor:pointer;font-size:11px;font-weight:bold;box-shadow:0 2px 4px rgba(0,0,0,0.2);transition:background 0.15s,box-shadow 0.15s}
        .top-btn:hover{background:linear-gradient(180deg,#428bca,#337ab9);color:#fff;border-color:#2a6496;box-shadow:0 3px 6px rgba(0,0,0,0.25)}
        .top-btn:active{background:#2a6496;box-shadow:0 1px 2px rgba(0,0,0,0.2)}
        .del-btn{background:linear-gradient(180deg,#d9534f,#c9302c);border:1px solid #b52b27;color:#fff;border-radius:4px;padding:2px 8px;cursor:pointer;font-size:11px;font-weight:bold;box-shadow:0 2px 4px rgba(0,0,0,0.2);transition:background 0.15s,box-shadow 0.15s}
        .del-btn:hover{background:linear-gradient(180deg,#e06b67,#d9534f);box-shadow:0 3px 6px rgba(0,0,0,0.25)}
        .del-btn:active{background:#b52b27;box-shadow:0 1px 2px rgba(0,0,0,0.2)}
        .toast{position:fixed;top:20px;left:50%;transform:translateX(-50%);background:#337ab9;color:#fff;padding:10px 24px;border-radius:4px;font-size:14px;opacity:0;transition:opacity 0.3s;pointer-events:none;z-index:9999;box-shadow:0 2px 8px rgba(0,0,0,0.2)}
        .toast.show{opacity:1}
        .browse-tabs{display:flex;gap:0;background:#2c3e50;border-bottom:1px solid #3a5068}
        .browse-tab{flex:1;padding:10px 0;background:transparent;border:none;border-top:2px solid transparent;border-bottom:2px solid transparent;color:#95a5a6;font-size:16px;font-weight:bold;cursor:pointer;transition:color 0.2s,background 0.2s,border-color 0.2s;position:relative}
        .browse-tab:hover{color:#bdc3c7;background:#243447}
        .browse-tab.active{color:#428bca;background:transparent;border-bottom:2px solid #428bca}
        .singer-panel{flex:1;overflow-y:auto;display:flex;flex-direction:column}
        .singer-letters{display:flex;flex-wrap:wrap;gap:3px;padding:6px 8px;background:#2c3e50;border-bottom:1px solid #3a5068;flex-shrink:0}
        .singer-letters span{padding:3px 8px;background:#243447;border:1px solid #3a5068;border-radius:3px;color:#bdc3c7;font-size:12px;cursor:pointer;transition:background 0.15s,color 0.15s,border-color 0.15s}
        .singer-letters span:hover,.singer-letters span.active{background:#428bca;color:#fff;border-color:#3578b5}
        .singer-list{flex:1;overflow-y:auto;padding:6px}
        .singer-grid{padding:6px;display:grid;grid-template-columns:repeat(auto-fill,minmax(120px,1fr));grid-template-rows:repeat(auto-fill,50px);gap:6px}
        .singer-btn{padding:6px 4px;background:#243447;border:1px solid #3a5068;border-left:3px solid #428bca;border-radius:4px;cursor:pointer;display:flex;flex-direction:column;align-items:center;justify-content:center;text-align:center;overflow:hidden;min-height:50px;max-height:50px;transition:background 0.15s,border-color 0.15s,box-shadow 0.15s;box-shadow:0 1px 3px rgba(0,0,0,0.1)}
        .singer-btn:hover{background:#2c3e50;border-color:#428bca;box-shadow:0 2px 6px rgba(0,0,0,0.2)}
        .singer-btn:active{background:#1e2d3d;box-shadow:0 1px 2px rgba(0,0,0,0.15)}
        .singer-btn .sname{font-size:13px;white-space:nowrap;overflow:hidden;text-overflow:ellipsis;width:100%;color:#bdc3c7}
        .singer-btn .scount{font-size:10px;color:#5bc0de}
        .cat-grid{padding:6px;display:grid;grid-template-columns:repeat(auto-fill,minmax(100px,1fr));grid-template-rows:repeat(auto-fill,50px);gap:6px}
        .cat-btn{padding:6px 4px;background:#243447;border:1px solid #3a5068;border-left:3px solid #5bc0de;border-radius:4px;cursor:pointer;display:flex;flex-direction:column;align-items:center;justify-content:center;text-align:center;overflow:hidden;min-height:50px;max-height:50px;transition:background 0.15s,border-color 0.15s,box-shadow 0.15s;box-shadow:0 1px 3px rgba(0,0,0,0.1)}
        .cat-btn:hover{background:#2c3e50;border-color:#5bc0de;box-shadow:0 2px 6px rgba(0,0,0,0.2)}
        .cat-btn:active{background:#1e2d3d;box-shadow:0 1px 2px rgba(0,0,0,0.15)}
        .cat-btn .cname{font-size:13px;white-space:nowrap;overflow:hidden;text-overflow:ellipsis;width:100%;color:#bdc3c7}
        .cat-btn .ccount{font-size:10px;color:#5bc0de}
        .singer-songs{padding:6px;display:grid;grid-template-columns:repeat(auto-fill,minmax(200px,1fr));grid-template-rows:repeat(auto-fill,60px);gap:8px}
        .singer-back{padding:8px 12px;background:linear-gradient(180deg,#5a6b7d,#4a5568);margin-bottom:6px;border-radius:4px;cursor:pointer;color:#ecf0f1;font-size:13px;font-weight:bold;text-align:center;box-shadow:0 2px 4px rgba(0,0,0,0.2);transition:background 0.15s,box-shadow 0.15s}
        .singer-back:hover{background:linear-gradient(180deg,#428bca,#337ab9);color:#fff;box-shadow:0 3px 6px rgba(0,0,0,0.25)}
        .singer-back:active{background:#2a6496;box-shadow:0 1px 2px rgba(0,0,0,0.2)}
        .pagination{padding:10px;background:#2c3e50;border-top:1px solid #3a5068;display:flex;justify-content:center;gap:8px;flex-wrap:wrap;max-height:80px;overflow-y:auto}
        .page-buttons{display:flex;gap:8px;flex-wrap:wrap;justify-content:center;align-items:center}
        .page-btn{padding:4px 10px;background:#243447;border:1px solid #3a5068;border-radius:3px;color:#ecf0f1;cursor:pointer;font-size:14px;font-weight:bold;box-shadow:0 1px 3px rgba(0,0,0,0.1);transition:background 0.15s,box-shadow 0.15s}
        .page-btn:hover{background:#428bca;color:#fff;border-color:#3578b5;box-shadow:0 2px 4px rgba(0,0,0,0.2)}
        .page-btn:active{background:#2a6496;box-shadow:0 1px 2px rgba(0,0,0,0.15)}
        .page-btn.active{background:#428bca;border-color:#3578b5;color:#fff}
        .page-btn:disabled{background:#1e2d3d;color:#5a6b7d;cursor:not-allowed;box-shadow:none;border-color:#2c3e50}
        .loading{text-align:center;color:#95a5a6;padding:20px}
        .nav-page{position:fixed;top:0;left:0;width:100%;height:100%;background:#1a252f;z-index:1000;display:none;overflow-y:auto}
        .nav-page.active{display:block}
        .nav-content{max-width:700px;margin:0 auto;padding:30px 20px}
        .nav-title{font-size:28px;margin-bottom:8px;color:#428bca;text-align:center}
        .nav-subtitle{font-size:14px;color:#95a5a6;text-align:center;margin-bottom:20px}
        .nav-section{margin-bottom:18px}
        .nav-section-title{font-size:15px;color:#428bca;margin-bottom:8px;border-bottom:2px solid #3a5068;padding-bottom:6px}
        .nav-card{background:#243447;border:1px solid #3a5068;border-radius:4px;padding:12px 16px;margin-bottom:8px;display:flex;align-items:flex-start;gap:10px;box-shadow:0 1px 3px rgba(0,0,0,0.1);transition:box-shadow 0.2s,border-color 0.2s}
        .nav-card:hover{box-shadow:0 2px 8px rgba(0,0,0,0.2);border-color:#4a6078}
        .nav-card-icon{font-size:20px;flex-shrink:0;line-height:1.3}
        .nav-card-body{flex:1;min-width:0}
        .nav-card-title{font-size:14px;font-weight:bold;margin-bottom:2px;color:#ecf0f1}
        .nav-card-detail{font-size:12px;color:#95a5a6;line-height:1.5}
        .nav-card-action{margin-top:6px}
        .nav-card-action a,.nav-card-action button{display:inline-block;padding:6px 16px;border:none;border-radius:4px;font-size:12px;cursor:pointer;text-decoration:none;margin-right:6px;margin-top:4px;transition:background 0.15s,box-shadow 0.15s;box-shadow:0 2px 4px rgba(0,0,0,0.2)}
        .nav-action-settings{background:linear-gradient(180deg,#428bca,#337ab9);color:#fff;border:1px solid #2a6496}
        .nav-action-settings:hover{background:linear-gradient(180deg,#4e97d1,#3578b5);box-shadow:0 3px 6px rgba(0,0,0,0.25)}
        .nav-action-settings:active{background:#2a6496;box-shadow:0 1px 2px rgba(0,0,0,0.2)}
        .nav-action-link{background:#243447;color:#5bc0de;text-decoration:underline;border:1px solid #3a5068;cursor:pointer}
        .nav-action-link:hover{color:#6ecff1;border-color:#4a6078}
        .nav-status-ok{color:#5cb85c}
        .nav-status-warn{color:#f0ad4e}
        .nav-status-err{color:#d9534f}
        .nav-track-notice{background:#243447;border:1px solid #3a5068;border-radius:4px;padding:12px 16px;margin-bottom:18px;font-size:13px;color:#95a5a6;line-height:1.6;box-shadow:0 1px 3px rgba(0,0,0,0.1)}
        .nav-track-notice input[type=text]{width:100%;padding:8px;background:#1a252f;border:1px solid #3a5068;border-radius:3px;color:#ecf0f1;font-size:13px;margin:8px 0;box-sizing:border-box}
        .nav-track-notice input[type=text]:focus{border-color:#428bca;outline:none}
        .nav-track-notice button{padding:6px 16px;background:linear-gradient(180deg,#428bca,#337ab9);border:1px solid #2a6496;border-radius:4px;color:#fff;font-size:13px;cursor:pointer;box-shadow:0 2px 4px rgba(0,0,0,0.2);transition:background 0.15s,box-shadow 0.15s}
        .nav-track-notice button:hover{background:linear-gradient(180deg,#4e97d1,#3578b5);box-shadow:0 3px 6px rgba(0,0,0,0.25)}
        .nav-skip{display:block;width:100%;padding:14px;background:linear-gradient(180deg,#428bca,#337ab9);color:#fff;border:1px solid #2a6496;border-radius:4px;font-size:18px;font-weight:bold;margin-top:20px;cursor:pointer;text-align:center;box-shadow:0 2px 4px rgba(0,0,0,0.2);transition:background 0.15s,box-shadow 0.15s}
        .nav-skip:hover{background:linear-gradient(180deg,#4e97d1,#3578b5);box-shadow:0 3px 6px rgba(0,0,0,0.25)}
        .nav-skip:active{background:#2a6496;box-shadow:0 1px 2px rgba(0,0,0,0.2)}
        .nav-skip-critical{background:linear-gradient(180deg,#d9534f,#c9302c);border-color:#b52b27}
        .nav-skip-critical:hover{background:linear-gradient(180deg,#e06b67,#d9534f)}
        .qr-modal{position:fixed;top:0;left:0;width:100vw;height:100vh;background:rgba(0,0,0,0.6);display:none;align-items:center;justify-content:center;z-index:10000}
        .qr-modal-content{background:#fff;border:1px solid #ddd;border-radius:4px;padding:30px;text-align:center;max-width:350px;box-shadow:0 4px 16px rgba(0,0,0,0.2)}
        .qr-modal-content h2{color:#337ab9;margin-bottom:15px;font-size:20px}
        .qr-modal-content img{border-radius:4px;margin:10px 0;box-shadow:0 1px 4px rgba(0,0,0,0.1)}
        .qr-modal-content .qr-info{color:#666;font-size:13px;margin:8px 0}
        .qr-modal-content .qr-status{font-size:14px;margin:10px 0}
        .qr-modal-content button{margin-top:15px;padding:8px 24px;border:1px solid #2a6496;border-radius:4px;background:linear-gradient(180deg,#428bca,#337ab9);color:#fff;cursor:pointer;font-size:14px;font-weight:bold;box-shadow:0 2px 4px rgba(0,0,0,0.2);transition:background 0.15s,box-shadow 0.15s}
        .qr-modal-content button:hover{background:linear-gradient(180deg,#4e97d1,#3578b5);box-shadow:0 3px 6px rgba(0,0,0,0.25)}
        .qr-modal-content button:active{background:#2a6496;box-shadow:0 1px 2px rgba(0,0,0,0.2)}
    </style>
</head>
<body>
<div class="toast" id="toast"></div>
<div id="statusBadge" style="position:fixed;bottom:8px;right:8px;background:rgba(26,37,47,0.9);color:#ecf0f1;padding:6px 14px;border-radius:4px;font-size:12px;z-index:9999;display:none;max-width:400px;pointer-events:none;border:1px solid rgba(255,255,255,0.08)"></div>
<div class="nav-page" id="navPage">
    <div class="nav-content">
        <div class="nav-title">欢迎使用 KTV 双屏点歌机</div>
        <div class="nav-subtitle" id="navSubtitle">正在加载系统状态...</div>
        <div id="navSelfCheckContent"></div>
        <div id="navTrackNotice" style="display:none"></div>
        <button class="nav-skip" id="navSkipBtn" onclick="skipNavPage()">进入点歌系统</button>
    </div>
</div>

<div class="wrap" id="mainPage">
    <div class="ktv-left">
        <div class="ktv-top">
            <input type="text" id="search" placeholder="搜索歌名/歌手..." onkeydown="handleSearchKeydown(event)" oninput="handleSearchInput()">
            <button id="searchBtn" onclick="filterList()">搜索</button>
            <button class="top-btn" onclick="openUpload()">上传歌曲</button>
            <button class="top-btn" id="btnQR" onclick="showQRCode()">扫码点歌</button>
            <button class="top-btn" onclick="openMissing()">缺歌登记</button>
            <div class="status-info">
                <span id="pageInfo">当前页: 1/1</span>
                <span id="totalInfo">总曲目: 0</span>
                <label class="toggle-switch" title="省流模式：视频实时转码低码率传输，节省带宽">
                    <input type="checkbox" id="streamMode" onchange="onStreamModeChange()">
                    <span class="toggle-slider"></span>
                    省流
                </label>
                <label class="toggle-switch" title="音乐模式：视频文件只播放音频，省流且听歌更专注">
                    <input type="checkbox" id="musicMode" onchange="onMusicModeChange()">
                    <span class="toggle-slider"></span>
                    音乐
                </label>
            </div>
        </div>
        <div class="browse-tabs">
            <button class="browse-tab active" id="tabSearch" onclick="switchBrowseTab('search')">搜索</button>
            <button class="browse-tab" id="tabSinger" onclick="switchBrowseTab('singer')">歌手</button>
            <button class="browse-tab" id="tabLanguage" onclick="switchBrowseTab('language')">语种</button>
            <button class="browse-tab" id="tabCategory" onclick="switchBrowseTab('category')">曲种</button>
            <button class="browse-tab" id="tabHot" onclick="switchBrowseTab('hot')">热播</button>
        </div>
        <div class="song-list" id="songList">
            <div class="loading">加载中...</div>
        </div>
        <div class="singer-panel" id="singerPanel" style="display:none">
            <div class="singer-letters" id="singerLetters"></div>
            <div class="singer-list" id="singerList"></div>
        </div>
        <div class="singer-panel" id="languagePanel" style="display:none">
            <div class="singer-list" id="languageList"></div>
        </div>
        <div class="singer-panel" id="categoryPanel" style="display:none">
            <div class="singer-list" id="categoryList"></div>
        </div>
        <div class="singer-panel" id="hotPanel" style="display:none">
            <div class="singer-list" id="hotList"></div>
        </div>
        <div class="pagination" id="pagination">
            <div class="page-buttons" id="pageButtons"></div>
        </div>
    </div>
    <div class="ktv-right">
        <div class="control-bar">
            <button class="control-btn" onclick="prevSong()">重播</button>
            <button class="control-btn" onclick="nextSong()">下一首</button>
            <button id="btnOrigin" class="control-btn active" onclick="switchTrack(0)">原唱</button>
            <button id="btnAcc" class="control-btn" onclick="switchTrack(1)">伴奏</button>
            <button class="control-btn" onclick="openSettings()">设置</button>
        </div>
        <div class="queue-area">
            <div class="queue-title">播放队列</div>
            <div id="queueList"></div>
            <button class="control-btn" style="width:100%;margin-top:8px;background:linear-gradient(180deg,#f0ad4e,#ec971f);border:1px solid #d58512;color:#fff;box-shadow:0 2px 4px rgba(0,0,0,0.2);font-weight:bold;font-size:15px" onclick="randomSong()">随机点歌</button>
        </div>
    </div>
</div>

</div>

<div class="qr-modal" id="qrModal">
    <div class="qr-modal-content">
        <h2>扫码点歌</h2>
        <img id="qrImage" src="" alt="二维码" width="250" height="250">
        <div class="qr-info" id="qrSessionId"></div>
        <div class="qr-info" id="qrUrl"></div>
        <div class="qr-status" id="qrStatus"></div>
        <button onclick="closeQRModal()">关闭</button>
    </div>
</div>

<script>
    var playerWin = null;
    // 已整合：音频与视频均使用同一个 playerWin（视频播放器内嵌 audio-player iframe）
    var queue = [];
    var currentPage = 1;
    var pageSize = 24;
    var totalItems = 0;
    var totalPages = 1;
    var currentKeyword = '';
    var columns = 6;
    var rows = 4;
    var transcodePollInterval = null;
    var currentPlayingIndex = -1;

    // 播放队列持久化（12小时有效期）
    var QUEUE_STORAGE_KEY = 'ktv_playQueue';
    var QUEUE_EXPIRE_MS = 12 * 60 * 60 * 1000; // 12小时

    function saveQueueToStorage() {
        try {
            var data = {
                queue: queue,
                currentPlayingIndex: currentPlayingIndex,
                timestamp: Date.now()
            };
            localStorage.setItem(QUEUE_STORAGE_KEY, JSON.stringify(data));
        } catch(e) {}
    }

    function loadQueueFromStorage() {
        try {
            var raw = localStorage.getItem(QUEUE_STORAGE_KEY);
            if (!raw) return null;
            var data = JSON.parse(raw);
            if (!data || !data.queue || !data.timestamp) return null;
            if (Date.now() - data.timestamp > QUEUE_EXPIRE_MS) {
                localStorage.removeItem(QUEUE_STORAGE_KEY);
                return null;
            }
            // 恢复时清理运行时状态，仅保留播放所需字段
            var restoredQueue = [];
            for (var i = 0; i < data.queue.length; i++) {
                var item = data.queue[i];
                restoredQueue.push({
                    path: item.path,
                    name: item.name,
                    type: item.type,
                    displayName: item.displayName || '',
                    status: 'checking',
                    transcodeProgress: 0,
                    requestKey: item.path
                });
            }
            return {
                queue: restoredQueue,
                currentPlayingIndex: (data.currentPlayingIndex >= 0 && data.currentPlayingIndex < restoredQueue.length) ? data.currentPlayingIndex : -1
            };
        } catch(e) { return null; }
    }

    function clearQueueStorage() {
        try { localStorage.removeItem(QUEUE_STORAGE_KEY); } catch(e) {}
    }

    // 定时检测播放窗口是否被关闭
    // 如果用户直接关闭播放窗口，无法触发 video.onended → postMessage("ended")
    // 此定时器作为兜底：窗口关闭时标记当前歌曲播放结束（不自动播放下一首）
    setInterval(function(){
        if(currentPlayingIndex < 0) return;
        var item = queue[currentPlayingIndex];
        if(!item) return;
        // 已整合：所有播放（音频/视频）都在 playerWin 中
        if(playerWin && playerWin.closed){
            // 播放窗口被关闭，视为当前歌曲播放结束
            console.log('[KTV] 播放窗口已关闭，标记当前歌曲结束');
            // 从列表移除当前已播放歌曲
            queue.splice(currentPlayingIndex, 1);
            currentPlayingIndex = -1;
            renderQueue();
            saveQueueToStorage();
        }
    }, 2000);
    var mySessionId = '';
    var lastTrackIndex = 0;

    function isStreamMode() {
        return document.getElementById('streamMode').checked;
    }

    function onStreamModeChange() {
        // 省流模式切换时，如果当前正在播放视频，重新播放当前曲目
        if (currentPlayingIndex >= 0 && currentPlayingIndex < queue.length) {
            playQueueItem(currentPlayingIndex);
        }
    }

    function isMusicMode() {
        return document.getElementById('musicMode').checked;
    }

    function onMusicModeChange() {
        // 音乐模式切换时，如果当前正在播放，重新播放当前曲目
        if (currentPlayingIndex >= 0 && currentPlayingIndex < queue.length) {
            playQueueItem(currentPlayingIndex);
        }
    }

    function isAudioFile(fileName) {
        var audioExtensions = ['.mp3', '.wav', '.flac', '.aac', '.m4a', '.m4r', '.alac', '.ogg', '.oga', '.opus', '.wma', '.ape', '.aiff', '.aif', '.amr', '.dvf', '.msv', '.dts', '.dff', '.dsf', '.sacd', '.tak', '.tta', '.wv', '.mka'];
        var ext = fileName.toLowerCase().substring(fileName.lastIndexOf('.'));
        return audioExtensions.indexOf(ext) !== -1;
    }

    function needsAudioTranscode(fileName) {
        var ext = fileName.toLowerCase().substring(fileName.lastIndexOf('.'));
        return ext === '.ape' || ext === '.wma' || ext === '.dts' || ext === '.dff' || ext === '.dsf' ||
            ext === '.aiff' || ext === '.aif' || ext === '.amr' || ext === '.dvf' || ext === '.msv' ||
            ext === '.sacd' || ext === '.tak' || ext === '.tta' || ext === '.wv' || ext === '.mka' || ext === '.oga';
    }

    function showPopupBlockedWarning() {
        var existing = document.getElementById('popupWarning');
        if(existing) existing.remove();
        var div = document.createElement('div');
        div.id = 'popupWarning';
        div.style.cssText = 'position:fixed;top:0;left:0;width:100%;height:100%;background:rgba(0,0,0,0.6);z-index:9999;display:flex;align-items:center;justify-content:center;';
    div.innerHTML = '<div style="background:#fff;border:1px solid #d9534f;border-radius:4px;padding:40px;max-width:500px;text-align:center;box-shadow:0 4px 16px rgba(0,0,0,0.2);">' +
        '<div style="font-size:48px;color:#d9534f;margin-bottom:20px;">&#9888;</div>' +
        '<div style="font-size:22px;color:#333;margin-bottom:15px;">播放窗口被浏览器阻止！</div>' +
        '<div style="font-size:16px;color:#666;margin-bottom:20px;line-height:1.8;">请在浏览器地址栏右侧点击弹窗阻止图标，<br>选择"始终允许来自此网站的弹出式窗口"，<br>然后刷新页面重试。</div>' +
        '<button onclick="document.getElementById(\'popupWarning\').remove()" style="padding:10px 30px;background:linear-gradient(180deg,#428bca,#337ab9);border:1px solid #2a6496;border-radius:4px;color:#fff;font-size:16px;cursor:pointer;box-shadow:0 2px 4px rgba(0,0,0,0.2);">我知道了</button>' +
            '</div>';
        document.body.appendChild(div);
    }

    function openPlayer(targetPage) {
        // 已整合：音频用 /audio-player，视频用 /player，同一个窗口（ktvPlayer）
        targetPage = targetPage || '/player';
        if(!playerWin || playerWin.closed){
            playerWin = window.open(targetPage, 'ktvPlayer', 'width=1280,height=720,menubar=no,toolbar=no,location=no,status=no');
            if(!playerWin || playerWin.closed){
                showPopupBlockedWarning();
            } else {
                // 同步当前队列和会话ID给新打开的播放器窗口（修复：不点扫码按钮切歌时播放器无二维码）
                setTimeout(function() {
                    if (playerWin && !playerWin.closed) {
                        playerWin.postMessage({action:"syncQueue",list:queue,currentPlayingIndex:currentPlayingIndex,sessionId:mySessionId},'*');
                    }
                }, 1500);
            }
        }
    }

    function openAudioPlayer() {
        // 已整合：音频走 /audio-player，视频走 /player，同一个窗口
        openPlayer('/audio-player');
    }

    var settingsWin = null;
    function openSettings() {
        if(!settingsWin || settingsWin.closed){
            settingsWin = window.open('/settings', 'ktvSettings', 'width=600,height=500,menubar=no,toolbar=no,location=no,status=no');
        }
    }

    var uploadWin = null;
    function openUpload() {
        if(!uploadWin || uploadWin.closed){
            uploadWin = window.open('/upload', 'ktvUpload', 'width=700,height=550,menubar=no,toolbar=no,location=no,status=no');
            if(!uploadWin || uploadWin.closed){
                showPopupBlockedWarning();
            }
        }
    }

    var missingWin = null;
    function openMissing() {
        if(!missingWin || missingWin.closed){
            missingWin = window.open('/missing', 'ktvMissing', 'width=500,height=550,menubar=no,toolbar=no,location=no,status=no');
        }
    }

    function showQRCode() {
        fetch('/api/qr/status').then(r=>r.json()).then(data => {
            if (!data.enabled) {
                alert('扫码点歌未启用，请在设置中配置二维码服务器地址');
                return;
            }
            if (!data.connected) {
                alert('未连接到二维码服务器，请检查配置');
                return;
            }
            if (!mySessionId) {
                alert('会话未注册，请刷新页面');
                return;
            }
            // 内置=与主程序同IP同端口；外接=外接二维码服务器地址
            var qrBase = data.qrUrlBase || ('http://' + data.qrServerAddr);
            var qrUrl = qrBase + '/m/' + mySessionId;
            var qrImgUrl = '/api/qr/image?url=' + encodeURIComponent(qrUrl);
            document.getElementById('qrModal').style.display = 'flex';
            document.getElementById('qrImage').src = qrImgUrl;
            document.getElementById('qrSessionId').textContent = '会话ID: ' + mySessionId;
            document.getElementById('qrUrl').textContent = qrUrl;
            document.getElementById('qrStatus').textContent = '已连接';
            document.getElementById('qrStatus').style.color = '#5cb85c';
        }).catch(e => {
            alert('获取二维码状态失败');
        });
    }

    function closeQRModal() {
        document.getElementById('qrModal').style.display = 'none';
    }

    var qrPollInterval = null;

    function startQRPoll() {
        if (qrPollInterval) return;
        qrPollInterval = setInterval(function() {
            if (!mySessionId) return;
            fetch('/api/qr/pending-songs?sessionId=' + encodeURIComponent(mySessionId)).then(r=>r.json()).then(data => {
                if (data.songs && data.songs.length > 0) {
                    data.songs.forEach(function(song) {
                        addToQueue(song.path, song.name, song.type);
                    });
                }
            });
            // 轮询手机端下发的遥控指令（切歌/重唱/播放暂停/音量）并执行
            fetch('/api/qr/control?sessionId=' + encodeURIComponent(mySessionId)).then(r=>r.json()).then(data => {
                if (data && data.controls && data.controls.length > 0) {
                    data.controls.forEach(function(cmd) {
                        executeRemoteControl(cmd);
                    });
                }
            }).catch(function(){});
        }, 2000);
    }

    // 执行手机端遥控指令
    function executeRemoteControl(cmd) {
        var action = cmd.action;
        if (action === 'next') {
            nextSong();
        } else if (action === 'restart') {
            if (playerWin && !playerWin.closed) {
                playerWin.postMessage({action:"restart"},'*');
            }
        } else if (action === 'togglePause') {
            if (playerWin && !playerWin.closed) {
                playerWin.postMessage({action:"togglePause"},'*');
            }
        } else if (action === 'volume') {
            var v = parseInt(cmd.value, 10);
            if (isFinite(v) && v >= 0 && v <= 100 && playerWin && !playerWin.closed) {
                // 更新本页音量滑块显示
                var volSlider = document.getElementById('volumeSlider');
                if (volSlider) volSlider.value = v;
                playerWin.postMessage({action:"setVolume", value:v},'*');
            }
        } else if (action === 'seek') {
            // 快进10秒
            var secs = cmd.value ? parseInt(cmd.value,10) : 10;
            if (playerWin && !playerWin.closed) {
                playerWin.postMessage({action:"seek", seconds:secs},'*');
            }
        } else if (action === 'track') {
            // 音轨/声道切换（原唱/伴奏 或 立体声/左/右），index 0/1/2
            var idx = parseInt(cmd.value, 10);
            if (!isNaN(idx) && playerWin && !playerWin.closed) {
                playerWin.postMessage({action:"switchTrack", index:idx},'*');
            }
        }
    }

    function stopQRPoll() {
        if (qrPollInterval) { clearInterval(qrPollInterval); qrPollInterval = null; }
    }

    function ensurePlayer(isAudio) {
        // 已整合：音频走 /audio-player，视频走 /player，同一个窗口（ktvPlayer）
        // 仅在窗口未开时打开；页面切换由 playQueueItem 通过 URL 参数完成
        if(!playerWin || playerWin.closed){
            var targetPage = isAudio ? '/audio-player' : '/player';
            openPlayer(targetPage);
            return false;
        }
        return true;
    }

    function getActivePlayerWin(item) {
        // 已整合：始终返回 playerWin
        return playerWin;
    }

    function switchTrack(trackIndex){
        lastTrackIndex = trackIndex;
        var o = document.getElementById("btnOrigin");
        var a = document.getElementById("btnAcc");
        if(trackIndex === 0){
            o.classList.add("active");
            a.classList.remove("active");
        }else{
            a.classList.add("active");
            o.classList.remove("active");
        }
        // 省流模式下，切换音轨需要重新播放（play消息已带新trackIndex，不再发switchTrack）
        if(isStreamMode() && currentPlayingIndex >= 0 && currentPlayingIndex < queue.length){
            playQueueItem(currentPlayingIndex);
            return;
        }
        // 音乐模式下，切换音轨需要重新播放（重新抽取音轨）
        if(isMusicMode() && currentPlayingIndex >= 0 && currentPlayingIndex < queue.length){
            playQueueItem(currentPlayingIndex);
            return;
        }
        if(playerWin && !playerWin.closed){
            playerWin.postMessage({action:"switchTrack",index:trackIndex},'*');
        }
    }

    function playNow(path,name,type){
        queue = [];
        currentPlayingIndex = -1;
        var queueItem = {path:path,name:name,type:type,status:"checking",transcodeProgress:0,requestKey:path};
        queue.push(queueItem);
        renderQueue();
        saveQueueToStorage();
        var isAudio = isAudioFile(name) || isMusicMode();
        ensurePlayer(isAudio);
        checkAndRequestTranscode(0);
        // 统计点播次数
        fetch('/api/increment-play?name=' + encodeURIComponent(path), {method:'POST'}).catch(function(){});
    }

    function addToQueue(path,name,type,displayName,insertNext){
        // 已点歌曲去重检查（按名称）
        for(var i=0;i<queue.length;i++){
            if(queue[i].name === name){
                showToast('队列中已有: ' + (displayName||name).replace(/\.[^.]+$/, ''));
                return;
            }
        }
        var songName = (displayName||name).replace(/\.[^.]+$/, '');
        showStatus('📋 加入队列: ' + songName);
        var queueItem = {path:path,name:name,type:type,displayName:displayName||'',status:"checking",transcodeProgress:0,requestKey:path};
        if(insertNext && currentPlayingIndex >= 0 && currentPlayingIndex < queue.length - 1){
            queue.splice(currentPlayingIndex + 1, 0, queueItem);
        } else {
            queue.push(queueItem);
        }
        renderQueue();
        saveQueueToStorage();

        var newIdx = insertNext && currentPlayingIndex >= 0 ? currentPlayingIndex + 1 : queue.length - 1;

        if(queue.length === 1 || (currentPlayingIndex === -1)){
            var isAudio = isAudioFile(name) || isMusicMode();
            ensurePlayer(isAudio);
        }

        checkAndRequestTranscode(newIdx);
        // 统计点播次数
        fetch('/api/increment-play?name=' + encodeURIComponent(path), {method:'POST'}).catch(function(){});
    }

    function checkAndRequestTranscode(idx){
        if(idx < 0 || idx >= queue.length) return;
        var item = queue[idx];

        // 省流模式下，视频文件无需预转码，直接标记ready
        if(isStreamMode() && !isAudioFile(item.name)){
            item.status = "ready";
            renderQueue();
            // 延迟触发自动播放，等待播放器窗口加载完成
            setTimeout(tryAutoPlay, 600);
            return;
        }

        // 音乐模式下，视频文件抽取音频流播放，无需预转码
        if(isMusicMode() && !isAudioFile(item.name)){
            item.status = "ready";
            renderQueue();
            setTimeout(tryAutoPlay, 600);
            return;
        }

        // 需要音频转码的文件（ape/wma等），实时转码，无需预转码
        if(isAudioFile(item.name) && needsAudioTranscode(item.name)){
            item.status = "ready";
            renderQueue();
            setTimeout(tryAutoPlay, 600);
            return;
        }

        // 启动磁盘休眠状态轮询：如果后端检测到磁盘休眠，显示提示
        if (window._diskSleepTimer) clearInterval(window._diskSleepTimer);
        window._diskSleepTimer = setInterval(function(){
            var sxhr = new XMLHttpRequest();
            sxhr.open('GET', '/api/disk-status', true);
            sxhr.onload = function(){
                if (sxhr.status === 200) {
                    try {
                        var sdata = JSON.parse(sxhr.responseText);
                        if (sdata.waking) {
                            showStatus('💤 硬盘已休眠，正在等待硬盘唤醒响应... (' + (sdata.elapsed||0) + 'ms)');
                        }
                    } catch(e) {}
                }
            };
            sxhr.send();
        }, 500);
        function stopDiskSleepPolling(){
            if (window._diskSleepTimer) { clearInterval(window._diskSleepTimer); window._diskSleepTimer = null; }
        }

        var xhr = new XMLHttpRequest();
        xhr.open('POST', '/api/transcode/check-and-add', true);
        xhr.setRequestHeader('Content-Type', 'application/json');
        xhr.onload = function() {
            stopDiskSleepPolling();
            if (xhr.status === 200) {
                try {
                    var data = JSON.parse(xhr.responseText);
                    if(data.needsTranscode){
                        item.status = data.status;
                        item.queuePosition = data.queuePosition;
                        if(data.codecInfo){
                            item.codecInfo = data.codecInfo;
                        }
                        showStatus('🔄 转码中: ' + (item.displayName||item.name).replace(/\.[^.]+$/, '') + (data.codecInfo ? ' (' + data.codecInfo + ')' : ''));
                    } else {
                        item.status = "ready";
                        showStatus('✅ 就绪: ' + (item.displayName||item.name).replace(/\.[^.]+$/, ''), 3000);
                    }
                    renderQueue();

                    if(item.status === "ready"){
                        tryAutoPlay();
                    }
                } catch(e) {
                    item.status = "ready";
                    renderQueue();
                    tryAutoPlay();
                }
            } else {
                item.status = "ready";
                renderQueue();
                tryAutoPlay();
            }
        };
        xhr.onerror = function(){
            stopDiskSleepPolling();
            item.status = "ready";
            renderQueue();
            tryAutoPlay();
        };
        xhr.send(JSON.stringify({fileName: item.path, requestKey: item.requestKey}));
    }

    function tryAutoPlay(){
        if(currentPlayingIndex >= 0 && currentPlayingIndex < queue.length) return;

        for(var i = 0; i < queue.length; i++){
            if(queue[i].status === "ready"){
                currentPlayingIndex = i;
                renderQueue();
                showStatus('▶ 播放: ' + (queue[i].displayName||queue[i].name).replace(/\.[^.]+$/, ''), 3000);
                playQueueItem(i);
                return;
            }
        }
    }

    function renderQueue(){
        var box = document.getElementById("queueList");
        var html = "";
        for(var i=0;i<queue.length;i++){
            var item = queue[i];
            var isCurrent = (i === currentPlayingIndex);
            html += '<div class="queue-item" style="' + (isCurrent ? 'border-left:3px solid #5cb85c;background:#243447;' : '') + '">';
            html += '<div style="flex: 1;">';
            html += '<span>'+(i+1)+'. '+(item.displayName||item.name)+'</span>';
            if(isCurrent){
                html += ' <span style="color:#5cb85c; margin-left: 8px;">正在播放</span>';
            } else if(item.status === "transcoding"){
                html += '<div class="transcode-progress">';
                html += '<div class="transcode-progress-bar"><div class="transcode-progress-fill" style="width:'+item.transcodeProgress+'%"></div></div>';
                html += '<span>正在转码: '+item.transcodeProgress+'%</span>';
                if(item.codecInfo){
                    html += ' <span style="color:#aaa; font-size:11px;">('+item.codecInfo+')</span>';
                }
                html += '</div>';
            } else if(item.status === "waiting"){
                html += '<span style="color:#f0ad4e; margin-left: 8px;">等待转码</span>';
                if(item.codecInfo){
                    html += ' <span style="color:#aaa; font-size:11px;">('+item.codecInfo+')</span>';
                }
                if(item.queuePosition !== undefined && item.queuePosition > 0){
                    html += ' <span style="color:#5bc0de;">(排队: 第'+item.queuePosition+'首)</span>';
                }
            } else if(item.status === "checking"){
                html += '<span style="color:#ccc; margin-left: 8px;">检查中...</span>';
            } else if(item.status === "ready"){
                if(!isCurrent){
                    html += '<span style="color:#5bc0de; margin-left: 8px;">已就绪</span>';
                }
            }
            // 显示轨道异常警告
            if(item.trackWarning){
                html += '<div style="color:#fff;font-size:12px;font-weight:bold;background:#d9534f;border:1px solid #c9302c;border-radius:3px;padding:2px 6px;margin-top:2px;">';
                if(item.trackWarning.noVideo) html += '🎬无画面 ';
                if(item.trackWarning.noAudio) html += '🔊无声音 ';
                html += item.trackWarning.message + '</div>';
            }
            html += '</div>';
            if(i !== currentPlayingIndex && i > 0){
                html += '<button class="top-btn" onclick="topQueue('+i+')" title="置顶到下一首">置顶</button>';
            }
            html += '<button class="del-btn" onclick="delQueue('+i+')">删除</button>';
            html += '</div>';
        }
        box.innerHTML = html;
        if(playerWin && !playerWin.closed){
            playerWin.postMessage({action:"syncQueue",list:queue,currentPlayingIndex:currentPlayingIndex,sessionId:mySessionId},'*');
        }
        // Send queue update to QR server
        fetch('/api/qr/queue-update', {
            method: 'POST',
            headers: {'Content-Type': 'application/json'},
            body: JSON.stringify({queue: queue.map(function(item) {
                return {name: item.name, status: item.status};
            }), currentPlayingIndex: currentPlayingIndex, sessionId: mySessionId})
        }).catch(function(){});
    }

    function topQueue(idx){
        var item = queue.splice(idx, 1)[0];
        var targetIdx = currentPlayingIndex >= 0 ? currentPlayingIndex + 1 : 0;
        queue.splice(targetIdx, 0, item);
        // 修正 currentPlayingIndex
        if(idx < currentPlayingIndex){
            currentPlayingIndex--;
        }
        // 现在 item 在 targetIdx（已调整后的位置）
        renderQueue();
        saveQueueToStorage();
    }

    function delQueue(idx){
        var wasPlaying = (idx === currentPlayingIndex);
        queue.splice(idx,1);
        if(wasPlaying){
            currentPlayingIndex = -1;
            tryAutoPlay();
        } else if(idx < currentPlayingIndex){
            currentPlayingIndex--;
        }
        renderQueue();
        if(queue.length === 0) clearQueueStorage(); else saveQueueToStorage();
    }

    function playQueueItem(idx){
        if(idx<0||idx>=queue.length) return;
        var item = queue[idx];
        var isAudio = isAudioFile(item.name);

        // 播放前检查文件轨道完整性（非音频文件才检查）
        if(!isAudio){
            checkAndWarnTracks(item.name, item.path, idx);
        }

        // 构建播放URL
        var url;
        if(isAudio){
            if(needsAudioTranscode(item.name)){
                url = '/api/music-mode-stream?name='+encodeURIComponent(item.path)+'&trackIndex=0&_t='+Date.now();
            } else {
                url = '/file?name='+encodeURIComponent(item.path);
            }
        } else if(isMusicMode()){
            // 音乐模式：视频文件抽取音频流，按音频页面处理
            url = '/api/music-mode-stream?name='+encodeURIComponent(item.path)+'&trackIndex='+lastTrackIndex+'&_t='+Date.now();
            isAudio = true;
        } else if(isStreamMode()){
            // 省流模式：使用流媒体实时转码
            url = '/api/stream?name='+encodeURIComponent(item.path)+'&trackIndex='+lastTrackIndex+'&quality=high&_t='+Date.now();
        } else {
            url = '/file?name='+encodeURIComponent(item.path);
        }

        var targetPage = isAudio ? '/audio-player' : '/player';

        // 判断是否需要切换页面：窗口未开/已关、当前页面不符、或页面仍在加载（刚打开）
        var needSwitch = false;
        if(!playerWin || playerWin.closed){
            needSwitch = true;
        } else {
            var currentPath = '';
            try {
                currentPath = playerWin.location.pathname;
                // 页面仍在加载（刚由 ensurePlayer 打开），用 URL 参数重定向以携带播放信息
                if(playerWin.document && playerWin.document.readyState !== 'complete') needSwitch = true;
            } catch(e) { needSwitch = true; }
            if(currentPath !== targetPage) needSwitch = true;
        }

        if(needSwitch){
            // 切换/打开页面，通过 URL 参数携带播放信息（避免 postMessage 在页面加载前丢失）
            var openUrl = targetPage + '?playUrl=' + encodeURIComponent(url)
                        + '&playName=' + encodeURIComponent(item.name)
                        + '&playPath=' + encodeURIComponent(item.path);
            if(!playerWin || playerWin.closed){
                playerWin = window.open(openUrl, 'ktvPlayer', 'width=1280,height=720,menubar=no,toolbar=no,location=no,status=no');
                if(!playerWin || playerWin.closed){
                    showPopupBlockedWarning();
                }
            } else {
                playerWin.location.replace(openUrl);
            }
            // 1.5s 后同步队列（新页面 message listener 应已就绪）
            setTimeout(function(){
                if(playerWin && !playerWin.closed){
                    playerWin.postMessage({action:"syncQueue",list:queue,currentPlayingIndex:currentPlayingIndex,sessionId:mySessionId},'*');
                }
            }, 1500);
            renderQueue();
            return;
        }

        // 同页面且已就绪，直接 postMessage play
        playerWin.postMessage({action:"play",url:url,type:item.type,name:item.name,path:item.path},'*');
        renderQueue();
    }

    // 检查文件轨道完整性，有异常时在播放列表中显著提示
    var trackWarningCache = {};
    function checkAndWarnTracks(songName, songPath, idx){
        // 使用缓存避免重复请求
        if(trackWarningCache[songPath] !== undefined) return;
        trackWarningCache[songPath] = null; // 标记已查询

        fetch('/api/check-tracks?name='+encodeURIComponent(songPath))
            .then(function(r){ return r.json(); })
            .then(function(data){
                if(data && data.message){
                    trackWarningCache[songPath] = data;
                    // 在播放列表中显示警告
                    var el = document.getElementById('queue-item-'+idx);
                    if(el){
                        var badge = el.querySelector('.track-warning');
                        if(!badge){
                            badge = document.createElement('div');
                            badge.className = 'track-warning';
                            badge.style.cssText = 'color:#fff;font-size:12px;font-weight:bold;padding:2px 6px;background:#d9534f;border:1px solid #c9302c;border-radius:3px;margin-top:2px;display:inline-block;';
                            el.appendChild(badge);
                        }
                        var icon = data.noVideo ? '🎬❌' : '';
                        icon += data.noAudio ? '🔊❌' : '';
                        badge.textContent = icon + ' ' + data.message;
                    }
                    // 控制台也输出
                    console.warn('[轨道异常] ' + songName + ': ' + data.message);
                }
            })
            .catch(function(){});
    }

    function prevSong(){
        if(queue.length===0)return;
        currentPlayingIndex = 0;
        playQueueItem(0);
        renderQueue();
        saveQueueToStorage();
    }

    function nextSong(){
        if(queue.length === 0) return;
        if(currentPlayingIndex >= 0 && currentPlayingIndex < queue.length){
            queue.splice(currentPlayingIndex, 1);
        }
        currentPlayingIndex = -1;

        // 从队列最前面扫描下一首可播放的歌曲
        var nextIdx = -1;
        for(var i = 0; i < queue.length; i++){
            if(queue[i].status === "ready"){
                nextIdx = i;
                break;
            }
        }

        if(nextIdx >= 0){
            currentPlayingIndex = nextIdx;
            showStatus('⏭ 切歌: ' + (queue[nextIdx].displayName||queue[nextIdx].name).replace(/\.[^.]+$/, ''), 3000);
            playQueueItem(nextIdx);
        }

        renderQueue();
        if(queue.length === 0) clearQueueStorage(); else saveQueueToStorage();
    }

    function randomSong(){
        fetch('/api/random-song').then(function(r){return r.json();}).then(function(data){
            if(data.success){
                addToQueue(data.path, data.name, data.type);
            } else {
                alert('曲库为空');
            }
        }).catch(function(){
            alert('获取随机歌曲失败');
        });
    }

    function sendToPlayer(msg){
        // 已整合：所有消息统一发送给 playerWin（其内部 iframe 会处理音频播放）
        if(playerWin && !playerWin.closed){
            playerWin.postMessage(msg,'*');
        }
    }

    function loadSongs(page, keyword) {
        var songList = document.getElementById("songList");
        songList.innerHTML = '<div class="loading">加载中...</div>';

        var xhr = new XMLHttpRequest();
        var url = '/api/songs?page=' + page + '&pageSize=' + pageSize;
        if (keyword) url += '&keyword=' + encodeURIComponent(keyword);

        xhr.open('GET', url, true);
        xhr.onload = function() {
            if (xhr.status === 200) {
                var data = JSON.parse(xhr.responseText);
                renderSongList(data.songs);
                totalItems = data.total;
                totalPages = data.totalPages;
                currentPage = data.page;
                updatePagination();
            } else {
                songList.innerHTML = '<div class="loading">加载失败</div>';
            }
        };
        xhr.onerror = function() {
            songList.innerHTML = '<div class="loading">加载失败</div>';
        };
        xhr.send();
    }

    function renderSongList(songs) {
        var songList = document.getElementById("songList");
        if (songs.length === 0) {
            songList.innerHTML = '<div class="loading">暂无曲目</div>';
            return;
        }

        var html = '';
        for (var i = 0; i < songs.length; i++) {
            var song = songs[i];
            var showName = song.displayName || song.name;
            var fontSize = calculateFontSize(showName);
            html += '<div class="song-item" data-path="' + song.path + '" data-type="' + song.type + '" onclick="addToQueue(\'' + song.path + '\',\'' + song.name + '\',\'' + song.type + '\',\'' + (showName !== song.name ? showName.replace(/'/g, "\\'") : '') + '\')">';
            html += '<span class="song-name" style="font-size:' + fontSize + 'px;">' + showName + '</span>';
            html += '</div>';
        }
        songList.innerHTML = html;
        // 显示点播次数（从热播数据中匹配）
        showPlayCounts(songs);
    }

    function calculateFontSize(text) {
        var baseSize = 16;
        var minSize = 10;
        var maxLines = 3;

        var songList = document.getElementById("songList");
        var buttonWidth = 180;

        if (songList) {
            var computedStyle = window.getComputedStyle(songList);
            var gap = parseInt(computedStyle.gap) || 8;
            var padding = parseInt(computedStyle.paddingLeft) || 6;

            var listWidth = songList.offsetWidth;
            var columns = Math.max(1, Math.floor(listWidth / (buttonWidth + gap)));

            if (columns > 0 && listWidth > 0) {
                buttonWidth = (listWidth - padding * 2 - gap * (columns - 1)) / columns - 12;
            }

            if (buttonWidth < 100) {
                buttonWidth = 100;
            }
        }

        var canvas = document.createElement('canvas');
        var ctx = canvas.getContext('2d');

        ctx.font = baseSize + 'px Microsoft YaHei';
        var textWidth = ctx.measureText(text).width;

        var availableWidth = buttonWidth * maxLines;

        if (textWidth <= availableWidth) {
            return baseSize;
        }

        var scaleFactor = availableWidth / textWidth;
        var fontSize = baseSize * scaleFactor;

        return Math.max(fontSize, minSize);
    }

    var searchTimeout = null;

    function filterList(){
        if (searchTimeout) {
            clearTimeout(searchTimeout);
            searchTimeout = null;
        }
        currentKeyword = document.getElementById("search").value;
        currentPage = 1;
        loadSongs(currentPage, currentKeyword);
    }

    function handleSearchKeydown(event) {
        if (event.key === "Enter") {
            filterList();
        }
    }

    function handleSearchInput() {
        if (searchTimeout) {
            clearTimeout(searchTimeout);
        }
        searchTimeout = setTimeout(function() {
            filterList();
        }, 4000);
    }

    function updatePagination(){
        var pagination = document.getElementById("pagination");

        document.getElementById("pageInfo").textContent = '当前页: ' + currentPage + '/' + totalPages;
        document.getElementById("totalInfo").textContent = '总曲目: ' + totalItems;

        if(totalPages <= 1){
            pagination.innerHTML = '';
            return;
        }

        var html = '';

        html += '<button class="page-btn" onclick="changePage(' + (currentPage-1) + ')" ' + (currentPage === 1 ? 'disabled' : '') + '>上一页</button>';

        var maxButtons = 10;
        if(totalPages <= maxButtons){
            for(var i=1;i<=totalPages;i++){
                html += '<button class="page-btn ' + (i === currentPage ? 'active' : '') + '" onclick="changePage(' + i + ')">' + i + '</button>';
            }
        } else {
            var step = Math.ceil(totalPages / maxButtons);
            for(var i=0;i<maxButtons;i++){
                var pageNum = (i + 1) * step;
                if(pageNum > totalPages) pageNum = totalPages;
                html += '<button class="page-btn ' + (pageNum === currentPage ? 'active' : '') + '" onclick="changePage(' + pageNum + ')">' + pageNum + '</button>';
            }
        }

        html += '<button class="page-btn" onclick="changePage(' + (currentPage+1) + ')" ' + (currentPage === totalPages ? 'disabled' : '') + '>下一页</button>';

        pagination.innerHTML = html;
    }

    function changePage(page){
        if(page < 1 || page > totalPages) return;
        currentPage = page;
        loadSongs(currentPage, currentKeyword);
    }

    window.addEventListener("message",function(e){
        if(e.data.action==="ended"){
            nextSong();
        } else if(e.data.action==="nextSong"){
            nextSong();
        } else if(e.data.action==="playByName"){
            playSongByName(e.data.name);
        } else if(e.data.action==="switchTrack"){
            // 播放器页面切换音轨（省流模式下由播放器发起）
            switchTrack(e.data.index);
        } else if(e.data.action==="trackMode"){
            // 播放器上报当前音轨模式（track/channel）与声道数，转发给手机端遥控显示对应按钮
            var tMode = e.data.mode || 'track';
            var tCh = parseInt(e.data.channels, 10);
            if (isNaN(tCh) || tCh <= 0) tCh = 2;
            var qs = '/api/qr/track-state?mode=' + encodeURIComponent(tMode) + '&channels=' + tCh;
            var qxhr = new XMLHttpRequest();
            qxhr.open('GET', qs, true);
            qxhr.send();
        }
    });

    function playSongByName(fileName) {
        var xhr = new XMLHttpRequest();
        xhr.open('GET', '/api/songs/search?name=' + encodeURIComponent(fileName), true);
        xhr.onload = function() {
            if (xhr.status === 200) {
                var data = JSON.parse(xhr.responseText);
                if (data && data.length > 0) {
                    var song = data[0];
                    playNow(song.path, song.name, song.type);
                }
            }
        };
        xhr.send();
    }

    function calculateOptimalLayout() {
        const songList = document.getElementById('songList');
        if (!songList) return;

        const rect = songList.getBoundingClientRect();
        const containerWidth = rect.width;
        const containerHeight = rect.height;

        const buttonWidth = 200 + 8;
        const buttonHeight = 60 + 8;

        columns = Math.floor(containerWidth / buttonWidth);
        rows = Math.floor(containerHeight / buttonHeight);

        columns = Math.max(1, columns);
        rows = Math.max(1, rows);

        pageSize = columns * rows;

        songList.style.gridTemplateColumns = "repeat(" + columns + ", minmax(200px, 1fr))";
        songList.style.gridTemplateRows = "repeat(" + rows + ", 60px)";

        if (totalItems > 0) {
            filterList();
        }
    }

    function pollTranscodeStatus() {
        var needPoll = false;
        for(var i = 0; i < queue.length; i++){
            if(queue[i].status === "waiting" || queue[i].status === "transcoding"){
                needPoll = true;
                break;
            }
        }
        if(!needPoll) return;

        var keysToCheck = [];
        for(var i = 0; i < queue.length; i++){
            if(queue[i].status === "waiting" || queue[i].status === "transcoding"){
                keysToCheck.push({index: i, requestKey: queue[i].requestKey});
            }
        }

        for(var k = 0; k < keysToCheck.length; k++){
            (function(idx, key){
                var xhr = new XMLHttpRequest();
                xhr.open('GET', '/api/transcode/status?requestKey=' + encodeURIComponent(key), true);
                xhr.onload = function() {
                    if (xhr.status === 200) {
                        try {
                            var data = JSON.parse(xhr.responseText);
                            console.log('[转码轮询] idx=' + idx + ' status=' + data.status + ' outputPath=' + (data.outputPath||'') + ' progress=' + data.progress);
                            if(idx < queue.length && queue[idx].requestKey === key){
                                queue[idx].status = data.status;
                                queue[idx].transcodeProgress = data.progress;
                                queue[idx].queuePosition = data.queuePosition;
                                if(data.codecInfo){
                                    queue[idx].codecInfo = data.codecInfo;
                                }
                                if(data.trackWarning){
                                    queue[idx].trackWarning = data.trackWarning;
                                }
                                if(data.outputPath && data.status === "completed"){
                                    queue[idx].path = data.outputPath;
                                    queue[idx].name = data.outputPath.split('/').pop();
                                    queue[idx].status = "ready";
                                    renderQueue();
                                    tryAutoPlay();
                                } else {
                                    renderQueue();
                                    if(data.status === "ready"){
                                        tryAutoPlay();
                                    }
                                }
                            }
                        } catch(e) {}
                    }
                };
                xhr.send();
            })(keysToCheck[k].index, keysToCheck[k].requestKey);
        }
    }

    function detectBrowser() {
        const userAgent = navigator.userAgent;
        if (userAgent.includes('Edg')) {
            return 'edge';
        } else if (userAgent.includes('Chrome') && !userAgent.includes('Edg')) {
            return 'chrome';
        } else if (userAgent.includes('Firefox')) {
            return 'firefox';
        } else if (userAgent.includes('Safari') && !userAgent.includes('Chrome')) {
            return 'safari';
        }
        return 'other';
    }

    function checkAudioTrackSupport() {
        const video = document.createElement('video');
        return video.audioTracks !== undefined && typeof video.audioTracks === 'object';
    }

    var selfCheckData = null;

    // renderWelcomePage 渲染欢迎页面（自检信息 + 音轨提示）
    function renderWelcomePage() {
        var navPage = document.getElementById('navPage');
        var mainPage = document.getElementById('mainPage');

        // 拉取自检数据
        fetch('/api/selfcheck').then(function(r){return r.json();}).then(function(data){
            selfCheckData = data;
            renderSelfCheckInfo(data);

            // 音轨切换检测
            if (!checkAudioTrackSupport()) {
                renderTrackNotice();
            }

            // 始终显示欢迎页面（让用户了解系统状态）
            navPage.classList.add('active');
            mainPage.style.display = 'none';

            // 如果有 critical 问题，修改跳过按钮样式强调
            if (data.overallStatus === 'critical') {
                var btn = document.getElementById('navSkipBtn');
                btn.className = 'nav-skip nav-skip-critical';
                btn.textContent = '仍有问题，强制进入';
            }
        }).catch(function(){
            // 自检接口失败，降级：仅检查音轨
            if (!checkAudioTrackSupport()) {
                renderTrackNotice();
                navPage.classList.add('active');
                mainPage.style.display = 'none';
            }
        });
    }

    // renderSelfCheckInfo 渲染自检信息卡片
    function renderSelfCheckInfo(data) {
        var container = document.getElementById('navSelfCheckContent');
        var subtitle = document.getElementById('navSubtitle');
        var html = '';

        // 更新副标题
        if (data.overallStatus === 'ok') {
            subtitle.innerHTML = '<span class="nav-status-ok">系统状态正常</span> | ' + data.arch;
        } else if (data.overallStatus === 'warning') {
            subtitle.innerHTML = '<span class="nav-status-warn">系统存在警告</span> | ' + data.arch;
        } else {
            subtitle.innerHTML = '<span class="nav-status-err">系统存在问题，请查看下方详情</span> | ' + data.arch;
        }

        // --- 阻断性问题（高亮显示在最上方） ---
        if (data.blockingIssues && data.blockingIssues.length > 0) {
            html += '<div class="nav-section">';
            html += '<div class="nav-section-title" style="color:#d9534f;border-bottom-color:#d9534f">需要处理的问题</div>';
            for (var i = 0; i < data.blockingIssues.length; i++) {
                var iss = data.blockingIssues[i];
                var icon = iss.level === 'critical' ? '<span class="nav-status-err">&#9888;</span>' : '<span class="nav-status-warn">&#9888;</span>';
                html += '<div class="nav-card" style="border-left:3px solid ' + (iss.level === 'critical' ? '#d9534f' : '#f0ad4e') + '">';
                html += '<div class="nav-card-icon">' + icon + '</div>';
                html += '<div class="nav-card-body">';
                html += '<div class="nav-card-title">' + escHtml(iss.title) + '</div>';
                html += '<div class="nav-card-detail">' + escHtml(iss.detail) + '</div>';
                if (iss.action) {
                    html += '<div class="nav-card-detail" style="color:#ccc;margin-top:4px">解决: ' + escHtml(iss.action) + '</div>';
                }
                html += '<div class="nav-card-action">';
                if (iss.actionType === 'settings') {
                    html += '<button class="nav-action-settings" onclick="openSettingsFromWelcome(\'' + escHtml(iss.actionURL) + '\')">打开设置</button>';
                } else if (iss.actionType === 'link' && iss.actionURL) {
                    html += '<a class="nav-action-link" href="' + escHtml(iss.actionURL) + '" target="_blank">下载/查看</a>';
                }
                html += '</div>';
                html += '</div></div>';
            }
            html += '</div>';
        }

        // --- FFmpeg / FFprobe ---
        html += '<div class="nav-section">';
        html += '<div class="nav-section-title">依赖组件</div>';
        // FFmpeg
        html += renderDepCard('FFmpeg', data.ffmpeg.found, data.ffmpeg.version || '', data.ffmpeg.path);
        // FFprobe
        html += renderDepCard('FFprobe', data.ffprobe.found, data.ffprobe.version || '', data.ffprobe.path);
        html += '</div>';

        // --- GPU 加速 ---
        html += '<div class="nav-section">';
        html += '<div class="nav-section-title">GPU 加速</div>';
        if (data.gpu.isGPU) {
            html += '<div class="nav-card"><div class="nav-card-icon"><span class="nav-status-ok">&#10003;</span></div>';
            html += '<div class="nav-card-body"><div class="nav-card-title">GPU 编码器: ' + escHtml(data.gpu.detectedEncoder) + '</div>';
            html += '<div class="nav-card-detail">GPU 硬件加速已启用，转码速度更快</div>';
            html += '</div></div>';
        } else {
            html += '<div class="nav-card"><div class="nav-card-icon"><span style="color:#7f8c8d">&#9432;</span></div>';
            html += '<div class="nav-card-body"><div class="nav-card-title">CPU 编码 (libx264)</div>';
            html += '<div class="nav-card-detail">GPU 加速不可用，将使用 CPU 编码（速度较慢但功能正常）</div>';
            html += '</div></div>';
        }
        // 各编码器详情
        if (data.gpu.encoders && data.gpu.encoders.length > 0) {
            for (var i = 0; i < data.gpu.encoders.length; i++) {
                var enc = data.gpu.encoders[i];
                var encIcon = enc.usable ? '<span class="nav-status-ok">&#10003;</span>' : '<span class="nav-status-err">&#10007;</span>';
                html += '<div class="nav-card" style="padding:8px 16px"><div class="nav-card-icon" style="font-size:14px">' + encIcon + '</div>';
                html += '<div class="nav-card-body"><div class="nav-card-detail">' + escHtml(enc.name) + ': ' + escHtml(enc.detail) + '</div>';
                html += '</div></div>';
            }
        }
        html += '</div>';

        // --- 曲库信息 ---
        html += '<div class="nav-section">';
        html += '<div class="nav-section-title">曲库信息</div>';
        html += '<div class="nav-card"><div class="nav-card-icon"><span style="font-size:18px">&#127925;</span></div>';
        html += '<div class="nav-card-body"><div class="nav-card-title">总曲目: ' + data.totalSongs + ' 首</div>';
        html += '</div></div>';
        for (var i = 0; i < data.mediaDirs.length; i++) {
            var d = data.mediaDirs[i];
            var dIcon, dTitle, dDetail;
            if (!d.exists) {
                dIcon = '<span class="nav-status-err">&#9888;</span>';
                dTitle = escHtml(d.path);
                dDetail = d.errorReason || '不存在';
            } else if (!data.mediaScanDone) {
                // 曲库尚未扫描，文件数未知
                dIcon = '<span style="color:#7f8c8d">&#9432;</span>';
                dTitle = escHtml(d.path);
                dDetail = '文件数量待扫描后统计';
            } else if (d.fileCount === 0) {
                dIcon = '<span class="nav-status-warn">&#9888;</span>';
                dTitle = escHtml(d.path);
                dDetail = '目录为空（无有效音视频文件）';
            } else {
                dIcon = '<span class="nav-status-ok">&#10003;</span>';
                dTitle = escHtml(d.path);
                dDetail = '视频 ' + d.videoCount + ' + 音频 ' + d.audioCount + ' = ' + d.fileCount + ' 首';
            }
            html += '<div class="nav-card" style="padding:8px 16px"><div class="nav-card-icon" style="font-size:14px">' + dIcon + '</div>';
            html += '<div class="nav-card-body"><div class="nav-card-detail" style="word-break:break-all">' + dTitle + ' — ' + dDetail + '</div>';
            html += '</div></div>';
        }
        html += '</div>';

        container.innerHTML = html;
    }

    function renderDepCard(name, found, version, path) {
        var icon = found ? '<span class="nav-status-ok">&#10003;</span>' : '<span class="nav-status-err">&#10007;</span>';
        var html = '<div class="nav-card"><div class="nav-card-icon">' + icon + '</div>';
        html += '<div class="nav-card-body">';
        if (found) {
            html += '<div class="nav-card-title">' + name + ': 已安装</div>';
            html += '<div class="nav-card-detail">' + escHtml(version) + '</div>';
        } else {
            html += '<div class="nav-card-title">' + name + ': 未找到</div>';
            html += '<div class="nav-card-detail">路径: ' + escHtml(path) + '</div>';
            html += '<div class="nav-card-action"><a class="nav-action-link" href="https://ffmpeg.org/download.html" target="_blank">下载 FFmpeg</a></div>';
        }
        html += '</div></div>';
        return html;
    }

    // renderTrackNotice 渲染音轨切换实验功能提示
    function renderTrackNotice() {
        var container = document.getElementById('navTrackNotice');
        var browser = detectBrowser();
        var settingsUrl = '';
        switch(browser) {
            case 'edge': settingsUrl = 'edge://flags/#enable-experimental-web-platform-features'; break;
            case 'chrome': settingsUrl = 'chrome://flags/#enable-experimental-web-platform-features'; break;
            case 'firefox':
            case 'safari':
                container.style.display = 'block';
                container.innerHTML = '<div style="padding:16px;border:2px solid #f0ad4e;border-radius:8px;background:rgba(240,173,78,0.1)">' +
                    '<div style="font-size:16px;font-weight:bold;color:#f0ad4e;margin-bottom:8px">&#9888; 重要提示：音轨切换功能</div>' +
                    '<div style="margin-bottom:8px">当前浏览器(' + browser + ')的音轨切换功能可能受限，建议使用 <strong>Chrome</strong> 或 <strong>Edge</strong> 以获得最佳体验。</div>' +
                    '</div>';
                return;
            default: return;
        }
        container.style.display = 'block';
        var html = '<div style="padding:16px;border:2px solid #f0ad4e;border-radius:8px;background:rgba(240,173,78,0.1)">';
        html += '<div style="font-size:16px;font-weight:bold;color:#f0ad4e;margin-bottom:8px">&#9888; 重要提示：音轨切换功能</div>';
        html += '<div style="margin-bottom:8px">当前浏览器未启用音轨切换实验功能，<strong>原唱/伴奏切换将无法使用</strong>。</div>';
        html += '<div style="margin-bottom:8px"><strong>开启步骤：</strong><br>';
        html += '1. 复制下方地址，粘贴到浏览器地址栏并回车<br>';
        html += '2. 找到 <code>Experimental Web Platform features</code>，设为 <strong>Enabled</strong><br>';
        html += '3. 点击页面底部"Relaunch"重启浏览器</div>';
        html += '<div style="display:flex;gap:8px;align-items:center;margin-top:8px">';
        html += '<input type="text" id="settingsUrlInput" value="' + escHtml(settingsUrl) + '" readonly style="flex:1;padding:6px 8px;border:1px solid #ccc;border-radius:4px;font-size:13px">';
        html += '<button onclick="copySettingsUrl()" style="padding:6px 16px;border:1px solid #2a6496;border-radius:4px;background:#428bca;color:#fff;cursor:pointer;font-size:13px">复制地址</button>';
        html += '</div>';
        html += '<label style="display:flex;align-items:center;gap:8px;margin-top:12px;cursor:pointer;font-size:14px">';
        html += '<input type="checkbox" id="trackNoticeConfirm" onchange="onTrackNoticeConfirm()" style="width:18px;height:18px">';
        html += '<span>我已阅读上述提示并了解如何开启音轨切换功能</span>';
        html += '</label>';
        html += '</div>';
        container.innerHTML = html;
        // 禁用进入按钮，直到用户确认
        var btn = document.getElementById('navSkipBtn');
        btn.disabled = true;
        btn.style.opacity = '0.4';
        btn.style.cursor = 'not-allowed';
        btn.textContent = '请先确认上方提示';
    }

    function onTrackNoticeConfirm() {
        var checkbox = document.getElementById('trackNoticeConfirm');
        var btn = document.getElementById('navSkipBtn');
        if (checkbox && checkbox.checked) {
            btn.disabled = false;
            btn.style.opacity = '1';
            btn.style.cursor = 'pointer';
            btn.textContent = '进入点歌系统';
        } else {
            btn.disabled = true;
            btn.style.opacity = '0.4';
            btn.style.cursor = 'not-allowed';
            btn.textContent = '请先确认上方提示';
        }
    }

    function copySettingsUrl() {
        var urlInput = document.getElementById('settingsUrlInput');
        if (urlInput) {
            urlInput.select();
            document.execCommand('copy');
            showToast('地址已复制！请粘贴到浏览器地址栏并按回车打开。');
        }
    }

    function openSettingsFromWelcome(url) {
        skipNavPage();
        openSettings();
    }

    function escHtml(s) {
        if (!s) return '';
        return String(s).replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;').replace(/"/g,'&quot;');
    }

    function skipNavPage() {
        var btn = document.getElementById('navSkipBtn');
        if (btn && btn.disabled) return; // 禁用时不允许进入
        var navPage = document.getElementById('navPage');
        var mainPage = document.getElementById('mainPage');
        navPage.classList.remove('active');
        mainPage.style.display = 'flex';
    }

    window.onload = function(){
        // 恢复之前的播放队列
        var savedData = loadQueueFromStorage();
        if(savedData && savedData.queue && savedData.queue.length > 0){
            queue = savedData.queue;
            currentPlayingIndex = -1; // 页面重载后播放窗口已不存在，不恢复播放状态
            renderQueue();
            // 对恢复的队列项逐个检查转码状态
            for(var i = 0; i < queue.length; i++){
                checkAndRequestTranscode(i);
            }
        }

        // 始终显示欢迎页面（展示系统自检信息 + 音轨提示）
        renderWelcomePage();

        calculateOptimalLayout();
        loadSongs(1, '');

        transcodePollInterval = setInterval(pollTranscodeStatus, 2000);

        // 注册本页面的独立会话ID
        fetch('/api/qr/register-session').then(r=>r.json()).then(data => {
            if (data.sessionId) {
                mySessionId = data.sessionId;
                console.log('[QR] 会话已注册: ' + mySessionId);
            }
            // 检查QR是否启用，启用则开始轮询
            return fetch('/api/qr/status');
        }).then(r=>r.json()).then(data => {
            if (data && data.enabled) {
                startQRPoll();
            }
        }).catch(function(){});
    };

    function showToast(msg) {
        var el = document.getElementById('toast');
        el.textContent = msg;
        el.className = 'toast show';
        setTimeout(function() { el.className = 'toast'; }, 2000);
    }

    // 状态标签：显示前后端操作进度
    var statusBadgeTimer = null;
    function showStatus(msg, duration) {
        var el = document.getElementById('statusBadge');
        if (!el) return;
        el.textContent = msg;
        el.style.display = 'block';
        if (statusBadgeTimer) clearTimeout(statusBadgeTimer);
        if (duration) {
            statusBadgeTimer = setTimeout(function() { el.style.display = 'none'; }, duration);
        }
    }
    function hideStatus() {
        var el = document.getElementById('statusBadge');
        if (el) el.style.display = 'none';
        if (statusBadgeTimer) clearTimeout(statusBadgeTimer);
    }

    // 歌手/语种/曲种浏览
    var singerData = null;
    var currentSingerLetter = '';
    var currentSingerName = '';
    var languageData = null;
    var categoryData = null;

    function switchBrowseTab(tab) {
        var tabs = ['search','singer','language','category','hot'];
        var panels = {search:'songList',singer:'singerPanel',language:'languagePanel',category:'categoryPanel',hot:'hotPanel'};
        for (var i = 0; i < tabs.length; i++) {
            var t = tabs[i];
            document.getElementById('tab' + t.charAt(0).toUpperCase() + t.slice(1)).className = 'browse-tab' + (t === tab ? ' active' : '');
            var panel = document.getElementById(panels[t]);
            if (panel) panel.style.display = t === tab ? '' : 'none';
        }
        document.getElementById('pagination').style.display = tab === 'search' ? '' : 'none';
        if (tab === 'singer' && !singerData) loadSingerIndex();
        if (tab === 'language' && !languageData) loadLanguageIndex();
        if (tab === 'category' && !categoryData) loadCategoryIndex();
        if (tab === 'hot') loadHotSongs();
    }

    var hotPlayCache = null;
    var hotPlayCacheTime = 0;
    function showPlayCounts(songs) {
        var now = Date.now();
        var doRender = function(countMap) {
            for (var i = 0; i < songs.length; i++) {
                var count = countMap[songs[i].path];
                if (count && count > 0) {
                    var items = document.querySelectorAll('.song-item[data-path="' + songs[i].path + '"]');
                    for (var j = 0; j < items.length; j++) {
                        if (!items[j].querySelector('.play-count-badge')) {
                            var badge = document.createElement('span');
                            badge.className = 'play-count-badge';
                            badge.style.cssText = 'position:absolute;bottom:2px;right:4px;font-size:10px;color:#5bc0de;';
                            badge.textContent = count + '次';
                            items[j].appendChild(badge);
                        }
                    }
                }
            }
        };

        if (hotPlayCache && now - hotPlayCacheTime < 30000) {
            doRender(hotPlayCache);
            return;
        }

        fetch('/api/hot-songs').then(function(r){return r.json();}).then(function(data){
            hotPlayCache = {};
            for (var i = 0; i < data.length; i++) {
                hotPlayCache[data[i].path] = data[i].count;
            }
            hotPlayCacheTime = now;
            doRender(hotPlayCache);
        }).catch(function(){});
    }

    function loadHotSongs() {
        var hotList = document.getElementById('hotList');
        hotList.innerHTML = '<div class="loading">加载中...</div>';
        fetch('/api/hot-songs').then(function(r){return r.json();}).then(function(data){
            if (!data || data.length === 0) {
                hotList.innerHTML = '<div style="text-align:center;color:#888;padding:40px 20px;font-size:16px">暂无热播数据<br><span style="font-size:13px;color:#555">歌曲被点播后会自动统计</span></div>';
                return;
            }
            var html = '<div style="padding:8px;color:#f0ad4e;font-size:15px;text-align:center;font-weight:bold">热播排行 <span style="color:#95a5a6;font-weight:normal">(' + data.length + '首)</span></div>';
            html += '<div class="singer-songs">';
            for (var i = 0; i < data.length; i++) {
                var s = data[i];
                var rank = i + 1;
                var rankColor = rank === 1 ? '#f0ad4e' : rank === 2 ? '#95a5a6' : rank === 3 ? '#cd7f32' : '#7f8c8d';
                var rankIcon = rank <= 3 ? '&#9733;' : '';
                var showName = s.name.replace(/\.[^.]+$/, '');
                var fontSize = calculateFontSize(showName);
                html += '<div class="song-item" onclick="addToQueue(\'' + s.path.replace(/\\/g, '/').replace(/'/g, "\\'") + '\',\'' + s.name.replace(/'/g, "\\'") + '\',\'video\')">';
                html += '<span style="position:absolute;top:2px;left:4px;font-size:11px;font-weight:bold;color:' + rankColor + '">' + rankIcon + rank + '</span>';
                html += '<span class="song-name" style="font-size:' + fontSize + 'px;padding-left:22px;">' + showName + '</span>';
                html += '<span style="position:absolute;bottom:2px;right:4px;font-size:10px;color:#5bc0de">' + s.count + '次</span>';
                html += '</div>';
            }
            html += '</div>';
            hotList.innerHTML = html;
        }).catch(function(){
            hotList.innerHTML = '<div style="text-align:center;color:#ff4444;padding:20px">加载失败</div>';
        });
    }

    function loadSingerIndex() {
        document.getElementById('singerLetters').innerHTML = '';
        document.getElementById('singerList').innerHTML = '<div class="loading" style="text-align:center;padding:30px;color:#5bc0de;font-size:16px">正在加载歌手索引...</div>';
        fetch('/api/singers').then(function(r){return r.json();}).then(function(data){
            singerData = data;
            renderSingerLetters();
            var letters = Object.keys(data).sort();
            if (letters.length > 0) selectSingerLetter(letters[0]);
        }).catch(function(){
            document.getElementById('singerList').innerHTML = '<div style="text-align:center;color:#d9534f;padding:20px">加载失败，请重试</div>';
        });
    }

    function renderSingerLetters() {
        if (!singerData) return;
        var letters = Object.keys(singerData).sort();
        var html = '';
        for (var i = 0; i < letters.length; i++) {
            html += '<span onclick="selectSingerLetter(\'' + letters[i] + '\')" id="letter_' + letters[i] + '">' + letters[i] + '</span>';
        }
        document.getElementById('singerLetters').innerHTML = html;
    }

    function selectSingerLetter(letter) {
        currentSingerLetter = letter;
        currentSingerName = '';
        var spans = document.getElementById('singerLetters').children;
        for (var i = 0; i < spans.length; i++) {
            spans[i].className = spans[i].id === 'letter_' + letter ? 'active' : '';
        }
        var singerList = document.getElementById('singerList');
        // 显示加载中提示（数据量大时渲染需要时间）
        singerList.innerHTML = '<div class="loading" style="text-align:center;padding:30px;color:#5bc0de;font-size:15px">正在加载歌手列表...</div>';
        // 用setTimeout让浏览器先渲染loading提示，再做实际渲染
        setTimeout(function() {
            var singers = singerData[letter] || [];
            var html = '<div class="singer-grid">';
            for (var i = 0; i < singers.length; i++) {
                html += '<div class="singer-btn" onclick="loadSingerSongs(\'' + singers[i].name.replace(/'/g, "\\'") + '\')">';
                html += '<span class="sname">' + singers[i].name + '</span>';
                html += '<span class="scount">' + singers[i].count + '首</span>';
                html += '</div>';
            }
            html += '</div>';
            if (singers.length === 0) {
                html = '<div style="text-align:center;color:#ccc;padding:20px">暂无歌手</div>';
            }
            singerList.innerHTML = html;
        }, 0);
    }

    function loadSingerSongs(singer) {
        currentSingerName = singer;
        var singerList = document.getElementById('singerList');
        singerList.innerHTML = '<div class="loading" style="text-align:center;padding:30px;color:#5bc0de;font-size:15px">正在加载 ' + singer + ' 的歌曲...</div>';
        fetch('/api/songs-by-singer?singer=' + encodeURIComponent(singer)).then(function(r){return r.json();}).then(function(songs){
            var html = '<div class="singer-back" onclick="selectSingerLetter(\'' + currentSingerLetter + '\')">&#8592; 返回歌手列表</div>';
            html += '<div style="padding:8px;color:#428bca;font-size:15px;text-align:center">' + singer + ' (' + songs.length + '首)</div>';
            html += '<div class="singer-songs">';
            for (var i = 0; i < songs.length; i++) {
                var s = songs[i];
                var showName = s.displayName || s.name;
                var fontSize = calculateFontSize(showName);
                html += '<div class="song-item" onclick="addToQueue(\'' + s.path + '\',\'' + s.name.replace(/'/g, "\\'") + '\',\'' + s.type + '\',\'' + (showName !== s.name ? showName.replace(/'/g, "\\'") : '') + '\')">';
                html += '<span class="song-name" style="font-size:' + fontSize + 'px;">' + showName + '</span>';
                html += '</div>';
            }
            html += '</div>';
            singerList.innerHTML = html;
        }).catch(function(){
            singerList.innerHTML = '<div style="text-align:center;color:#d9534f;padding:20px">加载失败，请重试</div>';
        });
    }

    function loadLanguageIndex() {
        fetch('/api/languages').then(function(r){return r.json();}).then(function(data){
            languageData = data;
            renderLanguageList(data);
        });
    }

    function renderLanguageList(data) {
        var html = '<div class="cat-grid">';
        for (var i = 0; i < data.length; i++) {
            html += '<div class="cat-btn" onclick="loadLanguageSongs(\'' + data[i].name.replace(/'/g, "\\'") + '\')">';
            html += '<span class="cname">' + data[i].name + '</span>';
            html += '<span class="ccount">' + data[i].count + '首</span>';
            html += '</div>';
        }
        html += '</div>';
        document.getElementById('languageList').innerHTML = html;
    }

    function loadLanguageSongs(lang) {
        fetch('/api/songs-by-language?language=' + encodeURIComponent(lang)).then(function(r){return r.json();}).then(function(songs){
            var html = '<div class="singer-back" onclick="renderLanguageList(languageData)">&#8592; 返回语种列表</div>';
            html += '<div style="padding:8px;color:#428bca;font-size:15px;text-align:center">' + lang + ' (' + songs.length + '首)</div>';
            html += '<div class="singer-songs">';
            for (var i = 0; i < songs.length; i++) {
                var s = songs[i];
                var showName = s.displayName || s.name;
                var fontSize = calculateFontSize(showName);
                html += '<div class="song-item" onclick="addToQueue(\'' + s.path + '\',\'' + s.name.replace(/'/g, "\\'") + '\',\'' + s.type + '\',\'' + (showName !== s.name ? showName.replace(/'/g, "\\'") : '') + '\')">';
                html += '<span class="song-name" style="font-size:' + fontSize + 'px;">' + showName + '</span>';
                html += '</div>';
            }
            html += '</div>';
            document.getElementById('languageList').innerHTML = html;
        });
    }

    function loadCategoryIndex() {
        fetch('/api/categories').then(function(r){return r.json();}).then(function(data){
            categoryData = data;
            renderCategoryList(data);
        });
    }

    function renderCategoryList(data) {
        var html = '<div class="cat-grid">';
        for (var i = 0; i < data.length; i++) {
            html += '<div class="cat-btn" onclick="loadCategorySongs(\'' + data[i].name.replace(/'/g, "\\'") + '\')">';
            html += '<span class="cname">' + data[i].name + '</span>';
            html += '<span class="ccount">' + data[i].count + '首</span>';
            html += '</div>';
        }
        html += '</div>';
        document.getElementById('categoryList').innerHTML = html;
    }

    function loadCategorySongs(cat) {
        fetch('/api/songs-by-category?category=' + encodeURIComponent(cat)).then(function(r){return r.json();}).then(function(songs){
            var html = '<div class="singer-back" onclick="renderCategoryList(categoryData)">&#8592; 返回曲种列表</div>';
            html += '<div style="padding:8px;color:#428bca;font-size:15px;text-align:center">' + cat + ' (' + songs.length + '首)</div>';
            html += '<div class="singer-songs">';
            for (var i = 0; i < songs.length; i++) {
                var s = songs[i];
                var showName = s.displayName || s.name;
                var fontSize = calculateFontSize(showName);
                html += '<div class="song-item" onclick="addToQueue(\'' + s.path + '\',\'' + s.name.replace(/'/g, "\\'") + '\',\'' + s.type + '\',\'' + (showName !== s.name ? showName.replace(/'/g, "\\'") : '') + '\')">';
                html += '<span class="song-name" style="font-size:' + fontSize + 'px;">' + showName + '</span>';
                html += '</div>';
            }
            html += '</div>';
            document.getElementById('categoryList').innerHTML = html;
        });
    }

    window.addEventListener('resize', function(){
        calculateOptimalLayout();
    });
</script>
</body>
</html>
`
	template.Must(template.New("ktv").Parse(tpl)).Execute(w, nil)
}

func SongSearchHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	name := r.URL.Query().Get("name")

	if name == "" {
		w.Write([]byte("[]"))
		return
	}

	if len(cachedFixedMediaList) == 0 && len(cachedUploadMediaList) == 0 {
		initMediaCache()
	}

	var results []MediaFile
	keywords := strings.Fields(strings.ToLower(name))
	for i, item := range getCachedMediaList() {
		lowerName := cachedLowerNames[i]
		lowerPinyin := cachedLowerPinyins[i]
		match := true
		for _, kw := range keywords {
			if !strings.Contains(lowerName, kw) && !strings.Contains(lowerPinyin, kw) {
				match = false
				break
			}
		}
		if match {
			results = append(results, item)
		}
	}

	json.NewEncoder(w).Encode(results)

	if len(results) == 0 {
		logZeroResultKeyword(name)
	}
}
