# JPT Announce

Cross-platform scheduled audio announcement application built with **Wails v3** (`v3.1.0`), **Go**, and **Vue 3 + TypeScript**.

The application UI and operational messages are in **Bahasa Indonesia**, designed with a responsive mobile-first layout for Windows desktop (primary target) and experimental Android builds from a single codebase.

## Features

- **Dashboard (`Dasbor`)**: Live scheduler status, active schedule count, audio library count, active playback status, and a real-time countdown to the next scheduled announcement.
- **Recurring Schedules (`Jadwal`)**: Create, edit, toggle, delete, and manually test announcements by time (`HH:MM`), days of the week, audio asset, and per-schedule volume (`0–100%`).
- **Local Audio Library (`Audio`)**: Import and preview local `.mp3` and `.wav` files powered by `gopxl/beep/v2` + `ebitengine/oto/v3` (pure Go on Windows; no FFmpeg binary required).
- **Windows Integration**:
  - **System Tray**: Tray icon and Bahasa Indonesia menu (*Buka JPT Announce*, *Pengumuman berikutnya*, *Putar audio*, *Pengaturan*, *Keluar*). Closing the window hides it to the tray instead of exiting.
  - **Autostart**: Optional Windows login autostart via `HKCU` registry (no administrator privileges required), launching with `--hidden` directly into the system tray.
- **Responsive Mobile-First UI**: Desktop sidebar layout and mobile bottom navigation with `viewport-fit=cover` and `env(safe-area-inset-*)` handling.
- **Activity Log**: Recent operational logs stored locally and surfaced in *Pengaturan*.

## Project Structure

```text
jpt-announcement/
├── main.go                     # Wails v3 entry point, service registration, --hidden flag
├── Taskfile.yml                # Wails v3 build/dev task entry point
├── build/                      # Wails v3 build config, icons, Windows & Android assets
├── shared/
│   ├── domain/                 # Data models & validation (Schedule, AudioAsset, AppConfig)
│   ├── config/                 # JSON persistence & managed local audio store
│   ├── scheduler/              # OS-independent time-based announcement scheduler
│   ├── announcement/           # Bound Wails v3 Service orchestrating app workflows
│   └── logging/                # Ring-buffer + file logger
├── platform/
│   ├── interfaces.go           # AudioPlayer, StartupManager, TrayController contracts
│   ├── audio/                  # Cross-platform gopxl/beep/v2 player
│   ├── runtime/                # Build-tagged platform capability & adapter wiring
│   ├── windows/                # Windows autostart, system tray, and audio adapters
│   └── android/                # Experimental Android audio adapter & no-op desktop stubs
└── frontend/                   # Vue 3 + TypeScript + Vite UI
```

## Prerequisites

- **Go** `1.26+`
- **Node.js** & **npm**
- **Wails v3 CLI** (`v3.1.0`)
- **Windows**: Microsoft Edge WebView2 Runtime (built into Windows 11; verify with `wails3 doctor`)

## Development & Build

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

### 3. Build for Windows

```powershell
wails3 build
```

This generates bindings, builds the frontend (`vue-tsc --noEmit && vite build`), embeds Windows manifest/icon resources, and outputs the GUI binary to:

```text
bin/JPTAnnounce.exe
```

You can launch it normally or minimized to the system tray:

```powershell
.\bin\JPTAnnounce.exe --hidden
```

### 4. Run Tests & Checks

```powershell
go test ./...
go vet ./...
cd frontend && npm run build
```

## Experimental Android Notes

- Android packaging tasks live under `build/android/Taskfile.yml` (via `wails3 task android:package` / `wails3 task android:bundle`), requiring Java JDK, Android SDK, and Android NDK (`build/android/Taskfile.yml` targets macOS/Linux/WSL build hosts).
- Desktop-only features (`StartupManager` and `TrayController`) are isolated behind build tags with safe no-op implementations on Android.
- **MVP Limitation**: On Android, scheduled playback works while the app is active in the foreground; background execution while minimized or screen-locked requires a native Android Foreground Service and is outside the current MVP scope.

## Local Data & Storage

Application data is stored per user under `os.UserConfigDir()/JPTAnnounce` (on Windows, `%AppData%\JPTAnnounce`):

- `config.json` — persistent application settings
- `schedules.json` — saved announcement schedules
- `audio/` — imported `.mp3` and `.wav` files
- `jpt-announce.log` — application activity log
