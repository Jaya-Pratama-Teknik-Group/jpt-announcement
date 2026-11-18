//go:build windows

package runtime

import (
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/services/notifications"

	"github.com/Jaya-Pratama-Teknik-Group/jpt-announcement/platform"
	winaudio "github.com/Jaya-Pratama-Teknik-Group/jpt-announcement/platform/windows/audio"
	winnotify "github.com/Jaya-Pratama-Teknik-Group/jpt-announcement/platform/windows/notify"
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

// NotifyHooks defines playback control callbacks wired from OS notification actions.
type NotifyHooks struct {
	OnPause  func()
	OnResume func()
	OnStop   func()
}

var winNotificationService = notifications.New()

// ExtraServices returns platform-specific Wails services to register at app startup.
func ExtraServices() []application.Service {
	return []application.Service{
		application.NewService(winNotificationService),
	}
}

// SetupNotifier constructs the Windows Toast notification adapter.
func SetupNotifier(hooks NotifyHooks) platform.Notifier {
	return winnotify.NewNotifier(winNotificationService, winnotify.Hooks{
		OnPause:  hooks.OnPause,
		OnResume: hooks.OnResume,
		OnStop:   hooks.OnStop,
	})
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
		SupportsAudioDucking:        true,
		SupportedAudioFormats:       []string{".mp3", ".mpeg", ".mpga", ".wav"},
	}
}

// SetupAudioDucker constructs the Windows WASAPI audio ducker.
func SetupAudioDucker() platform.AudioDucker {
	return winaudio.NewDucker()
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
