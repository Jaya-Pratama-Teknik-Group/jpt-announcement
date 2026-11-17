# JPT Announce — Rencana Teknis & Tinjauan Arsitektur (Wails v3)

Dokumen ini merangkum hasil verifikasi dokumentasi **Wails v3** (`v3.1.0` / `v3.x`), evaluasi solusi pemutaran audio lintas platform, perbedaan perilaku Windows vs Android, serta arsitektur kode tunggal untuk **JPT Announce**.

---

## 1. Verifikasi Dokumentasi Wails v3 (7 Poin Wajib)

### 1.1 Windows Autostart
- **API Resmi:** `app.Autostart` pada `github.com/wailsapp/wails/v3/pkg/application`:
  - `Enable() error` — mendaftarkan aplikasi untuk berjalan otomatis saat user login (idempoten).
  - `EnableWithOptions(opts application.AutostartOptions) error` — mendukung `Identifier` (string) dan `Arguments` (`[]string`, misalnya `[]string{"--hidden"}`).
  - `Disable() error` — menghapus pendaftaran autostart (idempoten).
  - `IsEnabled() (bool, error)` — mengecek status pendaftaran saat ini.
  - `Status() (AutostartStatus, error)` — mengembalikan status pendaftaran lengkap.
- **Mekanisme Windows:** Di balik layar menulis ke registry user-scope `HKEY_CURRENT_USER\Software\Microsoft\Windows\CurrentVersion\Run` (HKCU), sehingga **tidak memerlukan hak Administrator**.
- **Perilaku Start Minimized/Hidden:** Menggunakan `EnableWithOptions` dengan argumen `--hidden`, lalu membaca `os.Args` saat startup untuk memulai window dalam kondisi tersembunyi (`Hidden: true` / `window.Hide()`) sementara scheduler dan system tray tetap aktif.

### 1.2 Windows System Tray
- **API Resmi:** `app.SystemTray.New()`:
  - `systray.SetIcon(...)`, `systray.SetDarkModeIcon(...)`, `systray.SetLabel(...)`, `systray.SetMenu(menu)`.
  - Menu dibangun via `app.NewMenu()` dan `menu.Add("<label>").OnClick(func(ctx *application.Context) { ... })`.
- **Menahan Penutupan Window (Close-to-Tray):**
  - Listener biasa (`OnWindowEvent`) bersifat observasional dan tidak dapat membatalkan penutupan window.
  - Gunakan **Window Hook**: `window.RegisterHook(events.Common.WindowClosing, func(e *application.WindowEvent) { window.Hide(); e.Cancel() })`.
  - Tombol menu tray **"Keluar"** memanggil `app.Quit()` untuk menghentikan scheduler dan menutup aplikasi secara penuh.

### 1.3 Android Packaging (APK & AAB)
- **Status di Wails v3:** Dukungan mobile (Android & iOS) tersedia sebagai target **eksperimental** dalam satu basis kode Go + frontend.
- **Arsitektur Eksekusi:**
  - Frontend dirender melalui **Android WebView** dan disajikan secara in-process lewat `WebViewAssetLoader` yang terhubung langsung ke asset server Go (tanpa membuka port/server `localhost`).
  - Kode Go dikompilasi menjadi shared library C (`libwails.so`) menggunakan `GOOS=android` dan **Android NDK**, lalu dimuat oleh host Java/Gradle (`MainActivity`).
- **Format Output & Task CLI (`wails3 task`):**
  - **APK (pengujian lokal/perangkat):** `wails3 task android:package`, `wails3 task android:package:fat`, `wails3 task android:run`, `wails3 task android:run:device`, `wails3 task android:deploy-device`.
  - **AAB (distribusi Google Play):** `wails3 task android:assemble:aab`, `wails3 task android:assemble:aab:release`, `wails3 task android:bundle`, `wails3 task android:bundle:fat`.
  - Signing rilis menggunakan variabel lingkungan `ANDROID_KEYSTORE_FILE`, `ANDROID_KEYSTORE_PASSWORD`, `ANDROID_KEY_ALIAS`, `ANDROID_KEY_PASSWORD` (atau via Android Studio pada direktori `build/android/`).
- **Prasyarat Toolchain Android:**
  - Go 1.26+ dengan CGO aktif (terverifikasi: lingkungan saat ini menggunakan Go 1.27.4).
  - Java JDK (mis. OpenJDK 21 / JDK 11+ dengan `JAVA_HOME`).
  - Android SDK (`platform-tools`, `platforms;android-35`, `build-tools;35.0.0`) dan Android NDK (`26.3.x` / `r26d+`) dengan `ANDROID_HOME`.
  - **Catatan Limitasi Eksperimental:** Dukungan Android Wails v3 masih berstatus eksperimental; fitur desktop seperti system tray dan autostart tidak tersedia di Android dan harus diisolasi dari shared core.

### 1.4 Android Lifecycle Behavior
- Event siklus hidup Android (seperti `ActivityCreated`) dipetakan ke event terpadu Wails (`events.Common.ApplicationStarted`, dll.).
- Deteksi platform dapat dilakukan di Go via `application.System.IsAndroid()`, `application.System.IsWindows()`, `application.System.IsMobile()`, `application.System.IsDesktop()`, serta build tags (`//go:build android`, `//go:build windows`), dan di frontend via `@wailsio/runtime` (`System.IsAndroid()`, `System.IsMobile()`).
- **Batasan Penting Scheduler Latar Belakang Android:**
  - Berbeda dengan Windows (aplikasi tray residen yang dapat berjalan terus di latar belakang selama sesi login), Android secara agresif menghentikan atau membekukan proses saat aplikasi berpindah ke latar belakang atau layar terkunci (Doze / App Standby).
  - Eksekusi audio terjadwal di latar belakang Android memerlukan **Android Foreground Service**, notifikasi persisten, dan kemungkinan WakeLock / `AlarmManager` pada layer host Java/Kotlin.
  - **Keputusan MVP:** Sesuai cakupan awal, layanan background service Android **tidak** diimplementasikan pada MVP ini dan didokumentasikan sebagai ekstensi fase lanjutan di `platform/android/`.

### 1.5 Kemampuan Audio pada Android
- Di Android, kode Go (`libwails.so`) berjalan dengan CGO aktif melalui Android NDK, sehingga library audio Go yang mendukung **Oboe / AAudio / OpenSL ES** (seperti `ebitengine/oto/v3`) atau integrasi native NDK dapat memutar audio PCM langsung ke hardware audio Android.
- Sebagai alternatif atau cadangan di UI foreground Android, Android WebView juga mendukung elemen `<audio>` HTML5, namun pemutaran lewat Go di balik interface `platform` menjaga konsistensi arsitektur dengan Windows dan tidak bergantung pada kebijakan autoplay WebView.

### 1.6 Perilaku Aplikasi Responsif / Mobile
- `<meta name="viewport" content="width=device-width, initial-scale=1, viewport-fit=cover">` wajib dipasang di `index.html` agar konten dapat menyesuaikan layar penuh serta menghitung `env(safe-area-inset-*)`.
- CSS menggunakan variabel `env(safe-area-inset-top)`, `env(safe-area-inset-bottom)`, `env(safe-area-inset-left)`, `env(safe-area-inset-right)` untuk menghindari notch dan gesture bar di Android.
- Frontend (`Vue 3 + TypeScript + CSS responsif mobile-first`) memakai breakpoint adaptif:
  - **Desktop / Layar Lebar:** Sidebar navigasi, tata letak multi-kolom, tabel jadwal.
  - **Mobile / Jendela Kecil:** Navigasi bawah (bottom navigation), kartu ringkas, form bertumpuk, tombol ramah sentuhan (min-height >= 44px), dialog/drawer responsif.
  - Bahasa antarmuka seluruhnya **Bahasa Indonesia**.

### 1.7 Arsitektur Windows yang Didukung
- Wails v3 secara resmi mendukung Windows **`amd64`** dan **`arm64`** dengan runtime **WebView2**.
- Target rilis utama Windows: `JPTAnnounce.exe` (dan installer **NSIS** atau **MSIX** melalui pipeline packaging Wails v3 di `build/windows/`).

---

## 2. Evaluasi Solusi Audio Lintas Platform

| Kriteria | **`gopxl/beep/v2` + `ebitengine/oto/v3`** *(Dipilih)* | `gen2brain/malgo` (`miniaudio`) | FFmpeg (`os/exec` / binding) |
| :--- | :--- | :--- | :--- |
| **Kompatibilitas Windows** | Pure Go (`ebitengine/purego`), tanpa CGO/MinGW | Membutuhkan CGO (GCC/MinGW di Windows) | Eksternal binary / CGO |
| **Kompatibilitas Android** | Mendukung Android via Oboe (AAudio + fallback OpenSL ES) | Mendukung Android via OpenSL ES/AAudio (CGO) | Kompleks dikompilasi untuk NDK multi-ABI |
| **Dukungan MP3 / MPEG** | MP3 (MPEG-1/2/2.5 Layer III) via `hajimehoshi/go-mp3` (pure Go) | Built-in `dr_mp3` | Semua format MPEG |
| **Dukungan WAV** | Native pure Go (`beep/v2/wav`) | Built-in `dr_wav` | Ya |
| **Ukuran Aplikasi** | Ringan (~1–3 MB tambahan) | Ringan (~1–2 MB tambahan) | Sangat besar (~30–80+ MB) |
| **Lisensi** | Permisif: `beep` (MIT), `go-mp3` (Apache-2.0), `oto` (Apache-2.0) | Permisif: `malgo` (Unlicense), `miniaudio` (MIT-0 / Public Domain) | LGPL / GPL (berisiko jika static/codec paten) |
| **Keandalan Pemutaran** | Stabil: kontrol pause/resume/clear, volume, resampling terpadu | Stabil, namun callback PCM manual (lebih rendah levelnya) | Overhead proses eksternal |
| **Kemudahan Deployment** | Tidak perlu DLL/binary eksternal maupun compiler C di Windows | Butuh toolchain C saat build Windows | Harus bundel binary FFmpeg per platform |

**Kesimpulan Audio:**
Memilih **`gopxl/beep/v2`** (dengan `ebitengine/oto/v3`) di balik antarmuka `AudioPlayer` pada layer `platform/`.
- **Catatan format MPEG:** `hajimehoshi/go-mp3` mendukung standar MPEG-1/2/2.5 Layer III (`.mp3`). Untuk berkas `.wav` didukung secara native. Jika pengguna memilih berkas MPEG non-Layer-III (seperti `.mp2` atau `.m4a`/AAC), validator format akan memberikan pesan kesalahan yang jelas dalam Bahasa Indonesia tanpa menarik FFmpeg.

---

## 3. Arsitektur Paket & Pemisahan Platform

```text
jpt-announcement/
├── main.go                       # Entry point Wails v3 & inisialisasi platform
├── wails.yaml / Taskfile.yml     # Konfigurasi build Windows & Android
├── shared/
│   ├── domain/                   # Model data (Schedule, AudioAsset, AppConfig, Status)
│   ├── scheduler/                # Mesin penjadwalan berbasis waktu (independen OS)
│   ├── config/                   # Persistensi konfigurasi JSON di AppConfigDir
│   ├── announcement/             # Layanan aplikasi (CRUD jadwal, antrean, test audio)
│   └── logging/                  # Logger terstruktur ring buffer + file log
├── platform/
│   ├── interfaces.go             # Kontrak: AudioPlayer, StartupManager, TrayController
│   ├── windows/
│   │   ├── startup/              # Implementasi Wails app.Autostart
│   │   ├── tray/                 # Implementasi Wails app.SystemTray & WindowClosing hook
│   │   └── audio/                # Pemutar audio Windows (gopxl/beep/v2)
│   └── android/
│       ├── startup/              # No-op (autostart desktop tidak berlaku di Android)
│       ├── tray/                 # No-op (tray desktop tidak berlaku di Android)
│       └── audio/                # Pemutar audio Android (Oboe / isolated adapter)
└── frontend/                     # Vue 3 + TypeScript + CSS responsif mobile-first (Bahasa Indonesia)
```

---

## 4. Rencana Tahapan Implementasi

1. **Inisialisasi Proyek Wails v3 (`wails3 init`):** Menyiapkan struktur proyek Wails v3 + Vue 3 + TypeScript.
2. **Core Domain & Shared Services (`shared/`):** Model jadwal pengumuman, penyimpanan konfigurasi & berkas audio lokal, scheduler lintasan waktu, dan logger.
3. **Abstraksi Platform & Audio (`platform/`):** Antarmuka `AudioPlayer`, `StartupManager`, `TrayController` dengan implementasi Windows (`Autostart`, `SystemTray`, `WindowClosing` hook, `beep/v2`) dan isolasi build-tag untuk Android.
4. **Binding Wails v3 Service:** Mengekspos layanan pengumuman, jadwal, manajemen audio, status autostart, dan event real-time ke frontend.
5. **Frontend Responsif Mobile-First (Bahasa Indonesia):**
   - Dashboard status scheduler & pengumuman berikutnya
   - Daftar & editor jadwal (tambah/ubah/hapus/aktif-nonaktif)
   - Manajemen & uji coba audio lokal (`.mp3`, `.wav`)
   - Pengaturan aplikasi (termasuk toggle *"Jalankan otomatis saat Windows dimulai"* hanya pada platform Windows)
6. **Verifikasi Build & Uji Unit:** Pengujian unit `shared/scheduler`, `shared/config`, verifikasi `go test ./...`, kompilasi frontend, serta `wails3 build` untuk Windows.
