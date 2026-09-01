@echo off
setlocal

REM =======================================
REM   KTV Build Script (build 4 exes, no cleanup)
REM =======================================

set GOEXE=C:\Go1.20\bin\go.exe
if not exist "%GOEXE%" (
    echo [ERROR] Go compiler not found: %GOEXE%
    pause
    exit /b 1
)

set GOROOT=C:\Go1.20
set CGO_ENABLED=0
set ROOT=%~dp0

echo =======================================
echo   KTV Build Script (4 exes)
echo =======================================
echo.

echo [1/4] ktv.exe (amd64)...
set GOARCH=amd64
"%GOEXE%" build -ldflags="-s -w" -o "%ROOT%ktv.exe" .
if errorlevel 1 goto :fail

echo [2/4] qrserver.exe (amd64)...
pushd "%ROOT%qrserver"
"%GOEXE%" build -ldflags="-s -w" -o "%ROOT%qrserver.exe" .
if errorlevel 1 (popd & goto :fail) else (popd)

echo [3/4] ktv-32bit.exe (386)...
set GOARCH=386
"%GOEXE%" build -ldflags="-s -w" -o "%ROOT%ktv-32bit.exe" .
if errorlevel 1 goto :fail

echo [4/4] qrserver-32bit.exe (386)...
pushd "%ROOT%qrserver"
"%GOEXE%" build -ldflags="-s -w" -o "%ROOT%qrserver-32bit.exe" .
if errorlevel 1 (popd & goto :fail) else (popd)

echo.
echo =======================================
echo   Build done. Output:
echo =======================================
for %%F in ("%ROOT%ktv.exe" "%ROOT%ktv-32bit.exe" "%ROOT%qrserver.exe" "%ROOT%qrserver-32bit.exe") do (
    if exist "%%F" (
        echo   %%~nxF  [OK]
    ) else (
        echo   %%~nxF  [MISSING]
    )
)
goto :end

:fail
echo.
echo [FAILED] Build error.
pause
exit /b 1

:end
pause
endlocal
