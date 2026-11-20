package platform

import (
	"context"

	"github.com/Jaya-Pratama-Teknik-Group/jpt-announcement/shared/domain"
)

// AudioPlayer defines cross-platform audio playback operations implemented per target OS.
type AudioPlayer interface {
	Play(ctx context.Context, filePath string, volume int, onFinish func(error)) error
	Pause() error
	Resume() error
	Stop() error
	IsPlaying() bool
	IsPaused() bool
}

// AudioDucker temporarily lowers other active OS audio sessions while an announcement plays.
type AudioDucker interface {
	Duck(targetPercent int) error
	Restore() error
}

// LiveMicStreamer captures audio from the default microphone and streams it live to speakers.
type LiveMicStreamer interface {
	Start(ctx context.Context, volume int, onStop func(error)) error
	Stop() error
	IsActive() bool
}

// Notifier sends OS-level interactive notifications when scheduled audio plays.
type Notifier interface {
	NotifyPlayback(scheduleTitle, audioName string, paused bool)
	DismissPlayback()
}

// StartupManager abstracts OS login autostart registration (active on Windows, no-op on Android).
type StartupManager interface {
	Enable() error
	Disable() error
	IsEnabled() (bool, error)
}

// TrayController abstracts desktop system tray updates and window visibility actions.
type TrayController interface {
	UpdateNextAnnouncement(summary string)
	ShowWindow()
	HideWindow()
	NavigateTo(route string)
}

// RuntimeHost aggregates platform-specific adapters and capability flags.
type RuntimeHost struct {
	Capabilities platformCapabilitiesProvider
}

type platformCapabilitiesProvider = domain.PlatformCapabilities
