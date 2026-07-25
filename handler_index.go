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
        body{background:#080812;color:#fff}
        .wrap{display:flex;height:100vh}
        .ktv-left{width:66.67%;background:#0f1020;border-right:2px solid #1a1a30;display:flex;flex-direction:column}
        .ktv-top{display:flex;align-items:center;gap:10px;padding:10px 15px;background:#111328;border-bottom:1px solid #222444;min-height:56px}
        #search{flex:1;padding:8px 14px;background:#181a35;border:none;border-radius:6px;color:#fff;font-size:14px;max-width:300px}
        #search::placeholder{color:#888}
        #searchBtn{padding:8px 20px;background:#00aaff;border:none;border-radius:6px;color:#fff;font-size:14px;cursor:pointer}
        #searchBtn:hover{background:#0088cc}
        .top-btn{padding:8px 14px;background:#181a35;border:1px solid #2a2a55;border-radius:6px;color:#ccc;font-size:13px;cursor:pointer;white-space:nowrap}
        .top-btn:hover{background:#00aaff;color:#fff;border-color:#00aaff}
        .status-info{display:flex;gap:15px;font-size:14px;color:#00aaff;white-space:nowrap;align-items:center}
        .toggle-switch{display:inline-flex;align-items:center;gap:5px;cursor:pointer;font-size:13px;color:#888}
        .toggle-switch input{display:none}
        .toggle-slider{width:36px;height:20px;background:#333;border-radius:10px;position:relative;transition:background 0.3s}
        .toggle-slider::after{content:'';position:absolute;top:2px;left:2px;width:16px;height:16px;background:#888;border-radius:50%;transition:all 0.3s}
        .toggle-switch input:checked+.toggle-slider{background:#00aaff}
        .toggle-switch input:checked+.toggle-slider::after{left:18px;background:#fff}
        .song-list{flex:1;overflow-y:auto;padding:6px;display:grid;grid-template-columns:repeat(auto-fill, minmax(200px, 1fr));grid-template-rows:repeat(auto-fill, 60px);gap:8px}
        .song-item{padding:6px 4px;background:#181a35;border-radius:4px;cursor:pointer;display:flex;flex-direction:column;align-items:center;text-align:center;transition:all 0.3s ease;overflow:hidden;min-height:60px;max-height:60px;position:relative}
        .song-item:hover{background:#0066ff;transform:scale(1.02)}
        .ktv-right{width:33.33%;background:#080812;display:flex;flex-direction:column}
        .control-bar{padding:10px 15px;background:#0f1020;border-bottom:1px solid #222444;display:flex;justify-content:center;gap:15px;flex-wrap:wrap;min-height:56px;align-items:center}
        .control-btn{width:80px;height:36px;border-radius:6px;border:none;background:#181a35;color:#fff;font-size:16px;cursor:pointer}
        .control-btn:hover{background:#00aaff}
        .control-btn.active{background:#27ae60}
        .queue-area{flex:1;background:#111328;padding:15px;overflow-y:auto}
        .queue-title{color:#00aaff;margin-bottom:8px}
        .queue-item{padding:8px 12px;background:#181a35;margin:4px 0;border-radius:4px;display:flex;justify-content:space-between;align-items:center}
        .transcode-progress{font-size:12px;color:#ff9900;margin-left:8px}
        .transcode-progress-bar{width:60px;height:6px;background:#333;border-radius:3px;overflow:hidden;margin:4px 0}
        .transcode-progress-fill{height:100%;background:#ff9900;border-radius:3px;transition:width 0.3s}
        .top-btn{background:#0066cc;border:none;color:#fff;border-radius:3px;padding:2px 6px;cursor:pointer;font-size:11px}
        .top-btn:hover{background:#0088ff}
        .del-btn{background:#ff3333;border:none;color:#fff;border-radius:3px;padding:2px 6px;cursor:pointer}
        .toast{position:fixed;top:20px;left:50%;transform:translateX(-50%);background:rgba(0,170,255,0.9);color:#fff;padding:10px 24px;border-radius:6px;font-size:14px;opacity:0;transition:opacity 0.3s;pointer-events:none;z-index:9999}
        .toast.show{opacity:1}
        .browse-tabs{display:flex;gap:0;background:#111328;border-bottom:1px solid #222}
        .browse-tab{flex:1;padding:6px 0;background:transparent;border:none;color:#888;font-size:13px;cursor:pointer;border-bottom:2px solid transparent}
        .browse-tab.active{color:#0af;border-bottom-color:#0af}
        .singer-panel{flex:1;overflow-y:auto;display:flex;flex-direction:column}
        .singer-letters{display:flex;flex-wrap:wrap;gap:2px;padding:6px 8px;background:#0f1020;border-bottom:1px solid #222;flex-shrink:0}
        .singer-letters span{padding:2px 7px;background:#181a35;border-radius:3px;color:#aaa;font-size:12px;cursor:pointer}
        .singer-letters span:hover,.singer-letters span.active{background:#0af;color:#fff}
        .singer-list{flex:1;overflow-y:auto;padding:6px}
        .singer-grid{padding:6px;display:grid;grid-template-columns:repeat(auto-fill,minmax(120px,1fr));grid-template-rows:repeat(auto-fill,50px);gap:6px}
        .singer-btn{padding:6px 4px;background:#181a35;border-radius:4px;cursor:pointer;display:flex;flex-direction:column;align-items:center;justify-content:center;text-align:center;overflow:hidden;min-height:50px;max-height:50px;transition:all 0.2s}
        .singer-btn:hover{background:#0066ff;transform:scale(1.02)}
        .singer-btn .sname{font-size:13px;white-space:nowrap;overflow:hidden;text-overflow:ellipsis;width:100%}
        .singer-btn .scount{font-size:10px;color:#888}
        .cat-grid{padding:6px;display:grid;grid-template-columns:repeat(auto-fill,minmax(100px,1fr));grid-template-rows:repeat(auto-fill,50px);gap:6px}
        .cat-btn{padding:6px 4px;background:#181a35;border-radius:4px;cursor:pointer;display:flex;flex-direction:column;align-items:center;justify-content:center;text-align:center;overflow:hidden;min-height:50px;max-height:50px;transition:all 0.2s}
        .cat-btn:hover{background:#0066ff;transform:scale(1.02)}
        .cat-btn .cname{font-size:13px;white-space:nowrap;overflow:hidden;text-overflow:ellipsis;width:100%}
        .cat-btn .ccount{font-size:10px;color:#888}
        .singer-songs{padding:6px;display:grid;grid-template-columns:repeat(auto-fill,minmax(200px,1fr));grid-template-rows:repeat(auto-fill,60px);gap:8px}
        .singer-back{padding:8px 12px;background:#222444;margin-bottom:6px;border-radius:4px;cursor:pointer;color:#0af;font-size:13px;text-align:center}
        .singer-back:hover{background:#2a2a4e}
        .pagination{padding:10px;background:#111328;border-top:1px solid #222444;display:flex;justify-content:center;gap:8px;flex-wrap:wrap;max-height:80px;overflow-y:auto}
        .page-buttons{display:flex;gap:8px;flex-wrap:wrap;justify-content:center;align-items:center}
        .page-btn{padding:4px 8px;background:#222444;border:none;border-radius:4px;color:#fff;cursor:pointer;font-size:14px}
        .page-btn.active{background:#00aaff}
        .page-btn:disabled{background:#181a35;color:#666;cursor:not-allowed}
        .loading{text-align:center;color:#888;padding:20px}
        .nav-page{position:fixed;top:0;left:0;width:100%;height:100%;background:#080812;z-index:1000;display:none;overflow-y:auto}
        .nav-page.active{display:block}
        .nav-content{max-width:700px;margin:0 auto;padding:30px 20px}
        .nav-title{font-size:28px;margin-bottom:8px;color:#00aaff;text-align:center}
        .nav-subtitle{font-size:14px;color:#888;text-align:center;margin-bottom:20px}
        .nav-section{margin-bottom:18px}
        .nav-section-title{font-size:15px;color:#00aaff;margin-bottom:8px;border-bottom:1px solid #222444;padding-bottom:4px}
        .nav-card{background:#111328;border-radius:8px;padding:12px 16px;margin-bottom:8px;display:flex;align-items:flex-start;gap:10px}
        .nav-card-icon{font-size:20px;flex-shrink:0;line-height:1.3}
        .nav-card-body{flex:1;min-width:0}
        .nav-card-title{font-size:14px;font-weight:bold;margin-bottom:2px}
        .nav-card-detail{font-size:12px;color:#aaa;line-height:1.5}
        .nav-card-action{margin-top:6px}
        .nav-card-action a,.nav-card-action button{display:inline-block;padding:6px 16px;border:none;border-radius:4px;font-size:12px;cursor:pointer;text-decoration:none;margin-right:6px;margin-top:4px;transition:all 0.2s}
        .nav-action-settings{background:#00aaff;color:#fff}
        .nav-action-settings:hover{background:#0088cc}
        .nav-action-link{background:#333;color:#88ddff;text-decoration:underline;border:none;cursor:pointer}
        .nav-action-link:hover{color:#aaffff}
        .nav-status-ok{color:#00ff88}
        .nav-status-warn{color:#ffaa00}
        .nav-status-err{color:#ff4444}
        .nav-track-notice{background:#1a1a3e;border:1px solid #2a2a55;border-radius:8px;padding:12px 16px;margin-bottom:18px;font-size:13px;color:#ccc;line-height:1.6}
        .nav-track-notice input[type=text]{width:100%;padding:8px;background:#333;border:none;border-radius:4px;color:#fff;font-size:13px;margin:8px 0;box-sizing:border-box}
        .nav-track-notice button{padding:6px 16px;background:#00aaff;border:none;border-radius:4px;color:#fff;font-size:13px;cursor:pointer}
        .nav-track-notice button:hover{background:#0088cc}
        .nav-skip{display:block;width:100%;padding:12px;background:#222444;color:#aaa;border:none;border-radius:6px;font-size:16px;margin-top:20px;cursor:pointer;text-align:center}
        .nav-skip:hover{background:#333555;color:#ccc}
        .nav-skip-critical{background:#441111;color:#ff8888}
        .nav-skip-critical:hover{background:#552222;color:#ffaaaa}
        .qr-modal{position:fixed;top:0;left:0;width:100vw;height:100vh;background:rgba(0,0,0,0.8);display:none;align-items:center;justify-content:center;z-index:10000}
        .qr-modal-content{background:#1a1a3e;border-radius:12px;padding:30px;text-align:center;max-width:350px}
        .qr-modal-content h2{color:#00aaff;margin-bottom:15px;font-size:20px}
        .qr-modal-content img{border-radius:8px;margin:10px 0}
        .qr-modal-content .qr-info{color:#888;font-size:13px;margin:8px 0}
        .qr-modal-content .qr-status{font-size:14px;margin:10px 0}
        .qr-modal-content button{margin-top:15px;padding:8px 24px;border:none;border-radius:6px;background:#00aaff;color:#fff;cursor:pointer;font-size:14px}
        .qr-modal-content button:hover{background:#0088cc}
    </style>
</head>
<body>
<div class="toast" id="toast"></div>
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
            </div>
        </div>
        <div class="browse-tabs">
            <button class="browse-tab active" id="tabSearch" onclick="switchBrowseTab('search')">搜索</button>
            <button class="browse-tab" id="tabSinger" onclick="switchBrowseTab('singer')">歌手</button>
            <button class="browse-tab" id="tabLanguage" onclick="switchBrowseTab('language')">语种</button>
            <button class="browse-tab" id="tabCategory" onclick="switchBrowseTab('category')">曲种</button>
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
            <button class="control-btn" style="width:100%;margin-top:8px;background:#181a35;border:1px dashed #2a2a55" onclick="randomSong()">随机点歌</button>
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
    var audioPlayerWin = null;
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

    // 定时检测播放窗口是否被关闭
    // 如果用户直接关闭播放窗口，无法触发 video.onended → postMessage("ended")
    // 此定时器作为兜底：窗口关闭时标记当前歌曲播放结束（不自动播放下一首）
    setInterval(function(){
        if(currentPlayingIndex < 0) return;
        var item = queue[currentPlayingIndex];
        if(!item) return;
        var isAudio = isAudioFile(item.name);
        var win = isAudio ? audioPlayerWin : playerWin;
        if(win && win.closed){
            // 播放窗口被关闭，视为当前歌曲播放结束
            console.log('[KTV] 播放窗口已关闭，标记当前歌曲结束');
            // 从列表移除当前已播放歌曲
            queue.splice(currentPlayingIndex, 1);
            currentPlayingIndex = -1;
            renderQueue();
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

    function isAudioFile(fileName) {
        var audioExtensions = ['.mp3', '.wav', '.flac', '.aac', '.m4a', '.ogg', '.wma'];
        var ext = fileName.toLowerCase().substring(fileName.lastIndexOf('.'));
        return audioExtensions.indexOf(ext) !== -1;
    }

    function showPopupBlockedWarning() {
        var existing = document.getElementById('popupWarning');
        if(existing) existing.remove();
        var div = document.createElement('div');
        div.id = 'popupWarning';
        div.style.cssText = 'position:fixed;top:0;left:0;width:100%;height:100%;background:rgba(0,0,0,0.85);z-index:9999;display:flex;align-items:center;justify-content:center;';
        div.innerHTML = '<div style="background:#1a1a3e;border:2px solid #ff4444;border-radius:12px;padding:40px;max-width:500px;text-align:center;">' +
            '<div style="font-size:48px;color:#ff4444;margin-bottom:20px;">&#9888;</div>' +
            '<div style="font-size:22px;color:#fff;margin-bottom:15px;">播放窗口被浏览器阻止！</div>' +
            '<div style="font-size:16px;color:#ccc;margin-bottom:20px;line-height:1.8;">请在浏览器地址栏右侧点击弹窗阻止图标，<br>选择"始终允许来自此网站的弹出式窗口"，<br>然后刷新页面重试。</div>' +
            '<button onclick="document.getElementById(\'popupWarning\').remove()" style="padding:10px 30px;background:#00aaff;border:none;border-radius:6px;color:#fff;font-size:16px;cursor:pointer;">我知道了</button>' +
            '</div>';
        document.body.appendChild(div);
    }

    function openPlayer() {
        if(!playerWin || playerWin.closed){
            playerWin = window.open('/player', 'ktvPlayer', 'width=1280,height=720,menubar=no,toolbar=no,status=no');
            if(!playerWin || playerWin.closed){
                showPopupBlockedWarning();
            }
        }
    }

    function openAudioPlayer() {
        if(!audioPlayerWin || audioPlayerWin.closed){
            audioPlayerWin = window.open('/audio-player', 'ktvAudioPlayer', 'width=800,height=600,menubar=no,toolbar=no,status=no');
            if(!audioPlayerWin || audioPlayerWin.closed){
                showPopupBlockedWarning();
            }
        }
    }

    var settingsWin = null;
    function openSettings() {
        if(!settingsWin || settingsWin.closed){
            settingsWin = window.open('/settings', 'ktvSettings', 'width=600,height=500,menubar=no,toolbar=no,status=no');
        }
    }

    var uploadWin = null;
    function openUpload() {
        if(!uploadWin || uploadWin.closed){
            uploadWin = window.open('/upload', 'ktvUpload', 'width=700,height=550,menubar=no,toolbar=no,status=no');
            if(!uploadWin || uploadWin.closed){
                showPopupBlockedWarning();
            }
        }
    }

    var missingWin = null;
    function openMissing() {
        if(!missingWin || missingWin.closed){
            missingWin = window.open('/missing', 'ktvMissing', 'width=500,height=550,menubar=no,toolbar=no,status=no');
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
            var qrUrl = 'http://' + data.qrServerAddr + '/m/' + mySessionId;
            var qrImgUrl = '/api/qr/image?url=' + encodeURIComponent(qrUrl);
            document.getElementById('qrModal').style.display = 'flex';
            document.getElementById('qrImage').src = qrImgUrl;
            document.getElementById('qrSessionId').textContent = '会话ID: ' + mySessionId;
            document.getElementById('qrUrl').textContent = qrUrl;
            document.getElementById('qrStatus').textContent = '已连接';
            document.getElementById('qrStatus').style.color = '#00e676';
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
        }, 2000);
    }

    function stopQRPoll() {
        if (qrPollInterval) { clearInterval(qrPollInterval); qrPollInterval = null; }
    }

    function ensurePlayer(isAudio) {
        if(isAudio){
            if(!audioPlayerWin || audioPlayerWin.closed){
                openAudioPlayer();
                return false;
            }
            return true;
        } else {
            if(!playerWin || playerWin.closed){
                openPlayer();
                return false;
            }
            return true;
        }
    }

    function getActivePlayerWin(item) {
        var isAudio = isAudioFile(item.name);
        return isAudio ? audioPlayerWin : playerWin;
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
        var isAudio = isAudioFile(name);
        ensurePlayer(isAudio);
        checkAndRequestTranscode(0);
    }

    function addToQueue(path,name,type,displayName,insertNext){
        // 已点歌曲去重检查（按名称）
        for(var i=0;i<queue.length;i++){
            if(queue[i].name === name){
                showToast('队列中已有: ' + (displayName||name).replace(/\.[^.]+$/, ''));
                return;
            }
        }
        var queueItem = {path:path,name:name,type:type,displayName:displayName||'',status:"checking",transcodeProgress:0,requestKey:path};
        if(insertNext && currentPlayingIndex >= 0 && currentPlayingIndex < queue.length - 1){
            queue.splice(currentPlayingIndex + 1, 0, queueItem);
        } else {
            queue.push(queueItem);
        }
        renderQueue();

        var newIdx = insertNext && currentPlayingIndex >= 0 ? currentPlayingIndex + 1 : queue.length - 1;

        if(queue.length === 1 || (currentPlayingIndex === -1)){
            var isAudio = isAudioFile(name);
            ensurePlayer(isAudio);
        }

        checkAndRequestTranscode(newIdx);
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

        var xhr = new XMLHttpRequest();
        xhr.open('POST', '/api/transcode/check-and-add', true);
        xhr.setRequestHeader('Content-Type', 'application/json');
        xhr.onload = function() {
            if (xhr.status === 200) {
                try {
                    var data = JSON.parse(xhr.responseText);
                    if(data.needsTranscode){
                        item.status = data.status;
                        item.queuePosition = data.queuePosition;
                        if(data.codecInfo){
                            item.codecInfo = data.codecInfo;
                        }
                    } else {
                        item.status = "ready";
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
            html += '<div class="queue-item" style="' + (isCurrent ? 'border-left:3px solid #00ff00;' : '') + '">';
            html += '<div style="flex: 1;">';
            html += '<span>'+(i+1)+'. '+(item.displayName||item.name)+'</span>';
            if(isCurrent){
                html += ' <span style="color:#00ff00; margin-left: 8px;">正在播放</span>';
            } else if(item.status === "transcoding"){
                html += '<div class="transcode-progress">';
                html += '<div class="transcode-progress-bar"><div class="transcode-progress-fill" style="width:'+item.transcodeProgress+'%"></div></div>';
                html += '<span>正在转码: '+item.transcodeProgress+'%</span>';
                if(item.codecInfo){
                    html += ' <span style="color:#aaa; font-size:11px;">('+item.codecInfo+')</span>';
                }
                html += '</div>';
            } else if(item.status === "waiting"){
                html += '<span style="color:#ff9900; margin-left: 8px;">等待转码</span>';
                if(item.codecInfo){
                    html += ' <span style="color:#aaa; font-size:11px;">('+item.codecInfo+')</span>';
                }
                if(item.queuePosition !== undefined && item.queuePosition > 0){
                    html += ' <span style="color:#00aaff;">(排队: 第'+item.queuePosition+'首)</span>';
                }
            } else if(item.status === "checking"){
                html += '<span style="color:#888; margin-left: 8px;">检查中...</span>';
            } else if(item.status === "ready"){
                if(!isCurrent){
                    html += '<span style="color:#00ff00; margin-left: 8px;">已就绪</span>';
                }
            }
            // 显示轨道异常警告
            if(item.trackWarning){
                html += '<div style="color:#ff4444;font-size:12px;font-weight:bold;background:#fff3f3;border:1px solid #ff4444;border-radius:4px;padding:2px 6px;margin-top:2px;">';
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
            playerWin.postMessage({action:"syncQueue",list:queue,currentPlayingIndex:currentPlayingIndex},'*');
        }
        if(audioPlayerWin && !audioPlayerWin.closed){
            audioPlayerWin.postMessage({action:"syncQueue",list:queue,currentPlayingIndex:currentPlayingIndex},'*');
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
    }

    function playQueueItem(idx){
        if(idx<0||idx>=queue.length) return;
        var item = queue[idx];
        var isAudio = isAudioFile(item.name);

        // 播放前检查文件轨道完整性（非音频文件才检查）
        if(!isAudio){
            checkAndWarnTracks(item.name, item.path, idx);
        }

        var win = isAudio ? audioPlayerWin : playerWin;

        // 构建播放URL
        var url;
        if(isAudio){
            url = '/file?name='+encodeURIComponent(item.path);
        } else if(isStreamMode()){
            // 省流模式：使用流媒体实时转码
            url = '/api/stream?name='+encodeURIComponent(item.path)+'&trackIndex='+lastTrackIndex+'&quality=high&_t='+Date.now();
        } else {
            url = '/file?name='+encodeURIComponent(item.path);
        }

        if(!win || win.closed){
            ensurePlayer(isAudio);
            setTimeout(function(){
                var w = isAudio ? audioPlayerWin : playerWin;
                if(w && !w.closed){
                    w.postMessage({action:"play",url:url,type:item.type,name:item.name},'*');
                }
            }, 500);
            return;
        }
        win.postMessage({action:"play",url:url,type:item.type,name:item.name},'*');
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
                            badge.style.cssText = 'color:#ff4444;font-size:12px;font-weight:bold;padding:2px 6px;background:#fff3f3;border:1px solid #ff4444;border-radius:4px;margin-top:2px;display:inline-block;';
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
    }

    function nextSong(){
        if(queue.length === 0) return;
        if(currentPlayingIndex >= 0 && currentPlayingIndex < queue.length){
            queue.splice(currentPlayingIndex, 1);
            if(currentPlayingIndex >= queue.length){
                currentPlayingIndex = -1;
            }
        }
        renderQueue();
        if(currentPlayingIndex === -1){
            tryAutoPlay();
        } else {
            playQueueItem(currentPlayingIndex);
            renderQueue();
        }
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
        var isAudio = false;
        if(currentPlayingIndex >= 0 && currentPlayingIndex < queue.length){
            isAudio = isAudioFile(queue[currentPlayingIndex].name);
        } else if(msg.action === "play"){
            isAudio = isAudioFile(msg.name);
        }
        var win = isAudio ? audioPlayerWin : playerWin;
        if(win && !win.closed){
            win.postMessage(msg,'*');
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
            html += '<div class="nav-section-title" style="color:#ff4444;border-bottom-color:#ff4444">需要处理的问题</div>';
            for (var i = 0; i < data.blockingIssues.length; i++) {
                var iss = data.blockingIssues[i];
                var icon = iss.level === 'critical' ? '<span class="nav-status-err">&#9888;</span>' : '<span class="nav-status-warn">&#9888;</span>';
                html += '<div class="nav-card" style="border-left:3px solid ' + (iss.level === 'critical' ? '#ff4444' : '#ffaa00') + '">';
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
            html += '<div class="nav-card"><div class="nav-card-icon"><span style="color:#888">&#9432;</span></div>';
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
                dIcon = '<span style="color:#888">&#9432;</span>';
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
                container.innerHTML = '<span class="nav-status-warn">&#9888;</span> 当前浏览器的音轨切换功能可能受限，建议使用 Chrome 或 Edge 以获得最佳体验。';
                return;
            default: return;
        }
        container.style.display = 'block';
        var html = '<strong style="color:#ffaa00">音轨切换提示：</strong>当前浏览器未启用音轨切换实验功能。<br>';
        html += '请复制下方地址到浏览器地址栏，开启 <code>Experimental Web Platform features</code> 后刷新页面。<br>';
        html += '<input type="text" id="settingsUrlInput" value="' + escHtml(settingsUrl) + '" readonly>';
        html += '<button onclick="copySettingsUrl()">复制地址</button>';
        container.innerHTML = html;
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
        var navPage = document.getElementById('navPage');
        var mainPage = document.getElementById('mainPage');
        navPage.classList.remove('active');
        mainPage.style.display = 'flex';
    }

    window.onload = function(){
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

    // 歌手/语种/曲种浏览
    var singerData = null;
    var currentSingerLetter = '';
    var currentSingerName = '';
    var languageData = null;
    var categoryData = null;

    function switchBrowseTab(tab) {
        var tabs = ['search','singer','language','category'];
        var panels = {search:'songList',singer:'singerPanel',language:'languagePanel',category:'categoryPanel'};
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
    }

    function loadSingerIndex() {
        fetch('/api/singers').then(function(r){return r.json();}).then(function(data){
            singerData = data;
            renderSingerLetters();
            var letters = Object.keys(data).sort();
            if (letters.length > 0) selectSingerLetter(letters[0]);
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
            html = '<div style="text-align:center;color:#555;padding:20px">暂无歌手</div>';
        }
        document.getElementById('singerList').innerHTML = html;
    }

    function loadSingerSongs(singer) {
        currentSingerName = singer;
        fetch('/api/songs-by-singer?singer=' + encodeURIComponent(singer)).then(function(r){return r.json();}).then(function(songs){
            var html = '<div class="singer-back" onclick="selectSingerLetter(\'' + currentSingerLetter + '\')">&#8592; 返回歌手列表</div>';
            html += '<div style="padding:8px;color:#0af;font-size:15px;text-align:center">' + singer + ' (' + songs.length + '首)</div>';
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
            document.getElementById('singerList').innerHTML = html;
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
            html += '<div style="padding:8px;color:#0af;font-size:15px;text-align:center">' + lang + ' (' + songs.length + '首)</div>';
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
            html += '<div style="padding:8px;color:#0af;font-size:15px;text-align:center">' + cat + ' (' + songs.length + '首)</div>';
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
