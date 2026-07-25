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
*{margin:0;padding:0;background:#000}
body{width:100vw;height:100vh;overflow:hidden}
.video-container{position:relative;width:100vw;height:100vh;z-index:1}
video{width:100vw;height:100vh;object-fit:contain;z-index:1}
.lyrics-container{position:absolute;top:0;left:0;width:calc(100vw - 20px);height:85vh;display:flex;align-items:flex-start;justify-content:flex-start;z-index:10;overflow:hidden;pointer-events:none;margin-right:10px}
.lyrics{width:100%;height:100%;text-align:center;color:#fff;font-size:24px;line-height:1.8;text-shadow:2px 2px 4px rgba(0,0,0,0.8);overflow-y:auto;display:flex;flex-direction:column;align-items:center;pointer-events:auto;box-sizing:border-box;padding:20px 30px 20px 20px}
.lyrics-line{margin:10px 0;opacity:0.6}
.lyrics-line.active{opacity:1;font-size:28px;font-weight:bold;color:#00aaff}
.ctrl-bar{position:fixed;bottom:40px;left:40px;z-index:999;display:flex;gap:15px;opacity:0;transition:opacity 0.3s ease}
.track-btn{padding:12px 28px;border:none;border-radius:8px;font-size:18px;cursor:pointer;color:#fff}
.track-btn.active{background:#28a745}
.track-btn:not(.active){background:#007bff}
.transcode-btn{padding:12px 28px;border:none;border-radius:8px;font-size:18px;cursor:pointer;color:#fff;background:#e67e22}
.transcode-btn:hover{background:#d35400}
.transcode-btn:disabled{background:#666;cursor:not-allowed}
.tips{position:fixed;top:20px;left:50%;transform:translateX(-50%);background:rgba(0,0,0,0.8);color:#fff;padding:10px 30px;border-radius:6px;display:none;font-size:18px}
.transcode-progress{position:fixed;top:50%;left:50%;transform:translate(-50%,-50%);background:rgba(0,0,0,0.95);color:#fff;padding:30px 50px;border-radius:12px;display:none;flex-direction:column;align-items:center;z-index:1000;max-width:600px;max-height:80vh;overflow:hidden}
.progress-bar{width:300px;height:20px;background:#333;border-radius:10px;overflow:hidden;margin:10px 0}
.progress-fill{height:100%;background:#00aaff;width:0%}
.progress-text{font-size:16px;color:#fff;margin-bottom:10px}
.progress-log{width:100%;height:150px;background:#1a1a2e;border-radius:8px;padding:10px;font-family:Consolas,Monaco,monospace;font-size:12px;color:#00ff88;overflow-y:auto;white-space:pre-wrap}
.progress-command{width:100%;background:#2a2a4e;border-radius:8px;padding:10px;font-family:Consolas,Monaco,monospace;font-size:11px;color:#ffaa00;overflow-x:auto;margin-bottom:10px}
.progress-mediainfo{width:100%;background:#1a2a3e;border-radius:8px;padding:10px;font-family:Consolas,Monaco,monospace;font-size:12px;color:#00ccff;white-space:pre-wrap;margin-bottom:10px}
.play-info{position:fixed;top:20px;right:20px;background:rgba(0,0,0,0.8);color:#fff;padding:10px 20px;border-radius:6px;font-size:16px;z-index:100}
.play-info .current{color:#00ff00;margin-bottom:5px}
.play-info .next{color:#ffaa00}
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
<div id="trackWarningBar" style="display:none;background:#ff4444;color:#fff;padding:10px 16px;font-size:15px;font-weight:bold;text-align:center;animation:warnPulse 2s infinite">
  <span id="trackWarningText"></span>
</div>
<style>@keyframes warnPulse{0%,100%{opacity:1}50%{opacity:0.7}}</style>
<div class="play-info" id="playInfo" style="display:none">
  <div class="current" id="currentSong"></div>
  <div class="next" id="nextSong"></div>
</div>
<div class="ctrl-bar">
  <button class="track-btn active" id="btnOrigin" onclick="switchTrack(0)">原唱</button>
  <button class="track-btn" id="btnAcc" onclick="switchTrack(1)">伴奏</button>
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
var currentQueue = [];
var currentPlayingIndex = -1;
var audioCtx = null;
var mediaSource = null;
var splitter = null;
var merger = null;
var channelMode = 'stereo';
var lastTrackIndex = 0;
var lastVolume = 1;
var isStreamMode = false; // 是否为流媒体模式（省流模式）

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

  var lrcUrl = songName.replace(/\.[^.]+$/, '.lrc');
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

    document.getElementById("btnOrigin").classList.toggle("active", targetIdx === 0);
    document.getElementById("btnAcc").classList.toggle("active", targetIdx === 1);
    return true;
  }

  return false;
}

function switchTrack(targetIdx) {
  lastTrackIndex = targetIdx;

  // 流媒体模式：通知主页面重新播放（带新的trackIndex）
  if (isStreamMode) {
    document.getElementById("btnOrigin").classList.toggle("active", targetIdx === 0);
    document.getElementById("btnAcc").classList.toggle("active", targetIdx === 1);
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

  if (audioCtx && splitter && merger) {
    if (targetIdx === 0) {
      setupChannelRouting('left');
      channelMode = 'left';
      showTip("原唱模式：左声道");
    } else {
      setupChannelRouting('right');
      channelMode = 'right';
      showTip("伴奏模式：右声道");
    }
    document.getElementById("btnOrigin").classList.toggle("active", targetIdx === 0);
    document.getElementById("btnAcc").classList.toggle("active", targetIdx === 1);
    clearTimeout(retryTimer);
    return;
  }

  showTip("⏳ 音轨未加载，自动重试中...");

  clearTimeout(retryTimer);
  retryTimer = setInterval(function() {
    if (doSwitchTrack(targetIdx)) {
      clearTimeout(retryTimer);
      return;
    }

    if (audioCtx && splitter && merger) {
      if (targetIdx === 0) {
        setupChannelRouting('left');
        channelMode = 'left';
      } else {
        setupChannelRouting('right');
        channelMode = 'right';
      }
      document.getElementById("btnOrigin").classList.toggle("active", targetIdx === 0);
      document.getElementById("btnAcc").classList.toggle("active", targetIdx === 1);
      clearTimeout(retryTimer);
    }
  }, 200);
}

function isAudioFile(fileName) {
  var audioExtensions = ['.mp3', '.wav', '.flac', '.aac', '.m4a', '.ogg', '.wma'];
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
      alert('警告：当前媒体文件没有音频轨道或音频编码不被支持，可能无法正常播放声音');
      document.getElementById('btnTranscode').style.display = 'block';
    }

    if (video.videoWidth === 0 && video.videoHeight === 0) {
      alert('警告：当前媒体文件的视频轨道无法解码，可能是视频编码不被支持');
      document.getElementById('btnTranscode').style.display = 'block';
    }
  }, 1000);
}

function playVideo(url, name, type) {
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
  var isAudio = isAudioFile(name);

  // 检查文件轨道完整性（非音频文件）
  if (!isAudio) {
    fetch('/api/check-tracks?name=' + encodeURIComponent(name))
      .then(function(r){ return r.json(); })
      .then(function(data){
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
  // 重建video元素，清除所有旧事件监听器
  document.getElementById("videoBox").innerHTML = '<video autoplay controls style="display: block;"></video>';
  video = document.querySelector("video");
  video.src = url + '?t=' + Date.now();
  video.volume = lastVolume;

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
      // 流媒体模式：音轨已在URL中指定，只需更新按钮状态
      if (isStreamMode) {
        document.getElementById("btnOrigin").classList.toggle("active", lastTrackIndex === 0);
        document.getElementById("btnAcc").classList.toggle("active", lastTrackIndex === 1);
        return;
      }

      if (doSwitchTrack(lastTrackIndex)) {
        return;
      }

      if (!audioCtx) {
        initWebAudio();
      }
      if (audioCtx && splitter && merger) {
        if (lastTrackIndex === 0) {
          setupChannelRouting('left');
          channelMode = 'left';
        } else {
          setupChannelRouting('right');
          channelMode = 'right';
        }
        document.getElementById("btnOrigin").classList.toggle("active", lastTrackIndex === 0);
        document.getElementById("btnAcc").classList.toggle("active", lastTrackIndex === 1);
      }
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

  setTimeout(function() {
    document.getElementById('playInfo').style.display = 'none';
  }, 10000);
}

function updateNextSongDisplay() {
  var nextItem = null;
  if (currentPlayingIndex >= 0 && currentPlayingIndex < currentQueue.length - 1) {
    for (var i = currentPlayingIndex + 1; i < currentQueue.length; i++) {
      if (currentQueue[i].status === "ready") {
        nextItem = currentQueue[i];
        break;
      }
    }
  }
  var nextSongEl = document.getElementById('nextSong');
  if (nextItem) {
    nextSongEl.textContent = '下一首：' + nextItem.name;
  } else {
    nextSongEl.textContent = '下一首：暂无';
  }
}

window.addEventListener("message", function(e) {
  var data = e.data;
  if (data.action === "play") {
    playVideo(data.url, data.name, data.type);
  } else if (data.action === "switchTrack") {
    switchTrack(data.index);
  } else if (data.action === "syncQueue") {
    currentQueue = data.list;
    currentPlayingIndex = data.currentPlayingIndex !== undefined ? data.currentPlayingIndex : -1;
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
    text.textContent = icon + ' ' + data.message;
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
    var nextItem = null;
    if (currentPlayingIndex >= 0 && currentPlayingIndex < currentQueue.length - 1) {
      for (var i = currentPlayingIndex + 1; i < currentQueue.length; i++) {
        if (currentQueue[i].status === "ready") {
          nextItem = currentQueue[i];
          break;
        }
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

document.onkeydown = function(e) {
  if (e.key === "F11") {
    document.documentElement.requestFullscreen();
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
