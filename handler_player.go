package main

import "net/http"

func PlayerHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(`
<!DOCTYPE html>
<html>
<head>
<meta charset="UTF-8">
<title>KTV 播放屏</title>
<style>
*{margin:0;padding:0;background:linear-gradient(135deg,#1e2d3a 0%,#1a252f 40%,#151f28 100%)}
body{width:100vw;height:100vh;overflow:hidden}
.video-container{position:relative;width:100vw;height:100vh;z-index:1}
video{width:100vw;height:100vh;object-fit:contain;z-index:1}
.lyrics-container{position:absolute;top:0;left:0;width:calc(100vw - 20px);height:85vh;display:flex;align-items:flex-start;justify-content:flex-start;z-index:10;overflow:hidden;pointer-events:none;margin-right:10px}
.lyrics{width:100%;height:100%;text-align:center;color:#fff;font-size:26px;line-height:1.9;text-shadow:0 1px 2px rgba(0,0,0,0.5);overflow-y:auto;display:flex;flex-direction:column;align-items:center;pointer-events:auto;box-sizing:border-box;padding:20px 30px 20px 20px}
.lyrics-line{margin:10px 0;opacity:0.55;transition:all 0.4s cubic-bezier(.4,0,.2,1)}
.lyrics-line.active{opacity:1;font-size:32px;font-weight:bold;color:#5bc0de;text-shadow:0 1px 2px rgba(0,0,0,0.5)}
.ctrl-bar{position:fixed;bottom:40px;left:40px;z-index:999;display:flex;gap:18px;opacity:0;transition:opacity 0.3s ease;background:transparent !important}
.track-btn{padding:12px 32px;border:none;border-radius:6px;font-size:18px;font-weight:bold;cursor:pointer;color:#fff;letter-spacing:1px;transition:all 0.2s ease;transform:translateY(0);text-shadow:0 1px 2px rgba(0,0,0,0.3);line-height:1.2}
.track-btn .btn-sub{display:block;font-size:13px;font-weight:normal;opacity:0.9;letter-spacing:0;margin-top:2px;text-shadow:0 1px 1px rgba(0,0,0,0.4)}
.track-btn.active{background:#5cb85c;box-shadow:0 2px 4px rgba(0,0,0,0.3)}
.track-btn.active:hover{background:#4cae4c;box-shadow:0 3px 6px rgba(0,0,0,0.35)}
.track-btn.active:active{background:#449d44;box-shadow:0 1px 2px rgba(0,0,0,0.3)}
.track-btn:not(.active){background:#428bca;box-shadow:0 2px 4px rgba(0,0,0,0.3)}
.track-btn:not(.active):hover{background:#3276b1;box-shadow:0 3px 6px rgba(0,0,0,0.35)}
.track-btn:not(.active):active{background:#2d6ca2;box-shadow:0 1px 2px rgba(0,0,0,0.3)}
.transcode-btn{padding:12px 32px;border:none;border-radius:6px;font-size:18px;font-weight:bold;cursor:pointer;color:#fff;letter-spacing:1px;background:#f0ad4e;box-shadow:0 2px 4px rgba(0,0,0,0.3);transition:all 0.2s ease;transform:translateY(0);text-shadow:0 1px 2px rgba(0,0,0,0.3)}
.transcode-btn:hover{background:#ed9c28;box-shadow:0 3px 6px rgba(0,0,0,0.35)}
.transcode-btn:active{background:#d58512;box-shadow:0 1px 2px rgba(0,0,0,0.3)}
.transcode-btn:disabled{background:#555;box-shadow:0 1px 2px rgba(0,0,0,0.2);cursor:not-allowed;transform:translateY(0);color:#999;text-shadow:none}
.tips{position:fixed;top:20px;left:50%;transform:translateX(-50%);background:rgba(26,37,47,0.92);color:#fff;padding:12px 36px;border-radius:6px;display:none;font-size:18px;font-weight:bold;backdrop-filter:blur(12px);-webkit-backdrop-filter:blur(12px);box-shadow:0 2px 8px rgba(0,0,0,0.4);border:1px solid rgba(255,255,255,0.06)}
.transcode-progress{position:fixed;top:50%;left:50%;transform:translate(-50%,-50%);background:rgba(26,37,47,0.92);color:#fff;padding:30px 50px;border-radius:8px;display:none;flex-direction:column;align-items:center;z-index:1000;max-width:600px;max-height:80vh;overflow:hidden;backdrop-filter:blur(16px);-webkit-backdrop-filter:blur(16px);border:1px solid rgba(255,255,255,0.08);box-shadow:0 4px 16px rgba(0,0,0,0.5)}
.progress-bar{width:300px;height:20px;background:rgba(0,0,0,0.3);border-radius:10px;overflow:hidden;margin:10px 0;box-shadow:inset 0 1px 3px rgba(0,0,0,0.4)}
.progress-fill{height:100%;background:#428bca;width:0%;border-radius:10px}
.progress-text{font-size:16px;color:#fff;margin-bottom:10px;text-shadow:0 1px 2px rgba(0,0,0,0.3)}
.progress-log{width:100%;height:150px;background:rgba(0,0,0,0.3);border-radius:4px;padding:10px;font-family:Consolas,Monaco,monospace;font-size:12px;color:#aaa;overflow-y:auto;white-space:pre-wrap;border:1px solid rgba(255,255,255,0.05);box-shadow:inset 0 1px 3px rgba(0,0,0,0.3)}
.progress-command{width:100%;background:rgba(0,0,0,0.3);border-radius:4px;padding:10px;font-family:Consolas,Monaco,monospace;font-size:11px;color:#f0ad4e;overflow-x:auto;margin-bottom:10px;border:1px solid rgba(255,255,255,0.05);box-shadow:inset 0 1px 3px rgba(0,0,0,0.3)}
.progress-mediainfo{width:100%;background:rgba(0,0,0,0.3);border-radius:4px;padding:10px;font-family:Consolas,Monaco,monospace;font-size:12px;color:#5bc0de;white-space:pre-wrap;margin-bottom:10px;border:1px solid rgba(255,255,255,0.05);box-shadow:inset 0 1px 3px rgba(0,0,0,0.3)}
.play-info{position:fixed;top:20px;right:20px;background:rgba(26,37,47,0.9);color:#fff;padding:12px 24px;border-radius:6px;font-size:16px;font-weight:bold;z-index:100;backdrop-filter:blur(12px);-webkit-backdrop-filter:blur(12px);box-shadow:0 2px 8px rgba(0,0,0,0.4);border:1px solid rgba(255,255,255,0.06)}
.play-info .current{color:#5bc0de;margin-bottom:5px;text-shadow:0 1px 2px rgba(0,0,0,0.3)}
.play-info .next{color:#f0ad4e;text-shadow:0 1px 2px rgba(0,0,0,0.3)}
#trackWarningBar{background:#d9534f !important;box-shadow:0 2px 6px rgba(0,0,0,0.3) !important;border-radius:0 !important;position:fixed;top:0;left:0;right:0;z-index:2000;text-shadow:none}
</style>
</head>
<body>
<div class="video-container">
  <div id="videoBox"></div>
  <div class="lyrics-container" id="lyricsContainer">
    <div class="lyrics" id="lyrics"></div>
  </div>
</div>
<div class="tips" id="tipBox"></div>
<div id="trackWarningBar" style="display:none;background:#d9534f;color:#fff;padding:10px 16px;font-size:15px;font-weight:bold;text-align:center;animation:warnPulse 2s infinite">
  <span id="trackWarningText"></span>
</div>
<style>@keyframes warnPulse{0%,100%{opacity:1}50%{opacity:0.85}}</style>
<div class="play-info" id="playInfo" style="display:none">
  <div class="current" id="currentSong"></div>
  <div class="next" id="nextSong"></div>
  <div class="qr-block" id="qrBlock" style="display:none;margin-top:8px;text-align:center">
    <div style="font-size:13px;color:#5bc0de;margin-bottom:4px;text-shadow:0 1px 2px rgba(0,0,0,0.5)">手机扫码点歌</div>
    <img id="qrImage" src="" alt="二维码" width="140" height="140" style="display:block;margin:0 auto;border-radius:4px;background:#fff;padding:3px;box-shadow:0 2px 8px rgba(0,0,0,0.4)">
    <div id="qrNetworkTip" style="font-size:11px;color:#f0ad4e;margin-top:4px;line-height:1.4;text-shadow:0 1px 2px rgba(0,0,0,0.5)"></div>
  </div>
</div>
<div class="ctrl-bar">
  <button class="track-btn active" id="btnT0" onclick="switchTrack(0)">原唱</button>
  <button class="track-btn" id="btnT1" onclick="switchTrack(1)">伴奏</button>
  <button class="track-btn" id="btnT2" onclick="switchTrack(2)" style="display:none">右声道</button>
  <button class="transcode-btn" id="btnTranscode" onclick="startTranscode()" style="display:none">一键申请后台转码</button>
</div>
<div class="transcode-progress" id="transcodeProgress">
  <div class="progress-text">转码中...</div>
  <div class="progress-bar">
    <div class="progress-fill" id="progressFill"></div>
  </div>
  <div class="progress-text" id="progressPercent">0%</div>
  <div class="progress-mediainfo" id="progressMediaInfo"></div>
  <div class="progress-command" id="progressCommand"></div>
  <div class="progress-log" id="progressLog"></div>
</div>

<script>
var video = null;
var trackList = [];
var retryTimer = null;
var lyrics = [];
var currentLyricIndex = 0;
var currentFileName = '';
var currentFilePath = '';
var currentQueue = [];
var currentPlayingIndex = -1;
var audioCtx = null;
var mediaSource = null;
var splitter = null;
var merger = null;
var channelMode = 'stereo';
var lastTrackIndex = 0;
var lastVolume = 1;
var mediaAudioTrackCount = -1; // 从CheckTracks获取的音轨数，-1=未知
var mediaAudioChannels = -1;   // 从CheckTracks获取的声道数，-1=未知
var trackMode = 'track';       // 'track' = 多音轨模式（原唱/伴奏），'channel' = 单音轨立体声模式（立体声/左声道/右声道）
var isStreamMode = false; // 是否为流媒体模式（省流模式）
var currentSessionId = ''; // 从主页面同步的会话ID，用于显示点歌二维码

function showTip(text) {
  var tip = document.getElementById("tipBox");
  tip.innerText = text;
  tip.style.display = "block";
  clearTimeout(tip.timer);
  tip.timer = setTimeout(function() { tip.style.display = "none"; }, 10000);
}

function parseLRC(lrcText) {
  var lines = lrcText.split('\n');
  var result = [];
  var timeRegex = /\[(\d{2}):(\d{2}\.\d{2,3})\]/g;

  lines.forEach(function(line) {
    var times = [];
    var match;
    while ((match = timeRegex.exec(line)) !== null) {
      times.push(match);
    }
    var text = line.replace(timeRegex, '').trim();

    times.forEach(function(m) {
      var minutes = parseInt(m[1]);
      var seconds = parseFloat(m[2]);
      var time = minutes * 60 + seconds;
      result.push({ time: time, text: text });
    });
  });

  result.sort(function(a, b) { return a.time - b.time; });
  return result;
}

function decodeBuffer(uint8) {
  var utf8Text = new TextDecoder('utf-8').decode(uint8);
  var hasReplacement = utf8Text.indexOf('\uFFFD') !== -1;

  if (hasReplacement) {
    return new TextDecoder('gbk').decode(uint8);
  } else {
    var gbkTest = new TextDecoder('gbk').decode(uint8);
    var utf8Len = utf8Text.replace(/[^\x00-\x7F]/g, '').length;
    var gbkLen = gbkTest.replace(/[^\x00-\x7F]/g, '').length;
    if (gbkLen > utf8Len * 2) {
      return gbkTest;
    } else {
      return utf8Text;
    }
  }
}

function loadLyrics(songName) {
  var videoExtensions = ['.mp4', '.avi', '.mov', '.wmv', '.flv', '.mkv', '.mpg', '.mpeg', '.rm', '.rmvb', '.ts', '.webm'];
  var fileExtension = songName.toLowerCase().substring(songName.lastIndexOf('.'));

  if (videoExtensions.indexOf(fileExtension) !== -1) {
    document.getElementById("lyricsContainer").style.display = "none";
    document.querySelector(".ctrl-bar").style.display = "flex";
    return;
  }

  document.getElementById("lyricsContainer").style.display = "flex";
  document.querySelector(".ctrl-bar").style.display = "none";

  lyrics = [];
  currentLyricIndex = 0;
  document.getElementById("lyrics").innerHTML = '';

  var lrcUrl = (currentFilePath || songName).replace(/\.[^.]+$/, '.lrc');
  var xhr = new XMLHttpRequest();
  xhr.open('GET', '/file?name=' + encodeURIComponent(lrcUrl), true);
  xhr.responseType = 'arraybuffer';
  xhr.onload = function() {
    if (xhr.status === 200) {
      var uint8 = new Uint8Array(xhr.response);
      var text = decodeBuffer(uint8);
      lyrics = parseLRC(text);
      renderLyrics();
    } else {
      loadEmbeddedLyrics(songName);
    }
  };
  xhr.onerror = function() {
    loadEmbeddedLyrics(songName);
  };
  xhr.send();
}

function loadEmbeddedLyrics(songName) {
  var xhr = new XMLHttpRequest();
  xhr.open('GET', '/api/lyrics?fileName=' + encodeURIComponent(songName), true);
  xhr.responseType = 'arraybuffer';
  xhr.onload = function() {
    if (xhr.status === 200) {
      var uint8 = new Uint8Array(xhr.response);
      var text = decodeBuffer(uint8);
      lyrics = parseLRC(text);
      if (lyrics.length > 0) {
        renderLyrics();
      } else {
        document.getElementById("lyrics").innerHTML = '<div class="lyrics-line">暂无歌词</div>';
      }
    } else {
      document.getElementById("lyrics").innerHTML = '<div class="lyrics-line">暂无歌词</div>';
    }
  };
  xhr.onerror = function() {
    document.getElementById("lyrics").innerHTML = '<div class="lyrics-line">暂无歌词</div>';
  };
  xhr.send();
}

function renderLyrics() {
  var lyricsElement = document.getElementById("lyrics");
  lyricsElement.innerHTML = '';

  var topSpacer = document.createElement('div');
  topSpacer.style.height = '50vh';
  lyricsElement.appendChild(topSpacer);

  lyrics.forEach(function(lyric) {
    var line = document.createElement('div');
    line.className = 'lyrics-line';
    line.textContent = lyric.text;
    lyricsElement.appendChild(line);
  });

  var bottomSpacer = document.createElement('div');
  bottomSpacer.style.height = '50vh';
  lyricsElement.appendChild(bottomSpacer);

  setTimeout(function() {
    lyricsElement.scrollTop = 0;
  }, 100);
}

function updateLyrics() {
  if (!video || lyrics.length === 0) return;

  var currentTime = video.currentTime;
  var newIndex = currentLyricIndex;

  for (var i = 0; i < lyrics.length; i++) {
    if (lyrics[i].time <= currentTime) {
      newIndex = i;
    } else {
      break;
    }
  }

  if (newIndex !== currentLyricIndex) {
    var prevLine = document.querySelectorAll('.lyrics-line')[currentLyricIndex];
    if (prevLine) {
      prevLine.classList.remove('active');
    }

    var currentLine = document.querySelectorAll('.lyrics-line')[newIndex];
    if (currentLine) {
      currentLine.classList.add('active');

      var lyricsElement = document.getElementById('lyrics');
      var lineTop = currentLine.offsetTop;
      var scrollTop = lineTop - (lyricsElement.clientHeight / 2);

      lyricsElement.scrollTo({
        top: scrollTop,
        behavior: 'smooth'
      });
    }

    currentLyricIndex = newIndex;
  }
}

function setupChannelRouting(mode) {
  if (!audioCtx || !splitter || !merger) return;

  try {
    splitter.disconnect();
  } catch(e) {}

  if (mode === 'left') {
    splitter.connect(merger, 0, 0);
    splitter.connect(merger, 0, 1);
  } else if (mode === 'right') {
    splitter.connect(merger, 1, 0);
    splitter.connect(merger, 1, 1);
  } else {
    splitter.connect(merger, 0, 0);
    splitter.connect(merger, 1, 1);
  }
}

function initWebAudio() {
  if (!video) return;

  try {
    if (audioCtx) {
      audioCtx.close();
    }
  } catch(e) {}

  audioCtx = new (window.AudioContext || window.webkitAudioContext)();
  mediaSource = audioCtx.createMediaElementSource(video);
  splitter = audioCtx.createChannelSplitter(2);
  merger = audioCtx.createChannelMerger(2);

  mediaSource.connect(splitter);
  setupChannelRouting('stereo');
  merger.connect(audioCtx.destination);
  channelMode = 'stereo';
}

function doSwitchTrack(targetIdx) {
  if (!video) return false;

  trackList = Array.from(video.audioTracks || []);
  if (trackList.length >= 2) {
    trackList.forEach(function(track, idx) {
      track.enabled = (idx === targetIdx);
    });

    updateButtonStates(targetIdx);
    return true;
  }

  return false;
}

// 根据当前 trackMode 更新按钮文字、可见性、active 状态
function updateTrackButtons() {
  var btn0 = document.getElementById('btnT0');
  var btn1 = document.getElementById('btnT1');
  var btn2 = document.getElementById('btnT2');
  if (!btn0 || !btn1 || !btn2) return;

  if (mediaAudioTrackCount >= 2) {
    // 多音轨模式：原唱/伴奏（音轨X）
    trackMode = 'track';
    btn0.innerHTML = '原唱<span class="btn-sub">（音轨1）</span>';
    btn1.innerHTML = '伴奏<span class="btn-sub">（音轨2）</span>';
    btn2.style.display = 'none';
    if (lastTrackIndex > 1) lastTrackIndex = 0;
  } else if (mediaAudioTrackCount === 1 && mediaAudioChannels === 2) {
    // 单音轨立体声模式：立体声/左声道/右声道
    trackMode = 'channel';
    btn0.innerHTML = '立体声';
    btn1.innerHTML = '左声道';
    btn2.innerHTML = '右声道';
    btn2.style.display = 'inline-block';
    if (lastTrackIndex > 2) lastTrackIndex = 0;
  } else {
    // 未知或单声道：默认多音轨模式（仅显示2个按钮，等待canplay再决定是否走声道fallback）
    trackMode = 'track';
    btn0.innerHTML = '原唱';
    btn1.innerHTML = '伴奏';
    btn2.style.display = 'none';
    if (lastTrackIndex > 1) lastTrackIndex = 0;
  }

  updateButtonStates(lastTrackIndex);
}

function updateButtonStates(activeIdx) {
  var btn0 = document.getElementById('btnT0');
  var btn1 = document.getElementById('btnT1');
  var btn2 = document.getElementById('btnT2');
  if (btn0) btn0.classList.toggle('active', activeIdx === 0);
  if (btn1) btn1.classList.toggle('active', activeIdx === 1);
  if (btn2) btn2.classList.toggle('active', activeIdx === 2);
}

// 应用声道路由（channel模式），idx: 0=stereo 1=left 2=right
function applyChannelMode(idx) {
  if (!audioCtx) initWebAudio();
  if (!audioCtx || !splitter || !merger) return false;
  var mode = idx === 0 ? 'stereo' : (idx === 1 ? 'left' : 'right');
  setupChannelRouting(mode);
  channelMode = mode;
  return true;
}

function switchTrack(targetIdx) {
  lastTrackIndex = targetIdx;

  // 声道模式（单音轨立体声：立体声/左声道/右声道）
  if (trackMode === 'channel') {
    if (applyChannelMode(targetIdx)) {
      updateButtonStates(targetIdx);
      var tipText = targetIdx === 0 ? '立体声模式' : (targetIdx === 1 ? '左声道模式' : '右声道模式');
      showTip(tipText);
      clearTimeout(retryTimer);
      return;
    }
    // WebAudio 未就绪，重试
    showTip("⏳ 音频系统初始化中...");
    clearTimeout(retryTimer);
    var chanRetryCount = 0;
    retryTimer = setInterval(function() {
      chanRetryCount++;
      if (applyChannelMode(targetIdx)) {
        updateButtonStates(targetIdx);
        clearTimeout(retryTimer);
        return;
      }
      if (chanRetryCount >= 25) {
        clearTimeout(retryTimer);
        alert('音频系统初始化超时，无法切换声道。');
      }
    }, 200);
    return;
  }

  // 多音轨模式（原唱/伴奏）
  // 流媒体模式：通知主页面重新播放（带新的trackIndex）
  if (isStreamMode) {
    updateButtonStates(targetIdx);
    showTip(targetIdx === 0 ? "原唱模式" : "伴奏模式");
    if (window.opener && !window.opener.closed) {
      window.opener.postMessage({action: "switchTrack", index: targetIdx}, "*");
    }
    return;
  }

  if (doSwitchTrack(targetIdx)) {
    clearTimeout(retryTimer);
    return;
  }

  // 音轨切换失败，分析原因并提示
  // 1. 先判断浏览器是否支持 audioTracks API（实验功能是否开启）
  if (!video.audioTracks) {
    clearTimeout(retryTimer);
    var reason = '无法切换原唱/伴奏：当前浏览器不支持音轨切换功能。\n\n';
    if (mediaAudioTrackCount === 0) {
      reason += '注意：该歌曲本身也没有音频轨道（源文件问题）。\n\n';
    } else if (mediaAudioTrackCount === 1) {
      reason += '注意：该歌曲仅有1条音频轨道，即使开启实验功能也无法切换（源文件问题）。\n\n';
    }
    reason += '可能原因：\n';
    reason += '1. 未开启 Experimental Web Platform features 实验功能\n';
    reason += '2. 浏览器版本过旧不支持 AudioTracks API\n\n';
    reason += '解决方法：\n请在浏览器地址栏打开 chrome://flags/#enable-experimental-web-platform-features 并设为 Enabled，然后重启浏览器。';
    alert(reason);
    return;
  }

  // 2. 浏览器支持API，检查文件本身音轨数
  if (mediaAudioTrackCount === 0) {
    alert('该歌曲没有音频轨道，无法切换原唱/伴奏。\n\n这是源文件本身的问题，不是系统故障。');
    clearTimeout(retryTimer);
    return;
  }
  if (mediaAudioTrackCount === 1) {
    alert('该歌曲仅有1条音频轨道，无法切换原唱/伴奏。\n\n原唱/伴奏切换需要歌曲包含2条及以上音频轨道。\n这是源文件本身的问题，不是系统故障。');
    clearTimeout(retryTimer);
    return;
  }

  // 3. 音轨数未知或>=2但尚未加载，重试
  showTip("⏳ 音轨未加载，自动重试中...");

  clearTimeout(retryTimer);
  var retryCount = 0;
  var maxRetries = 25; // 5秒
  retryTimer = setInterval(function() {
    retryCount++;
    // check-tracks返回后可能切到了channel模式，转交channel模式处理
    if (trackMode === 'channel') {
      clearTimeout(retryTimer);
      switchTrack(targetIdx > 2 ? 0 : targetIdx);
      return;
    }
    if (doSwitchTrack(targetIdx)) {
      clearTimeout(retryTimer);
      return;
    }

    if (retryCount >= maxRetries) {
      clearTimeout(retryTimer);
      var reason = '音轨信息加载超时。\n\n';
      reason += '浏览器已支持音轨切换功能，但音轨信息迟迟未加载。\n';
      reason += '可能原因：视频文件的音频编码不被浏览器支持。\n\n';
      reason += '建议：点击下方"一键申请后台转码"按钮，将文件转为浏览器兼容的编码格式。';
      alert(reason);
    }
  }, 200);
}

function isAudioFile(fileName) {
  var audioExtensions = ['.mp3', '.wav', '.flac', '.aac', '.m4a', '.ogg', '.wma', '.ape'];
  var ext = fileName.toLowerCase().substring(fileName.lastIndexOf('.'));
  return audioExtensions.indexOf(ext) !== -1;
}

var metadataCheckTimer = null;
var switchingSong = false;
var nextSongCooldown = false;
var keydownRegistered = false;

function onVideoMetadataLoaded() {
  if (metadataCheckTimer) clearTimeout(metadataCheckTimer);
  metadataCheckTimer = setTimeout(function() {
    if (video.audioTracks && video.audioTracks.length === 0) {
      if (mediaAudioTrackCount === 0) {
        alert('警告：该歌曲没有音频轨道，无法正常播放声音。\n\n这是源文件本身的问题，不是系统故障。');
      } else {
        alert('警告：该歌曲有' + mediaAudioTrackCount + '条音频轨道，但音频编码不被浏览器支持，可能无法正常播放声音。\n\n建议：点击下方"一键申请后台转码"按钮转换编码。');
        document.getElementById('btnTranscode').style.display = 'block';
      }
    }

    if (video.videoWidth === 0 && video.videoHeight === 0) {
      alert('警告：当前媒体文件的视频轨道无法解码，可能是视频编码不被支持');
      document.getElementById('btnTranscode').style.display = 'block';
    }
  }, 1000);
}

function playVideo(url, name, type, path) {
  clearTimeout(retryTimer);
  switchingSong = true;
  trackList = [];
  lyrics = [];
  currentLyricIndex = 0;

  // 检测是否为流媒体模式（URL包含/api/stream）
  isStreamMode = url.indexOf('/api/stream') !== -1;

  try {
    if (audioCtx) {
      audioCtx.close();
      audioCtx = null;
      mediaSource = null;
      splitter = null;
      merger = null;
    }
  } catch(e) {}

  document.getElementById('btnTranscode').style.display = 'none';

  currentFileName = name;
  currentFilePath = path || '';
  var isAudio = isAudioFile(name);

  // 检查文件轨道完整性（非音频文件）
  if (!isAudio) {
    // 重置上一首的轨道信息，避免短暂闪烁
    mediaAudioTrackCount = -1;
    mediaAudioChannels = -1;
    var checkName = currentFilePath || name;
    if (!checkName) return;
    fetch('/api/check-tracks?name=' + encodeURIComponent(checkName))
      .then(function(r){ return r.json(); })
      .then(function(data){
        mediaAudioTrackCount = (data && data.audioTrackCount) ? data.audioTrackCount : 0;
        mediaAudioChannels = (data && data.audioChannels) ? data.audioChannels : 0;
        updateTrackButtons();
        if(data && data.message){
          showTrackWarning(data);
        } else {
          hideTrackWarning();
        }
      })
      .catch(function(){ hideTrackWarning(); });
  } else {
    hideTrackWarning();
  }

  // 先停止旧视频，终止旧的流媒体HTTP连接
  if (video) {
    video.pause();
    video.removeAttribute('src');
    video.load();
  }
  // 在重建video之前，保存当前全屏状态（自动切歌时保留全屏）
  var prevFullscreenElement = null;
  try {
    prevFullscreenElement = document.fullscreenElement || document.webkitFullscreenElement || document.mozFullScreenElement || document.msFullscreenElement;
  } catch(e) {}
  // 重建video元素，清除所有旧事件监听器
  document.getElementById("videoBox").innerHTML = '<video autoplay controls style="display: block;"></video>';
  video = document.querySelector("video");
  video.src = url + '?t=' + Date.now();
  video.volume = lastVolume;
  // 如果之前是video元素全屏（被替换会退出全屏），切歌后重新进入全屏
  // 注意：documentElement全屏不受video替换影响，无需重新请求
  // 同时：用户主动ESC退出后不再自动重新全屏（userExitedFullscreen标志）
  if (prevFullscreenElement && prevFullscreenElement !== document.documentElement && !userExitedFullscreen) {
    setTimeout(function() {
      if (!video) return;
      // 切歌瞬间用户未退出全屏，重新请求全屏（用documentElement更稳定）
      try {
        var el = document.documentElement;
        if (el.requestFullscreen) el.requestFullscreen();
        else if (el.webkitRequestFullscreen) el.webkitRequestFullscreen();
        else if (el.msRequestFullscreen) el.msRequestFullscreen();
        else if (el.mozRequestFullScreen) el.mozRequestFullScreen();
      } catch(e) {}
    }, 200);
  }

  // 注册快捷键（只注册一次）
  registerKeydownOnce();

  video.controls = true;
  video.style.display = 'block';

  if (isAudio) {
    document.querySelector(".ctrl-bar").style.display = "none";
  } else {
    document.querySelector(".ctrl-bar").style.display = "flex";
  }

  video.addEventListener("canplay", function() {
    switchingSong = false;
    setTimeout(function() {
      if (isAudio) return;

      // 根据已获取的音轨/声道信息更新按钮（check-tracks可能已返回）
      updateTrackButtons();

      // 声道模式：初始化WebAudio并应用当前选择的声道
      if (trackMode === 'channel') {
        if (!audioCtx) initWebAudio();
        if (applyChannelMode(lastTrackIndex)) {
          updateButtonStates(lastTrackIndex);
        }
        return;
      }

      // 多音轨模式 + 流媒体：音轨已在URL中指定，只需更新按钮状态
      if (isStreamMode) {
        updateButtonStates(lastTrackIndex);
        return;
      }

      // 多音轨模式 + 本地播放：尝试切换音轨
      if (doSwitchTrack(lastTrackIndex)) {
        return;
      }
      // 切换失败时（音轨未加载完成等）保持当前按钮状态，等待用户点击或自动重试
    }, 300);
  });

  function registerKeydownOnce() {
    if (keydownRegistered) return;
    keydownRegistered = true;
    document.addEventListener('keydown', function(e) {
      if (e.ctrlKey && e.key === 'ArrowDown') {
        e.preventDefault();
        if (e.repeat) return;
        if (nextSongCooldown) {
          console.log('[快捷键] 冷却中，忽略本次触发');
          return;
        }
        nextSongCooldown = true;
        setTimeout(function() { nextSongCooldown = false; }, 1000);
        console.log('[快捷键] Ctrl+↓ 触发切歌 ' + new Date().toLocaleTimeString('zh-CN', {hour12:false}));
        if (window.opener && !window.opener.closed) {
          window.opener.postMessage({action: "nextSong"}, "*");
        }
      }
    });
  }

  video.onended = function() {
    if (switchingSong) return;
    if (window.opener && !window.opener.closed) {
      window.opener.postMessage({action: "ended"}, "*");
    }
  };

  video.addEventListener("timeupdate", function() {
    handleTimeUpdate();
    updateLyrics();
  });

  video.onerror = function(e) {
    if (isTranscoding || switchingSong) {
      return;
    }
    if (!isAudio) {
      document.getElementById('btnTranscode').style.display = 'block';
    }
    var errorMsg = '';
    if (e.target.error) {
      var errorCode = e.target.error.code;
      if (isAudio && (errorCode === 4 || errorCode === 3)) {
        return;
      }
      switch(errorCode) {
        case 4:
          errorMsg = '该文件的视频编码不是浏览器支持编码，无法显示画面';
          break;
        case 3:
          errorMsg = '该文件的视频编码不是浏览器支持编码，无法显示画面';
          break;
        case 2:
          errorMsg = '网络错误，无法加载媒体文件';
          break;
        case 1:
          errorMsg = '媒体加载被中止';
          break;
        default:
          errorMsg = '媒体播放发生错误';
      }
    }
    if (errorMsg) {
      alert(errorMsg);
    }
  };

  if (!isAudio) {
    video.removeEventListener("loadedmetadata", onVideoMetadataLoaded);
    video.addEventListener("loadedmetadata", onVideoMetadataLoaded);
  }

  showTip("正在播放：" + name);

  loadLyrics(name);

  document.getElementById('playInfo').style.display = 'block';
  document.getElementById('currentSong').textContent = '正在播放：' + name;
  updateNextSongDisplay();
  updatePlayerQR();

  setTimeout(function() {
    document.getElementById('playInfo').style.display = 'none';
  }, 10000);
}

function updateNextSongDisplay() {
  var nextItem = null;
  // 从队列最前面扫描下一首可播放的歌曲（跳过当前播放的）
  for (var i = 0; i < currentQueue.length; i++) {
    if (i === currentPlayingIndex) continue;
    if (currentQueue[i].status === "ready") {
      nextItem = currentQueue[i];
      break;
    }
  }
  var nextSongEl = document.getElementById('nextSong');
  if (nextItem) {
    nextSongEl.textContent = '下一首：' + nextItem.name;
  } else {
    nextSongEl.textContent = '下一首：暂无';
  }
}

// isPrivateIP 判断IP是否属于内网地址
// 支持IPv4及简单IPv6（localhost/::1）
function isPrivateIP(ip) {
  if (!ip) return false;
  ip = ip.toLowerCase().trim();
  if (ip === 'localhost' || ip === '::1' || ip === '127.0.0.1') return true;
  // 去除可能存在的IPv6前缀 "::ffff:"
  ip = ip.replace(/^::ffff:/, '');
  var parts = ip.split('.');
  if (parts.length !== 4) return false;
  var a = parseInt(parts[0], 10), b = parseInt(parts[1], 10);
  if (isNaN(a) || isNaN(b)) return false;
  // 10.0.0.0/8
  if (a === 10) return true;
  // 172.16.0.0/12
  if (a === 172 && b >= 16 && b <= 31) return true;
  // 192.168.0.0/16
  if (a === 192 && b === 168) return true;
  // 169.254.0.0/16 (link-local)
  if (a === 169 && b === 254) return true;
  return false;
}

// extractIP 从 "ip:port" 或 "host:port" 中提取 IP/host
function extractIP(addr) {
  if (!addr) return '';
  // 处理 [ipv6]:port 形式
  if (addr.charAt(0) === '[') {
    var idx = addr.indexOf(']');
    return idx > 0 ? addr.substring(1, idx) : addr;
  }
  // 处理 IPv4 host:port
  var lastColon = addr.lastIndexOf(':');
  if (lastColon > 0) return addr.substring(0, lastColon);
  return addr;
}

// updatePlayerQR 智能显示二维码：服务器可用则显示，内网IP加提示
function updatePlayerQR() {
  var qrBlock = document.getElementById('qrBlock');
  var qrImg = document.getElementById('qrImage');
  var qrTip = document.getElementById('qrNetworkTip');
  if (!qrBlock || !qrImg || !qrTip) return;
  if (!currentSessionId) { qrBlock.style.display = 'none'; return; }

  fetch('/api/qr/status').then(function(r){return r.json();}).then(function(data){
    if (!data.enabled || !data.connected) {
      qrBlock.style.display = 'none';
      return;
    }
    var qrUrl = 'http://' + data.qrServerAddr + '/m/' + currentSessionId;
    var qrImgUrl = '/api/qr/image?url=' + encodeURIComponent(qrUrl);
    qrImg.src = qrImgUrl;
    var ip = extractIP(data.qrServerAddr);
    if (isPrivateIP(ip)) {
      qrTip.textContent = '手机必须与点歌电脑在同个网络内';
      qrTip.style.color = '#f0ad4e';
    } else {
      qrTip.textContent = '支持外网点歌';
      qrTip.style.color = '#5cb85c';
    }
    qrBlock.style.display = 'block';
  }).catch(function(){
    qrBlock.style.display = 'none';
  });
}

window.addEventListener("message", function(e) {
  var data = e.data;
  if (data.action === "play") {
    playVideo(data.url, data.name, data.type, data.path);
  } else if (data.action === "switchTrack") {
    switchTrack(data.index);
  } else if (data.action === "syncQueue") {
    currentQueue = data.list;
    currentPlayingIndex = data.currentPlayingIndex !== undefined ? data.currentPlayingIndex : -1;
    if (data.sessionId) currentSessionId = data.sessionId;
    updateNextSongDisplay();
  }
});

function showTrackWarning(data) {
  var bar = document.getElementById('trackWarningBar');
  var text = document.getElementById('trackWarningText');
  if (bar && text) {
    var icon = '';
    if (data.noVideo) icon += '🎬无画面 ';
    if (data.noAudio) icon += '🔊无声音 ';
    if (!data.noAudio && data.audioTrackCount === 1) icon += '🎵仅单音轨 ';
    text.textContent = icon + ' ' + data.message;
    // 单音轨用橙色警告，无音轨/无视频用红色
    bar.style.background = (data.noAudio || data.noVideo) ? '#d9534f' : '#f0ad4e';
    bar.style.display = 'block';
  }
}
function hideTrackWarning() {
  var bar = document.getElementById('trackWarningBar');
  if (bar) bar.style.display = 'none';
}

function handleTimeUpdate() {
  if (!video || isNaN(video.duration)) return;

  var remainingTime = video.duration - video.currentTime;
  if (remainingTime <= 10 && remainingTime > 9) {
    document.getElementById('playInfo').style.display = 'block';
    updateNextSongDisplay();
    updatePlayerQR();
    var nextItem = null;
    // 从队列最前面扫描下一首可播放的歌曲（跳过当前播放的）
    for (var i = 0; i < currentQueue.length; i++) {
      if (i === currentPlayingIndex) continue;
      if (currentQueue[i].status === "ready") {
        nextItem = currentQueue[i];
        break;
      }
    }
    if (nextItem) {
      showTip("即将播放：" + nextItem.name);
    } else {
      showTip("即将播放完毕，队列已空");
    }
  }
}

var transcodeInterval = null;
var isTranscoding = false;

function startTranscode() {
  var btn = document.getElementById('btnTranscode');
  var progress = document.getElementById('transcodeProgress');
  var progressFill = document.getElementById('progressFill');
  var progressPercent = document.getElementById('progressPercent');

  btn.disabled = true;
  btn.innerHTML = '转码中...';
  progress.style.display = 'flex';
  progressFill.style.width = '0%';
  progressPercent.textContent = '0%';

  isTranscoding = true;

  if (video) {
    video.pause();
    video.src = '';
  }

  var xhr = new XMLHttpRequest();
  xhr.open('POST', '/api/transcode', true);
  xhr.setRequestHeader('Content-Type', 'application/json');
  xhr.onload = function() {
    if (xhr.status === 200) {
      pollTranscodeProgress();
    } else {
      alert('转码任务创建失败');
      resetTranscodeUI();
    }
  };
  xhr.onerror = function() {
    alert('转码任务创建失败');
    resetTranscodeUI();
  };
  xhr.send(JSON.stringify({fileName: currentFileName}));
}

function pollTranscodeProgress() {
  transcodeInterval = setInterval(function() {
    var xhr = new XMLHttpRequest();
    xhr.open('GET', '/api/transcode/progress', true);
    xhr.onload = function() {
      if (xhr.status === 200) {
        var data = JSON.parse(xhr.responseText);
        var progressFill = document.getElementById('progressFill');
        var progressPercent = document.getElementById('progressPercent');
        var progressLog = document.getElementById('progressLog');
        var progressCommand = document.getElementById('progressCommand');
        var progressMediaInfo = document.getElementById('progressMediaInfo');

        progressFill.style.width = data.progress + '%';
        progressPercent.textContent = data.progress + '%';

        if (data.mediaInfo) {
          progressMediaInfo.textContent = '媒体信息:\n' + data.mediaInfo;
        }

        if (data.command) {
          progressCommand.textContent = 'ffmpeg命令: ' + data.command;
        }

        if (data.log) {
          progressLog.textContent = data.log;
          progressLog.scrollTop = progressLog.scrollHeight;
        }

        if (data.status === 'completed') {
          clearInterval(transcodeInterval);
          alert('转码完成！正在重新播放...');
          resetTranscodeUI();
          if (window.opener && !window.opener.closed) {
            window.opener.postMessage({action: "playByName", name: currentFileName}, "*");
          }
        } else if (data.status === 'error') {
          clearInterval(transcodeInterval);
          alert('转码失败: ' + data.message);
          resetTranscodeUI();
        }
      } else {
        clearInterval(transcodeInterval);
        resetTranscodeUI();
      }
    };
    xhr.onerror = function() {
      clearInterval(transcodeInterval);
      resetTranscodeUI();
    };
    xhr.send();
  }, 1000);
}

function resetTranscodeUI() {
  var btn = document.getElementById('btnTranscode');
  var progress = document.getElementById('transcodeProgress');

  btn.disabled = false;
  btn.innerHTML = '一键申请后台转码';
  progress.style.display = 'none';
  isTranscoding = false;
}

// 全屏状态跟踪：用户主动退出全屏后，切歌不再自动重新全屏
var userExitedFullscreen = false;
function getFullscreenElement() {
  try {
    return document.fullscreenElement || document.webkitFullscreenElement || document.mozFullScreenElement || document.msFullscreenElement;
  } catch(e) { return null; }
}
function exitFullscreenApi() {
  try {
    if (document.exitFullscreen) document.exitFullscreen();
    else if (document.webkitExitFullscreen) document.webkitExitFullscreen();
    else if (document.msExitFullscreen) document.msExitFullscreen();
    else if (document.mozCancelFullScreen) document.mozCancelFullScreen();
  } catch(e) {}
}
document.addEventListener('fullscreenchange', function() {
  if (!getFullscreenElement()) userExitedFullscreen = true;
  else userExitedFullscreen = false;
});
document.addEventListener('webkitfullscreenchange', function() {
  if (!getFullscreenElement()) userExitedFullscreen = true;
  else userExitedFullscreen = false;
});

document.onkeydown = function(e) {
  if (e.key === "F11") {
    e.preventDefault();
    if (getFullscreenElement()) exitFullscreenApi();
    else document.documentElement.requestFullscreen();
    return;
  }
  if (e.key === "Escape") {
    // 兜底：浏览器原生ESC如果未生效，主动调用API退出全屏
    if (getFullscreenElement()) {
      exitFullscreenApi();
    }
  }
};

document.addEventListener('mousemove', function() {
  var ctrlBar = document.querySelector('.ctrl-bar');
  ctrlBar.style.opacity = '1';
  clearTimeout(ctrlBar.hideTimer);
  ctrlBar.hideTimer = setTimeout(function() {
    ctrlBar.style.opacity = '0';
  }, 3000);
});

window.addEventListener('focus', function() {
  var ctrlBar = document.querySelector('.ctrl-bar');
  ctrlBar.style.opacity = '1';
  clearTimeout(ctrlBar.hideTimer);
  ctrlBar.hideTimer = setTimeout(function() {
    ctrlBar.style.opacity = '0';
  }, 3000);
});

window.addEventListener('load', function() {
  var ctrlBar = document.querySelector('.ctrl-bar');
  ctrlBar.style.opacity = '0';

  var urlParams = new URLSearchParams(window.location.search);
  var playFileName = urlParams.get('play');
  if (playFileName && window.opener && !window.opener.closed) {
    window.opener.postMessage({action: "playByName", name: decodeURIComponent(playFileName)}, "*");
  }
});
</script>
</body>
</html>
`))
}
