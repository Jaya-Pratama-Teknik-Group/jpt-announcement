//go:build android

package runtime

import (
	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/Jaya-Pratama-Teknik-Group/jpt-announcement/platform"
	"github.com/Jaya-Pratama-Teknik-Group/jpt-announcement/platform/android"
	androidaudio "github.com/Jaya-Pratama-Teknik-Group/jpt-announcement/platform/android/audio"
	androidnotify "github.com/Jaya-Pratama-Teknik-Group/jpt-announcement/platform/android/notify"
	"github.com/Jaya-Pratama-Teknik-Group/jpt-announcement/shared/domain"
)

// TrayHooks defines callbacks wired from desktop tray into shared services (unused on Android).
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

// ExtraServices returns platform-specific Wails services to register at app startup.
func ExtraServices() []application.Service {
	return nil
}

// SetupNotifier constructs the Android local notification adapter.
func SetupNotifier(_ NotifyHooks) platform.Notifier {
	return androidnotify.NewNotifier()
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
		SupportsAudioDucking:        false,
		SupportsLiveAnnouncement:    false,
		SupportedAudioFormats:       []string{".mp3", ".mpeg", ".mpga", ".wav"},
	}
}

// SetupAudioDucker returns a safe no-op audio ducker on Android.
func SetupAudioDucker() platform.AudioDucker {
	return android.NoopAudioDucker{}
}

// SetupLiveMic returns a safe no-op live microphone streamer on Android.
func SetupLiveMic() platform.LiveMicStreamer {
	return android.NoopLiveMicStreamer{}
}

// SetupAdapters initializes Android audio playback and safe no-op desktop adapters.
func SetupAdapters(_ *application.App, _ *application.WebviewWindow, _ TrayHooks) (platform.AudioPlayer, platform.StartupManager, platform.TrayController) {
	return androidaudio.NewPlayer(), android.NoopStartupManager{}, android.NoopTrayController{}
}
