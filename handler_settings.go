package main

import (
	"net/http"
)

func SettingsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(`
<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <title>KTV 点歌机设置</title>
    <style>
        *{margin:0;padding:0;box-sizing:border-box;font-family:Microsoft YaHei}
        body{background:#1a252f;color:#ecf0f1;padding:20px;min-height:100vh}
        .container{max-width:500px;margin:0 auto;padding:30px;background:#243447;border-radius:6px;border:1px solid #2f4562;box-shadow:0 2px 8px rgba(0,0,0,0.25)}
        .title{text-align:center;color:#337ab9;margin-bottom:30px;font-size:26px;letter-spacing:2px;text-shadow:0 1px 1px rgba(0,0,0,0.15)}
        .form-group{margin-bottom:20px;padding:18px;background:#243447;border-radius:4px;border:1px solid #2f4562;box-shadow:0 1px 4px rgba(0,0,0,0.15)}
        .form-group label{display:block;margin-bottom:8px;color:#95a5a6;font-size:13px}
        .form-group input{width:100%;padding:11px 14px;border:1px solid #2f4562;border-radius:4px;background:#0d1922;color:#ecf0f1;font-size:14px;transition:border-color .2s}
        .form-group input:focus{outline:none;border-color:#428bca}
        .form-group textarea{width:100%;padding:11px 14px;border:1px solid #2f4562;border-radius:4px;background:#0d1922;color:#ecf0f1;font-size:14px;min-height:100px;resize:vertical;transition:border-color .2s}
        .form-group textarea:focus{outline:none;border-color:#428bca}
        .form-group .hint{font-size:12px;color:#95a5a6;margin-top:6px}
        .form-group select{width:100%;padding:11px 14px;border:1px solid #2f4562;border-radius:4px;background:#0d1922;color:#ecf0f1;font-size:14px;transition:border-color .2s}
        .form-group select:focus{outline:none;border-color:#428bca}
        .btn{padding:13px 34px;border:none;border-radius:6px;color:#fff;font-size:16px;font-weight:bold;cursor:pointer;letter-spacing:1px;transition:background .15s,box-shadow .15s}
        .btn-primary{background:#5cb85c;box-shadow:0 2px 4px rgba(0,0,0,0.2)}
        .btn-primary:hover{background:#4cae4c;box-shadow:0 2px 6px rgba(0,0,0,0.25)}
        .btn-primary:active{background:#449d44;box-shadow:0 1px 2px rgba(0,0,0,0.2)}
        .btn-secondary{background:#428bca;box-shadow:0 2px 4px rgba(0,0,0,0.2)}
        .btn-secondary:hover{background:#357ebd;box-shadow:0 2px 6px rgba(0,0,0,0.25)}
        .btn-secondary:active{background:#3071a9;box-shadow:0 1px 2px rgba(0,0,0,0.2)}
        .btn-group{display:flex;gap:12px;justify-content:center;margin-top:30px}
        .status{text-align:center;margin-top:20px;padding:12px;border-radius:4px;font-weight:bold;letter-spacing:1px}
        .status.success{background:#5cb85c;color:#fff}
        .status.error{background:#d9534f;color:#fff}
        .status.hidden{display:none}
        h2{color:#337ab9 !important;text-shadow:0 1px 1px rgba(0,0,0,0.15) !important}
    </style>
</head>
<body>
    <div class="container">
        <h1 class="title">KTV 点歌机设置</h1>
        
        <div class="form-group">
            <label for="mediaDirs">曲库目录（多个目录用换行分隔）</label>
            <textarea id="mediaDirs" placeholder="请输入曲库目录路径，每行一个目录"></textarea>
            <div class="hint">例如：D:\Music\nE:\KTV\Songs</div>
        </div>
        
        <div class="form-group">
            <label for="port">服务器端口</label>
            <input type="text" id="port" placeholder="请输入端口号，如：82">
        </div>

        <h2 style="margin:25px 0 15px;font-size:18px">扫码点歌设置</h2>

        <div class="form-group">
            <label for="qrEnabled">启用扫码点歌</label>
            <select id="qrEnabled">
                <option value="true">启用</option>
                <option value="false">禁用</option>
            </select>
        </div>

        <div class="form-group">
            <label for="qrServerAddr">二维码服务器地址</label>
            <input type="text" id="qrServerAddr" placeholder="例如：123.45.67.89:8352">
            <div class="hint">公网服务器的IP:端口，QR服务程序运行的地址</div>
        </div>

        <div class="form-group">
            <label for="qrPassword">扫码点歌密码</label>
            <input type="text" id="qrPassword" placeholder="手机扫码后需要输入的密码（留空则无需密码）">
            <div class="hint">手机扫码点歌时需要输入此密码验证，防止陌生人点歌</div>
        </div>
        
        <div class="btn-group">
            <button class="btn btn-primary" onclick="saveSettings()">保存设置</button>
            <button class="btn btn-secondary" onclick="loadSettings()">刷新设置</button>
        </div>
        
        <div class="status hidden" id="status"></div>
    </div>

    <script>
        function loadSettings() {
            var xhr = new XMLHttpRequest();
            xhr.open('GET', '/api/config', true);
            xhr.onload = function() {
                if (xhr.status === 200) {
                    var config = JSON.parse(xhr.responseText);
                    document.getElementById('mediaDirs').value = config.mediaDirs.join('\n');
                    document.getElementById('port').value = config.port;
                    document.getElementById('qrEnabled').value = config.qrEnabled ? 'true' : 'false';
                    document.getElementById('qrServerAddr').value = config.qrServerAddr || '';
                    document.getElementById('qrPassword').value = config.qrPassword || '';
                }
            };
            xhr.send();
        }

        function saveSettings() {
            var mediaDirsText = document.getElementById('mediaDirs').value;
            var mediaDirs = mediaDirsText.split('\n').map(function(dir) {
                return dir.trim();
            }).filter(function(dir) {
                return dir !== '';
            });

            var port = document.getElementById('port').value;

            var config = {
                mediaDirs: mediaDirs,
                port: port,
                qrEnabled: document.getElementById('qrEnabled').value === 'true',
                qrServerAddr: document.getElementById('qrServerAddr').value.trim(),
                qrPassword: document.getElementById('qrPassword').value
            };
            
            var xhr = new XMLHttpRequest();
            xhr.open('POST', '/api/config', true);
            xhr.setRequestHeader('Content-Type', 'application/json');
            xhr.onload = function() {
                var status = document.getElementById('status');
                if (xhr.status === 200) {
                    status.className = 'status success';
                    status.textContent = '设置保存成功！需要重启服务器才能生效';
                } else {
                    status.className = 'status error';
                    status.textContent = '保存失败，请重试';
                }
                status.style.display = 'block';
                setTimeout(function() {
                    status.style.display = 'none';
                }, 3000);
            };
            xhr.onerror = function() {
                var status = document.getElementById('status');
                status.className = 'status error';
                status.textContent = '保存失败，请重试';
                status.style.display = 'block';
                setTimeout(function() {
                    status.style.display = 'none';
                }, 3000);
            };
            xhr.send(JSON.stringify(config));
        }

        window.onload = loadSettings;
    </script>
</body>
</html>
`))
}
