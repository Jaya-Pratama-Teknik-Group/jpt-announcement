//go:build windows

package runtime

import (
	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/Jaya-Pratama-Teknik-Group/jpt-announcement/platform"
	winaudio "github.com/Jaya-Pratama-Teknik-Group/jpt-announcement/platform/windows/audio"
	winstartup "github.com/Jaya-Pratama-Teknik-Group/jpt-announcement/platform/windows/startup"
	wintray "github.com/Jaya-Pratama-Teknik-Group/jpt-announcement/platform/windows/tray"
	"github.com/Jaya-Pratama-Teknik-Group/jpt-announcement/shared/domain"
)

// TrayHooks defines callbacks wired from the platform tray into shared services.
type TrayHooks struct {
	Icon           []byte
	OnPlayAudio    func()
	OnOpenSettings func()
	OnQuit         func()
}

// Capabilities returns runtime feature flags for Windows.
func Capabilities() domain.PlatformCapabilities {
	return domain.PlatformCapabilities{
		Platform:                    "windows",
		IsMobile:                    false,
		IsDesktop:                   true,
		SupportsAutostart:           true,
		SupportsSystemTray:          true,
		SupportsBackgroundScheduler: true,
		SupportsNativeFilePicker:    true,
		SupportedAudioFormats:       []string{".mp3", ".mpeg", ".mpga", ".wav"},
	}
}

// SetupAdapters initializes Windows audio, autostart, and system tray adapters.
func SetupAdapters(app *application.App, window *application.WebviewWindow, hooks TrayHooks) (platform.AudioPlayer, platform.StartupManager, platform.TrayController) {
	audioPlayer := winaudio.NewPlayer()
	startupMgr := winstartup.NewManager(app)
	trayCtrl := wintray.NewController(app, window, wintray.Actions{
		Icon:           hooks.Icon,
		OnPlayAudio:    hooks.OnPlayAudio,
		OnOpenSettings: hooks.OnOpenSettings,
		OnQuit:         hooks.OnQuit,
	})
	return audioPlayer, startupMgr, trayCtrl
}
