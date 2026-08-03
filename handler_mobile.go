package main

import (
	"net/http"
	"strings"
	"text/template"
)

func MobilePageHandler(w http.ResponseWriter, r *http.Request) {
	tpl := `<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width,initial-scale=1,maximum-scale=1,user-scalable=no">
<title>KTV 点歌</title>
<style>
*{margin:0;padding:0;box-sizing:border-box;font-family:Microsoft YaHei,-apple-system,sans-serif}
html,body{width:100%;height:100%;overflow:hidden;background:#1a252f;color:#ecf0f1}
.app{display:flex;flex-direction:column;height:100vh;width:100%}

/* 播放器区域 */
.player-area{width:100%;background:#0d1922;position:relative;flex-shrink:0;box-shadow:0 2px 6px rgba(0,0,0,0.3)}
.video-wrap{width:100%;aspect-ratio:16/9;background:#000;position:relative}
.video-wrap video,.video-wrap audio{width:100%;height:100%;object-fit:contain}
.audio-wrap{width:100%;padding:15px;background:#152029;box-shadow:inset 0 1px 0 rgba(255,255,255,0.03)}
.audio-wrap audio{width:100%}
.audio-title{text-align:center;color:#ecf0f1;font-size:16px;margin-bottom:8px;white-space:nowrap;overflow:hidden;text-overflow:ellipsis}
.player-controls{display:flex;justify-content:center;gap:10px;padding:8px 10px;background:#152029;flex-shrink:0;box-shadow:0 1px 4px rgba(0,0,0,0.2)}
.ctrl-btn{padding:8px 18px;border:none;border-radius:4px;font-size:14px;font-weight:bold;cursor:pointer;color:#fff;background:#34495e;box-shadow:0 1px 3px rgba(0,0,0,0.2);transition:background 0.15s}
.ctrl-btn:hover{background:#4a6278}
.ctrl-btn:active{background:#2c3e50}
.ctrl-btn.active{background:#428bca;box-shadow:0 1px 3px rgba(0,0,0,0.2);color:#fff}
.ctrl-btn.active:hover{background:#5a9bd5}

/* 标签栏 */
.tab-bar{display:flex;background:#152029;border-bottom:1px solid #253644;flex-shrink:0;box-shadow:0 1px 4px rgba(0,0,0,0.2)}
.tab-item{flex:1;padding:11px 0;text-align:center;font-size:15px;font-weight:bold;color:#95a5a6;cursor:pointer;border-bottom:3px solid transparent;transition:all 0.2s}
.tab-item.active{color:#428bca;border-bottom-color:#428bca}

/* 标签内容 */
.tab-content{flex:1;overflow:hidden;display:flex;flex-direction:column}
.tab-panel{display:none;flex:1;flex-direction:column;overflow:hidden}
.tab-panel.active{display:flex}

/* 点歌面板 - 2010s深蓝灰风格 */
.search-bar{display:flex;gap:8px;padding:10px 12px;background:#1e2d3a;border-bottom:1px solid #253644;flex-shrink:0}
.search-bar input{flex:1;padding:10px 14px;background:#0d1922;border:1px solid #34495e;border-radius:6px;color:#ecf0f1;font-size:15px;outline:none;box-shadow:inset 0 1px 3px rgba(0,0,0,0.3);transition:border-color 0.2s}
.search-bar input:focus{border-color:#428bca;box-shadow:inset 0 1px 3px rgba(0,0,0,0.3),0 0 0 2px rgba(66,139,202,0.25)}
.search-bar input::placeholder{color:#7f8c8d}
.search-bar button{padding:10px 20px;background:#428bca;border:none;border-radius:6px;color:#fff;font-size:15px;font-weight:bold;cursor:pointer;white-space:nowrap;box-shadow:0 1px 3px rgba(0,0,0,0.2);transition:background 0.15s}
.search-bar button:hover{background:#5a9bd5}
.search-bar button:active{background:#357ebd}
.song-list{flex:1;overflow-y:auto;padding:8px;-webkit-overflow-scrolling:touch}
.song-item{padding:12px 14px;background:#1e2d3a;margin-bottom:5px;border-radius:4px;font-size:14px;cursor:pointer;white-space:nowrap;overflow:hidden;text-overflow:ellipsis;box-shadow:0 1px 3px rgba(0,0,0,0.15);transition:background 0.15s;border-left:4px solid #428bca;border-top:1px solid #253644;border-right:1px solid #253644;border-bottom:1px solid #253644;color:#ecf0f1}
.song-item:hover{background:#273849;box-shadow:0 2px 6px rgba(0,0,0,0.2)}
.singer-letters{display:flex;flex-wrap:wrap;gap:4px;padding:10px;background:#1e2d3a;border-bottom:1px solid #253644;flex-shrink:0}
.singer-letters span{padding:5px 10px;background:#2c3e50;border-radius:4px;color:#bdc3c7;font-size:13px;font-weight:bold;cursor:pointer;box-shadow:0 1px 2px rgba(0,0,0,0.15);transition:all 0.15s;border:1px solid #34495e}
.singer-letters span:hover,.singer-letters span.active{background:#428bca;color:#fff;border-color:#428bca;box-shadow:0 1px 3px rgba(0,0,0,0.2)}
.singer-list{flex:1;overflow-y:auto;padding:6px;-webkit-overflow-scrolling:touch}
.singer-grid{padding:4px;display:grid;grid-template-columns:repeat(auto-fill,minmax(80px,1fr));gap:8px}
.singer-btn{padding:8px 4px;background:#1e2d3a;border-radius:4px;cursor:pointer;display:flex;flex-direction:column;align-items:center;justify-content:center;text-align:center;min-height:52px;box-shadow:0 1px 3px rgba(0,0,0,0.15);transition:background 0.15s;border:1px solid #253644}
.singer-btn:hover{background:#273849}
.singer-btn:active{background:#428bca}
.singer-btn .sname{font-size:12px;color:#ecf0f1;white-space:nowrap;overflow:hidden;text-overflow:ellipsis;width:100%}
.singer-btn .scount{font-size:10px;color:#95a5a6}
.cat-grid{padding:4px;display:grid;grid-template-columns:repeat(auto-fill,minmax(75px,1fr));gap:8px}
.cat-btn{padding:8px 4px;background:#1e2d3a;border-radius:4px;cursor:pointer;display:flex;flex-direction:column;align-items:center;justify-content:center;text-align:center;min-height:52px;box-shadow:0 1px 3px rgba(0,0,0,0.15);transition:background 0.15s;border:1px solid #253644}
.cat-btn:hover{background:#273849}
.cat-btn:active{background:#428bca}
.cat-btn .cname{font-size:12px;color:#ecf0f1;white-space:nowrap;overflow:hidden;text-overflow:ellipsis;width:100%}
.cat-btn .ccount{font-size:10px;color:#95a5a6}
.singer-back{padding:12px 14px;background:#1e2d3a;margin-bottom:6px;border-radius:4px;cursor:pointer;color:#428bca;font-size:14px;font-weight:bold;text-align:center;box-shadow:0 1px 3px rgba(0,0,0,0.15);transition:background 0.15s;border:1px solid #253644}
.singer-back:hover{background:#273849}
.song-item:active{background:#428bca;box-shadow:0 1px 2px rgba(0,0,0,0.15);border-left-color:#357ebd;border-top-color:#357ebd;border-right-color:#357ebd;border-bottom-color:#357ebd}
.pagination{padding:10px;background:#1e2d3a;display:flex;justify-content:center;gap:6px;flex-shrink:0;flex-wrap:wrap}
.page-btn{padding:6px 12px;background:#2c3e50;border:none;border-radius:4px;color:#ecf0f1;cursor:pointer;font-size:13px;font-weight:bold;box-shadow:0 1px 2px rgba(0,0,0,0.15);transition:background 0.15s}
.page-btn:hover{background:#34495e}
.page-btn.active{background:#428bca;box-shadow:0 1px 3px rgba(0,0,0,0.2)}
.page-btn:disabled{background:#1e2d3a;color:#546a7b;cursor:not-allowed;box-shadow:none}
.page-btn:not(:disabled):active{background:#357ebd}

/* 队列面板 - 2010s深蓝灰风格 */
.queue-list{flex:1;overflow-y:auto;padding:10px;-webkit-overflow-scrolling:touch}
.queue-title{color:#ecf0f1;font-size:15px;font-weight:bold;margin-bottom:10px}
.queue-item{padding:12px 14px;background:#1e2d3a;margin-bottom:5px;border-radius:4px;display:flex;justify-content:space-between;align-items:center;font-size:14px;box-shadow:0 1px 3px rgba(0,0,0,0.15);border:1px solid #253644;transition:background 0.15s}
.queue-item:hover{background:#273849}
.queue-item.playing{border-left:4px solid #428bca;border-top:1px solid #253644;border-right:1px solid #253644;border-bottom:1px solid #253644;box-shadow:0 1px 4px rgba(66,139,202,0.15)}
.queue-item .name{flex:1;white-space:nowrap;overflow:hidden;text-overflow:ellipsis;margin-right:8px;color:#ecf0f1}
.queue-item .status{font-size:12px;color:#95a5a6;white-space:nowrap;margin-right:8px}
.queue-item .status.playing{color:#428bca}
.queue-item .status.ready{color:#5cb85c}
.queue-item .status.transcoding{color:#f0ad4e}
.queue-item .status.waiting{color:#f0ad4e}
.top-btn{background:#5cb85c;border:none;color:#fff;border-radius:4px;padding:5px 10px;cursor:pointer;font-size:12px;font-weight:bold;flex-shrink:0;box-shadow:0 1px 2px rgba(0,0,0,0.15);transition:background 0.15s}
.top-btn:hover{background:#6ed06e}
.top-btn:active{background:#4cae4c}
.del-btn{background:#d9534f;border:none;color:#fff;border-radius:4px;padding:5px 10px;cursor:pointer;font-size:12px;font-weight:bold;flex-shrink:0;box-shadow:0 1px 2px rgba(0,0,0,0.15);transition:background 0.15s}
.del-btn:hover{background:#e06b5e}
.del-btn:active{background:#c9302c}
.empty-hint{text-align:center;color:#546a7b;padding:40px 20px;font-size:14px}
.btn-random{display:block;width:calc(100% - 20px);margin:12px auto;padding:14px;background:#f0ad4e;border:none;color:#1a252f;border-radius:6px;font-size:16px;font-weight:bold;cursor:pointer;flex-shrink:0;box-shadow:0 2px 6px rgba(0,0,0,0.2);transition:background 0.15s;letter-spacing:2px}
.btn-random:hover{background:#f5c06e}
.btn-random:active{background:#ec971f}
.toast{position:fixed;top:50%;left:50%;transform:translate(-50%,-50%);background:rgba(26,37,47,0.95);color:#ecf0f1;padding:14px 32px;border-radius:6px;font-size:16px;font-weight:bold;opacity:0;transition:opacity 0.3s;pointer-events:none;z-index:100;text-align:center;max-width:80%;box-shadow:0 4px 20px rgba(0,0,0,0.4);border:1px solid #34495e}
.toast.show{opacity:1}
.toast.error{background:rgba(185,74,72,0.95);border-color:#d9534f}
</style>
</head>
<body>
<div class="toast" id="toast"></div>
<div class="app">
    <!-- 播放器 -->
    <div class="player-area" id="playerArea">
        <div class="video-wrap" id="videoWrap" style="display:none">
            <video id="videoPlayer" playsinline></video>
        </div>
        <div class="audio-wrap" id="audioWrap" style="display:none">
            <div class="audio-title" id="audioTitle"></div>
            <audio id="audioPlayer" controls playsinline></audio>
        </div>
    </div>
    <div class="player-controls" id="playerControls">
        <button class="ctrl-btn" onclick="prevSong()">重播</button>
        <button class="ctrl-btn" onclick="nextSong()">下一首</button>
        <button class="ctrl-btn active" id="btnOrigin" onclick="switchTrack(0)">原唱</button>
        <button class="ctrl-btn" id="btnAcc" onclick="switchTrack(1)">伴奏</button>
        <span id="timeDisplay" style="color:#aaa;font-size:12px;line-height:30px;white-space:nowrap">00:00/00:00</span>
    </div>

    <!-- 标签栏 -->
    <div class="tab-bar">
        <div class="tab-item active" onclick="switchTab('songs')" id="tabSongs">点歌</div>
        <div class="tab-item" onclick="switchTab('singer')" id="tabSinger">歌手</div>
        <div class="tab-item" onclick="switchTab('language')" id="tabLanguage">语种</div>
        <div class="tab-item" onclick="switchTab('category')" id="tabCategory">曲种</div>
        <div class="tab-item" onclick="switchTab('hot')" id="tabHot">热播</div>
        <div class="tab-item" onclick="switchTab('queue')" id="tabQueue">队列</div>
    </div>

    <!-- 点歌面板 -->
    <div class="tab-content">
        <div class="tab-panel active" id="panelSongs">
            <div class="search-bar">
                <input type="text" id="searchInput" placeholder="搜索歌名/歌手..." onkeydown="if(event.key==='Enter')filterList()">
                <button onclick="filterList()">搜索</button>
            </div>
            <div class="song-list" id="songList">
                <div class="empty-hint">输入关键字搜索歌曲</div>
            </div>
            <div class="pagination" id="pagination"></div>
        </div>

        <!-- 歌手面板 -->
        <div class="tab-panel" id="panelSinger">
            <div class="singer-letters" id="singerLetters"></div>
            <div class="singer-list" id="singerList"></div>
        </div>

        <!-- 语种面板 -->
        <div class="tab-panel" id="panelLanguage">
            <div class="singer-list" id="languageList"></div>
        </div>

        <!-- 曲种面板 -->
        <div class="tab-panel" id="panelCategory">
            <div class="singer-list" id="categoryList"></div>
        </div>

        <!-- 热播面板 -->
        <div class="tab-panel" id="panelHot">
            <div class="singer-list" id="hotList"></div>
        </div>

        <!-- 队列面板 -->
        <div class="tab-panel" id="panelQueue">
            <div class="queue-list" id="queueList">
                <div class="empty-hint">暂无播放队列</div>
            </div>
            <button class="btn-random" onclick="randomSong()">随机点歌</button>
        </div>
    </div>
</div>

<script>
var queue = [];
var currentPlayingIndex = -1;
var currentPage = 1;
var pageSize = 30;
var totalItems = 0;
var totalPages = 1;
var currentKeyword = '';
var transcodePollInterval = null;
var lastTrackIndex = 0;
var lastVolume = 1;

function isAudioFile(fileName) {
    var ext = fileName.toLowerCase().substring(fileName.lastIndexOf('.'));
    return ['.mp3','.wav','.flac','.aac','.m4a','.ogg','.wma','.ape'].indexOf(ext) !== -1;
}

// 标签切换
var mobileSingerData = null;
var mobileSingerLetter = '';
var mobileLanguageData = null;
var mobileCategoryData = null;

function switchTab(tab) {
    var tabs = ['songs','singer','language','category','hot','queue'];
    for (var i = 0; i < tabs.length; i++) {
        var t = tabs[i];
        document.getElementById('tab' + t.charAt(0).toUpperCase() + t.slice(1)).className = 'tab-item' + (t === tab ? ' active' : '');
        var panel = document.getElementById('panel' + t.charAt(0).toUpperCase() + t.slice(1));
        if (panel) panel.className = 'tab-panel' + (t === tab ? ' active' : '');
    }
    if (tab === 'queue') renderQueue();
    if (tab === 'singer' && !mobileSingerData) loadMobileSingerIndex();
    if (tab === 'language' && !mobileLanguageData) loadMobileLanguageIndex();
    if (tab === 'category' && !mobileCategoryData) loadMobileCategoryIndex();
    if (tab === 'hot') loadHotSongs();
}

function loadMobileSingerIndex() {
    fetch('/api/singers').then(function(r){return r.json();}).then(function(data){
        mobileSingerData = data;
        var letters = Object.keys(data).sort();
        var html = '';
        for (var i = 0; i < letters.length; i++) {
            html += '<span onclick="selectMobileLetter(\'' + letters[i] + '\')" id="mletter_' + letters[i] + '">' + letters[i] + '</span>';
        }
        document.getElementById('singerLetters').innerHTML = html;
        if (letters.length > 0) selectMobileLetter(letters[0]);
    });
}

function selectMobileLetter(letter) {
    mobileSingerLetter = letter;
    var spans = document.getElementById('singerLetters').children;
    for (var i = 0; i < spans.length; i++) {
        spans[i].className = spans[i].id === 'mletter_' + letter ? 'active' : '';
    }
    var singers = mobileSingerData[letter] || [];
    var html = '<div class="singer-grid">';
    for (var i = 0; i < singers.length; i++) {
        html += '<div class="singer-btn" onclick="loadMobileSingerSongs(\'' + singers[i].name.replace(/'/g, "\\'") + '\')">';
        html += '<span class="sname">' + singers[i].name + '</span>';
        html += '<span class="scount">' + singers[i].count + '首</span>';
        html += '</div>';
    }
    html += '</div>';
    document.getElementById('singerList').innerHTML = html || '<div style="text-align:center;color:#555;padding:20px">暂无歌手</div>';
}

function loadMobileSingerSongs(singer) {
    fetch('/api/songs-by-singer?singer=' + encodeURIComponent(singer)).then(function(r){return r.json();}).then(function(songs){
        var html = '<div class="singer-back" onclick="selectMobileLetter(\'' + mobileSingerLetter + '\')">&#8592; 返回歌手列表</div>';
        html += '<div style="padding:8px;color:#0af;font-size:15px;text-align:center">' + singer + ' (' + songs.length + '首)</div>';
        for (var i = 0; i < songs.length; i++) {
            var s = songs[i];
            var showName = s.displayName || s.name;
            html += '<div class="song-item" onclick="addToQueue(\'' + s.path.replace(/'/g, "\\'") + '\',\'' + s.name.replace(/'/g, "\\'") + '\',\'' + s.type + '\',\'' + (showName !== s.name ? showName.replace(/'/g, "\\'") : '') + '\')">' + showName + '</div>';
        }
        document.getElementById('singerList').innerHTML = html;
    });
}

function loadMobileLanguageIndex() {
    fetch('/api/languages').then(function(r){return r.json();}).then(function(data){
        mobileLanguageData = data;
        renderMobileLanguageList(data);
    });
}

function renderMobileLanguageList(data) {
    var html = '<div class="cat-grid">';
    for (var i = 0; i < data.length; i++) {
        html += '<div class="cat-btn" onclick="loadMobileLanguageSongs(\'' + data[i].name.replace(/'/g, "\\'") + '\')">';
        html += '<span class="cname">' + data[i].name + '</span>';
        html += '<span class="ccount">' + data[i].count + '首</span>';
        html += '</div>';
    }
    html += '</div>';
    document.getElementById('languageList').innerHTML = html;
}

function loadMobileLanguageSongs(lang) {
    fetch('/api/songs-by-language?language=' + encodeURIComponent(lang)).then(function(r){return r.json();}).then(function(songs){
        var html = '<div class="singer-back" onclick="renderMobileLanguageList(mobileLanguageData)">&#8592; 返回语种列表</div>';
        html += '<div style="padding:8px;color:#0af;font-size:15px;text-align:center">' + lang + ' (' + songs.length + '首)</div>';
        for (var i = 0; i < songs.length; i++) {
            var s = songs[i];
            var showName = s.displayName || s.name;
            html += '<div class="song-item" onclick="addToQueue(\'' + s.path.replace(/'/g, "\\'") + '\',\'' + s.name.replace(/'/g, "\\'") + '\',\'' + s.type + '\',\'' + (showName !== s.name ? showName.replace(/'/g, "\\'") : '') + '\')">' + showName + '</div>';
        }
        document.getElementById('languageList').innerHTML = html;
    });
}

function loadHotSongs() {
    var hotList = document.getElementById('hotList');
    hotList.innerHTML = '<div class="empty-hint">加载中...</div>';
    fetch('/api/hot-songs').then(function(r){return r.json();}).then(function(data){
        if (!data || data.length === 0) {
            hotList.innerHTML = '<div class="empty-hint">暂无热播数据<br><span style="font-size:12px;color:#546a7b">歌曲被点播后会自动统计</span></div>';
            return;
        }
        var html = '<div style="padding:8px;color:#f0ad4e;font-size:15px;text-align:center;font-weight:bold">热播排行 <span style="color:#95a5a6;font-weight:normal">(' + data.length + '首)</span></div>';
        for (var i = 0; i < data.length; i++) {
            var s = data[i];
            var rank = i + 1;
            var rankColor = rank === 1 ? '#f0ad4e' : rank === 2 ? '#95a5a6' : rank === 3 ? '#cd7f32' : '#7f8c8d';
            var rankIcon = rank <= 3 ? '&#9733; ' : '';
            var showName = s.name.replace(/\.[^.]+$/, '');
            html += '<div class="song-item" onclick="addToQueue(\'' + s.path.replace(/\\/g, '/').replace(/'/g, "\\'") + '\',\'' + s.name.replace(/'/g, "\\'") + '\',\'video\')" style="position:relative;padding-left:36px">';
            html += '<span style="position:absolute;top:12px;left:8px;font-size:12px;font-weight:bold;color:' + rankColor + '">' + rankIcon + rank + '</span>';
            html += showName;
            html += '<span style="float:right;font-size:11px;color:#5bc0de;margin-left:8px">' + s.count + '次</span>';
            html += '</div>';
        }
        hotList.innerHTML = html;
    }).catch(function(){
        hotList.innerHTML = '<div class="empty-hint">加载失败</div>';
    });
}

function loadMobileCategoryIndex() {
    fetch('/api/categories').then(function(r){return r.json();}).then(function(data){
        mobileCategoryData = data;
        renderMobileCategoryList(data);
    });
}

function renderMobileCategoryList(data) {
    var html = '<div class="cat-grid">';
    for (var i = 0; i < data.length; i++) {
        html += '<div class="cat-btn" onclick="loadMobileCategorySongs(\'' + data[i].name.replace(/'/g, "\\'") + '\')">';
        html += '<span class="cname">' + data[i].name + '</span>';
        html += '<span class="ccount">' + data[i].count + '首</span>';
        html += '</div>';
    }
    html += '</div>';
    document.getElementById('categoryList').innerHTML = html;
}

function loadMobileCategorySongs(cat) {
    fetch('/api/songs-by-category?category=' + encodeURIComponent(cat)).then(function(r){return r.json();}).then(function(songs){
        var html = '<div class="singer-back" onclick="renderMobileCategoryList(mobileCategoryData)">&#8592; 返回曲种列表</div>';
        html += '<div style="padding:8px;color:#0af;font-size:15px;text-align:center">' + cat + ' (' + songs.length + '首)</div>';
        for (var i = 0; i < songs.length; i++) {
            var s = songs[i];
            var showName = s.displayName || s.name;
            html += '<div class="song-item" onclick="addToQueue(\'' + s.path.replace(/'/g, "\\'") + '\',\'' + s.name.replace(/'/g, "\\'") + '\',\'' + s.type + '\',\'' + (showName !== s.name ? showName.replace(/'/g, "\\'") : '') + '\')">' + showName + '</div>';
        }
        document.getElementById('categoryList').innerHTML = html;
    });
}

// 播放器
var videoEl = document.getElementById('videoPlayer');
var audioEl = document.getElementById('audioPlayer');
var mediaDuration = 0; // 服务端获取的媒体时长（流媒体无法从video.duration获取）

// JS估算播放时间（流媒体传输期间浏览器currentTime为0的workaround）
var playStartTime = 0;
var playOffset = 0;
var isVideoPlaying = false;

videoEl.addEventListener('ended', function() { nextSong(); });
audioEl.addEventListener('ended', function() { nextSong(); });

videoEl.addEventListener('volumechange', function() { lastVolume = videoEl.volume; });
audioEl.addEventListener('volumechange', function() { lastVolume = audioEl.volume; });

videoEl.addEventListener('canplay', function() {
    if (videoEl.audioTracks && videoEl.audioTracks.length > 0) {
        for (var i = 0; i < videoEl.audioTracks.length; i++) {
            videoEl.audioTracks[i].enabled = (i === lastTrackIndex);
        }
    }
});

videoEl.addEventListener('playing', function() {
    playStartTime = Date.now();
    isVideoPlaying = true;
});
videoEl.addEventListener('pause', function() {
    if (isVideoPlaying) {
        playOffset += (Date.now() - playStartTime) / 1000;
        isVideoPlaying = false;
    }
});

// 获取当前播放时间（优先用浏览器原生值，流媒体时用JS估算）
function getEstimatedCurrentTime() {
    // 视频播放时
    if (document.getElementById('videoWrap').style.display !== 'none') {
        if (videoEl.currentTime && videoEl.currentTime > 0 && isFinite(videoEl.currentTime)) {
            return videoEl.currentTime;
        }
        // 流媒体期间currentTime为0，用JS估算
        if (isVideoPlaying) {
            return playOffset + (Date.now() - playStartTime) / 1000;
        }
        return playOffset;
    }
    // 音频播放时
    if (document.getElementById('audioWrap').style.display !== 'none') {
        if (audioEl.currentTime && isFinite(audioEl.currentTime)) {
            return audioEl.currentTime;
        }
    }
    return 0;
}

function getEstimatedDuration() {
    // 视频
    if (document.getElementById('videoWrap').style.display !== 'none') {
        if (videoEl.duration && isFinite(videoEl.duration) && videoEl.duration > 0) {
            return videoEl.duration;
        }
        return mediaDuration;
    }
    // 音频
    if (document.getElementById('audioWrap').style.display !== 'none') {
        if (audioEl.duration && isFinite(audioEl.duration)) {
            return audioEl.duration;
        }
        return mediaDuration;
    }
    return 0;
}

// 更新进度显示
function formatTime(sec) {
    if (!sec || !isFinite(sec) || sec < 0) return '00:00';
    var m = Math.floor(sec / 60);
    var s = Math.floor(sec % 60);
    return (m < 10 ? '0' : '') + m + ':' + (s < 10 ? '0' : '') + s;
}

function updateTimeDisplay() {
    var current = getEstimatedCurrentTime();
    var total = getEstimatedDuration();
    document.getElementById('timeDisplay').textContent = formatTime(current) + '/' + formatTime(total);
}

// 用定时器高频更新进度（不依赖timeupdate事件，流媒体时该事件不可靠）
setInterval(updateTimeDisplay, 500);

function playMedia(url, name) {
    var isAudio = isAudioFile(name);
    // 先停止两个播放器，防止声音叠加
    videoEl.pause();
    videoEl.removeAttribute('src');
    videoEl.load();
    audioEl.pause();
    audioEl.removeAttribute('src');
    audioEl.load();
    // 重置播放时间估算
    playOffset = 0;
    playStartTime = 0;
    isVideoPlaying = false;

    if (isAudio) {
        document.getElementById('videoWrap').style.display = 'none';
        document.getElementById('audioWrap').style.display = 'block';
        document.getElementById('audioTitle').textContent = name.replace(/\.[^.]+$/, '');
        audioEl.src = url + '?t=' + Date.now();
        audioEl.volume = lastVolume;
        audioEl.play().catch(function(){});
    } else {
        document.getElementById('videoWrap').style.display = 'block';
        document.getElementById('audioWrap').style.display = 'none';
        videoEl.src = url + '?t=' + Date.now();
        videoEl.volume = lastVolume;
        videoEl.play().catch(function(){});
    }
}

function switchTrack(trackIndex) {
    lastTrackIndex = trackIndex;
    document.getElementById('btnOrigin').classList.toggle('active', trackIndex === 0);
    document.getElementById('btnAcc').classList.toggle('active', trackIndex === 1);

    // 优先尝试浏览器原生audioTracks API
    if (videoEl.audioTracks && videoEl.audioTracks.length > 1) {
        for (var i = 0; i < videoEl.audioTracks.length; i++) {
            videoEl.audioTracks[i].enabled = (i === trackIndex);
        }
        return;
    }

    // 视频文件：通过流媒体URL切换音轨
    if (currentPlayingIndex >= 0 && currentPlayingIndex < queue.length) {
        var item = queue[currentPlayingIndex];
        if (!isAudioFile(item.name)) {
            var url = '/api/stream?name=' + encodeURIComponent(item.path) + '&trackIndex=' + trackIndex + '&_t=' + Date.now();
            videoEl.src = url;
            videoEl.volume = lastVolume;
            videoEl.play().catch(function(){});
            showToast(trackIndex === 0 ? '已切换: 原唱' : '已切换: 伴奏');
        }
    }
}

// 队列操作
function playNow(path, name, type) {
    queue = [];
    currentPlayingIndex = -1;
    queue.push({path:path, name:name, type:type, status:"checking", transcodeProgress:0, requestKey:path});
    renderQueue();
    checkAndRequestTranscode(0);
}

function addToQueue(path, name, type, displayName) {
	    for (var i = 0; i < queue.length; i++) {
	        if (queue[i].name === name) {
	            showToast('队列中已有: ' + (displayName||name).replace(/\.[^.]+$/, ''), 'error');
	            return;
	        }
	    }
	    var idx = queue.length;
	    queue.push({path:path, name:name, type:type, displayName:displayName||'', status:"checking", transcodeProgress:0, requestKey:path});
	    showToast('已点: ' + (displayName||name).replace(/\.[^.]+$/, ''));
    renderQueue();
    updateQueueTab();
    checkAndRequestTranscode(idx);
}

function updateQueueTab() {
    document.getElementById('tabQueue').textContent = '队列（' + queue.length + '）';
}

function randomSong() {
    fetch('/api/random-song').then(function(r){return r.json();}).then(function(data){
        if (data.name && data.path) {
            addToQueue(data.path, data.name, data.type || 'video');
        } else {
            showToast('曲库为空', 'error');
        }
    }).catch(function(){ showToast('随机点歌失败', 'error'); });
}

function showToast(msg, type) {
    var el = document.getElementById('toast');
    el.textContent = msg;
    el.className = 'toast show' + (type ? ' ' + type : '');
    setTimeout(function() { el.className = 'toast'; }, 1500);
}

function checkAndRequestTranscode(idx) {
    if (idx < 0 || idx >= queue.length) return;
    var item = queue[idx];

    // 视频文件使用流媒体播放，无需预转码
    if (!isAudioFile(item.name)) {
        item.status = "ready";
        renderQueue();
        tryAutoPlay();
        return;
    }

    // 音频文件保持原有转码检查逻辑
    var xhr = new XMLHttpRequest();
    xhr.open('POST', '/api/transcode/check-and-add', true);
    xhr.setRequestHeader('Content-Type', 'application/json');
    xhr.onload = function() {
        if (xhr.status === 200) {
            try {
                var data = JSON.parse(xhr.responseText);
                if (data.needsTranscode) {
                    item.status = data.status;
                    item.queuePosition = data.queuePosition;
                    if (data.codecInfo) item.codecInfo = data.codecInfo;
                } else {
                    item.status = "ready";
                }
                renderQueue();
                if (item.status === "ready") {
                    tryAutoPlay();
                }
            } catch(e) {
                item.status = "ready";
                renderQueue();
                tryAutoPlay();
            }
        }
    };
    xhr.send(JSON.stringify({fileName: item.path, requestKey: item.requestKey}));
}

function tryAutoPlay() {
    if (currentPlayingIndex >= 0 && currentPlayingIndex < queue.length) return;
    for (var i = 0; i < queue.length; i++) {
        if (queue[i].status === "ready") {
            currentPlayingIndex = i;
            playQueueItem(i);
            return;
        }
    }
}

function prevSong() {
    if (queue.length === 0) return;
    currentPlayingIndex = 0;
    playQueueItem(0);
}

function playQueueItem(idx) {
    if (idx < 0 || idx >= queue.length) return;
    var item = queue[idx];
    mediaDuration = 0;

    if (isAudioFile(item.name)) {
        var url = '/file?name=' + encodeURIComponent(item.path);
        playMedia(url, item.name);
    } else {
        // 视频文件使用流媒体播放，通过trackIndex参数切换音轨
        var url = '/api/stream?name=' + encodeURIComponent(item.path) + '&trackIndex=' + lastTrackIndex + '&_t=' + Date.now();
        playMedia(url, item.name);
        // 异步获取时长（流媒体无法从video.duration获取）
        fetch('/api/media-duration?name=' + encodeURIComponent(item.path)).then(function(r){return r.json();}).then(function(data){
            if (data.duration && data.duration > 0) {
                mediaDuration = data.duration;
                updateTimeDisplay();
            }
        }).catch(function(){});
    }

    renderQueue();
    // 更新QR队列
    fetch('/api/qr/queue-update', {
        method: 'POST',
        headers: {'Content-Type': 'application/json'},
        body: JSON.stringify({queue: queue.map(function(it) { return {name: it.name, status: it.status}; }), currentPlayingIndex: currentPlayingIndex})
    }).catch(function(){});
}

function nextSong() {
    if (queue.length === 0) return;
    if (currentPlayingIndex >= 0 && currentPlayingIndex < queue.length) {
        queue.splice(currentPlayingIndex, 1);
        if (currentPlayingIndex >= queue.length) currentPlayingIndex = -1;
    }
    renderQueue();
    updateQueueTab();
    if (currentPlayingIndex === -1) {
        tryAutoPlay();
    } else {
        playQueueItem(currentPlayingIndex);
    }
}

function topQueue(idx) {
    var item = queue.splice(idx, 1)[0];
    var targetIdx = currentPlayingIndex >= 0 ? currentPlayingIndex + 1 : 0;
    queue.splice(targetIdx, 0, item);
    if (idx < currentPlayingIndex) currentPlayingIndex--;
    renderQueue();
}

function delQueue(idx) {
    var wasPlaying = (idx === currentPlayingIndex);
    queue.splice(idx, 1);
    if (wasPlaying) {
        currentPlayingIndex = -1;
        tryAutoPlay();
    } else if (idx < currentPlayingIndex) {
        currentPlayingIndex--;
    }
    renderQueue();
    updateQueueTab();
}

function renderQueue() {
    var box = document.getElementById('queueList');
    if (queue.length === 0) {
        box.innerHTML = '<div class="empty-hint">暂无播放队列</div>';
        return;
    }
    var html = '';
    for (var i = 0; i < queue.length; i++) {
        var item = queue[i];
        var isCurrent = (i === currentPlayingIndex);
        var statusText = '';
        var statusClass = '';
        if (isCurrent) {
            statusText = '播放中';
            statusClass = 'playing';
        } else if (item.status === 'transcoding') {
            statusText = '转码' + item.transcodeProgress + '%';
            statusClass = 'transcoding';
        } else if (item.status === 'waiting') {
            statusText = '等待转码';
            statusClass = 'waiting';
        } else if (item.status === 'checking') {
            statusText = '检查中';
        } else if (item.status === 'ready') {
            statusText = '已就绪';
            statusClass = 'ready';
        }
        html += '<div class="queue-item' + (isCurrent ? ' playing' : '') + '">';
        html += '<span class="name">' + (i+1) + '. ' + (item.displayName||item.name) + '</span>';
        html += '<span class="status ' + statusClass + '">' + statusText + '</span>';
        if (i !== currentPlayingIndex && i > 0) {
            html += '<button class="top-btn" onclick="topQueue(' + i + ')">置顶</button>';
        }
        html += '<button class="del-btn" onclick="delQueue(' + i + ')">删除</button>';
        html += '</div>';
    }
    box.innerHTML = html;
}

// 搜索
function filterList() {
    currentKeyword = document.getElementById('searchInput').value;
    currentPage = 1;
    loadSongs(currentPage, currentKeyword);
}

function loadSongs(page, keyword) {
    var songList = document.getElementById('songList');
    songList.innerHTML = '<div class="empty-hint">加载中...</div>';
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
            songList.innerHTML = '<div class="empty-hint">加载失败</div>';
        }
    };
    xhr.send();
}

function renderSongList(songs) {
    var songList = document.getElementById('songList');
    if (!songs || songs.length === 0) {
        songList.innerHTML = '<div class="empty-hint">' + (currentKeyword ? '未找到相关歌曲' : '输入关键字搜索歌曲') + '</div>';
        return;
    }
    var html = '';
    for (var i = 0; i < songs.length; i++) {
        var s = songs[i];
        var showName = s.displayName || s.name;
        html += '<div class="song-item" data-path="' + s.path + '" onclick="addToQueue(\'' + s.path.replace(/'/g, "\\'") + '\',\'' + s.name.replace(/'/g, "\\'") + '\',\'' + s.type + '\',\'' + (showName !== s.name ? showName.replace(/'/g, "\\'") : '') + '\')">' + showName + '</div>';
    }
    songList.innerHTML = html;
    showMobilePlayCounts(songs);
}

var mobileHotPlayCache = null;
var mobileHotPlayCacheTime = 0;
function showMobilePlayCounts(songs) {
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
                        badge.style.cssText = 'float:right;font-size:11px;color:#5bc0de;margin-left:8px;';
                        badge.textContent = count + '次';
                        items[j].appendChild(badge);
                    }
                }
            }
        }
    };
    if (mobileHotPlayCache && now - mobileHotPlayCacheTime < 30000) {
        doRender(mobileHotPlayCache);
        return;
    }
    fetch('/api/hot-songs').then(function(r){return r.json();}).then(function(data){
        mobileHotPlayCache = {};
        for (var i = 0; i < data.length; i++) {
            mobileHotPlayCache[data[i].path] = data[i].count;
        }
        mobileHotPlayCacheTime = now;
        doRender(mobileHotPlayCache);
    }).catch(function(){});
}

function updatePagination() {
    var pg = document.getElementById('pagination');
    if (totalPages <= 1) { pg.innerHTML = ''; return; }
    var html = '<button class="page-btn" onclick="changePage(' + (currentPage-1) + ')" ' + (currentPage===1?'disabled':'') + '>上一页</button>';
    var maxBtns = 5;
    if (totalPages <= maxBtns) {
        for (var i = 1; i <= totalPages; i++) {
            html += '<button class="page-btn' + (i===currentPage?' active':'') + '" onclick="changePage(' + i + ')">' + i + '</button>';
        }
    } else {
        var step = Math.ceil(totalPages / maxBtns);
        for (var i = 0; i < maxBtns; i++) {
            var p = (i + 1) * step;
            if (p > totalPages) p = totalPages;
            html += '<button class="page-btn' + (p===currentPage?' active':'') + '" onclick="changePage(' + p + ')">' + p + '</button>';
        }
    }
    html += '<button class="page-btn" onclick="changePage(' + (currentPage+1) + ')" ' + (currentPage===totalPages?'disabled':'') + '>下一页</button>';
    pg.innerHTML = html;
}

function changePage(page) {
    if (page < 1 || page > totalPages) return;
    loadSongs(page, currentKeyword);
}

// 转码轮询
function pollTranscodeStatus() {
    var needPoll = false;
    for (var i = 0; i < queue.length; i++) {
        if (queue[i].status === 'waiting' || queue[i].status === 'transcoding') { needPoll = true; break; }
    }
    if (!needPoll) return;
    for (var i = 0; i < queue.length; i++) {
        if (queue[i].status === 'waiting' || queue[i].status === 'transcoding') {
            (function(idx, key) {
                var xhr = new XMLHttpRequest();
                xhr.open('GET', '/api/transcode/status?requestKey=' + encodeURIComponent(key), true);
                xhr.onload = function() {
                    if (xhr.status === 200) {
                        try {
                            var data = JSON.parse(xhr.responseText);
                            if (idx < queue.length && queue[idx].requestKey === key) {
                                queue[idx].status = data.status;
                                queue[idx].transcodeProgress = data.progress;
                                queue[idx].queuePosition = data.queuePosition;
                                if (data.codecInfo) queue[idx].codecInfo = data.codecInfo;
                                if (data.outputPath && data.status === 'completed') {
                                    queue[idx].path = data.outputPath;
                                    queue[idx].name = data.outputPath.split('/').pop();
                                    queue[idx].status = 'ready';
                                    renderQueue();
                                    tryAutoPlay();
                                } else {
                                    renderQueue();
                                    if (data.status === 'ready') {
                                        tryAutoPlay();
                                    }
                                }
                            }
                        } catch(e) {}
                    }
                };
                xhr.send();
            })(i, queue[i].requestKey);
        }
    }
}

// QR轮询
var qrPollInterval = null;
function startQRPoll() {
    if (qrPollInterval) return;
    qrPollInterval = setInterval(function() {
        fetch('/api/qr/pending-songs').then(function(r){return r.json()}).then(function(data) {
            if (data.songs && data.songs.length > 0) {
                data.songs.forEach(function(song) { addToQueue(song.path, song.name, song.type); });
            }
        });
    }, 2000);
}

// 初始化
window.onload = function() {
    transcodePollInterval = setInterval(pollTranscodeStatus, 2000);
    fetch('/api/qr/status').then(function(r){return r.json()}).then(function(data) {
        if (data.enabled) startQRPoll();
    }).catch(function(){});
};
</script>
</body>
</html>`
	template.Must(template.New("mobile").Parse(tpl)).Execute(w, nil)
}

func isMobileBrowser(userAgent string) bool {
	ua := strings.ToLower(userAgent)
	return strings.Contains(ua, "android") ||
		strings.Contains(ua, "iphone") ||
		strings.Contains(ua, "ipad") ||
		strings.Contains(ua, "ipod") ||
		strings.Contains(ua, "mobile") ||
		strings.Contains(ua, "windows phone")
}
