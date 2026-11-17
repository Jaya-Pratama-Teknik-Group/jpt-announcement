//go:build windows

package startup

import (
	"github.com/wailsapp/wails/v3/pkg/application"
)

const autostartIdentifier = "JPTAnnounce"

// Manager wraps Wails v3 Autostart API for HKCU Windows login startup without admin rights.
type Manager struct {
	app *application.App
}

// NewManager creates a Windows startup manager backed by Wails v3 Autostart.
func NewManager(app *application.App) *Manager {
	return &Manager{app: app}
}

// Enable registers JPT Announce at Windows login with --hidden so it starts in the system tray.
func (m *Manager) Enable() error {
	return m.app.Autostart.EnableWithOptions(application.AutostartOptions{
		Identifier: autostartIdentifier,
		Arguments:  []string{"--hidden"},
	})
}

// Disable removes the Windows login startup registration.
func (m *Manager) Disable() error {
	return m.app.Autostart.Disable()
}

// IsEnabled checks whether Windows login autostart is currently registered.
func (m *Manager) IsEnabled() (bool, error) {
	return m.app.Autostart.IsEnabled()
}
