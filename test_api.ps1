$ErrorActionPreference = "Continue"
$log = "f:\Backup\Downloads\KTV32bit20260818\test_api_result.log"
"=== Test Start $(Get-Date) ===" | Out-File $log -Encoding utf8

function Test-Url {
    param($url, $method = "GET", $body = $null)
    $start = Get-Date
    try {
        if ($method -eq "POST") {
            $r = Invoke-WebRequest -Uri $url -Method POST -Body $body -ContentType "application/json" -UseBasicParsing -TimeoutSec 30
        } else {
            $r = Invoke-WebRequest -Uri $url -UseBasicParsing -TimeoutSec 30
        }
        $ms = ((Get-Date) - $start).TotalMilliseconds
        $msg = "OK status=$($r.StatusCode) ms=$([int]$ms) bodyLen=$($r.Content.Length)"
        if ($r.Content.Length -lt 500) { $msg += " body=$($r.Content)" }
        $msg
    } catch {
        $ms = ((Get-Date) - $start).TotalMilliseconds
        "ERR after $([int]$ms) ms: $($_.Exception.Message)"
    }
}

# 1. GET /
$r1 = Test-Url "http://localhost:82/"
"[1] GET /: $r1" | Out-File $log -Append -Encoding utf8

# 2. check-tracks 1st
$r2 = Test-Url "http://localhost:82/api/check-tracks?name=Twins-一半女生-国语-流行歌曲.mkv"
"[2] check-tracks 1st: $r2" | Out-File $log -Append -Encoding utf8

# 3. check-tracks 2nd (cache hit)
$r3 = Test-Url "http://localhost:82/api/check-tracks?name=Twins-一半女生-国语-流行歌曲.mkv"
"[3] check-tracks 2nd: $r3" | Out-File $log -Append -Encoding utf8

# 4. check-and-add 1st
$body1 = @{ fileName = "Twins-一半女生-国语-流行歌曲.mkv"; requestKey = "test-key-1" } | ConvertTo-Json
$r4 = Test-Url "http://localhost:82/api/check-and-add-transcode" -method "POST" -body $body1
"[4] check-and-add 1st: $r4" | Out-File $log -Append -Encoding utf8

# 5. check-and-add 2nd (should hit cache)
$body2 = @{ fileName = "Twins-一半女生-国语-流行歌曲.mkv"; requestKey = "test-key-2" } | ConvertTo-Json
$r5 = Test-Url "http://localhost:82/api/check-and-add-transcode" -method "POST" -body $body2
"[5] check-and-add 2nd: $r5" | Out-File $log -Append -Encoding utf8

"=== Test End $(Get-Date) ===" | Out-File $log -Append -Encoding utf8
