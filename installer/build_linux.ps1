# FluentA Linux Cross-Compilation & Packaging Script
$ErrorActionPreference = "Stop"

Write-Host "==========================================" -ForegroundColor Cyan
Write-Host "  Building FluentA for Linux (amd64)     " -ForegroundColor Cyan
Write-Host "==========================================" -ForegroundColor Cyan

$RootDir = Split-Path -Parent $PSScriptRoot
$DistLinux = Join-Path $RootDir "dist\linux"
$ArchiveOut = Join-Path $RootDir "dist\fluenta-linux-amd64.tar.gz"

if (Test-Path $DistLinux) {
    Remove-Item $DistLinux -Recurse -Force
}
New-Item -ItemType Directory -Path $DistLinux -Force | Out-Null

$env:GOOS = "linux"
$env:GOARCH = "amd64"
$env:CGO_ENABLED = "0"

# 1. Compile CLI
Write-Host "`n[1/3] Compiling fluenta CLI binary..." -ForegroundColor Yellow
$cliPath = Join-Path $DistLinux "fluenta-cli"
go build -ldflags="-s -w" -o $cliPath (Join-Path $RootDir "cmd\cli")

# 2. Compile Desktop GUI
Write-Host "[2/3] Compiling fluenta-desktop GUI binary..." -ForegroundColor Yellow
$deskPath = Join-Path $DistLinux "fluenta-desktop"
go build -ldflags="-s -w" -o $deskPath (Join-Path $RootDir "cmd\desktop")

# 3. Copy support files
Write-Host "[3/3] Copying desktop entry, icons, and installer script..." -ForegroundColor Yellow
Copy-Item (Join-Path $RootDir "deploy\linux\fluenta.desktop") -Destination $DistLinux
Copy-Item (Join-Path $RootDir "deploy\linux\install.sh") -Destination $DistLinux
Copy-Item (Join-Path $RootDir "desktop\ui\fluenta-logo.png") -Destination $DistLinux
Copy-Item (Join-Path $RootDir "PRIVACY.md") -Destination $DistLinux

# 4. Package as .tar.gz
Write-Host "Packaging dist\fluenta-linux-amd64.tar.gz..." -ForegroundColor Green
tar -czf $ArchiveOut -C $DistLinux .

$tarSize = (Get-Item $ArchiveOut).Length / 1MB
$roundSize = [math]::Round($tarSize, 2)
Write-Host "`nSUCCESS: Release archive created:" -ForegroundColor Green
Write-Host "  Path: $ArchiveOut"
Write-Host "  Size: $roundSize MB"
