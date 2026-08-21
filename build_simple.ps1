$env:GOROOT = "C:\Go1.20"
$env:GOARCH = "386"
$env:CGO_ENABLED = "0"

# 用 dist 子目录作为日志输出（项目根可能被沙箱限制）
$logDir = "C:\Users\Administrator\AppData\Local\Temp"
if (-not (Test-Path $logDir)) { $logDir = $env:TEMP }
if (-not (Test-Path $logDir)) { $logDir = "f:\Backup\Downloads\KTV32bit20260818\dist" }
$out = Join-Path $logDir "ktv_build_result.log"

"=== Build Start $(Get-Date) ===" | Out-File $out -Encoding utf8

$sw = [System.Diagnostics.Stopwatch]::StartNew()
& "C:\Go1.20\bin\go.exe" build -ldflags="-s -w" -o "f:\Backup\Downloads\KTV32bit20260818\dist\ktv32\ktv.exe" . 2>&1 | Out-File $out -Append -Encoding utf8
$sw.Stop()
"EXIT=$LASTEXITCODE elapsed=$($sw.ElapsedMilliseconds)ms" | Out-File $out -Append -Encoding utf8

"=== Build End $(Get-Date) ===" | Out-File $out -Append -Encoding utf8

$f = Get-Item "f:\Backup\Downloads\KTV32bit20260818\dist\ktv32\ktv.exe"
"ktv32.exe: $($f.Length) bytes, lastWrite=$($f.LastWriteTime)" | Out-File $out -Append -Encoding utf8

Write-Host ("LOG_FILE=" + $out)
Write-Host ("EXIT=" + $LASTEXITCODE)
