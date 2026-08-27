package main

import "net/http"

func AudioPlayerHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(`
<!DOCTYPE html>
<html>
<head>
<meta charset="UTF-8">
<title>KTV 音频播放</title>
<style>
*{margin:0;padding:0;box-sizing:border-box}
body{width:100vw;height:100vh;overflow:hidden;background:#1a252f;display:flex;flex-direction:column;font-family:Microsoft YaHei,sans-serif;color:#fff}
.song-title{font-size:26px;padding:12px 20px;text-align:center;overflow:hidden;text-overflow:ellipsis;white-space:nowrap;color:#fff;text-shadow:0 1px 3px rgba(0,0,0,0.4);background:#1e2d3a;border-bottom:1px solid #2c3e50}
.content-area{flex:1;min-height:0;display:flex;flex-direction:column;overflow:hidden;position:relative}
.lyrics-container{flex:1;overflow-y:auto;overflow-x:hidden;text-align:center;padding:0 20px;mask-image:linear-gradient(transparent,black 10%,black 90%,transparent);-webkit-mask-image:linear-gradient(transparent,black 10%,black 90%,transparent)}
.lyrics-line{margin:10px 0;opacity:0.35;font-size:18px;transition:all 0.4s ease;white-space:nowrap;display:block;width:fit-content;margin-left:auto;margin-right:auto;max-width:100%;color:rgba(255,255,255,0.5);text-shadow:0 1px 2px rgba(0,0,0,0.3)}
.lyrics-line.active{opacity:1;font-size:24px;font-weight:bold;color:#5bc0de;text-shadow:0 1px 3px rgba(0,0,0,0.3)}
.visual-container{flex:1;display:flex;flex-direction:column;min-height:0;position:relative}
.visual-canvas-wrap{flex:1;min-height:0;position:relative;background:#0f1923;border-radius:6px;overflow:hidden;box-shadow:0 2px 8px rgba(0,0,0,0.3);outline:1px solid #2c3e50;outline-offset:-1px}
.visual-canvas-wrap canvas{width:100%;height:100%;display:block}
.toolbar{position:absolute;top:8px;right:8px;display:flex;gap:5px;z-index:10;flex-wrap:wrap;justify-content:flex-end}
.toolbar button,.toolbar select{padding:5px 10px;border:1px solid #2c3e50;border-radius:6px;background:#428bca;color:#fff;font-size:12px;cursor:pointer;font-family:inherit;box-shadow:0 2px 4px rgba(0,0,0,0.25);transition:background 0.15s ease,border-color 0.15s ease}
.toolbar button:hover,.toolbar select:hover{background:#3276b1;border-color:#2c3e50;box-shadow:0 2px 6px rgba(0,0,0,0.3)}
.toolbar button:active,.toolbar select:active{background:#286090;box-shadow:0 1px 2px rgba(0,0,0,0.2)}
.toolbar button.active{background:#5cb85c;border-color:#4cae4c;box-shadow:0 2px 4px rgba(0,0,0,0.25)}
.toolbar button.active:hover{background:#4cae4c;box-shadow:0 2px 6px rgba(0,0,0,0.3)}
.toolbar button.active:active{background:#449d44;box-shadow:0 1px 2px rgba(0,0,0,0.2)}
.toolbar select{appearance:auto;-webkit-appearance:menulist;background:#6c757d;border-color:#5a6268}
.toolbar select:hover{background:#5a6268}
.toolbar select option{background:#1a252f;color:#fff}
.toolbar input[type="color"]{width:30px;height:24px;border:1px solid #2c3e50;border-radius:6px;background:#6c757d;cursor:pointer;padding:0;vertical-align:middle;box-shadow:0 2px 4px rgba(0,0,0,0.25)}
.player-controls{display:flex;align-items:center;gap:12px;padding:12px 20px;background:#1e2d3a;border-top:1px solid #2c3e50;box-shadow:0 -2px 8px rgba(0,0,0,0.2)}
.progress-bar{flex:1;height:10px;background:#2c3e50;border-radius:5px;cursor:pointer;position:relative;box-shadow:inset 0 1px 3px rgba(0,0,0,0.3)}
.progress-fill{height:100%;background:#428bca;border-radius:5px;width:0%;transition:width 0.3s;box-shadow:0 1px 0 rgba(255,255,255,0.15) inset}
.time-display{font-size:14px;color:rgba(255,255,255,0.7);white-space:nowrap;text-shadow:0 1px 2px rgba(0,0,0,0.3)}
.volume-control{display:flex;align-items:center;gap:6px}
.volume-control input[type="range"]{width:80px;accent-color:#428bca;height:8px;background:transparent;border-radius:4px}
.volume-control input[type="range"]::-webkit-slider-runnable-track{height:8px;background:#2c3e50;border-radius:4px;box-shadow:inset 0 1px 3px rgba(0,0,0,0.3)}
.volume-control input[type="range"]::-webkit-slider-thumb{-webkit-appearance:none;width:16px;height:16px;border-radius:50%;background:#428bca;border:2px solid #3276b1;box-shadow:0 2px 4px rgba(0,0,0,0.25);margin-top:-5px;cursor:pointer}
.tips{position:fixed;top:20px;left:50%;transform:translateX(-50%);background:#1e2d3a;color:#fff;padding:12px 36px;border-radius:6px;display:none;font-size:18px;border:1px solid #2c3e50;box-shadow:0 4px 12px rgba(0,0,0,0.35);text-shadow:0 1px 2px rgba(0,0,0,0.3)}
.play-info{position:fixed;top:20px;right:20px;background:#1e2d3a;color:#fff;padding:12px 24px;border-radius:6px;font-size:16px;z-index:100;border:1px solid #2c3e50;box-shadow:0 4px 12px rgba(0,0,0,0.35)}
.play-info .current{color:#5cb85c;margin-bottom:6px;text-shadow:0 1px 2px rgba(0,0,0,0.3)}
.play-info .next{color:#f0ad4e;text-shadow:0 1px 2px rgba(0,0,0,0.3)}
.fullscreen-overlay{position:fixed;top:0;left:0;width:100vw;height:100vh;background:#1a252f;z-index:9999;display:none;flex-direction:column;padding:10px}
.fullscreen-overlay .content-area{flex:1;min-height:0;padding:0 20px}
.fullscreen-overlay .lyrics-container{flex:1;overflow-x:hidden}
.fullscreen-overlay .visual-canvas-wrap{flex:1}
.fullscreen-overlay .toolbar{top:12px;right:12px}
.fullscreen-overlay .toolbar button,.fullscreen-overlay .toolbar select{font-size:14px;padding:8px 14px}
.exit-fullscreen-hint{position:absolute;bottom:15px;left:50%;transform:translateX(-50%);font-size:13px;color:rgba(255,255,255,0.3);pointer-events:none;text-shadow:0 1px 2px rgba(0,0,0,0.3)}

/* Theme: 智慧蓝 */
.theme-smart-blue .lyrics-line.active{color:#5bc0de;text-shadow:0 1px 3px rgba(0,0,0,0.3)}
/* Theme: 青春少女 */
.theme-youth-pink .lyrics-line.active{color:#ff6b9d;text-shadow:0 1px 3px rgba(0,0,0,0.3)}
.theme-youth-pink .lyrics-line{opacity:0.25}
/* Theme: 复古金 */
.theme-retro-gold .lyrics-line.active{color:#f0ad4e;text-shadow:0 1px 3px rgba(0,0,0,0.3)}
.theme-retro-gold .lyrics-line{opacity:0.25}
/* Theme: 翡翠绿 */
.theme-emerald .lyrics-line.active{color:#5cb85c;text-shadow:0 1px 3px rgba(0,0,0,0.3)}
.theme-emerald .lyrics-line{opacity:0.25}
/* Theme: 烈焰红 */
.theme-fire-red .lyrics-line.active{color:#d9534f;text-shadow:0 1px 3px rgba(0,0,0,0.3)}
.theme-fire-red .lyrics-line{opacity:0.25}
/* Theme: 紫罗兰 */
.theme-violet .lyrics-line.active{color:#9b59b6;text-shadow:0 1px 3px rgba(0,0,0,0.3)}
.theme-violet .lyrics-line{opacity:0.25}

/* Transition: 直接切换 */
.trans-direct .lyrics-line{transition:color 0.2s,opacity 0.2s}
/* Transition: 平滑缩放 */
.trans-zoom .lyrics-line{transition:all 0.5s cubic-bezier(0.25,0.46,0.45,0.94);transform-origin:center center}
.trans-zoom .lyrics-line.active{transform:scale(1.15)}
/* Transition: 淡入淡出 */
.trans-fade .lyrics-line{transition:all 0.6s ease}
.trans-fade .lyrics-line.active{opacity:1;letter-spacing:2px}
/* Transition: 滑入 */
.trans-slide .lyrics-line{transition:all 0.5s cubic-bezier(0.25,0.46,0.45,0.94);transform:translateX(0);opacity:0.4}
.trans-slide .lyrics-line.active{transform:translateX(10px);opacity:1}
/* Transition: 弹跳 */
.trans-bounce .lyrics-line{transition:all 0.5s cubic-bezier(0.68,-0.55,0.265,1.55);transform-origin:center center}
.trans-bounce .lyrics-line.active{transform:scale(1.12)}
</style>
</head>
<body>
<div id="fpsDisplay" style="position:fixed;top:8px;left:8px;z-index:2147483000;background:rgba(0,0,0,0.55);color:#4caf50;font:12px Consolas,monospace;padding:2px 8px;border-radius:4px;pointer-events:none;user-select:none">FPS: --</div>
<div class="song-title" id="songTitle">等待播放...</div>
<div class="content-area" id="contentArea">
  <div class="lyrics-container theme-smart-blue trans-zoom" id="lyricsContainer" style="display:none">
    <div id="lyrics"></div>
  </div>
  <div class="visual-container" id="visualContainer">
    <div class="visual-canvas-wrap" id="canvasWrap">
      <canvas id="mainCanvas"></canvas>
    </div>
  </div>
  <div class="toolbar" id="toolbar" style="display:none">
    <button id="btnLyrics" onclick="userSetView('lyrics')">歌词</button>
    <button id="btnWaveform" class="active" onclick="userSetView('waveform')">波形</button>
    <button id="btnSpectrumBar" onclick="userSetView('spectrumbar')">频谱条</button>
    <button id="btnSpectrumCurve" onclick="userSetView('spectrumcurve')">频谱曲线</button>
    <select id="selWaveformTime" onchange="setWaveformTime(this.value)" title="波形时长" style="display:none">
      <option value="15">15ms</option>
      <option value="50">50ms</option>
      <option value="100" selected>100ms</option>
      <option value="200">200ms</option>
      <option value="500">500ms</option>
      <option value="1000">1s</option>
    </select>
    <select id="selSpectrumBars" onchange="setSpectrumBars(this.value)" title="频谱条数" style="display:none">
      <option value="64">64条</option>
      <option value="128">128条</option>
      <option value="256" selected>256条</option>
      <option value="512">512条</option>
    </select>
    <button id="btnPeakColor" onclick="togglePeakColoring()" title="峰值着色" style="display:none" class="active">峰值着色</button>
    <select id="selSpectrumColor" onchange="setSpectrumColor(this.value)" title="频谱颜色" style="display:none">
      <option value="0,100,255">蓝色</option>
      <option value="0,200,200">青色</option>
      <option value="0,200,80">绿色</option>
      <option value="150,50,255">紫色</option>
      <option value="255,50,50">红色</option>
      <option value="255,140,0">橙色</option>
      <option value="255,200,0">黄色</option>
      <option value="custom">自定义</option>
    </select>
    <input type="color" id="customSpectrumColor" value="#0064ff" onchange="setSpectrumColorCustom(this.value)" title="自定义颜色" style="display:none">
    <select id="selCurveStyle" onchange="setCurveStyle(this.value)" title="曲线样式" style="display:none">
      <option value="fast">快速曲线</option>
      <option value="smooth">平滑曲线</option>
    </select>
    <select id="selTheme" onchange="setTheme(this.value)" title="歌词配色" style="display:none">
      <option value="smart-blue">智慧蓝</option>
      <option value="youth-pink">青春少女</option>
      <option value="retro-gold">复古金</option>
      <option value="emerald">翡翠绿</option>
      <option value="fire-red">烈焰红</option>
      <option value="violet">紫罗兰</option>
    </select>
    <select id="selTransition" onchange="setTransition(this.value)" title="切换效果" style="display:none">
      <option value="zoom">平滑缩放</option>
      <option value="direct">直接切换</option>
      <option value="fade">淡入淡出</option>
      <option value="slide">滑入</option>
      <option value="bounce">弹跳</option>
    </select>
    <button onclick="toggleFullscreen()" title="全屏">⛶</button>
  </div>
</div>
<div class="player-controls">
  <span class="time-display" id="currentTime">0:00</span>
  <div class="progress-bar" id="progressBar" onclick="seekAudio(event)">
    <div class="progress-fill" id="progressFill"></div>
  </div>
  <span class="time-display" id="totalTime">0:00</span>
  <div class="volume-control">
    <span>&#128264;</span>
    <input type="range" id="volumeSlider" min="0" max="100" value="100" oninput="setVolume(this.value)">
  </div>
</div>
<div class="tips" id="tipBox"></div>
<div class="play-info" id="playInfo" style="display:none">
  <div class="current" id="currentSong"></div>
  <div class="next" id="nextSong"></div>
</div>
<div class="fullscreen-overlay" id="fullscreenOverlay">
  <div id="fpsDisplayFs" style="position:fixed;top:8px;left:8px;z-index:2147483000;background:rgba(0,0,0,0.55);color:#4caf50;font:12px Consolas,monospace;padding:2px 8px;border-radius:4px;pointer-events:none;user-select:none">FPS: --</div>
  <div class="content-area" id="fsContentArea">
    <div class="lyrics-container theme-smart-blue trans-zoom" id="fsLyricsContainer" style="display:none">
      <div id="fsLyrics"></div>
    </div>
    <div class="visual-container" id="fsVisualContainer">
      <div class="visual-canvas-wrap">
        <canvas id="fullscreenCanvas"></canvas>
      </div>
    </div>
    <div class="toolbar" id="fsToolbar">
      <button id="btnLyricsFs" onclick="userSetView('lyrics')">歌词</button>
      <button id="btnWaveformFs" class="active" onclick="userSetView('waveform')">波形</button>
      <button id="btnSpectrumBarFs" onclick="userSetView('spectrumbar')">频谱条</button>
      <button id="btnSpectrumCurveFs" onclick="userSetView('spectrumcurve')">频谱曲线</button>
      <select id="selWaveformTimeFs" onchange="setWaveformTime(this.value)" title="波形时长" style="display:none">
        <option value="15">15ms</option>
        <option value="50">50ms</option>
        <option value="100" selected>100ms</option>
        <option value="200">200ms</option>
        <option value="500">500ms</option>
        <option value="1000">1s</option>
      </select>
      <select id="selSpectrumBarsFs" onchange="setSpectrumBars(this.value)" title="频谱条数" style="display:none">
        <option value="64">64条</option>
        <option value="128">128条</option>
        <option value="256" selected>256条</option>
        <option value="512">512条</option>
      </select>
      <button id="btnPeakColorFs" onclick="togglePeakColoring()" title="峰值着色" style="display:none" class="active">峰值着色</button>
      <select id="selSpectrumColorFs" onchange="setSpectrumColor(this.value)" title="频谱颜色" style="display:none">
        <option value="0,100,255">蓝色</option>
        <option value="0,200,200">青色</option>
        <option value="0,200,80">绿色</option>
        <option value="150,50,255">紫色</option>
        <option value="255,50,50">红色</option>
        <option value="255,140,0">橙色</option>
        <option value="255,200,0">黄色</option>
        <option value="custom">自定义</option>
      </select>
      <input type="color" id="customSpectrumColorFs" value="#0064ff" onchange="setSpectrumColorCustom(this.value)" title="自定义颜色" style="display:none">
      <select id="selCurveStyleFs" onchange="setCurveStyle(this.value)" title="曲线样式" style="display:none">
        <option value="fast">快速曲线</option>
        <option value="smooth">平滑曲线</option>
      </select>
      <select id="selThemeFs" onchange="setTheme(this.value)" title="歌词配色" style="display:none">
        <option value="smart-blue">智慧蓝</option>
        <option value="youth-pink">青春少女</option>
        <option value="retro-gold">复古金</option>
        <option value="emerald">翡翠绿</option>
        <option value="fire-red">烈焰红</option>
        <option value="violet">紫罗兰</option>
      </select>
      <select id="selTransitionFs" onchange="setTransition(this.value)" title="切换效果" style="display:none">
        <option value="zoom">平滑缩放</option>
        <option value="direct">直接切换</option>
        <option value="fade">淡入淡出</option>
        <option value="slide">滑入</option>
        <option value="bounce">弹跳</option>
      </select>
      <button onclick="toggleFullscreen()">✕</button>
    </div>
  </div>
  <div class="exit-fullscreen-hint">按 ESC 或 F11 退出全屏</div>
</div>

<audio id="audioPlayer" autoplay></audio>

<script>
var audio = document.getElementById('audioPlayer');
var lyrics = [];
var currentLyricIndex = 0;
var currentFileName = '';
var currentFilePath = '';
var currentQueue = [];
var currentPlayingIndex = -1;
var hasLyrics = false;
var audioCtx = null;
var analyser = null;
var audioSource = null;
var animId = null;
var currentView = 'waveform';
var userSelectedView = false; // 用户是否手动选择过视图
var isFullscreen = false;
var currentTheme = 'smart-blue';
var currentTransition = 'zoom';
var waveformTimeMs = 100;

var spectrumSettings = {
  fftSize: 4096, barCount: 256, showPeaks: true,
  topDb: -20, bottomDb: -72,
  colorTop1: [255,0,0], colorPct20: [255,255,0], colorPct40: [0,100,255],
  thresholdPct20: 10, thresholdPct40: 30
};
var dynamicTopDb = 0; // 动态上限，初始=默认topDb
var defaultTopDb = 0;  // 默认上限（从spectrumSettings.topDb初始化）
var spectrumPeakData = [];
var spectrumPeakTimes = [];
var PEAK_HOLD_TIME_MS = 300;
var PEAK_HOLD_DECAY_RATE = 10;
var peakColoring = true;
var spectrumColor = [0, 100, 255];
var curveStyle = 'fast';

// 预分配的缓冲区，避免每帧GC
var freqDataBuf = null;   // Float32Array for frequency data
var timeDataBuf = null;   // Uint8Array for time domain data
var barHeightsBuf = null; // Float64Array for normalized bar heights (0~1)
var barAmpDbBuf = null;   // Float64Array for raw ampDb values
var barColorsBuf = null;  // Array for bar colors
var pointsBuf = null;     // Array for curve points

// 缓存的canvas尺寸，避免每帧重算
var cachedCanvasW = 0, cachedCanvasH = 0, cachedDpr = 1;

// 缓存的bin范围，避免每帧重复log/exp计算
var cachedBinRanges = null;
var cachedBinRangeKey = '';

// 分析器是否接入音频图（仅可视化需要；非视觉视图/暂停时断开以省FFT计算）
var analyserEnabled = false;

// 画布父容器缓存的CSS尺寸（由ResizeObserver更新，避免每帧触发布局）
var cachedViewW = 0, cachedViewH = 0;

// 频谱曲线渐变缓存
var cachedCurveGrad = null;
var cachedCurveGradKey = '';

function ensureBuffers() {
  if (!analyser) return;
  var bins = analyser.frequencyBinCount;
  if (!freqDataBuf || freqDataBuf.length !== bins) {
    freqDataBuf = new Float32Array(bins);
  }
  if (!timeDataBuf || timeDataBuf.length !== analyser.fftSize) {
    timeDataBuf = new Uint8Array(analyser.fftSize);
  }
  var numBars = spectrumSettings.barCount;
  if (!barHeightsBuf || barHeightsBuf.length !== numBars) {
    barHeightsBuf = new Float64Array(numBars);
  }
  if (!barAmpDbBuf || barAmpDbBuf.length !== numBars) {
    barAmpDbBuf = new Float64Array(numBars);
  }
  if (!barColorsBuf || barColorsBuf.length !== numBars) {
    barColorsBuf = new Array(numBars);
  }
  if (!pointsBuf || pointsBuf.length !== numBars) {
    pointsBuf = new Array(numBars);
    for (var i = 0; i < numBars; i++) pointsBuf[i] = {x:0, y:0};
  }
}

function buildBinRanges(numBars, minFreq, maxFreq, binRes) {
  var key = numBars+'_'+minFreq+'_'+maxFreq+'_'+binRes;
  if (cachedBinRangeKey === key && cachedBinRanges) return cachedBinRanges;
  var logMin = Math.log(minFreq), logMax = Math.log(maxFreq);
  var ranges = new Array(numBars);
  for (var i = 0; i < numBars; i++) {
    var fStart = Math.exp(logMin + (i / numBars) * (logMax - logMin));
    var fEnd = Math.exp(logMin + ((i + 1) / numBars) * (logMax - logMin));
    var bStart = Math.max(0, Math.floor(fStart / binRes));
    var bEnd = Math.ceil(fEnd / binRes);
    if (bEnd <= bStart) bEnd = bStart + 1;
    ranges[i] = {start: bStart, end: bEnd};
  }
  cachedBinRanges = ranges;
  cachedBinRangeKey = key;
  return ranges;
}

function setAnalyserEnabled(on) {
  if (!audioSource || !analyser) return;
  if (on === analyserEnabled) return;
  analyserEnabled = on;
  if (on) {
    // 接入可视化分析分支：source -> analyser -> destination
    audioSource.disconnect();
    audioSource.connect(analyser);
    analyser.connect(audioCtx.destination);
  } else {
    // 切回直连：source -> destination
    // analyser 从图中移除，不再被拉取，生成的FFT计算开销归零，同时保证声音仍正常输出
    analyser.disconnect();
    audioSource.disconnect();
    audioSource.connect(audioCtx.destination);
  }
}

function initAudioAnalyser() {
  if (audioCtx) return;
  audioCtx = new (window.AudioContext || window.webkitAudioContext)();
  analyser = audioCtx.createAnalyser();
  analyser.fftSize = spectrumSettings.fftSize;
  analyser.smoothingTimeConstant = 0.7;
  analyser.minDecibels = -120;
  analyser.maxDecibels = 0;
  audioSource = audioCtx.createMediaElementSource(audio);
  // 默认可视化视图，走分析分支；非视觉视图/暂停时由setView/事件自动断开
  setAnalyserEnabled(true);
  // 初始化动态范围上限
  defaultTopDb = spectrumSettings.topDb;
  dynamicTopDb = defaultTopDb;
}

function pauseAnalyserForIdle() { setAnalyserEnabled(false); }
function resumeAnalyserForPlay() { setAnalyserEnabled(currentView !== 'lyrics'); }

function userSetView(view) {
  userSelectedView = true;
  setView(view);
}

function setView(view) {
  currentView = view;
  var isLyrics = view === 'lyrics';
  var isVisual = view !== 'lyrics';
  var isSpectrumBar = view === 'spectrumbar';
  var isSpectrumCurve = view === 'spectrumcurve';
  var isSpectrum = isSpectrumBar || isSpectrumCurve;

  document.getElementById('lyricsContainer').style.display = (isLyrics && !isFullscreen) ? '' : 'none';
  document.getElementById('visualContainer').style.display = (isVisual && !isFullscreen) ? '' : 'none';
  document.getElementById('fsLyricsContainer').style.display = (isLyrics && isFullscreen) ? '' : 'none';
  document.getElementById('fsVisualContainer').style.display = (isVisual && isFullscreen) ? '' : 'none';

  var lyricsBtns = ['btnLyrics','btnLyricsFs'];
  var waveBtns = ['btnWaveform','btnWaveformFs'];
  var specBarBtns = ['btnSpectrumBar','btnSpectrumBarFs'];
  var specCurveBtns = ['btnSpectrumCurve','btnSpectrumCurveFs'];
  lyricsBtns.forEach(function(id){var e=document.getElementById(id);if(e)e.classList.toggle('active',view==='lyrics');});
  waveBtns.forEach(function(id){var e=document.getElementById(id);if(e)e.classList.toggle('active',view==='waveform');});
  specBarBtns.forEach(function(id){var e=document.getElementById(id);if(e)e.classList.toggle('active',isSpectrumBar);});
  specCurveBtns.forEach(function(id){var e=document.getElementById(id);if(e)e.classList.toggle('active',isSpectrumCurve);});

  // Show/hide context controls based on view
  ['selWaveformTime','selWaveformTimeFs'].forEach(function(id){var e=document.getElementById(id);if(e)e.style.display=view==='waveform'?'':'none';});
  ['selSpectrumBars','selSpectrumBarsFs'].forEach(function(id){var e=document.getElementById(id);if(e)e.style.display=isSpectrum?'':'none';});
  ['btnPeakColor','btnPeakColorFs'].forEach(function(id){var e=document.getElementById(id);if(e)e.style.display=isSpectrumBar?'':'none';});
  ['selSpectrumColor','selSpectrumColorFs'].forEach(function(id){var e=document.getElementById(id);if(e)e.style.display=isSpectrum?'':'none';});
  ['customSpectrumColor','customSpectrumColorFs'].forEach(function(id){var e=document.getElementById(id);if(e){var selId=id.replace('custom','sel');var sel=document.getElementById(selId);e.style.display=(isSpectrum&&sel&&sel.value==='custom')?'':'none';}});
  ['selCurveStyle','selCurveStyleFs'].forEach(function(id){var e=document.getElementById(id);if(e)e.style.display=isSpectrumCurve?'':'none';});
  ['selTheme','selThemeFs'].forEach(function(id){var e=document.getElementById(id);if(e)e.style.display=view==='lyrics'?'':'none';});
  ['selTransition','selTransitionFs'].forEach(function(id){var e=document.getElementById(id);if(e)e.style.display=view==='lyrics'?'':'none';});

  // 切换到波形视图时重置帧时间，避免dt过大导致推入大量重复数据
  if (view === 'waveform') {
    lastWaveformFrameTime = 0;
  }

  setAnalyserEnabled(isVisual);

  // 启动绘制前必须先取消已有循环：drawFrame 内部会递归 rAF，若直接重复调用会导致
  // 多条 rAF 链叠加（帧率虚高 120/240/360fps、GPU/CPU 重复烧）——见实测复现
  if (isVisual && analyser) { stopAnim(); drawFrame(); }
  else { stopAnim(); }

  if (isLyrics && hasLyrics) {
    var container = isFullscreen ? document.getElementById('fsLyricsContainer') : document.getElementById('lyricsContainer');
    var lines = container.querySelectorAll('.lyrics-line');
    if (lines[currentLyricIndex]) {
      var lineTop = lines[currentLyricIndex].offsetTop - (isFullscreen ? document.getElementById('fsLyrics') : document.getElementById('lyrics')).offsetTop;
      container.scrollTo({top: lineTop - container.clientHeight/2 + 20, behavior:'smooth'});
    }
  }
}

function setTheme(theme) {
  currentTheme = theme;
  ['lyricsContainer','fsLyricsContainer'].forEach(function(id){
    var el = document.getElementById(id);
    el.className = el.className.replace(/theme-\S+/g, '');
    el.classList.add('theme-'+theme);
  });
  ['selTheme','selThemeFs'].forEach(function(id){var e=document.getElementById(id);if(e)e.value=theme;});
}

function setTransition(trans) {
  currentTransition = trans;
  ['lyricsContainer','fsLyricsContainer'].forEach(function(id){
    var el = document.getElementById(id);
    el.className = el.className.replace(/trans-\S+/g, '');
    el.classList.add('trans-'+trans);
  });
  ['selTransition','selTransitionFs'].forEach(function(id){var e=document.getElementById(id);if(e)e.value=trans;});
}

var lastWaveformFrameTime = 0; // 上一帧的时间戳

function setWaveformTime(val) {
  waveformTimeMs = parseInt(val);
  // 重置环形缓冲区
  waveRingLen = 0; waveRingHead = 0;
  ['selWaveformTime','selWaveformTimeFs'].forEach(function(id){var e=document.getElementById(id);if(e)e.value=val;});
}

function setSpectrumBars(val) {
  spectrumSettings.barCount = parseInt(val);
  spectrumPeakData = []; spectrumPeakTimes = [];
  cachedBinRangeKey = ''; // 使bin范围缓存失效
  // 根据条数调整fftSize以提供足够的频率分辨率
  if (analyser) {
    var needed = spectrumSettings.barCount * 8; // 每条至少8个bin
    var fs = 2048;
    while (fs < needed * 2 && fs < 16384) fs *= 2;
    if (fs !== analyser.fftSize) analyser.fftSize = fs;
  }
  ['selSpectrumBars','selSpectrumBarsFs'].forEach(function(id){var e=document.getElementById(id);if(e)e.value=val;});
}

function togglePeakColoring() {
  peakColoring = !peakColoring;
  ['btnPeakColor','btnPeakColorFs'].forEach(function(id){var e=document.getElementById(id);if(e)e.classList.toggle('active',peakColoring);});
}

function setSpectrumColor(val) {
  if (val === 'custom') {
    ['customSpectrumColor','customSpectrumColorFs'].forEach(function(id){var e=document.getElementById(id);if(e)e.style.display='';});
    // trigger custom color from current input value
    var inp = document.getElementById('customSpectrumColor');
    if (inp) setSpectrumColorCustom(inp.value);
    return;
  }
  spectrumColor = val.split(',').map(Number);
  ['customSpectrumColor','customSpectrumColorFs'].forEach(function(id){var e=document.getElementById(id);if(e)e.style.display='none';});
  ['selSpectrumColor','selSpectrumColorFs'].forEach(function(id){var e=document.getElementById(id);if(e)e.value=val;});
}

function setSpectrumColorCustom(hex) {
  var r = parseInt(hex.substr(1,2),16);
  var g = parseInt(hex.substr(3,2),16);
  var b = parseInt(hex.substr(5,2),16);
  spectrumColor = [r, g, b];
  ['customSpectrumColor','customSpectrumColorFs'].forEach(function(id){var e=document.getElementById(id);if(e)e.value=hex;});
  ['selSpectrumColor','selSpectrumColorFs'].forEach(function(id){var e=document.getElementById(id);if(e)e.value='custom';});
}

function setCurveStyle(val) {
  curveStyle = val;
  ['selCurveStyle','selCurveStyleFs'].forEach(function(id){var e=document.getElementById(id);if(e)e.value=val;});
}

function measureCanvasSize() {
  var canvas = isFullscreen ? document.getElementById('fullscreenCanvas') : document.getElementById('mainCanvas');
  if (!canvas || !canvas.parentElement) return;
  var rect = canvas.parentElement.getBoundingClientRect();
  cachedViewW = Math.round(rect.width);
  cachedViewH = Math.round(rect.height);
  // 强制下一帧重建 backing（尺寸已变）
  cachedCanvasW = 0;
}

function sizeCanvas(canvas) {
  var dpr = window.devicePixelRatio || 1;
  // 性能优化：HiDPI/4K 屏幕下 dpr 可能=2/3，导致 canvas backing 像素总量爆炸（GPU 杀手）
  // 限制 dpr 上限为 1.5，并在总像素超过阈值时进一步降采样
  if (dpr > 1.5) dpr = 1.5;
  // 用缓存的CSS尺寸（由ResizeObserver维护），避免每帧 getBoundingClientRect 触发同步布局
  if (!cachedViewW || !cachedViewH) measureCanvasSize();
  var w = cachedViewW, h = cachedViewH;
  // 仅在尺寸变化时更新canvas，避免每帧触发重绘
  var pw = Math.round(w * dpr), ph = Math.round(h * dpr);
  // 压低可视化内部分辨率上限（960×540 ≈ 约 51.8万像素）：
  // 波形/频谱是矢量条与曲线，低分辨率下显示仍清晰，栅格化开销明显降低
  var maxPixels = 960 * 540;
  if (pw * ph > maxPixels) {
    var scale = Math.sqrt(maxPixels / (pw * ph));
    pw = Math.max(1, Math.round(pw * scale));
    ph = Math.max(1, Math.round(ph * scale));
    dpr = pw / w;  // 实际生效 dpr
  }
  if (canvas.width !== pw || canvas.height !== ph || cachedCanvasW !== w || cachedCanvasH !== h) {
    canvas.width = pw;
    canvas.height = ph;
    cachedCanvasW = w; cachedCanvasH = h; cachedDpr = dpr;
    spectrumPeakData = []; spectrumPeakTimes = [];
  }
  return {w: w, h: h, dpr: dpr};
}

// FPS 帧率显示（左上角观测用）：利用 rAF 自带时间戳，每0.5秒更新，按帧率分级着色
// 同时更新普通窗口与全屏两个显示位（全屏元素位于 fullscreenOverlay 内，body 下的会被全屏层遮挡）
var fpsFrames = 0, fpsLastTime = 0, fpsEl = null, fpsElFs = null;
function updateFps(timestamp) {
  if (!fpsEl) fpsEl = document.getElementById('fpsDisplay');
  if (!fpsElFs) fpsElFs = document.getElementById('fpsDisplayFs');
  fpsFrames++;
  if (timestamp - fpsLastTime >= 500) {
    var fps = Math.round(fpsFrames * 1000 / (timestamp - fpsLastTime));
    var color = fps >= 50 ? '#4caf50' : (fps >= 30 ? '#ffc107' : '#f44336');
    var txt = 'FPS: ' + fps;
    if (fpsEl) { fpsEl.textContent = txt; fpsEl.style.color = color; }
    if (fpsElFs) { fpsElFs.textContent = txt; fpsElFs.style.color = color; }
    fpsFrames = 0;
    fpsLastTime = timestamp;
  }
}

function drawFrame(timestamp) {
  if (!analyser) { animId = requestAnimationFrame(drawFrame); return; }
  ensureBuffers();
  var canvas = isFullscreen ? document.getElementById('fullscreenCanvas') : document.getElementById('mainCanvas');
  var info = sizeCanvas(canvas);
  var ctx = canvas.getContext('2d');
  ctx.setTransform(info.dpr, 0, 0, info.dpr, 0, 0);
  ctx.clearRect(0, 0, info.w, info.h);
  if (currentView === 'waveform') drawWaveform(ctx, info.w, info.h);
  else if (currentView === 'spectrumbar') drawSpectrum(ctx, info.w, info.h);
  else if (currentView === 'spectrumcurve') drawSpectrumCurve(ctx, info.w, info.h);
  updateFps(timestamp || performance.now());
  animId = requestAnimationFrame(drawFrame);
}

// 波形环形缓冲区
var waveRingBuf = null;   // Uint8Array环形缓冲
var waveRingHead = 0;     // 写入位置
var waveRingLen = 0;      // 有效数据长度
var waveRingCap = 0;      // 容量

function ensureWaveRingBuf(cap) {
  if (waveRingCap >= cap) {
    // 容量足够，但如果当前有效数据超过新需求，截断
    if (waveRingLen > cap) waveRingLen = cap;
    return;
  }
  waveRingCap = cap;
  waveRingBuf = new Uint8Array(cap);
  waveRingHead = 0;
  waveRingLen = 0;
}

function waveRingPush(data, count) {
  for (var i = 0; i < count; i++) {
    waveRingBuf[waveRingHead] = data[i];
    waveRingHead = (waveRingHead + 1) % waveRingCap;
  }
  waveRingLen = Math.min(waveRingLen + count, waveRingCap);
}

function waveRingGet(idx) {
  var start = (waveRingHead - waveRingLen + idx + waveRingCap * 2) % waveRingCap;
  return waveRingBuf[start];
}

function drawWaveform(ctx, w, h) {
  var sampleRate = audioCtx.sampleRate;
  var samplesNeeded = Math.round(sampleRate * waveformTimeMs / 1000);

  // 从analyser获取当前帧的时域数据（使用预分配缓冲）
  analyser.getByteTimeDomainData(timeDataBuf);

  // 计算自上一帧以来新增的样本数，只追加新数据避免重复
  var now = performance.now();
  var dt = lastWaveformFrameTime ? (now - lastWaveformFrameTime) / 1000 : 0;
  lastWaveformFrameTime = now;
  var newSamples = Math.min(Math.round(dt * sampleRate), timeDataBuf.length);
  if (newSamples > 0) {
    ensureWaveRingBuf(samplesNeeded);
    // 只追加最新的newSamples个样本
    var start = timeDataBuf.length - newSamples;
    waveRingPush(timeDataBuf.subarray(start), newSamples);
  }

  var midY = h / 2;
  ctx.strokeStyle = 'rgba(0,170,255,0.2)';
  ctx.lineWidth = 1;
  ctx.beginPath(); ctx.moveTo(0, midY); ctx.lineTo(w, midY); ctx.stroke();

  ctx.lineWidth = 2;
  ctx.strokeStyle = '#00aaff';
  ctx.beginPath();
  var len = waveRingLen;
  if (len > 1) {
    // 降采样：样本数远超画布宽度时，每像素取该范围内 |v-128| 最大的样本作为代表值
    // lineTo 调用数从 len 降到 ~w（画布宽度），GPU 命令数大幅减少
    var displayPoints = Math.min(w, len);
    // 计算第一个样本的物理索引（避免循环内每样本都取模）
    var startPhys = (waveRingHead - waveRingLen + waveRingCap * 2) % waveRingCap;
    if (len > w * 2) {
      var samplesPerPixel = len / displayPoints;
      var xStep = w / (displayPoints - 1);
      for (var i = 0; i < displayPoints; i++) {
        var s = Math.floor(i * samplesPerPixel);
        var e = Math.min(Math.floor((i + 1) * samplesPerPixel), len);
        var maxDev = -1, repV = 128;
        for (var j = s; j < e; j++) {
          var pi = startPhys + j;
          if (pi >= waveRingCap) pi -= waveRingCap;
          var v = waveRingBuf[pi];
          var dev = v < 128 ? 128 - v : v - 128;
          if (dev > maxDev) { maxDev = dev; repV = v; }
        }
        var y = midY + (repV / 128.0 - 1.0) * midY * 0.9;
        if (i === 0) ctx.moveTo(0, y); else ctx.lineTo(i * xStep, y);
      }
    } else {
      var sliceW = w / (len - 1);
      for (var i = 0; i < len; i++) {
        var pi = startPhys + i;
        if (pi >= waveRingCap) pi -= waveRingCap;
        var v = waveRingBuf[pi] / 128.0;
        var y = midY + (v - 1.0) * midY * 0.9;
        if (i === 0) ctx.moveTo(0, y); else ctx.lineTo(i * sliceW, y);
      }
    }
  }
  ctx.stroke();
}

function lerpColor(c1, c2, t) {
  return [Math.round(c1[0]+(c2[0]-c1[0])*t), Math.round(c1[1]+(c2[1]-c1[1])*t), Math.round(c1[2]+(c2[2]-c1[2])*t)];
}

// 将频率值转为画布x坐标（对数刻度）
function freqToX(freq, w, minFreq, maxFreq) {
  return (Math.log(freq/minFreq) / Math.log(maxFreq/minFreq)) * w;
}

// 绘制频率刻度（底部）
function drawFreqScale(ctx, w, h, minFreq, maxFreq) {
  var scaleH = 16;
  var labels = [100, 200, 500, 1000, 2000, 5000, 10000];
  ctx.fillStyle = 'rgba(255,255,255,0.3)';
  ctx.font = '10px sans-serif';
  ctx.textAlign = 'center';
  for (var i = 0; i < labels.length; i++) {
    var f = labels[i];
    if (f < minFreq || f > maxFreq) continue;
    var x = freqToX(f, w, minFreq, maxFreq);
    // 刻度线
    ctx.strokeStyle = 'rgba(255,255,255,0.15)';
    ctx.beginPath(); ctx.moveTo(x, 0); ctx.lineTo(x, h - scaleH); ctx.stroke();
    // 标签
    var txt = f >= 1000 ? (f/1000) + 'k' : f + '';
    ctx.fillText(txt, x, h - 3);
  }
}

function drawSpectrum(ctx, w, h) {
  ctx.fillStyle = '#000';
  ctx.fillRect(0, 0, w, h);
  var scaleH = 16;
  var drawH = h - scaleH;

  // 使用预分配缓冲区
  analyser.getFloatFrequencyData(freqDataBuf);
  var sampleRate = audioCtx.sampleRate;
  var fftSize = analyser.fftSize;
  var binCount = freqDataBuf.length;
  var binRes = sampleRate / fftSize;
  var minFreq = 50, maxFreq = 15000;
  var numBars = spectrumSettings.barCount;
  var barW = w / numBars;
  var bottomDb = spectrumSettings.bottomDb;

  // 使用缓存的bin范围
  var ranges = buildBinRanges(numBars, minFreq, maxFreq, binRes);

  // 先扫描一遍计算ampDb并找到最大值
  var maxAmpDb = -Infinity;
  for (var i = 0; i < numBars; i++) {
    var range = (i === 0) ? {start:0, end:ranges[0].end} : ranges[i];
    var totalPow = 0, validBins = 0;
    for (var b = range.start; b <= range.end && b < binCount; b++) {
      var dbVal = freqDataBuf[b];
      if (dbVal > -200 && dbVal !== -Infinity) { totalPow += Math.pow(10, dbVal/10); validBins++; }
    }
    if (validBins > 0 && totalPow > 0) {
      var ampDb = 10 * Math.log10(totalPow);
      barAmpDbBuf[i] = ampDb;
      if (ampDb > maxAmpDb) maxAmpDb = ampDb;
    } else {
      barAmpDbBuf[i] = -Infinity;
    }
  }

  // 动态范围扩展：如果最大值超过当前上限，扩展上限
  if (maxAmpDb > dynamicTopDb) {
    dynamicTopDb = maxAmpDb + 3; // 留3dB余量
  }

  // 用动态topDb计算barHeightsBuf
  var dbRange = dynamicTopDb - bottomDb;
  for (var i = 0; i < numBars; i++) {
    if (barAmpDbBuf[i] > -Infinity) {
      barHeightsBuf[i] = (Math.max(bottomDb, Math.min(dynamicTopDb, barAmpDbBuf[i]))-bottomDb)/dbRange;
    } else {
      barHeightsBuf[i] = 0;
    }
  }

  // 计算颜色（恢复原始：每帧重算，峰值着色与颜色选择即时生效）
  if (peakColoring) {
    // 排名着色：先收集非低频条索引，按高度排序
    var rankedIdx = [];
    for (var i = 0; i < numBars; i++) {
      var fEnd = (i === 0) ? Math.exp(Math.log(minFreq)+(1/numBars)*(Math.log(maxFreq)-Math.log(minFreq))) : 0;
      if (fEnd <= 125 && i === 0) { barColorsBuf[i] = spectrumColor; continue; }
      // 简化：125Hz以下（约前2条）统一基底色
      var fE = Math.exp(Math.log(minFreq)+((i+1)/numBars)*(Math.log(maxFreq)-Math.log(minFreq)));
      if (fE <= 125) { barColorsBuf[i] = spectrumColor; continue; }
      rankedIdx.push(i);
    }
    // 按高度降序排列rankedIdx
    rankedIdx.sort(function(a,b){return barHeightsBuf[b]-barHeightsBuf[a];});
    var cY = spectrumSettings.colorTop1, cG = spectrumSettings.colorPct20, cB = spectrumColor;
    var total = rankedIdx.length;
    var p20 = Math.floor(total*(spectrumSettings.thresholdPct20/100));
    var p40 = Math.floor(total*(spectrumSettings.thresholdPct40/100));
    for (var r = 0; r < total; r++) {
      var idx = rankedIdx[r];
      var color;
      if (r === 0) color = cY;
      else if (r <= p20) color = lerpColor(cY, cG, r/(p20||1));
      else if (r <= p40) color = lerpColor(cG, cB, (r-p20)/((p40-p20)||1));
      else color = cB;
      barColorsBuf[idx] = color;
    }
  } else {
    for (var i = 0; i < numBars; i++) barColorsBuf[i] = spectrumColor;
  }

  // 绘制频谱条和峰值
  var now = performance.now();
  for (var i = 0; i < numBars; i++) {
    var bh = barHeightsBuf[i];
    var barH = bh * drawH;
    if (barH > 0) {
      var c = barColorsBuf[i];
      ctx.fillStyle = 'rgb('+c[0]+','+c[1]+','+c[2]+')';
      ctx.fillRect(i*barW, drawH-barH, barW, barH);
    }
    if (spectrumSettings.showPeaks && barAmpDbBuf[i] > -Infinity) {
      if (barAmpDbBuf[i] > (spectrumPeakData[i]||-Infinity)) { spectrumPeakData[i]=barAmpDbBuf[i]; spectrumPeakTimes[i]=now; }
      else if (spectrumPeakData[i]!==undefined) {
        if (now-spectrumPeakTimes[i] >= PEAK_HOLD_TIME_MS) {
          spectrumPeakData[i] -= PEAK_HOLD_DECAY_RATE*(1/60);
          if (spectrumPeakData[i]<bottomDb) { delete spectrumPeakData[i]; delete spectrumPeakTimes[i]; continue; }
        }
      }
      if (spectrumPeakData[i]!==undefined && spectrumPeakData[i]>bottomDb) {
        var pDb = Math.max(bottomDb, Math.min(dynamicTopDb, spectrumPeakData[i]));
        var pY = drawH - ((pDb-bottomDb)/dbRange)*drawH;
        if (pY < drawH-3) {
          var alpha = (now-spectrumPeakTimes[i])<PEAK_HOLD_TIME_MS ? 1 : 0.5;
          ctx.fillStyle = 'rgba(255,255,255,'+alpha+')';
          ctx.fillRect(i*barW, pY-1, barW, 2);
        }
      }
    }
  }
  drawFreqScale(ctx, w, h, minFreq, maxFreq);
  ctx.strokeStyle = '#30363d'; ctx.lineWidth = 1; ctx.strokeRect(0, 0, w, drawH);
}

function computeBarHeights(numBars, floatData, binRes, binCount, minFreq, maxFreq) {
  var topDb = spectrumSettings.topDb, bottomDb = spectrumSettings.bottomDb;
  var dbRange = topDb - bottomDb;
  var ranges = buildBinRanges(numBars, minFreq, maxFreq, binRes);
  for (var i = 0; i < numBars; i++) {
    var range = (i === 0) ? {start:0, end:ranges[0].end} : ranges[i];
    var totalPow = 0, validBins = 0;
    for (var b = range.start; b <= range.end && b < binCount; b++) {
      var dbVal = floatData[b];
      if (dbVal > -200 && dbVal !== -Infinity) { totalPow += Math.pow(10, dbVal/10); validBins++; }
    }
    if (validBins > 0 && totalPow > 0) {
      var ampDb = 10 * Math.log10(totalPow);
      barHeightsBuf[i] = (Math.max(bottomDb, Math.min(topDb, ampDb))-bottomDb)/dbRange;
    } else {
      barHeightsBuf[i] = 0;
    }
  }
}

function drawSpectrumCurve(ctx, w, h) {
  ctx.fillStyle = '#000';
  ctx.fillRect(0, 0, w, h);
  var scaleH = 16;
  var drawH = h - scaleH;

  analyser.getFloatFrequencyData(freqDataBuf);
  var sampleRate = audioCtx.sampleRate;
  var fftSize = analyser.fftSize;
  var binCount = freqDataBuf.length;
  var binRes = sampleRate / fftSize;
  var minFreq = 50, maxFreq = 15000;
  var numBars = spectrumSettings.barCount;
  computeBarHeights(numBars, freqDataBuf, binRes, binCount, minFreq, maxFreq);

  // 使用预分配的points数组
  for (var i = 0; i < numBars; i++) {
    pointsBuf[i].x = (i + 0.5) * (w / numBars);
    pointsBuf[i].y = drawH - barHeightsBuf[i] * drawH;
  }

  var colorStr = 'rgb('+spectrumColor[0]+','+spectrumColor[1]+','+spectrumColor[2]+')';

  // 绘制填充区域（曲线到底部）
  ctx.beginPath();
  if (curveStyle === 'smooth' && numBars > 2) {
    ctx.moveTo(pointsBuf[0].x, pointsBuf[0].y);
    for (var i = 0; i < numBars - 1; i++) {
      var cp1x = (pointsBuf[i].x + pointsBuf[i+1].x) / 2;
      var cp1y = pointsBuf[i].y;
      var cp2x = cp1x;
      var cp2y = pointsBuf[i+1].y;
      ctx.bezierCurveTo(cp1x, cp1y, cp2x, cp2y, pointsBuf[i+1].x, pointsBuf[i+1].y);
    }
  } else {
    ctx.moveTo(pointsBuf[0].x, pointsBuf[0].y);
    for (var i = 1; i < numBars; i++) {
      ctx.lineTo(pointsBuf[i].x, pointsBuf[i].y);
    }
  }
  ctx.lineTo(pointsBuf[numBars-1].x, drawH);
  ctx.lineTo(pointsBuf[0].x, drawH);
  ctx.closePath();
  var gradKey = drawH + '|' + spectrumColor[0]+','+spectrumColor[1]+','+spectrumColor[2];
  if (!cachedCurveGrad || cachedCurveGradKey !== gradKey) {
    cachedCurveGrad = ctx.createLinearGradient(0, 0, 0, drawH);
    var r = spectrumColor[0], g = spectrumColor[1], b = spectrumColor[2];
    cachedCurveGrad.addColorStop(0, 'rgba('+r+','+g+','+b+',0.4)');
    cachedCurveGrad.addColorStop(1, 'rgba('+r+','+g+','+b+',0.02)');
    cachedCurveGradKey = gradKey;
  }
  ctx.fillStyle = cachedCurveGrad;
  ctx.fill();

  // 绘制曲线线条
  ctx.beginPath();
  if (curveStyle === 'smooth' && numBars > 2) {
    ctx.moveTo(pointsBuf[0].x, pointsBuf[0].y);
    for (var i = 0; i < numBars - 1; i++) {
      var cp1x = (pointsBuf[i].x + pointsBuf[i+1].x) / 2;
      var cp1y = pointsBuf[i].y;
      var cp2x = cp1x;
      var cp2y = pointsBuf[i+1].y;
      ctx.bezierCurveTo(cp1x, cp1y, cp2x, cp2y, pointsBuf[i+1].x, pointsBuf[i+1].y);
    }
  } else {
    ctx.moveTo(pointsBuf[0].x, pointsBuf[0].y);
    for (var i = 1; i < numBars; i++) {
      ctx.lineTo(pointsBuf[i].x, pointsBuf[i].y);
    }
  }
  ctx.strokeStyle = colorStr;
  ctx.lineWidth = 2;
  ctx.stroke();

  drawFreqScale(ctx, w, h, minFreq, maxFreq);
  ctx.strokeStyle = '#30363d'; ctx.lineWidth = 1; ctx.strokeRect(0, 0, w, drawH);
}

function stopAnim() { if(animId){cancelAnimationFrame(animId);animId=null;} }

function syncLyricsToFullscreen() {
  var fsLyrics = document.getElementById('fsLyrics');
  var mainLyrics = document.getElementById('lyrics');
  fsLyrics.innerHTML = mainLyrics.innerHTML;
  var fsContainer = document.getElementById('fsLyricsContainer');
  fsContainer.className = document.getElementById('lyricsContainer').className;
  var lines = fsContainer.querySelectorAll('.lyrics-line');
  if (lines[currentLyricIndex]) {
    var lineTop = lines[currentLyricIndex].offsetTop - fsLyrics.offsetTop;
    fsContainer.scrollTop = lineTop - fsContainer.clientHeight/2 + 20;
  }
}

function toggleFullscreen() {
  if (!isFullscreen) {
    isFullscreen = true;
    document.getElementById('fullscreenOverlay').style.display = 'flex';
    if (currentView === 'lyrics' && hasLyrics) syncLyricsToFullscreen();
    measureCanvasSize();
    setView(currentView);
    setTimeout(autoScaleLyricsFont, 50);
    try { document.getElementById('fullscreenOverlay').requestFullscreen(); } catch(e){}
    spectrumPeakData = []; spectrumPeakTimes = [];
  } else {
    isFullscreen = false;
    document.getElementById('fullscreenOverlay').style.display = 'none';
    setView(currentView);
    setTimeout(autoScaleLyricsFont, 50);
    if (document.fullscreenElement) try{document.exitFullscreen();}catch(e){}
    spectrumPeakData = []; spectrumPeakTimes = [];
  }
}

document.addEventListener('fullscreenchange', function() {
  if (!document.fullscreenElement && isFullscreen) {
    isFullscreen = false;
    document.getElementById('fullscreenOverlay').style.display = 'none';
    setView(currentView);
  }
});

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

function loadLyrics(songName) {
  lyrics = []; currentLyricIndex = 0; hasLyrics = false;
  document.getElementById("lyrics").innerHTML = '';
  document.getElementById("fsLyrics").innerHTML = '';
  // 用户未手动选择过视图时，切歌先回退到波形
  if (!userSelectedView) {
    setView('waveform');
  }
  var lrcUrl = (currentFilePath || songName).replace(/\.[^.]+$/, '.lrc');
  var xhr = new XMLHttpRequest();
  xhr.open('GET', '/file?name=' + encodeURIComponent(lrcUrl), true);
  xhr.responseType = 'arraybuffer';
  xhr.onload = function() {
    if (xhr.status === 200) {
      lyrics = parseLRC(decodeBuffer(new Uint8Array(xhr.response)));
      if (lyrics.length > 0) {
        hasLyrics=true; renderLyrics();
        if (!userSelectedView) setView('lyrics');
      } else loadEmbeddedLyrics(songName);
    } else loadEmbeddedLyrics(songName);
  };
  xhr.onerror = function() { loadEmbeddedLyrics(songName); };
  xhr.send();
}

function loadEmbeddedLyrics(songName) {
  var xhr = new XMLHttpRequest();
  xhr.open('GET', '/api/lyrics?fileName=' + encodeURIComponent(songName), true);
  xhr.responseType = 'arraybuffer';
  xhr.onload = function() {
    if (xhr.status === 200) {
      lyrics = parseLRC(decodeBuffer(new Uint8Array(xhr.response)));
      if (lyrics.length > 0) {
        hasLyrics=true; renderLyrics();
        if (!userSelectedView) setView('lyrics');
      } else {
        if (!userSelectedView) setView('waveform');
      }
    } else {
      if (!userSelectedView) setView('waveform');
    }
  };
  xhr.onerror = function() {
    if (!userSelectedView) setView('waveform');
  };
  xhr.send();
}

function renderLyrics() {
  ['lyrics','fsLyrics'].forEach(function(elId) {
    var el = document.getElementById(elId);
    el.innerHTML = '';
    var top = document.createElement('div'); top.style.height = '80px'; el.appendChild(top);
    lyrics.forEach(function(lyric) {
      var line = document.createElement('div');
      line.className = 'lyrics-line';
      line.textContent = lyric.text;
      el.appendChild(line);
    });
    var bot = document.createElement('div'); bot.style.height = '80px'; el.appendChild(bot);
  });
  autoScaleLyricsFont();
  setTimeout(function() {
    document.getElementById('lyricsContainer').scrollTop = 0;
    document.getElementById('fsLyricsContainer').scrollTop = 0;
  }, 100);
}

function autoScaleLyricsFont() {
  if (lyrics.length === 0) return;
  var longestText = '';
  lyrics.forEach(function(l){ if(l.text.length > longestText.length) longestText = l.text; });
  if (!longestText) return;

  var containers = [
    {container: document.getElementById('lyricsContainer'), isFs: false},
    {container: document.getElementById('fsLyricsContainer'), isFs: true}
  ];

  containers.forEach(function(item) {
    var container = item.container;
    var targetWidth = container.clientWidth * 0.8;
    if (targetWidth <= 0) return;

    var canvas = document.createElement('canvas');
    var ctx = canvas.getContext('2d');

    var minBase = 14;
    var maxBase = item.isFs ? 72 : 56;
    var activeRatio = 1.33;

    var bestBase = minBase;
    for (var fs = maxBase; fs >= minBase; fs--) {
      var activeFs = Math.round(fs * activeRatio);
      ctx.font = 'bold ' + activeFs + 'px Microsoft YaHei,sans-serif';
      if (ctx.measureText(longestText).width <= targetWidth) {
        bestBase = fs;
        break;
      }
    }

    var bestActive = Math.round(bestBase * activeRatio);
    var styleId = item.isFs ? 'fsLyricsScaleStyle' : 'lyricsScaleStyle';
    var existing = document.getElementById(styleId);
    if (existing) existing.remove();
    var style = document.createElement('style');
    style.id = styleId;
    style.textContent = '#' + container.id + ' .lyrics-line{font-size:' + bestBase + 'px} #' + container.id + ' .lyrics-line.active{font-size:' + bestActive + 'px}';
    document.head.appendChild(style);
  });
}

function updateLyrics() {
  if (!audio || lyrics.length === 0) return;
  var ct = audio.currentTime, newIdx = currentLyricIndex;
  for (var i = 0; i < lyrics.length; i++) { if (lyrics[i].time <= ct) newIdx = i; else break; }
  if (newIdx !== currentLyricIndex) {
    ['lyricsContainer','fsLyricsContainer'].forEach(function(containerId) {
      var container = document.getElementById(containerId);
      var lines = container.querySelectorAll('.lyrics-line');
      if (lines[currentLyricIndex]) lines[currentLyricIndex].classList.remove('active');
      if (lines[newIdx]) {
        lines[newIdx].classList.add('active');
        var lyricsInner = container.querySelector('div');
        var lineTop = lines[newIdx].offsetTop - (lyricsInner ? lyricsInner.offsetTop : 0);
        container.scrollTo({ top: lineTop - container.clientHeight/2 + 20, behavior: 'smooth' });
      }
    });
    currentLyricIndex = newIdx;
  }
}

function formatTime(s) { if(isNaN(s))return '0:00'; var m=Math.floor(s/60),sec=Math.floor(s%60); return m+':'+(sec<10?'0':'')+sec; }

function seekAudio(event) {
  if (!audio || !isFinite(audio.duration) || audio.duration <= 0) return;
  var bar = document.getElementById('progressBar');
  var rect = bar.getBoundingClientRect();
  audio.currentTime = ((event.clientX - rect.left) / rect.width) * audio.duration;
}

function setVolume(v) { if(audio) audio.volume = v/100; }

function playAudio(url, name, path) {
  currentFileName = name;
  currentFilePath = path || '';
  document.getElementById('songTitle').textContent = name;
  if (!audioCtx) initAudioAnalyser();
  // 切歌时恢复默认动态上限
  dynamicTopDb = defaultTopDb;
  audio.src = url + '?t=' + Date.now();
  audio.volume = document.getElementById('volumeSlider').value / 100;
  audio.play().catch(function(){});
  if (audioCtx && audioCtx.state === 'suspended') audioCtx.resume();

  // 自动切歌后必须重启可视化绘制循环：上首播完的 onended 会 stopAnim()，
  // 若此处不重启，波形/频谱画布会一直空白（手动切视图 setView 会拉起，故只有自动切歌出现）。
  if (currentView !== 'lyrics' && analyser) { stopAnim(); drawFrame(); }

  document.getElementById('toolbar').style.display = 'flex';

  audio.ontimeupdate = function() {
    document.getElementById('progressFill').style.width = '0%';
    document.getElementById('currentTime').textContent = formatTime(audio.currentTime);
    if (isFinite(audio.duration) && audio.duration > 0) {
      document.getElementById('progressFill').style.width = (audio.currentTime/audio.duration*100)+'%';
      document.getElementById('totalTime').textContent = formatTime(audio.duration);
    } else {
      document.getElementById('totalTime').textContent = '直播流';
    }
    updateLyrics();
  };
  audio.onended = function() { stopAnim(); if(window.opener&&!window.opener.closed) window.opener.postMessage({action:"ended"},"*"); else if(window.parent&&window.parent!==window) window.parent.postMessage({action:"ended"},"*"); };
  audio.onerror = function() { stopAnim(); };

  showTip("正在播放：" + name);
  loadLyrics(name);
  document.getElementById('playInfo').style.display = 'block';
  document.getElementById('currentSong').textContent = '正在播放：' + name;
  updateNextSongDisplay();
  setTimeout(function(){ document.getElementById('playInfo').style.display='none'; }, 10000);
}

function updateNextSongDisplay() {
  var nextItem = null;
  if (currentPlayingIndex >= 0 && currentPlayingIndex < currentQueue.length - 1) {
    for (var i = currentPlayingIndex + 1; i < currentQueue.length; i++) {
      if (currentQueue[i].status === "ready") { nextItem = currentQueue[i]; break; }
    }
  }
  document.getElementById('nextSong').textContent = nextItem ? '下一首：' + nextItem.name : '下一首：暂无';
}

// 暂停时断开分析器以省FFT计算；恢复播放时按当前视图决定是否重连（非歌词视图才需要）
audio.addEventListener('pause', function(){ pauseAnalyserForIdle(); });
audio.addEventListener('play', function(){ resumeAnalyserForPlay(); });

// 监听画布父容器尺寸变化（仅在实际变化时重新测量，替代每帧 getBoundingClientRect）
measureCanvasSize();
[document.getElementById('mainCanvas'), document.getElementById('fullscreenCanvas')]
  .forEach(function(canvas){ if (canvas && canvas.parentElement) {
    if (window.ResizeObserver) {
      new ResizeObserver(function(){ measureCanvasSize(); }).observe(canvas.parentElement);
    } else {
      window.addEventListener('resize', function(){ measureCanvasSize(); });
    }
  }});

window.addEventListener("message", function(e) {
  if (e.data.action === "play") playAudio(e.data.url, e.data.name, e.data.path);
  else if (e.data.action === "syncQueue") { currentQueue=e.data.list; currentPlayingIndex=e.data.currentPlayingIndex!==undefined?e.data.currentPlayingIndex:-1; updateNextSongDisplay(); }
  else if (e.data.action === "stopAnim") { stopAnim(); }  // 父窗口切到视频模式时通知停止可视化
});

document.onkeydown = function(e) {
  if (e.key === "F11") { e.preventDefault(); toggleFullscreen(); }
  else if (e.key === "Escape" && isFullscreen) { toggleFullscreen(); }
  else if (e.key === " " && e.target === document.body) { e.preventDefault(); if(audio){if(audio.paused)audio.play();else audio.pause();} }
};

window.addEventListener('resize', function() { spectrumPeakData=[]; spectrumPeakTimes=[]; autoScaleLyricsFont(); });

// 启动时检查 URL 参数，自动播放（主页面切歌时通过 URL 参数携带播放信息）
window.addEventListener('load', function() {
  var urlParams = new URLSearchParams(window.location.search);
  var autoPlayUrl = urlParams.get('playUrl');
  if (autoPlayUrl) {
    var autoPlayName = urlParams.get('playName') || '';
    var autoPlayPath = urlParams.get('playPath') || '';
    // 清除 URL 参数，避免刷新时重复播放
    history.replaceState(null, '', location.pathname);
    setTimeout(function() {
      playAudio(autoPlayUrl, autoPlayName, autoPlayPath);
    }, 100);
  }
});
</script>
</body>
</html>
`))
}
