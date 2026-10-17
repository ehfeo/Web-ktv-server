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
<link rel="icon" href="/favicon.ico">
<style>
*{margin:0;padding:0;box-sizing:border-box;font-family:Microsoft YaHei,-apple-system,sans-serif}
html,body{width:100%;height:100%;overflow:hidden;background:#1a252f;color:#ecf0f1}
.app{display:flex;flex-direction:column;height:100vh;width:100%}

/* 播放器区域 */
.player-area{width:100%;background:#0d1922;position:relative;flex-shrink:0;box-shadow:0 2px 6px rgba(0,0,0,0.3)}
.video-wrap{width:100%;aspect-ratio:16/9;background:#000;position:relative}
.video-wrap video,.video-wrap audio{width:100%;height:100%;object-fit:contain}
.audio-wrap{width:100%;padding:12px 14px;background:#152029;box-shadow:inset 0 1px 0 rgba(255,255,255,0.03);display:flex;flex-direction:column;gap:10px;flex-shrink:0}
.audio-wrap audio{width:100%;flex-shrink:0}
.audio-title{text-align:center;color:#ecf0f1;font-size:15px;white-space:nowrap;overflow:hidden;text-overflow:ellipsis;flex-shrink:0}
/* 电平表（LED 条） */
.meter-wrap{flex-shrink:0;background:#0b1419;border:1px solid #253644;border-radius:4px;padding:6px 9px;box-shadow:inset 0 1px 3px rgba(0,0,0,0.4);display:flex;flex-direction:column;gap:3px}
.meter-row{display:flex;align-items:center;gap:6px}
.meter-label{width:12px;font-size:10px;color:#7f8c8d;flex-shrink:0;text-align:center;font-weight:bold}
.meter-leds{display:flex;gap:2px;height:11px;width:100%}
.meter-scale-row{margin-bottom:1px}
.meter-scale{position:relative;flex:1;height:9px;font-size:9px;color:#6b7b85;font-weight:bold}
.meter-scale span{position:absolute;top:0;transform:translateX(-50%);line-height:9px;text-shadow:0 1px 1px rgba(0,0,0,0.5)}
.meter-scale .scale-end{left:100%;transform:translateX(-100%)}
.meter-led{flex:1;background:#1c2b38;border-radius:2px;box-shadow:inset 0 1px 2px rgba(0,0,0,0.5);transition:background 0.05s}
.meter-led.on{background:#5cf06a;box-shadow:0 0 4px rgba(92,240,106,0.5)}
.meter-led.yellow{background:#ffd600;box-shadow:0 0 4px rgba(255,214,0,0.5)}
.meter-led.red{background:#ff3355;box-shadow:0 0 5px rgba(255,51,85,0.6)}
.meter-led.peak{background:#ff9f43;box-shadow:0 0 5px rgba(255,159,67,0.9);outline:1px solid #fff;outline-offset:1px}
/* 歌词 */
.mobile-lyrics{height:170px;overflow:hidden;position:relative;background:rgba(10,18,24,0.6);border-radius:4px;border:1px solid #253644;flex-shrink:0}
.mobile-lyrics-inner{position:absolute;left:0;right:0;top:0;text-align:center;transition:transform 0.35s ease;will-change:transform}
.mline{font-size:16px;color:#7f8c8d;padding:6px 12px;font-weight:bold;text-shadow:0 1px 2px rgba(0,0,0,0.5);transition:all 0.2s}
.mline.current{color:#ffd600;font-size:20px}
.no-lyrics{text-align:center;color:#546a7b;line-height:170px;font-size:13px}
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
/* 歌词选择弹窗 */
.lc-modal{position:fixed;inset:0;z-index:200;background:rgba(0,0,0,0.65);display:flex;align-items:center;justify-content:center;padding:20px}
.lc-dialog{width:92%;max-width:420px;max-height:78vh;background:#1e2d3a;border:1px solid #2c3e50;border-radius:8px;overflow:hidden;display:flex;flex-direction:column;box-shadow:0 4px 20px rgba(0,0,0,0.5)}
.lc-head{display:flex;align-items:center;justify-content:space-between;padding:10px 14px;border-bottom:1px solid #2c3e50;color:#fff;font-weight:bold;font-size:15px}
.lc-head button{background:#d9534f;color:#fff;border:none;border-radius:4px;padding:5px 14px;cursor:pointer;font-weight:bold}
.lc-inputs{display:flex;gap:8px;align-items:center;padding:10px 14px;border-bottom:1px solid #2c3e50;flex-wrap:wrap}
.lc-inputs input{flex:1;min-width:110px;background:#0d1922;border:1px solid #34495e;border-radius:4px;color:#ecf0f1;padding:8px 10px;font-size:14px;outline:none}
.lc-inputs input:focus{border-color:#428bca}
.lc-inputs button{background:#428bca;color:#fff;border:none;border-radius:4px;padding:9px 16px;cursor:pointer;font-weight:bold;white-space:nowrap}
.lc-list{flex:1;overflow-y:auto;padding:8px;-webkit-overflow-scrolling:touch}
.lc-item{padding:10px 12px;margin:4px 0;background:#243744;border:1px solid #2c3e50;border-radius:4px;cursor:pointer;color:#fff;display:flex;align-items:center;gap:8px;overflow:hidden;white-space:nowrap}
.lc-item:active{background:#428bca}
.lc-item .badge{background:#428bca;color:#fff;border-radius:3px;padding:2px 6px;font-size:12px;white-space:nowrap;flex-shrink:0}
.lc-item .info{overflow:hidden;text-overflow:ellipsis;white-space:nowrap}
</style>
</head>
<body>
<div class="toast" id="toast"></div>
<div class="lc-modal" id="errModal" style="display:none">
    <div class="lc-dialog" style="padding:26px 24px;align-items:center;text-align:center;overflow:visible">
        <div style="font-size:38px;color:#d9534f;margin-bottom:8px;line-height:1">&#9888;</div>
        <div style="font-size:17px;color:#fff;font-weight:bold;margin-bottom:8px">播放失败</div>
        <div style="font-size:14px;color:#aebfce;line-height:1.7;margin-bottom:18px">该曲目文件损坏或无法解码，<br>5 秒后自动播放下一首</div>
        <button onclick="errSkipNow()" style="background:#428bca;color:#fff;border:none;border-radius:6px;padding:10px 28px;font-size:15px;font-weight:bold;cursor:pointer">立即播放下一首</button>
    </div>
</div>
<div class="lc-modal" id="lcModal" style="display:none">
    <div class="lc-dialog">
        <div class="lc-head">选择歌词<button onclick="closeMobileLyricPicker()">关闭</button></div>
        <div class="lc-inputs">
            <input id="lcTitle" placeholder="歌名">
            <input id="lcArtist" placeholder="歌手">
            <button onclick="mobileLyricSearch()">搜索歌词</button>
        </div>
        <div class="lc-list" id="lcList"><div class="empty-hint" style="padding:30px">输入歌名/歌手后点击“搜索歌词”</div></div>
    </div>
</div>
<div class="app">
    <!-- 播放器 -->
    <div class="player-area" id="playerArea">
        <div class="video-wrap" id="videoWrap" style="display:none">
            <video id="videoPlayer" playsinline></video>
        </div>
        <div class="audio-wrap" id="audioWrap" style="display:none">
            <div class="audio-title" id="audioTitle"></div>
            <div class="meter-wrap">
                <div class="meter-row meter-scale-row"><span class="meter-label"></span><div class="meter-scale"><span style="left:0">-30</span><span style="left:33.333%">-20</span><span style="left:66.667%">-10</span><span class="scale-end">0</span></div></div>
                <div class="meter-row"><span class="meter-label">L</span><div class="meter-leds" id="meterLedsL"></div></div>
                <div class="meter-row"><span class="meter-label">R</span><div class="meter-leds" id="meterLedsR"></div></div>
            </div>
            <div class="mobile-lyrics" id="mobileLyrics"><div class="no-lyrics">播放音频时显示歌词与电平表</div></div>
            <audio id="audioPlayer" controls playsinline></audio>
        </div>
    </div>
    <div class="player-controls" id="playerControls">
        <button class="ctrl-btn" onclick="prevSong()">重播</button>
        <button class="ctrl-btn" onclick="nextSong()">下一首</button>
        <button class="ctrl-btn active" id="btnOrigin" onclick="switchTrack(0)">原唱</button>
        <button class="ctrl-btn" id="btnAcc" onclick="switchTrack(1)">伴奏</button>
        <button class="ctrl-btn" id="btnLyricPick" onclick="openMobileLyricPicker()">选歌词</button>
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
    return ['.mp3','.wav','.flac','.aac','.m4a','.m4r','.alac','.ogg','.oga','.opus','.wma','.ape','.aiff','.aif','.amr','.dvf','.msv','.dts','.dff','.dsf','.sacd','.tak','.tta','.wv','.mka'].indexOf(ext) !== -1;
}
// 判断该音频文件是否需要实时转码才能在手机端播放：
// 浏览器无法直接解码 dts/dff/dsf/wma/ape 等，必须经过 ffmpeg 转码为 AAC 流式输出。
function needsLiveAudioTranscode(fileName) {
    var ext = fileName.toLowerCase().substring(fileName.lastIndexOf('.'));
    // mp3/wav/flac/aac/m4a/ogg/opus 浏览器可原生播放，走 /file
    // dts/dff/dsf/wma/ape/aiff/amr/dvf/msv/sacd/tak/tta/wv/mka 需要转码
    var need = ['.dts','.dff','.dsf','.wma','.ape','.aiff','.aif','.amr','.dvf','.msv','.sacd','.tak','.tta','.wv','.mka'];
    return need.indexOf(ext) !== -1;
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

// ===== 歌手索引分页 =====
// /api/singers 现在按字母+分页返回：不带 letter 只回 {"letters":[...]} 轻量字母条，
// 带 letter&page 返回该字母下的歌手分页，避免全量歌手一次性拉取+渲染导致手机卡顿。
var mSingerLetter = '';
var mSingerPage = 1;
var mSingerTotal = 0;
var mSingerTotalPages = 1;

function loadMobileSingerIndex() {
    mobileSingerData = { letters: [] };
    fetch('/api/singers').then(function(r){return r.json();}).then(function(data){
        mobileSingerData = data;
        var letters = (data.letters || []).sort();
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
    mSingerLetter = letter;
    mSingerPage = 1;
    var spans = document.getElementById('singerLetters').children;
    for (var i = 0; i < spans.length; i++) {
        spans[i].className = spans[i].id === 'mletter_' + letter ? 'active' : '';
    }
    // 切换字母后让歌手名单回到最顶部
    window.scrollTo(0, 0);
    loadMobileSingers(letter, 1);
}

function loadMobileSingers(letter, page) {
    mSingerLetter = letter;
    mSingerPage = page;
    var listEl = document.getElementById('singerList');
    listEl.innerHTML = '<div style="text-align:center;color:#555;padding:20px">加载中...</div>';
    var xhr = new XMLHttpRequest();
    var url = '/api/singers?letter=' + encodeURIComponent(letter) + '&page=' + page + '&pageSize=' + pageSize;
    xhr.open('GET', url, true);
    xhr.onload = function() {
        if (xhr.status !== 200) { listEl.innerHTML = '<div style="text-align:center;color:#555;padding:20px">加载失败</div>'; return; }
        var data = JSON.parse(xhr.responseText);
        mSingerTotal = data.total;
        mSingerTotalPages = data.totalPages;
        var singers = data.items || [];
        var html = '<div class="singer-grid">';
        for (var i = 0; i < singers.length; i++) {
            html += '<div class="singer-btn" onclick="loadMobileSingerSongs(\'' + singers[i].name.replace(/'/g, "\\'") + '\')">';
            html += '<span class="sname">' + singers[i].name + '</span>';
            html += '<span class="scount">' + singers[i].count + '首</span>';
            html += '</div>';
        }
        html += '</div>';
        if (singers.length === 0) html = '<div style="text-align:center;color:#555;padding:20px">暂无歌手</div>';
        else html += buildSingerPagination();
        listEl.innerHTML = html;
    };
    xhr.send();
}

function sChangePage(page) {
    if (page < 1 || page > mSingerTotalPages) return;
    loadMobileSingers(mSingerLetter, page);
}

function buildSingerPagination() {
    if (mSingerTotalPages <= 1) return '';
    var html = '<div style="display:flex;justify-content:center;padding:12px 0">';
    html += '<button class="page-btn" onclick="sChangePage(' + (mSingerPage - 1) + ')" ' + (mSingerPage === 1 ? 'disabled' : '') + '>上一页</button>';
    var maxBtns = 5;
    if (mSingerTotalPages <= maxBtns) {
        for (var i = 1; i <= mSingerTotalPages; i++) {
            html += '<button class="page-btn' + (i === mSingerPage ? ' active' : '') + '" onclick="sChangePage(' + i + ')">' + i + '</button>';
        }
    } else {
        var step = Math.ceil(mSingerTotalPages / maxBtns);
        for (var i = 0; i < maxBtns; i++) {
            var p = (i + 1) * step;
            if (p > mSingerTotalPages) p = mSingerTotalPages;
            html += '<button class="page-btn' + (p === mSingerPage ? ' active' : '') + '" onclick="sChangePage(' + p + ')">' + p + '</button>';
        }
    }
    html += '<button class="page-btn" onclick="sChangePage(' + (mSingerPage + 1) + ')" ' + (mSingerPage === mSingerTotalPages ? 'disabled' : '') + '>下一页</button>';
    html += '</div>';
    return html;
}

// ===== 分类歌单分页（按歌手/语种/曲种）=====
// 分类下歌曲可能成千上万，一次全量渲染会让手机浏览器卡住；
// 改为与搜索一致的分页加载，既省流量又降低浏览器压力。
var mSongType = '';        // 'singer' | 'language' | 'category'
var mSongKey = '';         // 当前选中的分类键（歌手名/语种/曲种）
var mSongPage = 1;
var mSongTotal = 0;
var mSongTotalPages = 1;

// 分类歌单的“返回”头部，随分类类型不同而回退到对应索引列表
function mSongBackHtml() {
    if (mSongType === 'singer') {
        return '<div class="singer-back" onclick="selectMobileLetter(\'' + mobileSingerLetter + '\')">&#8592; 返回歌手列表</div>';
    }
    if (mSongType === 'language') {
        return '<div class="singer-back" onclick="renderMobileLanguageList(mobileLanguageData)">&#8592; 返回语种列表</div>';
    }
    return '<div class="singer-back" onclick="renderMobileCategoryList(mobileCategoryData)">&#8592; 返回曲种列表</div>';
}

// 分类歌单目标容器，随类型不同选对应的面板
function mSongTarget() {
    if (mSongType === 'singer') return document.getElementById('singerList');
    if (mSongType === 'language') return document.getElementById('languageList');
    return document.getElementById('categoryList');
}

// 统一的分页加载入口：type 为 singer/language/category，key 为具体分类值
function loadMobileSongs(type, key, page) {
    mSongType = type;
    mSongKey = key;
    mSongPage = page;
    var target = mSongTarget();
    target.innerHTML = mSongBackHtml() + '<div class="empty-hint">加载中...</div>';

    var path = '/api/songs-by-singer?singer=';
    if (type === 'language') path = '/api/songs-by-language?language=';
    else if (type === 'category') path = '/api/songs-by-category?category=';

    var url = path + encodeURIComponent(key) + '&page=' + page + '&pageSize=' + pageSize;
    var xhr = new XMLHttpRequest();
    xhr.open('GET', url, true);
    xhr.onload = function() {
        if (xhr.status !== 200) {
            target.innerHTML = mSongBackHtml() + '<div class="empty-hint">加载失败</div>';
            return;
        }
        var data = JSON.parse(xhr.responseText);
        mSongTotal = data.total;
        mSongTotalPages = data.totalPages;
        var songs = data.songs;
        var html = mSongBackHtml();
        html += '<div style="padding:8px;color:#0af;font-size:15px;text-align:center">' + key + ' (' + mSongTotal + '首)</div>';
        if (!songs || songs.length === 0) {
            html += '<div class="empty-hint">暂无歌曲</div>';
        } else {
            for (var i = 0; i < songs.length; i++) {
                var s = songs[i];
                var showName = s.displayName || s.name;
                html += '<div class="song-item" data-path="' + s.path + '" onclick="addToQueue(\'' + s.path.replace(/'/g, "\\'") + '\',\'' + s.name.replace(/'/g, "\\'") + '\',\'' + s.type + '\',\'' + (showName !== s.name ? showName.replace(/'/g, "\\'") : '') + '\')">' + showName + '</div>';
            }
            html += buildMobilePagination();
        }
        target.innerHTML = html;
        if (songs && songs.length > 0) showMobilePlayCounts(songs);
    };
    xhr.send();
}

function mChangePage(page) {
    if (page < 1 || page > mSongTotalPages) return;
    loadMobileSongs(mSongType, mSongKey, page);
}

function buildMobilePagination() {
    if (mSongTotalPages <= 1) return '';
    var html = '<div style="display:flex;justify-content:center;padding:12px 0">';
    html += '<button class="page-btn" onclick="mChangePage(' + (mSongPage - 1) + ')" ' + (mSongPage === 1 ? 'disabled' : '') + '>上一页</button>';
    var maxBtns = 5;
    if (mSongTotalPages <= maxBtns) {
        for (var i = 1; i <= mSongTotalPages; i++) {
            html += '<button class="page-btn' + (i === mSongPage ? ' active' : '') + '" onclick="mChangePage(' + i + ')">' + i + '</button>';
        }
    } else {
        var step = Math.ceil(mSongTotalPages / maxBtns);
        for (var i = 0; i < maxBtns; i++) {
            var p = (i + 1) * step;
            if (p > mSongTotalPages) p = mSongTotalPages;
            html += '<button class="page-btn' + (p === mSongPage ? ' active' : '') + '" onclick="mChangePage(' + p + ')">' + p + '</button>';
        }
    }
    html += '<button class="page-btn" onclick="mChangePage(' + (mSongPage + 1) + ')" ' + (mSongPage === mSongTotalPages ? 'disabled' : '') + '>下一页</button>';
    html += '</div>';
    return html;
}

function loadMobileSingerSongs(singer) {
    loadMobileSongs('singer', singer, 1);
}

// ===== 语种/曲种索引分页 =====
// /api/languages、/api/categories 已分页返回 {items,total,...}，前端同样分页渲染，
// 避免零碎残障的语种/曲种条目过多导致卡顿。
var mLangPage = 1;
var mLangTotalPages = 1;
var mCatPage = 1;
var mCatTotalPages = 1;

function loadMobileLanguageIndex() {
    mLangPage = 1;
    loadMobileLanguages(1);
}

function loadMobileLanguages(page) {
    mLangPage = page;
    var el = document.getElementById('languageList');
    el.innerHTML = '<div class="empty-hint">加载中...</div>';
    var xhr = new XMLHttpRequest();
    xhr.open('GET', '/api/languages?page=' + page + '&pageSize=' + pageSize, true);
    xhr.onload = function() {
        if (xhr.status !== 200) { el.innerHTML = '<div class="empty-hint">加载失败</div>'; return; }
        var data = JSON.parse(xhr.responseText);
        mobileLanguageData = data;
        mLangTotalPages = data.totalPages;
        renderMobileLanguageList(data);
    };
    xhr.send();
}

function renderMobileLanguageList(data) {
    var items = data && data.items ? data.items : (data || []);
    var html = '<div class="cat-grid">';
    for (var i = 0; i < items.length; i++) {
        html += '<div class="cat-btn" onclick="loadMobileLanguageSongs(\'' + items[i].name.replace(/'/g, "\\'") + '\')">';
        html += '<span class="cname">' + items[i].name + '</span>';
        html += '<span class="ccount">' + items[i].count + '首</span>';
        html += '</div>';
    }
    html += '</div>';
    if (items.length === 0) html = '<div class="empty-hint">暂无语种</div>';
    else html += '<div style="display:flex;justify-content:center;padding:12px 0">' + buildCatPagination('lang') + '</div>';
    document.getElementById('languageList').innerHTML = html;
}

function loadMobileLanguageSongs(lang) {
    loadMobileSongs('language', lang, 1);
}

function loadMobileCategoryIndex() {
    mCatPage = 1;
    loadMobileCategories(1);
}

function loadMobileCategories(page) {
    mCatPage = page;
    var el = document.getElementById('categoryList');
    el.innerHTML = '<div class="empty-hint">加载中...</div>';
    var xhr = new XMLHttpRequest();
    xhr.open('GET', '/api/categories?page=' + page + '&pageSize=' + pageSize, true);
    xhr.onload = function() {
        if (xhr.status !== 200) { el.innerHTML = '<div class="empty-hint">加载失败</div>'; return; }
        var data = JSON.parse(xhr.responseText);
        mobileCategoryData = data;
        mCatTotalPages = data.totalPages;
        renderMobileCategoryList(data);
    };
    xhr.send();
}

function renderMobileCategoryList(data) {
    var items = data && data.items ? data.items : (data || []);
    var html = '<div class="cat-grid">';
    for (var i = 0; i < items.length; i++) {
        html += '<div class="cat-btn" onclick="loadMobileCategorySongs(\'' + items[i].name.replace(/'/g, "\\'") + '\')">';
        html += '<span class="cname">' + items[i].name + '</span>';
        html += '<span class="ccount">' + items[i].count + '首</span>';
        html += '</div>';
    }
    html += '</div>';
    if (items.length === 0) html = '<div class="empty-hint">暂无曲种</div>';
    else html += '<div style="display:flex;justify-content:center;padding:12px 0">' + buildCatPagination('cat') + '</div>';
    document.getElementById('categoryList').innerHTML = html;
}

function catChangePage(kind, page) {
    if (kind === 'lang') {
        if (page < 1 || page > mLangTotalPages) return;
        loadMobileLanguages(page);
    } else {
        if (page < 1 || page > mCatTotalPages) return;
        loadMobileCategories(page);
    }
}

function buildCatPagination(kind) {
    var cur = kind === 'lang' ? mLangPage : mCatPage;
    var total = kind === 'lang' ? mLangTotalPages : mCatTotalPages;
    if (total <= 1) return '';
    var html = '<button class="page-btn" onclick="catChangePage(\'' + kind + '\',' + (cur - 1) + ')" ' + (cur === 1 ? 'disabled' : '') + '>上一页</button>';
    var maxBtns = 5;
    if (total <= maxBtns) {
        for (var i = 1; i <= total; i++) {
            html += '<button class="page-btn' + (i === cur ? ' active' : '') + '" onclick="catChangePage(\'' + kind + '\',' + i + ')">' + i + '</button>';
        }
    } else {
        var step = Math.ceil(total / maxBtns);
        for (var i = 0; i < maxBtns; i++) {
            var p = (i + 1) * step;
            if (p > total) p = total;
            html += '<button class="page-btn' + (p === cur ? ' active' : '') + '" onclick="catChangePage(\'' + kind + '\',' + p + ')">' + p + '</button>';
        }
    }
    html += '<button class="page-btn" onclick="catChangePage(\'' + kind + '\',' + (cur + 1) + ')" ' + (cur === total ? 'disabled' : '') + '>下一页</button>';
    return html;
}

function loadMobileCategorySongs(cat) {
    loadMobileSongs('category', cat, 1);
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

// 播放器
var videoEl = document.getElementById('videoPlayer');
var audioEl = document.getElementById('audioPlayer');
var mediaDuration = 0; // 服务端获取的媒体时长（流媒体无法从video.duration获取）

// 媒体加载失败时探测响应体：v视频/音频经 /api/stream 实时转码推流时，
// 若服务端因并发推流过多或内存不足会被拒绝，需给出明确提示而非安静失败。
function probeMobileMediaError(target) {
  var src = '';
  try { src = target.currentSrc || target.src; } catch(e) {}
  if (!src) return;
  var probe = new XMLHttpRequest();
  probe.open('GET', src, false); // 同步探测，读取错误响应体
  probe.withCredentials = false;
  try { probe.send(); } catch(e) { return; }
  if (probe.status === 403 || probe.status === 503 || probe.status === 500) {
    var msg = '';
    try {
      var body = probe.responseText || '';
      if (body.charAt(0) === '{') {
        var obj = JSON.parse(body);
        msg = (obj && obj.message) || '';
      }
    } catch(e) {}
    if (msg) {
      if (typeof showToast === 'function') showToast(msg);
      else alert(msg);
    }
  }
}
// 播放失败自动跳过：弹窗提示，5 秒后自动播队列下一首（nextSong 会移除当前失败项，队列空则停，不会死循环）
var _errModalTimer = null;
function closeErrModal() {
  var m = document.getElementById('errModal');
  if (m) m.style.display = 'none';
  clearTimeout(_errModalTimer);
}
function errSkipNow() { closeErrModal(); nextSong(); }
function handleMobilePlayError(el) {
  if (el.__skipHandled) return;
  el.__skipHandled = true;
  try { el.removeAttribute('src'); } catch (e) {}
  var m = document.getElementById('errModal');
  if (!m) {
    showToast('该曲目播放失败，已自动播放下一首', 'error');
    setTimeout(function(){ nextSong(); }, 300);
    return;
  }
  m.style.display = 'flex';
  clearTimeout(_errModalTimer);
  _errModalTimer = setTimeout(function(){ closeErrModal(); nextSong(); }, 5000);
}
if (!window._mobileMediaErrorProbeInstalled) {
  window._mobileMediaErrorProbeInstalled = true;
  // 用捕获阶段监听，覆盖 videoEl 以及每次重建的 <audio> 元素
  document.addEventListener('error', function(e) {
    var t = e.target;
    if (!t || (t.tagName !== 'VIDEO' && t.tagName !== 'AUDIO')) return;
    probeMobileMediaError(t);
    handleMobilePlayError(t);
  }, true);
}

// 每次音频播放重建一个全新的 <audio> 元素并重新绑定事件。
// 参考 fftplayer：iOS/安卓上反复复用同一元素更换 src 后 MediaElementSource 常失效(有声音但 analyser 无数据)，
// 用全新元素 + 每次重建 source 规避。旧 source 一并断开销毁。
function freshMobileAudioEl() {
  var wrap = document.getElementById('audioWrap');
  if (!wrap) return audioEl;
  var old = document.getElementById('audioPlayer');
  if (old) {
    // 断开并丢弃旧的 WebAudio 源节点(旧元素已移除但 source 仍挂在 AudioContext 图上，
    // 不断开会随播放曲目累积导致主线程越来越卡、电平表滞后)
    if (old.__meshSrc) { try { old.__meshSrc.disconnect(); } catch (e) {} old.__meshSrc = null; }
    old.removeAttribute('src');
    try { old.load(); } catch (e) {}
    old.remove();
  }
  var na = document.createElement('audio');
  na.id = 'audioPlayer';
  na.controls = true;
  na.playsinline = true;
  na.addEventListener('ended', function() { nextSong(); });
  na.addEventListener('volumechange', function() { lastVolume = na.volume; });
  wrap.appendChild(na);
  // 断开并丢弃旧的 WebAudio 源(旧元素已移除)，新元素后续由 ensureMobileSource 重新接入
  if (mobileMediaSource) { try { mobileMediaSource.disconnect(); } catch (e) {} mobileMediaSource = null; }
  audioEl = na;
  return na;
}

// 每次播放都走缓存/降级模式(已确认浏览器劫持网络流)。
// 优先用 MediaSource 分段流式：MSE 同样为本地解码、不会被劫持，且边下载边播不用等整首，
// 既保证电平表又能快速起播。MSE 不可用时直接回退 http 流式(不再整首缓存，避免长时间等待)。
function startBlobFallback(name, url) {
  showToast('使用缓存模式播放...');
  // 实时转码流(经 ffmpeg 推流，如 dts 走的 music-mode-stream)：本身是流，直接播放即可，
  // 整首缓存一个实时流既无意义又很慢。
  if (url.indexOf('/api/music-mode-stream') !== -1 || url.indexOf('/api/stream') !== -1) {
    playbackHttp(url, name);
    return;
  }
  var mime = mseMimeForName(name);
  if (mime && window.MediaSource) {
    try {
      startMseStream(name, url, mime);
      return;
    } catch (e) { console.warn('MSE 启动失败，尝试直连流式:', e); }
  }
  playbackHttp(url, name);
}

// 直接 http 流式播放（实时转码流 / MSE 不可用时的回退）
function playbackHttp(url, name) {
  audioEl = freshMobileAudioEl();
  var u = (url.indexOf('blob:') === 0) ? url : (url + '?t=' + Date.now());
  audioEl.src = u;
  audioEl.volume = lastVolume;
  audioEl.play().catch(function(){});
  loadMobileLyrics(currentNameForLyrics, currentLyricsPath);
  startMobileAnalyzer(audioEl);
}

// 根据文件扩展名返回可用于 MediaSource 的 MIME，无法确定(如实时转码流)则返回空串交由直连
function mseMimeForName(name) {
  var ext = name.toLowerCase().substring(name.lastIndexOf('.'));
  if (ext === '.mp3') return 'audio/mpeg';
  if (ext === '.flac') return 'audio/flac';
  if (ext === '.wav') return 'audio/wav';
  if (ext === '.m4a' || ext === '.aac' || ext === '.alac') return 'audio/mp4';
  if (ext === '.ogg' || ext === '.oga') return 'audio/ogg';
  if (ext === '.opus') return 'audio/ogg;codecs=opus';
  return '';
}

// MediaSource 分段流式播放：边下载边 appendBuffer，收到一段就尝试播放，
// 无需等整首缓存完成，显著缩短大文件首响等待。本地解码、不受浏览器劫持，电平表正常。
// 启动失败或超时未开始均回退 http 直连流式。
function startMseStream(name, url, mime) {
  var MS = window.MediaSource || window.WebKitMediaSource;
  var ms = new MS();
  var objUrl = URL.createObjectURL(ms);
  var mseEl = freshMobileAudioEl();
  mseEl.src = objUrl;
  var sb = null;
  var pending = [];
  var appending = false;
  var started = false;
  var finished = false;
  var fellBack = false;

  function endStream() {
    if (!finished) { try { if (ms.readyState === 'open') ms.endOfStream(); } catch (e) {} finished = true; }
  }
  function revoke() { try { URL.revokeObjectURL(objUrl); } catch (e) {} }
  function fallbackHttp() {
    if (fellBack) return;
    fellBack = true;
    revoke();
    try { mseEl.removeAttribute('src'); mseEl.load(); } catch (e2) {}
    playbackHttp(url, name);
  }
  function scheduleNext() {
    if (appending || !pending.length || !sb) return;
    var data = pending.shift();
    if (data && data.byteLength) {
      appending = true;
      try { sb.appendBuffer(data); } catch (e) { appending = false; scheduleNext(); }
    } else {
      scheduleNext();
    }
  }
  function queue(data) {
    pending.push(data);
    if (!sb) return;
    scheduleNext();
  }

  ms.addEventListener('sourceopen', function() {
    try {
      sb = ms.addSourceBuffer(mime);
    } catch (e) {
      // MIME 不受支持，回退 http 直连
      fallbackHttp();
      return;
    }
    sb.addEventListener('updateend', function() {
      appending = false;
      if (!started) {
        started = true;
        mseEl.volume = lastVolume;
        mseEl.play().catch(function(){});
      }
      scheduleNext();
    });
    sb.addEventListener('error', function() { endStream(); });
    loadMobileLyrics(currentNameForLyrics, currentLyricsPath);
    startMobileAnalyzer(mseEl);

    fetch(url).then(function(r) {
      var body = r.body;
      if (body && typeof body.getReader === 'function') {
        var reader = body.getReader();
        function pump() {
          reader.read().then(function(rs) {
            if (rs.done) { endStream(); return; }
            queue(rs.value);
            pump();
          }).catch(function() { endStream(); });
        }
        pump();
      } else {
        // 无 ReadableStream API (部分国产旧浏览器)，直接回退 http 直连，
        // 不做整首缓存 blob，浏览器本身会流式下载、不阻塞首响。
        fallbackHttp();
      }
    }).catch(function() { endStream(); });
  });

  // 超时保护：一段时间没开始播放则回退 http 直连，避免永久空白
  setTimeout(function() {
    if (!started) fallbackHttp();
  }, 8000);
}

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
        // 记录当前媒体信息，供“无信号自动降级为 blob 缓存”使用
        lastMediaName = name;
        lastMediaUrl = url;
        // 已确认劫持：直接走 blob 缓存播放(免 2.5s 无信号等待)，提升体验
        if (meterFallbackForced && url.indexOf('blob:') !== 0) {
            startBlobFallback(name, url);
        } else {
            // 每次点播用全新 audio 元素，避免复用元素后 MediaElementSource 在手机端失效(有声音但无电平数据)
            audioEl = freshMobileAudioEl();
            // blob URL(本地/降级缓存)不能追加 ?t= 查询串，只有 http(s) 流才加时间戳防缓存
            audioEl.src = (url.indexOf('blob:') === 0) ? url : (url + '?t=' + Date.now());
            audioEl.volume = lastVolume;
            audioEl.play().catch(function(){});
            // 音频播放：加载歌词并启动电平表
            loadMobileLyrics(currentNameForLyrics, currentLyricsPath);
            startMobileAnalyzer(audioEl);
        }
    } else {
        document.getElementById('videoWrap').style.display = 'block';
        document.getElementById('audioWrap').style.display = 'none';
        videoEl.src = url + '?t=' + Date.now();
        videoEl.volume = lastVolume;
        videoEl.play().catch(function(){});
        // 视频：停止歌词同步与电平表
        stopMobileAnalyzer();
        setNoLyrics('视频播放，不显示歌词');
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

// 歌词手动选择（接口同桌面端：/api/lyrics/meta|candidates|save）
var mobileLyricPickBase = '';
function openMobileLyricPicker() {
    // 用原始完整文件名作为歌词查询基准
    mobileLyricPickBase = currentNameForLyrics || '';
    document.getElementById('lcModal').style.display = 'flex';
    document.getElementById('lcTitle').value = '';
    document.getElementById('lcArtist').value = '';
    document.getElementById('lcList').innerHTML = '<div class="empty-hint" style="padding:30px">正在预填歌名/歌手…</div>';
    fetch('/api/lyrics/meta?fileName=' + encodeURIComponent(mobileLyricPickBase))
        .then(function(r) { return r.json(); })
        .then(function(j) {
            if (j) {
                if (j.title) document.getElementById('lcTitle').value = j.title;
                if (j.artist) document.getElementById('lcArtist').value = j.artist;
            }
            document.getElementById('lcList').innerHTML = '<div class="empty-hint" style="padding:30px">输入歌名/歌手后点击"搜索歌词"</div>';
        })
        .catch(function() {
            document.getElementById('lcList').innerHTML = '<div class="empty-hint" style="padding:30px">输入歌名/歌手后点击"搜索歌词"</div>';
        });
}
function closeMobileLyricPicker() {
    document.getElementById('lcModal').style.display = 'none';
}
function mobileLyricSearch() {
    var fileName = mobileLyricPickBase;
    var t = encodeURIComponent(document.getElementById('lcTitle').value.trim());
    var a = encodeURIComponent(document.getElementById('lcArtist').value.trim());
    var list = document.getElementById('lcList');
    list.innerHTML = '<div class="empty-hint" style="padding:30px">正在搜索各接口歌词候选…</div>';
    fetch('/api/lyrics/candidates?fileName=' + encodeURIComponent(fileName) + '&title=' + t + '&artist=' + a)
        .then(function(res) { return res.json(); })
        .then(function(arr) { renderMobileLyricPicker(arr); })
        .catch(function() { list.innerHTML = '<div class="empty-hint" style="padding:30px">搜索失败，请重试</div>'; });
}
function renderMobileLyricPicker(arr) {
    var list = document.getElementById('lcList');
    list.innerHTML = '';
    if (!arr || !arr.length) {
        list.innerHTML = '<div class="empty-hint" style="padding:30px">没有可用的歌词候选</div>';
        return;
    }
    arr.forEach(function(c) {
        var item = document.createElement('div');
        item.className = 'lc-item';
        var badge = document.createElement('span');
        badge.className = 'badge';
        badge.textContent = c.source;
        var info = document.createElement('span');
        info.className = 'info';
        var dur = '';
        if (c.duration > 0) { var m=Math.floor(c.duration/60), s=Math.floor(c.duration%60); dur = ' [' + m + ':' + (s<10?'0':'') + s + ']'; }
        info.textContent = (c.title || '') + (c.artist ? ' - ' + c.artist : '') + dur;
        item.appendChild(badge);
        item.appendChild(info);
        item.onclick = function() { mobileApplyLyricPicker(c.lyrics, c.source); };
        list.appendChild(item);
    });
}
function mobileApplyLyricPicker(lrcText, source) {
    mobileLyrics = parseLRC(lrcText);
    if (!mobileLyrics.length) { showToast('该候选没有时间轴歌词', 'error'); return; }
    currentLyricIndex = 0;
    renderMobileLyrics();
    showToast('歌词来源：手动选择(' + source + ')');
    closeMobileLyricPicker();
    // 保存手动歌词覆盖本地 .lrc
    fetch('/api/lyrics/save?fileName=' + encodeURIComponent(mobileLyricPickBase), {
        method: 'POST',
        body: lrcText
    }).catch(function(){});
}

// 队列操作
function playNow(path, name, type) {
    queue = [];
    currentPlayingIndex = -1;
    queue.push({path:path, name:name, type:type, origName:name, status:"checking", transcodeProgress:0, requestKey:path});
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
	    queue.push({path:path, name:name, type:type, origName:name, displayName:displayName||'', status:"checking", transcodeProgress:0, requestKey:path});
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
                        showToast('💤 硬盘已休眠，正在等待唤醒... (' + (sdata.elapsed||0) + 'ms)', 'info');
                    }
                } catch(e) {}
            }
        };
        sxhr.send();
    }, 500);
    function stopDiskSleepPolling(){
        if (window._diskSleepTimer) { clearInterval(window._diskSleepTimer); window._diskSleepTimer = null; }
    }

    // 音频文件保持原有转码检查逻辑
    var xhr = new XMLHttpRequest();
    xhr.open('POST', '/api/transcode/check-and-add', true);
    xhr.setRequestHeader('Content-Type', 'application/json');
    xhr.onload = function() {
        stopDiskSleepPolling();
        if (xhr.status === 403) {
            try {
                var errMsg = JSON.parse(xhr.responseText);
                errMsg = errMsg && errMsg.message ? errMsg.message : '该资源正在审核中，暂不可点播。';
                item.status = "blocked";
                item.blockMsg = errMsg;
                renderQueue();
                if (typeof showToast === 'function') showToast(errMsg);
            } catch(e) {}
            return;
        }
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
    xhr.onerror = function(){ stopDiskSleepPolling(); };
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
        var url;
        // 浏览器不能直接解码的音频（dts/dff/etc）走实时转码为 AAC 流，否则播放不出来
        if (needsLiveAudioTranscode(item.name)) {
            url = '/api/music-mode-stream?name=' + encodeURIComponent(item.path) + '&trackIndex=' + lastTrackIndex + '&_t=' + Date.now();
        } else {
            url = '/file?name=' + encodeURIComponent(item.path);
        }
        currentNameForLyrics = item.origName || item.name;
        currentLyricsPath = item.path; // 歌词本地 .lrc 按服务器路径查找（与桌面端 currentFilePath 一致）
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
    // 首个用户手势时预热 AudioContext，确保电平表有数据/不静音
    var warm = function() { warmMobileAudio(); };
    document.addEventListener('touchstart', warm, {once:true, passive:true});
    document.addEventListener('click', warm, {once:true});
};

// ==================== 歌词 + 电平表（音频播放时） ====================
var currentNameForLyrics = '';
var currentLyricsPath = ''; // 当前音频的服务器路径（用于本地 .lrc 查找，等价桌面端 currentFilePath）
var mobileLyrics = [];
var currentLyricIndex = 0;

// LRC 解析（同桌面端）
function parseLRC(lrcText) {
  var lines = lrcText.split('\n');
  var result = [];
  var timeRegex = /\[(\d{2}):(\d{2}\.\d{2,3})\]/g;
  lines.forEach(function(line) {
    var times = [], match;
    while ((match = timeRegex.exec(line)) !== null) times.push(match);
    var text = line.replace(timeRegex, '').trim();
    times.forEach(function(m) { result.push({ time: parseInt(m[1])*60+parseFloat(m[2]), text: text }); });
  });
  result.sort(function(a, b) { return a.time - b.time; });
  return result;
}

function decodeBuffer(uint8) {
  var utf8Text = new TextDecoder('utf-8').decode(uint8);
  if (utf8Text.indexOf('\uFFFD') !== -1) return new TextDecoder('gbk').decode(uint8);
  var gbkTest = new TextDecoder('gbk').decode(uint8);
  return gbkTest.replace(/[^\x00-\x7F]/g, '').length > utf8Text.replace(/[^\x00-\x7F]/g, '').length * 2 ? gbkTest : utf8Text;
}

function setNoLyrics(msg) {
  mobileLyrics = [];
  currentLyricIndex = 0;
  var box = document.getElementById('mobileLyrics');
  if (box) box.innerHTML = '<div class="no-lyrics">' + (msg||'暂无歌词') + '</div>';
}

// 加载歌词：先本地 .lrc（按服务器路径），再 /api/lyrics（内嵌或在线搜索，传完整文件名）
// songName = 完整文件名（带扩展名）；basePath = 服务器路径（用于本地 .lrc 定位，等价桌面端 currentFilePath）
function loadMobileLyrics(songName, basePath) {
  setNoLyrics('加载歌词中...');
  basePath = basePath || currentLyricsPath;
  if (basePath) {
    // 1) 本地 .lrc：基于服务器路径构造（与桌面端一致）
    var lrcUrl = basePath.replace(/\.[^.]+$/, '.lrc');
    var xhr = new XMLHttpRequest();
    xhr.open('GET', '/file?name=' + encodeURIComponent(lrcUrl) + '&_t=' + Date.now(), true);
    xhr.responseType = 'arraybuffer';
    xhr.onload = function() {
      if (xhr.status === 200) {
        mobileLyrics = parseLRC(decodeBuffer(new Uint8Array(xhr.response)));
        if (mobileLyrics.length > 0) { renderMobileLyrics(); return; }
      }
      loadEmbeddedMobileLyrics(songName);
    };
    xhr.onerror = function() { loadEmbeddedMobileLyrics(songName); };
    xhr.send();
  } else {
    loadEmbeddedMobileLyrics(songName);
  }
}

function loadEmbeddedMobileLyrics(songName) {
  if (!songName) { setNoLyrics('暂无歌词'); return; }
  var xhr = new XMLHttpRequest();
  xhr.open('GET', '/api/lyrics?fileName=' + encodeURIComponent(songName) + '&_t=' + Date.now(), true);
  xhr.responseType = 'arraybuffer';
  xhr.onload = function() {
    if (xhr.status === 200) {
      mobileLyrics = parseLRC(decodeBuffer(new Uint8Array(xhr.response)));
      if (mobileLyrics.length > 0) { renderMobileLyrics(); return; }
    }
    setNoLyrics('暂无歌词');
  };
  xhr.onerror = function() { setNoLyrics('暂无歌词'); };
  xhr.send();
}

function renderMobileLyrics() {
  var box = document.getElementById('mobileLyrics');
  if (!box || mobileLyrics.length === 0) return;
  var html = '<div class="mobile-lyrics-inner" id="mobileLyricsInner">';
  html += '<div style="height:60px"></div>';
  mobileLyrics.forEach(function(l) {
    html += '<div class="mline" data-time="' + l.time + '">' + l.text + '</div>';
  });
  html += '<div style="height:60px"></div>';
  html += '</div>';
  box.innerHTML = html;
  currentLyricIndex = 0;
}

// 歌词同步滚动（复用 getEstimatedCurrentTime 的播放时间）
function updateMobileLyrics(time) {
  if (!mobileLyrics || mobileLyrics.length === 0) return;
  var idx = 0;
  for (var i = 0; i < mobileLyrics.length; i++) {
    if (mobileLyrics[i].time <= time) idx = i; else break;
  }
  if (idx === currentLyricIndex) return;
  currentLyricIndex = idx;
  var inner = document.getElementById('mobileLyricsInner');
  if (!inner) return;
  var lines = inner.querySelectorAll('.mline');
  for (var i = 0; i < lines.length; i++) {
    lines[i].className = 'mline' + (i === idx ? ' current' : '');
  }
  if (lines[idx]) {
    var box = document.getElementById('mobileLyrics');
    var lineTop = lines[idx].offsetTop;
    if (box) inner.style.transform = 'translateY(' + (box.clientHeight/2 - lineTop - lines[idx].offsetHeight/2) + 'px)';
  }
}

var mobileLyricTimer = null;

// 电平表：完全复刻 fftplayer 的 WebAudio 采集方式（该文件在手机上已验证可用）。
// 关键：AudioContext/Analyser/ScriptProcessor 及 createMediaElementSource 必须在用户手势栈内创建，
// 我们的点播由异步转码回调触发(脱离手势)，故把整条链的建立绑定到页面首次触摸事件上。
var mobileAudioCtx = null;
var mobileMediaSource = null;
var meterScript = null;         // ScriptProcessor 采集裸采样(fftplayer 方案，手机可靠)
var meterAnalyser = null;       // 时域直通输出节点：保证 createMediaElementSource 后有声；不读频域，零 FFT 开销
var meterAnim = 0;
var meterData = { leftRms: -90, rightRms: -90 };
var meterLedsL = null;
var meterLedsR = null;

// 保存当前待接入 WebAudio 的 audio 元素(playMedia 里设置)
var meterTargetAudio = null;

// 无信号自动降级为 blob 缓存(绕过夸克/自带浏览器对网络流的媒体接管)
var lastMediaName = '';
var lastMediaUrl = '';
var meterHadSignal = false;      // 曾经拿到过信号(正常浏览器，无需降级)
var meterPlayStartedAt = 0;      // 本次播放开始时间(用于判断"播了一阵仍无信号")
var meterFallingBack = false;    // 正在降级(防止重复触发)
var meterFallbackForced = false; // 本会话已确认劫持，后续直接走 blob(不重播等待)

// 调音台风格电平表状态：LED 显示值(0~1) + 峰值保持值(0~1)
var meterDispL = 0, meterDispR = 0;
var meterPeakL = 0, meterPeakR = 0;
var meterPeakAtL = 0, meterPeakAtR = 0;   // 峰值标记的开始时间(ms)
var meterFallAtL = 0, meterFallAtR = 0;   // 主灯串上次回落时间(ms)，每50ms回落1格
var METER_LED_COUNT = 30;                 // 单行 LED 格数

// 持久化键：确认浏览器劫持(网络流无信号)后记录，下次直接走 blob 提升体验
var FALLBACK_KEY = 'ktv_mobile_meter_fallback';
try {
  if (localStorage.getItem(FALLBACK_KEY) === '1') meterFallbackForced = true;
} catch (e) {}

// 在首次用户手势(点击/触摸)内建立 AudioContext 图。与 fftplayer.initAudioContext 一致。
function initMobileAudioGraph() {
  if (mobileAudioCtx) {
    if (mobileAudioCtx.state === 'suspended') mobileAudioCtx.resume().catch(function(){});
    return;
  }
  var AC = window.AudioContext || window.webkitAudioContext;
  if (!AC) return;
  try {
    mobileAudioCtx = new AC();
    // 时域直通节点：把源信号送到扬声器，保证 createMediaElementSource 重路由后仍有声。
    // Analyser 仅在读取频域数据时才做 FFT；这里从不读频域，零开销，纯 time-domain 直通。
    meterAnalyser = mobileAudioCtx.createAnalyser();
    meterAnalyser.fftSize = 2048;
    meterAnalyser.smoothingTimeConstant = 0;
    meterAnalyser.connect(mobileAudioCtx.destination);
    meterScript = mobileAudioCtx.createScriptProcessor(256, 2, 2);
    meterScript.onaudioprocess = function(e) {
      // 仅采集峰值，不回写输出(回写会与 meterAnalyser 直通形成双出声叠加，产生咔哒声)。
      // 出声完全由上方 meterAnalyser(时域直通到 destination)负责。
      var l = e.inputBuffer.getChannelData(0);
      var r = e.inputBuffer.getChannelData(1);
      var lMax = -1, lMin = 1, rMax = -1, rMin = 1;
      for (var i = 0; i < l.length; i++) {
        var a = l[i]; if (a > lMax) lMax = a; if (a < lMin) lMin = a;
        var b = r[i]; if (b > rMax) rMax = b; if (b < rMin) rMin = b;
      }
      // 峰值电平表：本帧最大值-最小值的一半即峰值幅度，不再做 RMS 累加，反应更即时应手
      // clamp 到 ≤1（0dBFS），防止解码器/VB 增益后样本超界导致峰值刻度越界
      var lpeak=(lMax-lMin)/2, rpeak=(rMax-rMin)/2;
      if (lpeak > 1) lpeak = 1; if (rpeak > 1) rpeak = 1;
      meterData.leftRms = peakDb(lpeak);
      meterData.rightRms = peakDb(rpeak);
    };
    meterScript.connect(mobileAudioCtx.destination);

    if (mobileAudioCtx.state === 'suspended') mobileAudioCtx.resume().catch(function(){});
  } catch (e) {}
}

function peakDb(peak) {
  return peak > 0 ? 20 * Math.log10(peak) : -90;
}

// 把给定 audio 元素接入 WebAudio 图(同 fftplayer.setupAudioSource)。
function attachMobileSource(audio) {
  if (!mobileAudioCtx || !audio) return;
  if (audio.__srcConnected) return;
  try {
    var src = mobileAudioCtx.createMediaElementSource(audio);
    audio.__srcConnected = true;
    audio.__meshSrc = src;   // 保存引用，供 freshMobileAudioEl 重建时 disconnect 防止累积泄漏
    src.connect(meterAnalyser);   // 主出声路径(时域直通)
    src.connect(meterScript);     // 采集路径(峰值电平)
  } catch (e) { try { audio.__srcConnected = false; } catch (_) {} }
}

// 建立一次性的首次触摸监听，保证 AudioContext 图在手势栈内创建。
// 注意：这里【不】立即 createMediaElementSource（远程流此时媒体未就绪，提前绑定移动端拿不到数据），
// 真正的 source 绑定延迟到媒体 canplay/loadedmetadata 之后再执行。
function bindFirstGesture() {
  function arm() {
    initMobileAudioGraph();
    document.removeEventListener('touchstart', arm);
    document.removeEventListener('mousedown', arm);
    document.removeEventListener('pointerdown', arm);
  }
  document.addEventListener('touchstart', arm);
  document.addEventListener('mousedown', arm);
  document.addEventListener('pointerdown', arm);
}

// 本地文件路径：blob 同步可播，media readyState 已就绪→立即绑定有效(参考文件即可做)。
// 在网络流路径：play() 返回时媒体尚未就绪，必须等 canplay 后才 createMediaElementSource，
// 否则记 iOS/安卓 WebKit 拿不到任何 analyser 数据(有声音但电平表全暗)。
// 幂等：同一元素只绑一次。
function bindMobileSourceWhenReady(el) {
  if (!el) return;
  if (el.__srcConnected) return;
  if (el.readyState >= 3) { // HAVE_FUTURE_DATA：数据充足，直接绑
    attachMobileSource(el);
    return;
  }
  function attach() {
    el.removeEventListener('canplay', attach);
    el.removeEventListener('loadeddata', attach);
    if (!el.__srcConnected) attachMobileSource(el);
  }
  el.addEventListener('canplay', attach);
  el.addEventListener('loadeddata', attach);
}

function startMobileAnalyzer(audio) {
  var target = audio || audioEl;
  // 重置降级检测状态(每次点播重新计时)
  meterHadSignal = false;
  meterPlayStartedAt = 0;
  meterFallingBack = false;
  // 就绪后再绑定 source(远程流关键：等 canplay)
  bindMobileSourceWhenReady(target);
  if (meterLedsL === null) {
    meterLedsL = document.getElementById('meterLedsL');
    meterLedsR = document.getElementById('meterLedsR');
    var html = '';
    for (var i = 0; i < 30; i++) html += '<div class="meter-led"></div>';
    if (meterLedsL) meterLedsL.innerHTML = html;
    if (meterLedsR) meterLedsR.innerHTML = html;
  }
  if (mobileAudioCtx && mobileAudioCtx.state === 'suspended') mobileAudioCtx.resume().catch(function(){});
  meterTargetAudio = target;
  // 记录开始时间(等 canplay 后再计，避免源未就绪期误判)
  if (target) {
    target.addEventListener('playing', function __mstart() {
      target.removeEventListener('playing', __mstart);
      meterPlayStartedAt = Date.now();
    });
  }
  if (!meterAnim) requestAnimationFrame(mobileMeterLoop);
}

function stopMobileAnalyzer() {
  if (meterAnim) { cancelAnimationFrame(meterAnim); meterAnim = 0; }
}

function mobileMeterLoop() {
  meterAnim = requestAnimationFrame(mobileMeterLoop);
  if (document.getElementById('audioWrap').style.display !== 'block') return;
  updateMobileLyrics(getEstimatedCurrentTime());
  if (!meterScript || meterLedsL === null) return;
  var now = Date.now();
  // 调音台风格：信号≥显示立即跟随；信号<显示每50ms回落1格；峰值标记保持0.5秒
  meterDispL = meterSmooth(meterDispL, meterData.leftRms, now, true);
  meterDispR = meterSmooth(meterDispR, meterData.rightRms, now, false);
  // 峰值标记：显示值创新高则更新，并保持 0.5 秒
  updateMeterPeak('L'); updateMeterPeak('R');
  renderMeterLeds(meterLedsL, meterDispL, meterPeakL);
  renderMeterLeds(meterLedsR, meterDispR, meterPeakR);
  // 无信号自动降级检测：网络流(非blob)在夸克/自带浏览器会被媒体接管抓不到数据，
  // 播放一阵仍无信号则自动 fetch 为 blob 重播(等效本地文件，三浏览器均出电平表)。
  if (!meterFallingBack && meterPlayStartedAt && !meterHadSignal) {
    var cur = audioEl && (audioEl.currentSrc || audioEl.src);
    if (cur && cur.indexOf('blob:') !== 0 && lastMediaUrl) {
      var silent = (meterData.leftRms < -89) && (meterData.rightRms < -89);
      if (silent && (Date.now() - meterPlayStartedAt > 2500)) {
        // 确认劫持：记录持久化，本浏览器之后直接走 blob(免等待)
        meterFallbackForced = true;
        try { localStorage.setItem(FALLBACK_KEY, '1'); } catch (e) {}
        meterFallingBack = true;
        showToast('浏览器不支持流式电平，切换缓存模式...');
        fetch(lastMediaUrl)
          .then(function(r){ return r.blob(); })
          .then(function(b){
            playMedia(URL.createObjectURL(b), lastMediaName);
            meterFallingBack = false;
          })
          .catch(function(){ meterFallingBack = false; });
      }
    }
  }
}

// dB -> 0~1 显示值（范围 -30dB ~ 0dB）
function dbToNorm(db) {
  var norm = (db + 30) / 30;
  if (norm < 0) norm = 0; if (norm > 1) norm = 1;
  return norm;
}

// 显示值平滑：信号电平 ≥ 显示值 → 立即更新(快速上升)；信号 < 显示值 → 每50ms回落1个灯
function meterSmooth(disp, db, now, isLeft) {
  var target = dbToNorm(db);
  if (target >= disp) {
    // 快速上升：立即跟随，并重置回落计时
    if (isLeft) meterFallAtL = now; else meterFallAtR = now;
    return target;
  }
  // 缓慢回落：每 50ms 掉 1 格
  var lastFall = isLeft ? meterFallAtL : meterFallAtR;
  if (now - lastFall >= 50) {
    if (isLeft) meterFallAtL = now; else meterFallAtR = now;
    var step = 1 / METER_LED_COUNT;
    var next = disp - step;
    return next < 0 ? 0 : next;
  }
  return disp;
}

// 峰值标记：显示值创新高则记录峰值与时间，保持 0.5 秒后随回落
function updateMeterPeak(ch) {
  var disp = ch === 'L' ? meterDispL : meterDispR;
  var now = Date.now();
  var isLeft = ch === 'L';
  var peak = isLeft ? meterPeakL : meterPeakR;
  var at = isLeft ? meterPeakAtL : meterPeakAtR;
  if (disp > peak || (now - at > 500)) {
    if (isLeft) { meterPeakL = disp; meterPeakAtL = now; }
    else { meterPeakR = disp; meterPeakAtR = now; }
  }
}

// 电平渲染：亮灯数=disp，峰值格=peak(带白色描边标记，模拟调音台峰值保持)
function renderMeterLeds(leds, disp, peak) {
  if (!leds) return;
  var total = leds.children.length;
  var onCount = Math.round(disp * total);
  // 峰值格 clamp 到条内最后一格：满刻度(0dB)时显示在最右格，绝不越界/消失
  var peakIdx = Math.round(peak * total);
  if (peakIdx >= total) peakIdx = total - 1;
  for (var i = 0; i < total; i++) {
    var led = leds.children[i];
    var cls = 'meter-led';
    var p = i / total;
    if (i < onCount) {
      cls = p > 0.85 ? 'meter-led red' : (p > 0.70 ? 'meter-led yellow' : 'meter-led on');
    }
    // 峰值标记：保持 0.5 秒，显示在信号最前端
    if (i === peakIdx) {
      cls += ' peak';
    }
    if (led.className !== cls) led.className = cls;
  }
}

// 页面加载后立即挂首次触摸预热
bindFirstGesture();

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
