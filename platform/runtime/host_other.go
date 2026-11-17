//go:build !windows && !android

package runtime

import (
	"errors"

	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/Jaya-Pratama-Teknik-Group/jpt-announcement/platform"
	sharedaudio "github.com/Jaya-Pratama-Teknik-Group/jpt-announcement/platform/audio"
	"github.com/Jaya-Pratama-Teknik-Group/jpt-announcement/shared/domain"
)

// TrayHooks defines callbacks wired from desktop tray into shared services.
type TrayHooks struct {
	OnPlayAudio    func()
	OnOpenSettings func()
	OnQuit         func()
}

type unsupportedStartup struct{}

func (unsupportedStartup) Enable() error          { return errors.New("autostart hanya didukung di Windows") }
func (unsupportedStartup) Disable() error         { return nil }
func (unsupportedStartup) IsEnabled() (bool, error) { return false, nil }

type noopTray struct {
	window *application.WebviewWindow
	app    *application.App
}

func (n noopTray) UpdateNextAnnouncement(string) {}
func (n noopTray) ShowWindow() {
	if n.window != nil {
		n.window.Show()
		n.window.Focus()
	}
}
func (n noopTray) HideWindow() {
	if n.window != nil {
		n.window.Hide()
	}
}
func (n noopTray) NavigateTo(route string) {
	if n.app != nil {
		n.app.Event.Emit("app:navigate", route)
	}
}

// Capabilities returns runtime feature flags for non-Windows desktop hosts.
func Capabilities() domain.PlatformCapabilities {
	return domain.PlatformCapabilities{
		Platform:                    "desktop",
		IsMobile:                    false,
		IsDesktop:                   true,
		SupportsAutostart:           false,
		SupportsSystemTray:          false,
		SupportsBackgroundScheduler: true,
		SupportsNativeFilePicker:    true,
		SupportedAudioFormats:       []string{".mp3", ".mpeg", ".mpga", ".wav"},
	}
}

// SetupAdapters initializes fallback desktop adapters.
func SetupAdapters(app *application.App, window *application.WebviewWindow, _ TrayHooks) (platform.AudioPlayer, platform.StartupManager, platform.TrayController) {
	return sharedaudio.NewBeepPlayer(), unsupportedStartup{}, noopTray{window: window, app: app}
}
