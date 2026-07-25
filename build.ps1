# build.ps1 — KTV 双屏点歌机一键构建脚本
# 使用 Go 1.20.14 编译，原生支持 Windows 7 SP1+
#
# 用法: .\build.ps1
# 前提: C:\Go1.20 已安装 Go 1.20.14
# 输出:
#   dist\ktv\ktv.exe    (64位, Win7+兼容)
#   dist\ktv32\ktv.exe  (32位, Win7+兼容)

$ErrorActionPreference = "Stop"

$Go120 = "C:\Go1.20\bin\go.exe"
if (-not (Test-Path $Go120)) {
    Write-Host "[错误] 未找到 Go 1.20: $Go120" -ForegroundColor Red
    Write-Host "请安装 Go 1.20.14 到 C:\Go1.20" -ForegroundColor Yellow
    exit 1
}

Write-Host "=======================================" -ForegroundColor Cyan
Write-Host "  KTV双屏点歌机 - 构建 (Go 1.20 Win7兼容)" -ForegroundColor Cyan
Write-Host "=======================================" -ForegroundColor Cyan

# 清理输出目录
if (Test-Path "dist") { Remove-Item -Recurse -Force "dist" }
New-Item -ItemType Directory -Path "dist\ktv" -Force | Out-Null
New-Item -ItemType Directory -Path "dist\ktv32" -Force | Out-Null

# ---- 64位构建 ----
Write-Host ""
Write-Host "[1/4] 编译 64位版本 (amd64)..." -ForegroundColor Yellow
$env:GOROOT = "C:\Go1.20"
$env:GOARCH = "amd64"
$env:CGO_ENABLED = "0"
& $Go120 build -ldflags="-s -w" -o "dist\ktv\ktv.exe" .
if ($LASTEXITCODE -ne 0) { Write-Host "64位编译失败!" -ForegroundColor Red; exit 1 }
Write-Host "[OK] 64位编译完成" -ForegroundColor Green

# ---- 32位构建 ----
Write-Host ""
Write-Host "[2/4] 编译 32位版本 (386)..." -ForegroundColor Yellow
$env:GOARCH = "386"
& $Go120 build -ldflags="-s -w" -o "dist\ktv32\ktv.exe" .
if ($LASTEXITCODE -ne 0) { Write-Host "32位编译失败!" -ForegroundColor Red; exit 1 }
Write-Host "[OK] 32位编译完成" -ForegroundColor Green

# 清理环境变量
Remove-Item Env:\GOARCH -ErrorAction SilentlyContinue
Remove-Item Env:\CGO_ENABLED -ErrorAction SilentlyContinue
Remove-Item Env:\GOROOT -ErrorAction SilentlyContinue

# ---- 复制运行依赖 ----
Write-Host ""
Write-Host "[3/4] 复制运行依赖文件..." -ForegroundColor Yellow
$deps = @("ffmpeg.exe", "ffprobe.exe")
foreach ($dep in $deps) {
    if (Test-Path $dep) {
        Copy-Item $dep "dist\ktv\$dep" -Force
        Copy-Item $dep "dist\ktv32\$dep" -Force
        Write-Host "  [OK] $dep" -ForegroundColor Green
    } else {
        Write-Host "  [跳过] $dep 不存在" -ForegroundColor DarkGray
    }
}

# ---- 文件大小 ----
Write-Host ""
Write-Host "[4/4] 构建结果:" -ForegroundColor Yellow
$exe64 = Get-Item "dist\ktv\ktv.exe" -ErrorAction SilentlyContinue
$exe32 = Get-Item "dist\ktv32\ktv.exe" -ErrorAction SilentlyContinue
if ($exe64) { Write-Host ("  64位: {0:N1} MB" -f ($exe64.Length/1MB)) -ForegroundColor White }
if ($exe32) { Write-Host ("  32位: {0:N1} MB" -f ($exe32.Length/1MB)) -ForegroundColor White }

Write-Host ""
Write-Host "=======================================" -ForegroundColor Cyan
Write-Host "  构建完成！" -ForegroundColor Green
Write-Host "=======================================" -ForegroundColor Cyan
Write-Host ""
Write-Host "64位: dist\ktv\" -ForegroundColor White
Write-Host "32位: dist\ktv32\" -ForegroundColor White
