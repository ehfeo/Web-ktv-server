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

        <div class="form-group">
            <label for="audioBitrate">音频实时转码码率（kbps）</label>
            <input type="number" id="audioBitrate" min="32" max="512" placeholder="请输入码率，范围 32-512，默认 192">
            <div class="hint">用于播放 wma / ape / dts / dff / dsf 及多音轨抽取等需要转码的音频。数值越高音质越好但占用带宽与CPU更多，AAC 编码上限为 512k，建议 128-320。</div>
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
            <label for="qrMode">二维码服务器类型</label>
            <select id="qrMode" onchange="updateQRMode()">
                <option value="internal">内置（随本程序运行，共用IP和端口）</option>
                <option value="external">外接（使用独立二维码服务器）</option>
            </select>
            <div class="hint" id="qrModeHint"></div>
        </div>

        <div class="form-group" id="intQRNotice">
            <label>内置服务器说明</label>
            <div class="hint" style="line-height:1.7">
                ◆ 内置模式把二维码服务器直接集成到本程序，<b>与主服务器共用同一个IP和端口</b>，无需另跑程序、无需开额外端口。<br>
                ◆ 适用场景：<b>手机与点歌电脑在同一个局域网内</b>（店内、家里内网扫码点歌），扫码即可直接访问。<br>
                ◆ 由于共用主服务器端口，异地（公网）扫码可能受网络/端口限制，无法保证访问成功。
            </div>
        </div>

        <div class="form-group" id="extQRGroup">
            <label for="qrServerAddr">二维码服务器地址</label>
            <input type="text" id="qrServerAddr" placeholder="例如：123.45.67.89:8352">
            <div class="hint" style="line-height:1.7">
                ◆ 外接模式需要<b>自行部署并运行</b>独立的二维码服务器（项目内附带的 qrserver 程序），地址填写该服务器的 IP:端口。<br>
                ◆ 适用场景：手机与点歌电脑<b>不在同一局域网</b>，需要通过公网远程扫码点歌。<br>
                <b style="color:#f0ad4e">⚠ 重要：选择外接时，必须自行保证扫码端（手机）能访问到你所填写的外接二维码服务器——该服务器必须有公网IP，或已通过端口映射/内网穿透公开到公网；否则手机扫码将打不开点歌页面。</b>
            </div>
        </div>

        <div class="form-group">
            <label for="qrPassword">扫码点歌密码</label>
            <input type="text" id="qrPassword" placeholder="手机扫码后需要输入的密码（留空则无需密码）">
            <div class="hint">手机扫码点歌时需要输入此密码验证，防止陌生人点歌</div>
        </div>

        <div class="form-group">
            <label>允许手机遥控</label>
            <label class="switch-row">
                <input type="checkbox" id="qrCtrlEnabled">
                <span>允许手机端远程控制播放（切歌 / 播放暂停 / 重唱 / 音量）</span>
            </label>
            <div class="hint">开启后，手机扫码点歌页的“遥控”标签页可操控主控端播放。为防止他人捣乱，建议仅在需要时开启；关闭后手机端会提示“主控端未开启手机遥控权限”。</div>
        </div>
        
        <div class="btn-group">
            <button class="btn btn-primary" onclick="saveSettings()">保存设置</button>
            <button class="btn btn-secondary" onclick="loadSettings()">刷新设置</button>
        </div>
        
        <div class="status hidden" id="status"></div>
    </div>

    <script>
        function updateQRMode() {
            var mode = document.getElementById('qrMode').value;
            var intNotice = document.getElementById('intQRNotice');
            var extGroup = document.getElementById('extQRGroup');
            var hint = document.getElementById('qrModeHint');
            if (mode === 'internal') {
                intNotice.style.display = 'block';
                extGroup.style.display = 'none';
                hint.textContent = '已选内置：手机扫码走本程序同IP同端口，适合局域网内点歌。';
            } else {
                intNotice.style.display = 'none';
                extGroup.style.display = 'block';
                hint.textContent = '已选外接：请填写外部二维码服务器地址，并确保手机能访问到它。';
            }
        }

        function loadSettings() {
            var xhr = new XMLHttpRequest();
            xhr.open('GET', '/api/config', true);
            xhr.onload = function() {
                if (xhr.status === 200) {
                    var config = JSON.parse(xhr.responseText);
                    document.getElementById('mediaDirs').value = config.mediaDirs.join('\n');
                    document.getElementById('port').value = config.port;
                    document.getElementById('audioBitrate').value = config.audioTranscodeBitrate;
                    document.getElementById('qrEnabled').value = config.qrEnabled ? 'true' : 'false';
                    document.getElementById('qrMode').value = config.qrMode === 'internal' ? 'internal' : 'external';
                    document.getElementById('qrServerAddr').value = config.qrServerAddr || '';
                    document.getElementById('qrPassword').value = config.qrPassword || '';
                    document.getElementById('qrCtrlEnabled').checked = !!config.qrCtrlEnabled;
                    updateQRMode();
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

            var audioBitrate = parseInt(document.getElementById('audioBitrate').value, 10);
            if (isNaN(audioBitrate)) audioBitrate = 192;
            if (audioBitrate < 32) audioBitrate = 32;
            if (audioBitrate > 512) audioBitrate = 512;

            var config = {
                mediaDirs: mediaDirs,
                port: port,
                audioTranscodeBitrate: audioBitrate,
                qrEnabled: document.getElementById('qrEnabled').value === 'true',
                qrMode: document.getElementById('qrMode').value,
                qrServerAddr: document.getElementById('qrServerAddr').value.trim(),
                qrPassword: document.getElementById('qrPassword').value,
                qrCtrlEnabled: document.getElementById('qrCtrlEnabled').checked
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
