@echo off
chcp 65001 >nul
cd /d "%~dp0"
echo =======================================
echo   KTV 构建脚本 ( bypass 执行策略 )
echo =======================================
echo.
powershell.exe -ExecutionPolicy Bypass -NoProfile -File "%~dp0build.ps1"
echo.
echo 按任意键退出...
pause >nul
