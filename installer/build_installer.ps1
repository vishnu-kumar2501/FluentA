<#
.SYNOPSIS
    Builds the FluentA release binary and compiles the Inno Setup installer.
.DESCRIPTION
    1. Runs Go code generators (rsrc_windows_amd64.syso).
    2. Builds FluentA.exe with release flags (-s -w -H windowsgui).
    3. Detects Inno Setup 6 (ISCC.exe).
    4. Compiles installer\FluentASetup.iss into dist\FluentASetup.exe.
#>

[CmdletBinding()]
param(
    [switch]$SkipBuild
)

$ErrorActionPreference = 'Stop'
$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$RootDir = Split-Path -Parent $ScriptDir

Push-Location $RootDir
try {
    Write-Host "=============================================" -ForegroundColor Cyan
    Write-Host "      FluentA Production Installer Builder    " -ForegroundColor Cyan
    Write-Host "=============================================" -ForegroundColor Cyan

    # 1. Build FluentA.exe if requested
    if (-not $SkipBuild) {
        Write-Host "`n[1/3] Generating Windows PE metadata & manifest..." -ForegroundColor Yellow
        $env:GOOS = 'windows'
        $env:GOARCH = 'amd64'
        $env:CGO_ENABLED = '0'
        $env:GOCACHE = Join-Path $env:TEMP 'fluenta-gocache'
        & 'go' generate ./cmd/desktop
        if ($LASTEXITCODE -ne 0) { throw "go generate failed with exit code $LASTEXITCODE" }

        Write-Host "[2/3] Compiling release binary FluentA.exe..." -ForegroundColor Yellow
        & 'go' build -trimpath -ldflags="-s -w -H windowsgui" -o FluentA.exe ./cmd/desktop
        if ($LASTEXITCODE -ne 0) { throw "go build failed with exit code $LASTEXITCODE" }

        $exeInfo = Get-Item 'FluentA.exe'
        Write-Host "      Built FluentA.exe ($([math]::Round($exeInfo.Length / 1MB, 2)) MB)" -ForegroundColor Green
    } else {
        Write-Host "`n[1/2] Skipping Go compilation (-SkipBuild specified)..." -ForegroundColor Gray
    }

    # 2. Locate Inno Setup Compiler (ISCC.exe)
    Write-Host "`n[3/3] Looking for Inno Setup Compiler (ISCC.exe)..." -ForegroundColor Yellow
    $isccCandidates = @(
        (Get-Command 'iscc.exe' -ErrorAction SilentlyContinue | Select-Object -ExpandProperty Source -ErrorAction SilentlyContinue),
        'C:\Program Files (x86)\Inno Setup 6\ISCC.exe',
        'C:\Program Files\Inno Setup 6\ISCC.exe',
        'C:\Program Files (x86)\Inno Setup 7\ISCC.exe',
        'C:\Program Files\Inno Setup 7\ISCC.exe',
        'C:\ProgramData\chocolatey\bin\iscc.exe',
        "$env:LOCALAPPDATA\Programs\Inno Setup 7\ISCC.exe",
        "$env:LOCALAPPDATA\Programs\Inno Setup 6\ISCC.exe"
    ) | Where-Object { $_ -and (Test-Path $_) }

    $iscc = $isccCandidates | Select-Object -First 1

    if (-not $iscc) {
        Write-Host "`n[WARNING] Inno Setup 6 (ISCC.exe) was not found on your system." -ForegroundColor Red
        Write-Host "To compile the installer into an EXE, install Inno Setup using one of:" -ForegroundColor Yellow
        Write-Host "   winget install JRSoftware.InnoSetup" -ForegroundColor White
        Write-Host "   choco install innosetup" -ForegroundColor White
        Write-Host "   or download from: https://jrsoftware.org/isdl.php" -ForegroundColor White
        Write-Host "`nThe installer script is ready at:" -ForegroundColor Cyan
        Write-Host "   $ScriptDir\FluentASetup.iss" -ForegroundColor White
        return
    }

    Write-Host "      Found ISCC: $iscc" -ForegroundColor Green

    $distDir = Join-Path $RootDir 'dist'
    if (-not (Test-Path $distDir)) {
        New-Item -ItemType Directory -Path $distDir | Out-Null
    }

    $issPath = Join-Path $ScriptDir 'FluentASetup.iss'
    Write-Host "      Compiling $issPath..." -ForegroundColor Yellow
    & $iscc "/Qp" $issPath
    if ($LASTEXITCODE -ne 0) { throw "ISCC compilation failed with exit code $LASTEXITCODE" }

    $setupExe = Join-Path $distDir 'FluentASetup.exe'
    if (Test-Path $setupExe) {
        $docsSetup = Join-Path $RootDir 'docs\FluentASetup.exe'
        if (Test-Path (Join-Path $RootDir 'docs')) {
            Copy-Item $setupExe -Destination $docsSetup -Force
        }
        $setupInfo = Get-Item $setupExe
        Write-Host "`n=============================================" -ForegroundColor Green
        Write-Host "  SUCCESS: FluentASetup.exe created!" -ForegroundColor Green
        Write-Host "  Path: $($setupInfo.FullName)" -ForegroundColor White
        Write-Host "  Size: $([math]::Round($setupInfo.Length / 1MB, 2)) MB" -ForegroundColor White
        if (Test-Path $docsSetup) {
            Write-Host "  Copied to docs site: $docsSetup" -ForegroundColor Cyan
        }
        Write-Host "=============================================" -ForegroundColor Green
    }
} finally {
    Pop-Location
}

