package main

import (
	"embed"
	"log"
	"os"
	"path/filepath"
	"slices"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"

	"github.com/Jaya-Pratama-Teknik-Group/jpt-announcement/platform"
	platformrt "github.com/Jaya-Pratama-Teknik-Group/jpt-announcement/platform/runtime"
	"github.com/Jaya-Pratama-Teknik-Group/jpt-announcement/shared/announcement"
	"github.com/Jaya-Pratama-Teknik-Group/jpt-announcement/shared/config"
	"github.com/Jaya-Pratama-Teknik-Group/jpt-announcement/shared/domain"
	"github.com/Jaya-Pratama-Teknik-Group/jpt-announcement/shared/logging"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	startHidden := slices.Contains(os.Args[1:], "--hidden") || slices.Contains(os.Args[1:], "--minimized")

	store, err := config.NewStore("")
	if err != nil {
		log.Fatalf("gagal menginisialisasi penyimpanan: %v", err)
	}
	logger := logging.New(filepath.Join(store.BaseDir(), "jpt-announce.log"))

	caps := platformrt.Capabilities()
	svc := announcement.NewService(announcement.Options{
		Store:        store,
		Logger:       logger,
		Capabilities: caps,
	})

	var app *application.App
	app = application.New(application.Options{
		Name:        "JPT Pengumuman Terjadwal",
		Description: "Aplikasi Penjadwalan dan Pemutaran Pengumuman Audio",
		Services: []application.Service{
			application.NewService(svc),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		OnShutdown: func() {
			svc.Shutdown()
		},
	})

	window := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:     "JPT Pengumuman Terjadwal",
		Width:     1120,
		Height:    760,
		MinWidth:  360,
		MinHeight: 560,
		Hidden:    startHidden && caps.SupportsSystemTray,
		URL:       "/",
	})

	audioPlayer, startupMgr, trayCtrl := platformrt.SetupAdapters(app, window, platformrt.TrayHooks{
		OnPlayAudio: func() {
			if err := svc.PlayDefaultOrFirstAudio(); err != nil {
				logger.Warn("tray", "Putar audio dari tray gagal: "+err.Error())
			}
		},
		OnQuit: func() {
			svc.Shutdown()
		},
	})

	// Attach adapters into the already-registered service instance before starting scheduler.
	svc = attachRuntimeAdapters(svc, store, logger, caps, audioPlayer, startupMgr, trayCtrl, app)
	_ = svc // svc pointer registered in Services was updated in place; see helper below.

	app.Event.OnApplicationEvent(events.Common.ApplicationStarted, func(_ *application.ApplicationEvent) {
		svc.Start()
	})

	if err := app.Run(); err != nil {
		logger.Error("main", "Aplikasi berhenti dengan kesalahan: "+err.Error())
		os.Exit(1)
	}
}

func attachRuntimeAdapters(
	svc *announcement.Service,
	_ *config.Store,
	_ *logging.Logger,
	caps domain.PlatformCapabilities,
	audioPlayer platform.AudioPlayer,
	startupMgr platform.StartupManager,
	trayCtrl platform.TrayController,
	app *application.App,
) *announcement.Service {
	svc.SetAdapters(audioPlayer, startupMgr, trayCtrl)
	if caps.SupportsNativeFilePicker {
		svc.SetFilePicker(func() (string, error) {
			return app.Dialog.OpenFile().
				SetTitle("Pilih Berkas Audio Pengumuman").
				AddFilter("Berkas Audio (*.mp3;*.mpeg;*.mpga;*.wav)", "*.mp3;*.mpeg;*.mpga;*.wav").
				PromptForSingleSelection()
		})
	}
	svc.SetOnChanged(func(state domain.DashboardState) {
		app.Event.Emit("state:updated", state)
	})
	return svc
}
