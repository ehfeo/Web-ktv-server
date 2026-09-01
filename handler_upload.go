package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

var allowedExtensions = []string{".mp3", ".wav", ".flac", ".aac", ".m4a", ".m4r", ".alac", ".ogg", ".oga", ".opus", ".wma", ".ape", ".aiff", ".aif", ".amr", ".dvf", ".msv", ".dts", ".dff", ".dsf", ".sacd", ".tak", ".tta", ".wv", ".mka", ".mp4", ".mkv", ".avi", ".mov", ".wmv", ".rm", ".rmvb", ".ts", ".m2ts", ".mts", ".m2t", ".3gp", ".3g2", ".m4v", ".vob", ".ogv", ".asf", ".divx", ".f4v", ".mxf", ".wtv", ".webm", ".mpg", ".mpeg", ".flv"}

func UploadPageHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(`
<!DOCTYPE html>
<html>
<head>
<meta charset="UTF-8">
<title>上传歌曲</title>
<link rel="icon" href="/favicon.ico">
<style>
*{margin:0;padding:0;box-sizing:border-box}
body{width:100vw;height:100vh;background:#1a252f;display:flex;flex-direction:column;align-items:center;font-family:Microsoft YaHei,sans-serif;color:#ecf0f1;padding:30px}
h1{font-size:26px;margin-bottom:20px;color:#337ab9;letter-spacing:2px;font-weight:700}
.upload-area{width:100%;max-width:600px;border:2px dashed #428bca;border-radius:8px;padding:40px 20px;text-align:center;cursor:pointer;transition:all 0.2s;margin-bottom:20px;background:rgba(66,139,202,0.04);box-shadow:0 1px 4px rgba(0,0,0,0.15)}
.upload-area:hover,.upload-area.dragover{border-color:#337ab9;background:rgba(66,139,202,0.08);box-shadow:0 2px 8px rgba(0,0,0,0.2)}
.upload-area .icon{font-size:52px;margin-bottom:10px;opacity:0.7}
.upload-area .text{font-size:17px;color:#ecf0f1;font-weight:600}
.upload-area .hint{font-size:12px;color:#95a5a6;margin-top:8px}
.file-input-wrap{width:100%;max-width:600px;margin-bottom:15px;text-align:center}
.file-input-wrap input[type="file"]{color:#95a5a6;font-size:14px}
.file-input-wrap label{display:inline-block;padding:10px 28px;background:#428bca;border-radius:6px;color:#fff;cursor:pointer;font-size:15px;font-weight:600;letter-spacing:0.5px;box-shadow:0 2px 4px rgba(0,0,0,0.15);transition:all 0.15s;border:none}
.file-input-wrap label:hover{background:#337ab9;box-shadow:0 2px 6px rgba(0,0,0,0.2)}
.file-input-wrap label:active{background:#2a6496;box-shadow:0 1px 2px rgba(0,0,0,0.15)}
.file-list{width:100%;max-width:600px;flex:1;overflow-y:auto;margin-bottom:15px}
.file-item{display:flex;align-items:center;gap:10px;padding:10px 14px;background:#243342;border-radius:6px;margin-bottom:6px;font-size:14px;box-shadow:0 1px 3px rgba(0,0,0,0.12);border:1px solid #2f4254;transition:all 0.15s}
.file-item:hover{border-color:#428bca;box-shadow:0 2px 6px rgba(0,0,0,0.18)}
.file-item .name{flex:1;overflow:hidden;text-overflow:ellipsis;white-space:nowrap;color:#ecf0f1}
.file-item .size{color:#95a5a6;font-size:12px;white-space:nowrap}
.file-item .remove{cursor:pointer;color:#d9534f;font-size:20px;padding:0 4px;transition:all 0.15s;font-weight:700}
.file-item .remove:hover{color:#c9302c}
.file-item.uploading .name{color:#428bca}
.file-item.done .name{color:#5cb85c}
.file-item.error .name{color:#d9534f}
.progress-wrap{width:100%;max-width:600px;margin-bottom:6px}
.progress-bar{height:5px;background:#2f4254;border-radius:3px;overflow:hidden}
.progress-bar .fill{height:100%;background:#428bca;border-radius:3px;width:0%;transition:width 0.2s}
.btn-row{display:flex;gap:12px;width:100%;max-width:600px}
button{padding:12px 28px;border:none;border-radius:6px;font-size:15px;cursor:pointer;font-family:inherit;font-weight:600;letter-spacing:0.5px;transition:all 0.15s}
.btn-upload{background:#5cb85c;color:#fff;flex:1;box-shadow:0 2px 4px rgba(0,0,0,0.15);border:none}
.btn-upload:hover{background:#449d44;box-shadow:0 2px 6px rgba(0,0,0,0.2)}
.btn-upload:active{background:#398439;box-shadow:0 1px 2px rgba(0,0,0,0.15)}
.btn-upload:disabled{background:#3a4a56;color:#6a7a86;cursor:not-allowed;box-shadow:none;transform:none}
.btn-clear{background:#428bca;color:#fff;box-shadow:0 2px 4px rgba(0,0,0,0.15);border:none}
.btn-clear:hover{background:#337ab9;box-shadow:0 2px 6px rgba(0,0,0,0.2)}
.btn-clear:active{background:#2a6496;box-shadow:0 1px 2px rgba(0,0,0,0.15)}
.status{margin-top:10px;font-size:13px;color:#95a5a6;text-align:center}
</style>
</head>
<body>
<h1>上传歌曲</h1>
<div class="upload-area" id="dropZone">
  <div class="icon">&#128193;</div>
  <div class="text">拖拽文件到此处</div>
  <div class="hint">支持格式：MP3, WAV, FLAC, AAC, M4A, OGG, WMA, MP4, MKV, AVI, MOV, WMV, RM, RMVB, TS, WEBM, MPG, MPEG, FLV</div>
</div>
<div class="file-input-wrap">
  <label for="fileInput">选择文件</label>
  <input type="file" id="fileInput" multiple>
</div>
<div class="file-list" id="fileList"></div>
<div class="progress-wrap" id="progressWrap" style="display:none">
  <div class="progress-bar"><div class="fill" id="progressFill"></div></div>
</div>
<div class="btn-row">
  <button class="btn-upload" id="btnUpload" disabled>开始上传</button>
  <button class="btn-clear" onclick="clearAll()">清空</button>
</div>
<div class="status" id="status"></div>

<script>
var filesToUpload = [];
var uploading = false;
var currentUploadIdx = -1;

var allowedExts = ['.mp3','.wav','.flac','.aac','.m4a','.m4r','.alac','.ogg','.oga','.opus','.wma','.ape','.aiff','.aif','.amr','.dvf','.msv','.dts','.dff','.dsf','.sacd','.tak','.tta','.wv','.mka','.mp4','.mkv','.avi','.mov','.wmv','.rm','.rmvb','.ts','.m2ts','.mts','.m2t','.3gp','.3g2','.m4v','.vob','.ogv','.asf','.divx','.f4v','.mxf','.wtv','.webm','.mpg','.mpeg','.flv'];

function isAllowedFile(name) {
  var dotIdx = name.lastIndexOf('.');
  if (dotIdx < 0) return false;
  var ext = name.substring(dotIdx).toLowerCase();
  return allowedExts.indexOf(ext) !== -1;
}

function formatSize(bytes) {
  if (bytes < 1024) return bytes + ' B';
  if (bytes < 1048576) return (bytes/1024).toFixed(1) + ' KB';
  return (bytes/1048576).toFixed(1) + ' MB';
}

function addFiles(fileList) {
  var count = 0;
  for (var i = 0; i < fileList.length; i++) {
    var f = fileList[i];
    if (!isAllowedFile(f.name)) {
      console.log('跳过不支持的文件: ' + f.name);
      continue;
    }
    var dup = false;
    for (var j = 0; j < filesToUpload.length; j++) {
      if (filesToUpload[j].name === f.name && filesToUpload[j].size === f.size) { dup = true; break; }
    }
    if (!dup) {
      filesToUpload.push({name: f.name, size: f.size, file: f, status: 'pending', progress: 0});
      count++;
    }
  }
  console.log('添加了 ' + count + ' 个文件，共 ' + filesToUpload.length + ' 个待上传');
  renderList();
}

function renderList() {
  var html = '';
  var pendingCount = 0;
  for (var i = 0; i < filesToUpload.length; i++) {
    var f = filesToUpload[i];
    if (f.status === 'pending') pendingCount++;
    var cls = f.status === 'done' ? 'done' : f.status === 'error' ? 'error' : f.status === 'uploading' ? 'uploading' : '';
    html += '<div class="file-item ' + cls + '">';
    html += '<span class="name">' + f.name + '</span>';
    html += '<span class="size">' + formatSize(f.size) + '</span>';
    if (f.status === 'pending') html += '<span class="remove" onclick="removeFile(' + i + ')">&times;</span>';
    if (f.status === 'done') html += '<span style="color:#00e676">&#10003;</span>';
    if (f.status === 'error') html += '<span style="color:#ff4444">&#10007;</span>';
    html += '</div>';
  }
  document.getElementById('fileList').innerHTML = html;
  document.getElementById('btnUpload').disabled = uploading || pendingCount === 0;
}

function removeFile(idx) {
  filesToUpload.splice(idx, 1);
  renderList();
}

function clearAll() {
  if (uploading) return;
  filesToUpload = [];
  renderList();
  document.getElementById('status').textContent = '';
  document.getElementById('progressWrap').style.display = 'none';
}

function startUpload() {
  if (uploading) return;
  uploading = true;
  currentUploadIdx = -1;
  renderList();
  uploadNext();
}

function uploadNext() {
  currentUploadIdx++;
  while (currentUploadIdx < filesToUpload.length && filesToUpload[currentUploadIdx].status !== 'pending') {
    currentUploadIdx++;
  }

  if (currentUploadIdx >= filesToUpload.length) {
    uploading = false;
    renderList();
    document.getElementById('progressWrap').style.display = 'none';
    var done = 0, err = 0;
    for (var i = 0; i < filesToUpload.length; i++) {
      if (filesToUpload[i].status === 'done') done++;
      if (filesToUpload[i].status === 'error') err++;
    }
    document.getElementById('status').textContent = '上传完成！成功 ' + done + ' 个' + (err > 0 ? '，失败 ' + err + ' 个' : '');
    if (window.opener && !window.opener.closed) {
      window.opener.postMessage({action:'uploadComplete'}, '*');
    }
    return;
  }

  var item = filesToUpload[currentUploadIdx];
  item.status = 'uploading';
  item.progress = 0;
  renderList();
  document.getElementById('progressWrap').style.display = '';
  document.getElementById('progressFill').style.width = '0%';
  document.getElementById('status').textContent = '正在上传：' + item.name;

  var formData = new FormData();
  formData.append('file', item.file);

  var xhr = new XMLHttpRequest();
  xhr.open('POST', '/api/upload', true);

  xhr.upload.onprogress = function(e) {
    if (e.lengthComputable) {
      var pct = Math.round((e.loaded / e.total) * 100);
      item.progress = pct;
      document.getElementById('progressFill').style.width = pct + '%';
    }
  };

  xhr.onload = function() {
    console.log('上传响应: status=' + xhr.status + ' body=' + xhr.responseText);
    if (xhr.status === 200) {
      item.status = 'done';
      item.progress = 100;
    } else {
      item.status = 'error';
    }
    renderList();
    uploadNext();
  };

  xhr.onerror = function() {
    console.log('上传网络错误');
    item.status = 'error';
    renderList();
    uploadNext();
  };

  xhr.send(formData);
}

document.getElementById('fileInput').addEventListener('change', function(e) {
  console.log('文件选择变更, 文件数: ' + e.target.files.length);
  addFiles(e.target.files);
});

document.getElementById('btnUpload').addEventListener('click', function() {
  startUpload();
});

var dropZone = document.getElementById('dropZone');
dropZone.addEventListener('dragover', function(e) { e.preventDefault(); dropZone.classList.add('dragover'); });
dropZone.addEventListener('dragleave', function() { dropZone.classList.remove('dragover'); });
dropZone.addEventListener('drop', function(e) {
  e.preventDefault();
  dropZone.classList.remove('dragover');
  console.log('拖拽文件数: ' + e.dataTransfer.files.length);
  addFiles(e.dataTransfer.files);
});
</script>
</body>
</html>
`))
}

func UploadAPIHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	r.ParseMultipartForm(500 << 20) // 500MB max

	file, handler, err := r.FormFile("file")
	if err != nil {
		fmt.Printf("[上传] 读取文件失败: %v\n", err)
		http.Error(w, "读取文件失败", http.StatusBadRequest)
		return
	}
	defer file.Close()

	ext := strings.ToLower(filepath.Ext(handler.Filename))
	allowed := false
	for _, a := range allowedExtensions {
		if ext == a {
			allowed = true
			break
		}
	}
	if !allowed {
		fmt.Printf("[上传] 不支持的文件格式: %s\n", ext)
		http.Error(w, "不支持的文件格式", http.StatusBadRequest)
		return
	}

	exePath, err := os.Executable()
	if err != nil {
		fmt.Printf("[上传] 获取程序路径失败: %v\n", err)
		http.Error(w, "服务器错误", http.StatusInternalServerError)
		return
	}
	uploadDir := filepath.Join(filepath.Dir(exePath), "uploads")

	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		fmt.Printf("[上传] 创建上传目录失败: %v\n", err)
		http.Error(w, "服务器错误", http.StatusInternalServerError)
		return
	}

	destPath := filepath.Join(uploadDir, handler.Filename)

	// 如果文件已存在，添加序号
	counter := 1
	originalPath := destPath
	for {
		if _, err := os.Stat(destPath); os.IsNotExist(err) {
			break
		}
		nameWithoutExt := strings.TrimSuffix(originalPath, ext)
		destPath = fmt.Sprintf("%s_%d%s", nameWithoutExt, counter, ext)
		counter++
	}

	dst, err := os.Create(destPath)
	if err != nil {
		fmt.Printf("[上传] 创建文件失败: %v\n", err)
		http.Error(w, "服务器错误", http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	written, err := io.Copy(dst, file)
	if err != nil {
		fmt.Printf("[上传] 写入文件失败: %v\n", err)
		os.Remove(destPath)
		http.Error(w, "服务器错误", http.StatusInternalServerError)
		return
	}

	fmt.Printf("[上传] 文件上传成功: %s (%.1f MB)\n", destPath, float64(written)/1048576)

	// 重新扫描上传目录，让新文件可以被搜索到
	rescanUploadDir()

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Write([]byte(`{"status":"ok","file":"` + handler.Filename + `"}`))
}
