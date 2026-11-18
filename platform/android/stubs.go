//go:build android

package android

import "errors"

// NoopStartupManager isolates desktop login autostart from Android builds.
type NoopStartupManager struct{}

func (NoopStartupManager) Enable() error {
	return errors.New("autostart desktop tidak tersedia di Android")
}

func (NoopStartupManager) Disable() error {
	return nil
}

func (NoopStartupManager) IsEnabled() (bool, error) {
	return false, nil
}

// NoopTrayController isolates desktop system tray behavior from Android builds.
type NoopTrayController struct{}

func (NoopTrayController) UpdateNextAnnouncement(string) {}
func (NoopTrayController) ShowWindow()                   {}
func (NoopTrayController) HideWindow()                   {}
func (NoopTrayController) NavigateTo(string)             {}

// NoopAudioDucker is a safe no-op audio ducker on Android builds.
type NoopAudioDucker struct{}

func (NoopAudioDucker) Duck(int) error { return nil }
func (NoopAudioDucker) Restore() error { return nil }
