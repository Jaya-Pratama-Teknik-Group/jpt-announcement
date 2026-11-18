# =============================================================================
# build-installer.ps1
# Script untuk membangun installer Windows (.exe) JPT Announce
#
# CARA PAKAI:
#   1. Buka PowerShell sebagai Administrator
#   2. Jalankan: .\scripts\build-installer.ps1
#
# Output: bin\JPT Announce-amd64-setup.exe
# =============================================================================

$ErrorActionPreference = "Stop"
$Root = $PSScriptRoot | Split-Path -Parent

function Write-Step([string]$msg) {
    Write-Host ""
    Write-Host "==> $msg" -ForegroundColor Cyan
}

function Write-OK([string]$msg) {
    Write-Host "    [OK] $msg" -ForegroundColor Green
}

function Write-Fail([string]$msg) {
    Write-Host "    [GAGAL] $msg" -ForegroundColor Red
    exit 1
}

# ---------------------------------------------------------------------------
# Langkah 1: Pastikan NSIS terinstal
# ---------------------------------------------------------------------------
Write-Step "Memeriksa instalasi NSIS..."

$nsisExe = ""
$candidates = @(
    "C:\Program Files (x86)\NSIS\makensis.exe",
    "C:\Program Files\NSIS\makensis.exe"
)
foreach ($c in $candidates) {
    if (Test-Path $c) { $nsisExe = $c; break }
}

if (-not $nsisExe) {
    Write-Host "    NSIS tidak ditemukan. Mengunduh dan menginstal NSIS..." -ForegroundColor Yellow

    # Cek Chocolatey atau Winget, jika tidak ada unduh manual
    if (Get-Command choco -ErrorAction SilentlyContinue) {
        choco install nsis -y --no-progress
    } elseif (Get-Command winget -ErrorAction SilentlyContinue) {
        winget install NSIS.NSIS --silent --accept-package-agreements --accept-source-agreements
    } else {
        $nsisUrl = "https://downloads.sourceforge.net/project/nsis/NSIS%203/3.10/nsis-3.10-setup.exe"
        $nsisInstaller = "$env:TEMP\nsis-setup.exe"
        Write-Host "    Mengunduh NSIS dari $nsisUrl ..." -ForegroundColor Yellow
        Invoke-WebRequest -Uri $nsisUrl -OutFile $nsisInstaller -UseBasicParsing
        Start-Process $nsisInstaller -ArgumentList "/S" -Wait
        Remove-Item $nsisInstaller -Force
    }

    # Cek ulang
    foreach ($c in $candidates) {
        if (Test-Path $c) { $nsisExe = $c; break }
    }

    if (-not $nsisExe) {
        Write-Fail "NSIS tetap tidak ditemukan setelah instalasi. Install manual dari https://nsis.sourceforge.io/Download"
    }
}

Write-OK "NSIS ditemukan: $nsisExe"

# Tambahkan ke PATH sesi ini agar makensis bisa dipanggil langsung
$nsisDir = Split-Path $nsisExe
if ($env:PATH -notlike "*$nsisDir*") {
    $env:PATH = "$nsisDir;$env:PATH"
}

# ---------------------------------------------------------------------------
# Langkah 2: Build frontend
# ---------------------------------------------------------------------------
Write-Step "Membangun frontend Vue 3..."
Push-Location "$Root\frontend"
try {
    npm run build
    if ($LASTEXITCODE -ne 0) { Write-Fail "npm run build gagal" }
    Write-OK "Frontend berhasil dibangun"
} finally {
    Pop-Location
}

# ---------------------------------------------------------------------------
# Langkah 3: Buat WebView2 bootstrapper (jika belum ada)
# ---------------------------------------------------------------------------
Write-Step "Menyiapkan WebView2 bootstrapper..."
$webview2 = "$Root\build\windows\nsis\MicrosoftEdgeWebview2Setup.exe"
if (-not (Test-Path $webview2)) {
    Push-Location "$Root\build\windows\nsis"
    wails3 generate webview2bootstrapper -dir "$Root\build\windows\nsis"
    Pop-Location
}
Write-OK "WebView2 bootstrapper siap"

# ---------------------------------------------------------------------------
# Langkah 4: Build .exe aplikasi (production)
# ---------------------------------------------------------------------------
Write-Step "Membangun aplikasi (production binary)..."
Push-Location "$Root\build"
wails3 generate syso -arch amd64 -icon windows/icon.ico -manifest windows/wails.exe.manifest -info windows/info.json -out ..\wails_windows_amd64.syso
Pop-Location

Push-Location $Root
go build -tags production -trimpath -buildvcs=false -ldflags="-w -s -H windowsgui" -o "bin\JPTAnnounce.exe" .
if ($LASTEXITCODE -ne 0) { Write-Fail "go build gagal" }
Remove-Item "$Root\*.syso" -Force -ErrorAction SilentlyContinue
Write-OK "Binary berhasil dibangun: bin\JPTAnnounce.exe"

# ---------------------------------------------------------------------------
# Langkah 5: Jalankan NSIS untuk membuat installer
# ---------------------------------------------------------------------------
Write-Step "Membuat installer NSIS..."
Push-Location "$Root\build\windows\nsis"
try {
    & $nsisExe "-DARG_WAILS_AMD64_BINARY=$Root\bin\JPTAnnounce.exe" project.nsi
    if ($LASTEXITCODE -ne 0) { Write-Fail "makensis gagal (exit code $LASTEXITCODE)" }
} finally {
    Pop-Location
}

# ---------------------------------------------------------------------------
# Langkah 6: Tampilkan hasil
# ---------------------------------------------------------------------------
$output = Get-ChildItem "$Root\bin\*-setup.exe" | Sort-Object LastWriteTime -Descending | Select-Object -First 1
if ($output) {
    $sizeMB = [math]::Round($output.Length / 1MB, 2)
    Write-Host ""
    Write-Host "===============================================================" -ForegroundColor Green
    Write-Host "  INSTALLER BERHASIL DIBUAT!" -ForegroundColor Green
    Write-Host "  File   : $($output.FullName)" -ForegroundColor White
    Write-Host "  Ukuran : $sizeMB MB" -ForegroundColor White
    Write-Host "===============================================================" -ForegroundColor Green
    Write-Host ""
    Write-Host "Distribusikan file '$($output.Name)' ke pengguna." -ForegroundColor Yellow
    Write-Host "Installer akan menginstal ke: C:\Program Files\JPT\JPT Announce" -ForegroundColor Yellow
} else {
    Write-Fail "File installer tidak ditemukan di bin\ setelah build"
}
