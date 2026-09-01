# Generate favicon.ico for KTV app.
# Design: saturated purple gradient rounded square + bold yellow "KTV" laid out
# diagonally (top-left -> bottom-right) with black outline. High contrast, readable at small sizes.
# Outputs a multi-size ICO (16/32/48/64/128/256) using PNG-compressed ICO entries (Vista+).
$ErrorActionPreference = "Stop"
Add-Type -AssemblyName System.Drawing

function New-RoundedPath([float]$x, [float]$y, [float]$w, [float]$h, [float]$r) {
    $p = New-Object System.Drawing.Drawing2D.GraphicsPath
    $d = $r * 2
    $p.AddArc($x, $y, $d, $d, 180, 90)
    $p.AddArc($x + $w - $d, $y, $d, $d, 270, 90)
    $p.AddArc($x + $w - $d, $y + $h - $d, $d, $d, 0, 90)
    $p.AddArc($x, $y + $h - $d, $d, $d, 90, 90)
    $p.CloseFigure()
    return $p
}

# Renders the KTV icon onto the given Graphics at 256-unit canvas.
function Draw-Icon([System.Drawing.Graphics]$g) {
    $g.SmoothingMode = [System.Drawing.Drawing2D.SmoothingMode]::AntiAlias
    $g.TextRenderingHint = [System.Drawing.Text.TextRenderingHint]::AntiAlias
    $g.PixelOffsetMode = [System.Drawing.Drawing2D.PixelOffsetMode]::HighQuality

    # rounded-square clip (nothing bleeds to transparent area)
    $bgPath = New-RoundedPath 0 0 256 256 56
    $g.SetClip($bgPath)

    # saturated purple gradient background
    $left = [System.Drawing.Color]::FromArgb(255,150,50,255)   # #9632FF
    $right = [System.Drawing.Color]::FromArgb(255,55,6,120)    # #370678
    $bgBrush = New-Object System.Drawing.Drawing2D.LinearGradientBrush(
        (New-Object System.Drawing.RectangleF(0,0,256,256)), $left, $right, 90)
    $g.FillPath($bgBrush, $bgPath)
    $bgBrush.Dispose()

    # rotate + so K lands top-left and V bottom-right, then draw centered (positive = tips right end down on the tile)
    $g.TranslateTransform(128,128)
    $g.RotateTransform(30)
    $g.TranslateTransform(-128,-128)

    $font = New-Object System.Drawing.Font("Arial Black", 56, [System.Drawing.FontStyle]::Bold, [System.Drawing.GraphicsUnit]::Pixel)
    $yellowBrush = New-Object System.Drawing.SolidBrush([System.Drawing.Color]::FromArgb(255,255,214,0))   # #FFD600
    $blackBrush  = New-Object System.Drawing.SolidBrush([System.Drawing.Color]::FromArgb(255,30,6,40))
    # use MeasureString + PointF overload so the string is NEVER truncated by a layout rect
    $m = $g.MeasureString("KTV", $font)
    $sx = (256 - $m.Width) / 2
    $sy = (256 - $m.Height) / 2
    foreach ($o in @(@(-3,-3),@(3,-3),@(-3,3),@(3,3),@(-3,0),@(3,0),@(0,-3),@(0,3))) {
        $g.DrawString("KTV", $font, $blackBrush, ($sx + $o[0]), ($sy + $o[1]))
    }
    $g.DrawString("KTV", $font, $yellowBrush, $sx, $sy)
    $g.ResetClip()

    $yellowBrush.Dispose(); $blackBrush.Dispose(); $font.Dispose()
}

$sizes = @(256,128,64,48,32,16)
$pngPaths = @()
foreach ($sz in $sizes) {
    $bmp = New-Object System.Drawing.Bitmap($sz, $sz, [System.Drawing.Imaging.PixelFormat]::Format32bppArgb)
    $g = [System.Drawing.Graphics]::FromImage($bmp)
    $g.ScaleTransform(($sz/256.0), ($sz/256.0))
    Draw-Icon $g
    $g.Dispose()
    $p = Join-Path $env:TEMP ("favicon_{0}.png" -f $sz)
    $bmp.Save($p, [System.Drawing.Imaging.ImageFormat]::Png)
    $bmp.Dispose()
    $pngPaths += ,$p
    Write-Host ("rendered {0}x{0}" -f $sz)
}

# --- Pack the PNG entries into a multi-size ICO ---
$sizesCount = $sizes.Count
$bw = New-Object System.IO.BinaryWriter([System.IO.MemoryStream]::new())
$bw.Write([uint16]0); $bw.Write([uint16]1); $bw.Write([uint16]$sizesCount)
$icondir = $bw.BaseStream.ToArray(); $bw.Dispose()

$entrySize = 16
$offset = 6 + ($sizesCount * $entrySize)
$entries = @()
$blobs = @()
for ($j=0; $j -lt $sizesCount; $j++) {
    $sz = $sizes[$j]
    $blob = [System.IO.File]::ReadAllBytes($pngPaths[$j])
    $blobs += ,$blob
    $dim = if ($sz -ge 256) { 0 } else { $sz }
    $bw = New-Object System.IO.BinaryWriter([System.IO.MemoryStream]::new())
    $bw.Write([byte]$dim)
    $bw.Write([byte]$dim)
    $bw.Write([byte]0); $bw.Write([byte]0)
    $bw.Write([uint16]1)
    $bw.Write([uint16]32)
    $bw.Write([uint32]$blob.Length)
    $bw.Write([uint32]$offset)
    $entries += $bw.BaseStream.ToArray(); $bw.Dispose()
    $offset += $blob.Length
}

$ico = $icondir
foreach ($e in $entries) { $ico += $e }
foreach ($b in $blobs) { $ico += $b }

$out = Join-Path $PSScriptRoot "favicon.ico"
[System.IO.File]::WriteAllBytes($out, $ico)
Write-Host ("WROTE " + $out + " size=" + $ico.Length)

Copy-Item $pngPaths[0] (Join-Path $PSScriptRoot "_favicon_preview.png") -Force
foreach ($p in $pngPaths) { if (Test-Path $p) { Remove-Item $p -Force } }
Write-Host "DONE"