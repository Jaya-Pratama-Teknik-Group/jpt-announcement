//go:build android

package runtime

import (
	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/Jaya-Pratama-Teknik-Group/jpt-announcement/platform"
	"github.com/Jaya-Pratama-Teknik-Group/jpt-announcement/platform/android"
	androidaudio "github.com/Jaya-Pratama-Teknik-Group/jpt-announcement/platform/android/audio"
	"github.com/Jaya-Pratama-Teknik-Group/jpt-announcement/shared/domain"
)

// TrayHooks defines callbacks wired from desktop tray into shared services (unused on Android).
type TrayHooks struct {
	Icon           []byte
	OnPlayAudio    func()
	OnOpenSettings func()
	OnQuit         func()
}

// Capabilities returns runtime feature flags for experimental Android builds.
func Capabilities() domain.PlatformCapabilities {
	return domain.PlatformCapabilities{
		Platform:                    "android",
		IsMobile:                    true,
		IsDesktop:                   false,
		SupportsAutostart:           false,
		SupportsSystemTray:          false,
		SupportsBackgroundScheduler: false,
		SupportsNativeFilePicker:    false,
		SupportedAudioFormats:       []string{".mp3", ".mpeg", ".mpga", ".wav"},
	}
}

// SetupAdapters initializes Android audio playback and safe no-op desktop adapters.
func SetupAdapters(_ *application.App, _ *application.WebviewWindow, _ TrayHooks) (platform.AudioPlayer, platform.StartupManager, platform.TrayController) {
	return androidaudio.NewPlayer(), android.NoopStartupManager{}, android.NoopTrayController{}
}
