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
        body{background:#080812;color:#fff;padding:20px}
        .container{max-width:500px;margin:0 auto}
        .title{text-align:center;color:#00aaff;margin-bottom:30px;font-size:24px}
        .form-group{margin-bottom:20px}
        .form-group label{display:block;margin-bottom:8px;color:#aaa}
        .form-group input{width:100%;padding:10px;border:none;border-radius:6px;background:#181a35;color:#fff;font-size:14px}
        .form-group textarea{width:100%;padding:10px;border:none;border-radius:6px;background:#181a35;color:#fff;font-size:14px;min-height:100px;resize:vertical}
        .form-group .hint{font-size:12px;color:#666;margin-top:5px}
        .btn{padding:12px 30px;border:none;border-radius:6px;color:#fff;font-size:16px;cursor:pointer;margin-right:10px}
        .btn-primary{background:#00aaff}
        .btn-primary:hover{background:#0088dd}
        .btn-secondary{background:#222444}
        .btn-secondary:hover{background:#333555}
        .btn-group{display:flex;gap:10px;justify-content:center;margin-top:30px}
        .status{text-align:center;margin-top:20px;padding:10px;border-radius:6px}
        .status.success{background:#27ae60;color:#fff}
        .status.error{background:#e74c3c;color:#fff}
        .status.hidden{display:none}
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

        <h2 style="color:#00aaff;margin:25px 0 15px;font-size:18px">扫码点歌设置</h2>

        <div class="form-group">
            <label for="qrEnabled">启用扫码点歌</label>
            <select id="qrEnabled" style="width:100%;padding:10px;border:none;border-radius:6px;background:#181a35;color:#fff;font-size:14px">
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
