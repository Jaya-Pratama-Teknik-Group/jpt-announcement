# JPT Announce

Cross-platform scheduled audio announcement application built with **Wails v3** (`v3.1.0`), **Go**, and **Vue 3 + TypeScript**.

The application UI and operational messages are in **Bahasa Indonesia**, designed with a responsive mobile-first layout for **Windows desktop** (primary target) and **experimental Android** builds from a single codebase.

## Features

- **Dashboard (`Dasbor`)**: Live scheduler status, active schedule count, audio library count, active playback status, and a real-time 24-hour countdown to the next scheduled announcement.
- **Multi-Time Recurring Schedules (`Jadwal`)**:
  - Create, edit, toggle, delete, and manually test announcements with **multiple trigger times per schedule** using a dedicated **24-hour (`00:00–23:59`)** time selector, per-day-of-week selection, and per-schedule volume (`0–100%`).
  - **Preview Audio in Form**: Play/stop the selected audio directly inside the schedule editor modal using the configured volume.
  - **Export & Import Schedules (`.json`)**: Export schedules into a portable JSON bundle and import them onto another device. Imported schedules start **disabled (`Nonaktif`)** without an assigned audio file and cannot be activated until an audio asset is selected.
- **Local Audio Library (`Audio`)**: Clean list view to import, preview, and remove local `.mp3`, `.mpeg`, `.mpga`, and `.wav` files powered by `gopxl/beep/v2` + `ebitengine/oto/v3` (pure Go on Windows; no FFmpeg binary required; includes pure-Go MPEG-PS audio stream demuxing for `.mpeg` files).
- **Windows Integration**:
  - **System Tray**: Custom rounded JPT tray icon and Bahasa Indonesia menu (*Buka JPT Announce*, *Pengumuman berikutnya*, *Putar audio*, *Pengaturan*, *Keluar*). Closing the window hides it to the tray instead of exiting.
  - **Autostart**: Optional Windows login autostart via `HKCU` registry (no administrator privileges required), launching with `--hidden` directly into the system tray.
- **Responsive Mobile-First UI**: Offline **HugeIcons** (`@iconify/vue`), JPT brand logo & soft pastel menu accents, custom confirmation/form modals, desktop sidebar, and mobile bottom navigation (`viewport-fit=cover` and `env(safe-area-inset-*)`).
- **Activity Log**: Recent operational logs stored locally and surfaced in *Pengaturan*.

## Project Structure

```text
jpt-announcement/
├── main.go                     # Wails v3 entry point, embedded icon/assets, --hidden flag
├── Taskfile.yml                # Wails v3 build/dev task entry point
├── build/                      # Wails v3 build config, rounded icons, Windows & Android assets
├── shared/
│   ├── domain/                 # Data models, 24h multi-time & export validation
│   ├── config/                 # JSON persistence & managed local audio store
│   ├── scheduler/              # OS-independent multi-time announcement scheduler
│   ├── announcement/           # Bound Wails v3 Service (schedules, audio, export/import)
│   └── logging/                # Ring-buffer + file logger
├── platform/
│   ├── interfaces.go           # AudioPlayer, StartupManager, TrayController contracts
│   ├── audio/                  # Cross-platform gopxl/beep/v2 player + MPEG-PS demuxer
│   ├── runtime/                # Build-tagged platform capability & adapter wiring
│   ├── windows/                # Windows autostart, system tray, and audio adapters
│   └── android/                # Experimental Android audio adapter & no-op desktop stubs
└── frontend/                   # Vue 3 + TypeScript + Vite UI (HugeIcons + JPT branding)
```

---

## Building on Windows

### Prerequisites (Windows)

- **Go** `1.26+`
- **Node.js** & **npm**
- **Wails v3 CLI** (`v3.1.0`)
- **Microsoft Edge WebView2 Runtime** (pre-installed on Windows 11)
- Verify environment readiness with:

```powershell
wails3 doctor
```

### 1. Install Frontend Dependencies

```powershell
cd frontend
npm install
cd ..
```

### 2. Run in Development Mode

```powershell
wails3 dev
```

Starts the Vite dev server on `http://127.0.0.1:9245` with Go hot-reload. Press `F12` (`Ctrl+Shift+I`) inside the app window to open WebView2 DevTools.

### 3. Build Windows Executable (`.exe`)

```powershell
wails3 build
```

This runs `go mod tidy`, generates icons & Wails bindings, builds the frontend (`vue-tsc --noEmit && vite build`), embeds the Windows manifest/icon `.syso`, and compiles the GUI executable (`-H windowsgui`) to:

```text
bin/JPTAnnounce.exe
```

Launch normally or minimized directly to the system tray:

```powershell
.\bin\JPTAnnounce.exe
.\bin\JPTAnnounce.exe --hidden
```

*(Optional installers: `wails3 package` builds an NSIS installer if NSIS is installed, or `wails3 task windows:package FORMAT=msix` for MSIX.)*

### 4. Run Tests & Checks

```powershell
go test ./...
go vet ./...
cd frontend && npm run build
```

---

## Building for Android (APK / AAB — Experimental)

### Build Host Requirement

`build/android/Taskfile.yml` executes bash scripts and checks `uname -s` for **Linux** or **Darwin (macOS)**. On a Windows machine, run Android builds inside **WSL2 (Ubuntu/Linux)** (or on a Linux/macOS host).

### Prerequisites (inside WSL2 / Linux / macOS)

- **Go** `1.26+`, **Node.js** & **npm**, and **Wails v3 CLI** (`wails3`)
- **Java JDK 11+** (e.g., OpenJDK 17 or 21) with `JAVA_HOME` set
- **Android SDK & NDK** with `ANDROID_HOME` (or `ANDROID_SDK_ROOT`) set:
  - Android SDK platform matching `compileSdk = 36` / `targetSdk = 36` in `build/android/app/build.gradle`
  - Android NDK (e.g., `ndk;26.3.11579264` or `ANDROID_NDK_HOME` set)
  - Optional dependency helper on Linux/macOS: `wails3 task android:install:deps`

### Build APK Commands

Run from the project root inside WSL2 / Linux / macOS:

```bash
# 1. Release APK for physical Android devices (arm64-v8a) -> bin/JPTAnnounce.apk
wails3 task android:package

# 2. Universal / Fat Release APK (arm64-v8a + x86_64)
wails3 task android:package:fat

# 3. Debug APK (arm64-v8a)
wails3 task android:build ARCH=arm64
wails3 task android:assemble:apk

# 4. Build & install directly to a connected device via ADB (USB debugging)
wails3 task android:deploy-device

# 5. Play Store App Bundle (AAB)
wails3 task android:bundle
```

### Android Signing, App ID & MVP Notes

- **Signing**: Without keystore environment variables, `android:package` signs the APK using the debug keystore (suitable for local testing/sideloading). For release signing, set `ANDROID_KEYSTORE_FILE`, `ANDROID_KEYSTORE_PASSWORD`, `ANDROID_KEY_ALIAS`, and `ANDROID_KEY_PASSWORD`.
- **Package ID**: Configured as `applicationId = "com.wails.app"` in `build/android/app/build.gradle` (`APP_ID` in `build/android/Taskfile.yml`); update both before public distribution if you want a custom package ID (e.g., `id.jpt.announce`).
- **MVP Limitation**: On Android, desktop features (`StartupManager`, `TrayController`) are isolated behind build tags as safe no-ops, and scheduled announcements fire while the app is open in the foreground. Background execution while minimized or screen-locked requires a native Android Foreground Service (not yet included in the MVP).

---

## Local Data & Storage

Application data is stored per user under `os.UserConfigDir()/JPTAnnounce` (on Windows, `%AppData%\JPTAnnounce`):

- `config.json` — persistent application settings
- `schedules.json` — saved announcement schedules
- `audio/` — imported `.mp3`, `.mpeg`, `.mpga`, and `.wav` files
- `jpt-announce.log` — application activity log
