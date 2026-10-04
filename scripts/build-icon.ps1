param(
    [string]$Source = "docs/assets/sound-snoofer-tray.png",
    [string]$Output = "internal/desktop/tray.ico"
)
$ErrorActionPreference = "Stop"
Add-Type -AssemblyName System.Drawing
Push-Location (Split-Path $PSScriptRoot -Parent)
try {
    $sourceImage = [System.Drawing.Image]::FromFile((Join-Path $PWD $Source))
    $sizes = @(16, 20, 24, 32, 48, 64, 256)
    $frames = [System.Collections.Generic.List[byte[]]]::new()
    try {
        foreach ($size in $sizes) {
            $bitmap = [System.Drawing.Bitmap]::new($size, $size, [System.Drawing.Imaging.PixelFormat]::Format32bppArgb)
            $graphics = [System.Drawing.Graphics]::FromImage($bitmap)
            $stream = [System.IO.MemoryStream]::new()
            try {
                $graphics.Clear([System.Drawing.Color]::Transparent)
                $graphics.InterpolationMode = [System.Drawing.Drawing2D.InterpolationMode]::HighQualityBicubic
                $graphics.PixelOffsetMode = [System.Drawing.Drawing2D.PixelOffsetMode]::HighQuality
                $graphics.DrawImage($sourceImage, [System.Drawing.Rectangle]::new(0, 0, $size, $size))
                $bitmap.Save($stream, [System.Drawing.Imaging.ImageFormat]::Png)
                $frames.Add($stream.ToArray())
            } finally {
                $stream.Dispose()
                $graphics.Dispose()
                $bitmap.Dispose()
            }
        }
    } finally { $sourceImage.Dispose() }
    $file = [System.IO.File]::Create((Join-Path $PWD $Output))
    $writer = [System.IO.BinaryWriter]::new($file)
    try {
        $writer.Write([uint16]0)
        $writer.Write([uint16]1)
        $writer.Write([uint16]$sizes.Count)
        $offset = 6 + 16 * $sizes.Count
        for ($i = 0; $i -lt $sizes.Count; $i++) {
            $dimension = [byte]($sizes[$i] % 256)
            $writer.Write($dimension)
            $writer.Write($dimension)
            $writer.Write([byte]0)
            $writer.Write([byte]0)
            $writer.Write([uint16]1)
            $writer.Write([uint16]32)
            $writer.Write([uint32]$frames[$i].Length)
            $writer.Write([uint32]$offset)
            $offset += $frames[$i].Length
        }
        foreach ($frame in $frames) { $writer.Write($frame) }
    } finally { $writer.Dispose() }
} finally { Pop-Location }
