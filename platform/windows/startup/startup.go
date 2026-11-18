//go:build windows

package startup

import (
	"fmt"
	"os"
	"strings"

	"golang.org/x/sys/windows/registry"

	"github.com/wailsapp/wails/v3/pkg/application"
)

const autostartIdentifier = "JPTAnnounce"
const autostartRegPath = `Software\Microsoft\Windows\CurrentVersion\Run`

// Manager wraps Wails v3 Autostart API for HKCU Windows login startup without admin rights.
type Manager struct {
	app *application.App
}

// NewManager creates a Windows startup manager backed by Wails v3 Autostart.
func NewManager(app *application.App) *Manager {
	return &Manager{app: app}
}

// Enable registers JPT Announce at Windows login with --hidden so it starts in the system tray.
// Always uses the current executable's path so moving the .exe auto-heals the registry entry.
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

// IsEnabled checks whether Windows login autostart is currently registered and
// whether the recorded path matches the running executable. If the path is stale
// (e.g. the .exe was moved after the last Enable call) it re-registers so the
// entry always points to the correct location.
func (m *Manager) IsEnabled() (bool, error) {
	enabled, err := m.app.Autostart.IsEnabled()
	if err != nil || !enabled {
		return enabled, err
	}

	// Verify the registry path is up-to-date with the current executable location.
	exePath, exeErr := os.Executable()
	if exeErr != nil {
		return true, nil // cannot determine path; assume OK
	}

	key, err := registry.OpenKey(registry.CURRENT_USER, autostartRegPath, registry.READ)
	if err != nil {
		return true, nil
	}
	defer key.Close()

	val, _, err := key.GetStringValue(autostartIdentifier)
	if err != nil {
		return true, nil
	}

	// val is typically: "C:\path\to\JPTAnnounce.exe --hidden"
	// Compare only the exe portion (before the first space that starts a flag).
	recordedExe := strings.SplitN(val, " --", 2)[0]
	if !strings.EqualFold(recordedExe, fmt.Sprintf("%s", exePath)) {
		// Path mismatch — silently re-register with current path.
		_ = m.Enable()
	}

	return true, nil
}
