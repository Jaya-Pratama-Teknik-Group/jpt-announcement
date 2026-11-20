# =============================================================================
# build-android.ps1
# Script untuk membangun APK Android JPT Announce langsung di Windows
# menggunakan Android Studio SDK/NDK dan JDK 17+ (tanpa WSL2).
#
# CARA PAKAI:
#   .\scripts\build-android.ps1          # Release APK (arm64-v8a)
#   .\scripts\build-android.ps1 -Fat     # Universal APK (arm64-v8a + x86_64)
#
# Output: bin\JPTAnnounce.apk
# =============================================================================

param(
    [switch]$Fat,
    [switch]$DebugBuild
)

$ErrorActionPreference = "Stop"
$Root = $PSScriptRoot | Split-Path -Parent

Write-Host ""
Write-Host "==> Memeriksa Android SDK, NDK, dan JDK..." -ForegroundColor Cyan

$sdkRoot = @($env:ANDROID_HOME, $env:ANDROID_SDK_ROOT, "$env:LOCALAPPDATA\Android\Sdk") |
    Where-Object { $_ -and (Test-Path $_) } |
    Select-Object -First 1
if (-not $sdkRoot) {
    Write-Error "Android SDK tidak ditemukan. Set ANDROID_HOME atau instal via Android Studio."
}
$env:ANDROID_HOME = $sdkRoot
$env:ANDROID_SDK_ROOT = $sdkRoot

$ndkRoot = $env:ANDROID_NDK_HOME
if (-not $ndkRoot -or -not (Test-Path $ndkRoot)) {
    $ndkRoot = Get-ChildItem "$sdkRoot\ndk" -Directory -ErrorAction SilentlyContinue |
        Sort-Object Name -Descending |
        Where-Object { Test-Path "$($_.FullName)\toolchains\llvm\prebuilt\windows-x86_64\bin" } |
        Select-Object -First 1 -ExpandProperty FullName
}
if (-not $ndkRoot -or -not (Test-Path $ndkRoot)) {
    Write-Error "Android NDK tidak ditemukan di $sdkRoot\ndk. Instal NDK via Android Studio SDK Manager."
}

$javaCandidates = @(
    $env:JAVA_HOME,
    "C:\Program Files\Android\Android Studio\jbr",
    (Get-ChildItem "C:\Program Files\Android\Android Studio*\jbr" -Directory -ErrorAction SilentlyContinue | Sort-Object FullName -Descending | Select-Object -First 1 -ExpandProperty FullName),
    (Get-ChildItem "C:\Program Files\Eclipse Adoptium\jdk-*" -Directory -ErrorAction SilentlyContinue | Sort-Object Name -Descending | Select-Object -First 1 -ExpandProperty FullName),
    (Get-ChildItem "C:\Program Files\Java\jdk-2*" -Directory -ErrorAction SilentlyContinue | Sort-Object Name -Descending | Select-Object -First 1 -ExpandProperty FullName),
    (Get-ChildItem "C:\Program Files\Java\jdk-17*" -Directory -ErrorAction SilentlyContinue | Sort-Object Name -Descending | Select-Object -First 1 -ExpandProperty FullName)
) | Where-Object { $_ -and (Test-Path "$_\bin\java.exe") }

$javaHome = $javaCandidates | Select-Object -First 1
if (-not $javaHome) {
    Write-Error "JDK 17+ tidak ditemukan. Set JAVA_HOME ke JDK 17/21."
}
$env:JAVA_HOME = $javaHome
$env:PATH = "$javaHome\bin;$env:PATH"

Write-Host "    [OK] ANDROID_HOME = $sdkRoot" -ForegroundColor Green
Write-Host "    [OK] ANDROID_NDK  = $ndkRoot" -ForegroundColor Green
Write-Host "    [OK] JAVA_HOME    = $javaHome" -ForegroundColor Green

Write-Host ""
Write-Host "==> Membangun frontend Vue 3..." -ForegroundColor Cyan
Push-Location "$Root\frontend"
try {
    npm run build
    if ($LASTEXITCODE -ne 0) { throw "npm run build gagal" }
} finally {
    Pop-Location
}

Write-Host ""
Write-Host "==> Menghasilkan Android overlay Wails v3..." -ForegroundColor Cyan
Push-Location $Root
try {
    wails3 android overlay:gen -out build/android/overlay.json -config build/config.yml
    if ($LASTEXITCODE -ne 0) { throw "wails3 android overlay:gen gagal" }

    $toolchainBin = "$ndkRoot\toolchains\llvm\prebuilt\windows-x86_64\bin"
    $targets = @(
        @{ GoArch = "arm64"; JniDir = "arm64-v8a"; Target = "aarch64-linux-android21" }
    )
    if ($Fat) {
        $targets += @{ GoArch = "amd64"; JniDir = "x86_64"; Target = "x86_64-linux-android21" }
    }

    $oldPath = $env:PATH
    $env:PATH = "$toolchainBin;$oldPath"

    foreach ($t in $targets) {
        Write-Host ""
        Write-Host "==> Mengompilasi Go libwails.so ($($t.GoArch))..." -ForegroundColor Cyan
        $outDir = "$Root\build\android\app\src\main\jniLibs\$($t.JniDir)"
        New-Item -ItemType Directory -Force -Path $outDir | Out-Null

        # Invoke clang.exe directly with --target so spaces in user profile paths
        # do not break cgo or NDK .cmd wrappers.
        $env:CGO_ENABLED = "1"
        $env:GOOS = "android"
        $env:GOARCH = $t.GoArch
        $env:CC = "clang.exe --target=$($t.Target)"
        $env:CXX = "clang++.exe --target=$($t.Target)"

        if ($DebugBuild) {
            go build -buildmode=c-shared -overlay build/android/overlay.json -tags android,debug -buildvcs=false -gcflags="all=-l" -o "$outDir\libwails.so" .
        } else {
            go build -buildmode=c-shared -overlay build/android/overlay.json -tags production,android -trimpath -buildvcs=false -ldflags="-w -s" -o "$outDir\libwails.so" .
        }
        if ($LASTEXITCODE -ne 0) { throw "go build android ($($t.GoArch)) gagal" }
    }
} finally {
    if ($oldPath) { $env:PATH = $oldPath }
    Remove-Item Env:CGO_ENABLED, Env:GOOS, Env:GOARCH, Env:CC, Env:CXX -ErrorAction SilentlyContinue
    Pop-Location
}

Write-Host ""
Write-Host "==> Merakit APK dengan Gradle..." -ForegroundColor Cyan
Push-Location "$Root\build\android"
try {
    $gradleTask = if ($DebugBuild) { "assembleDebug" } else { "assembleRelease" }
    & .\gradlew.bat $gradleTask
    if ($LASTEXITCODE -ne 0) { throw "gradlew $gradleTask gagal" }
} finally {
    Pop-Location
}

New-Item -ItemType Directory -Force -Path "$Root\bin" | Out-Null
$apkSrc = if ($DebugBuild) {
    "$Root\build\android\app\build\outputs\apk\debug\app-debug.apk"
} else {
    "$Root\build\android\app\build\outputs\apk\release\app-release.apk"
}
$apkDst = "$Root\bin\JPTAnnounce.apk"
Copy-Item -Path $apkSrc -Destination $apkDst -Force

$sizeMB = [math]::Round((Get-Item $apkDst).Length / 1MB, 2)
Write-Host ""
Write-Host "===============================================================" -ForegroundColor Green
Write-Host "  APK ANDROID BERHASIL DIBUAT!" -ForegroundColor Green
Write-Host "  File   : $apkDst" -ForegroundColor White
Write-Host "  Ukuran : $sizeMB MB" -ForegroundColor White
Write-Host "===============================================================" -ForegroundColor Green
