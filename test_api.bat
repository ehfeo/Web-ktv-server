@echo off
chcp 65001 >nul
echo === Test Start %time% ===
echo.
echo [1] GET /
curl -s -o nul -w "status=%%{http_code} size=%%{size_download}\n" http://localhost:82/
echo.
echo [2] check-tracks first call
curl -s -w "\nstatus=%%{http_code} time=%%{time_total}s\n" "http://localhost:82/api/check-tracks?name=Twins-一半女生-国语-流行歌曲.mkv"
echo.
echo [3] check-tracks second call (should hit cache)
curl -s -w "\nstatus=%%{http_code} time=%%{time_total}s\n" "http://localhost:82/api/check-tracks?name=Twins-一半女生-国语-流行歌曲.mkv"
echo.
echo [4] check-and-add-transcode
curl -s -X POST -H "Content-Type: application/json" -d "{\"fileName\":\"Twins-一半女生-国语-流行歌曲.mkv\",\"requestKey\":\"test-key-1\"}" -w "\nstatus=%%{http_code} time=%%{time_total}s\n" http://localhost:82/api/check-and-add-transcode
echo.
echo [5] check-and-add-transcode 2nd (should hit cache)
curl -s -X POST -H "Content-Type: application/json" -d "{\"fileName\":\"Twins-一半女生-国语-流行歌曲.mkv\",\"requestKey\":\"test-key-2\"}" -w "\nstatus=%%{http_code} time=%%{time_total}s\n" http://localhost:82/api/check-and-add-transcode
echo.
echo === Test End %time% ===
