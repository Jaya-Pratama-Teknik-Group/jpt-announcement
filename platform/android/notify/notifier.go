//go:build android

package notify

import (
	"encoding/json"
	"fmt"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// AndroidNotifier posts native Android local notifications when scheduled audio plays.
type AndroidNotifier struct{}

// NewNotifier constructs an AndroidNotifier using the Wails Android bridge.
func NewNotifier() *AndroidNotifier {
	return &AndroidNotifier{}
}

// NotifyPlayback posts a local Android notification with the schedule and audio details.
func (n *AndroidNotifier) NotifyPlayback(scheduleTitle, audioName string, paused bool) {
	title := "Pengumuman Sedang Diputar"
	if paused {
		title = "Pengumuman Dijeda"
	}
	body := fmt.Sprintf("Jadwal: %s — Audio: %s", scheduleTitle, audioName)
	if scheduleTitle == "" {
		body = fmt.Sprintf("Audio: %s", audioName)
	}
	payload, err := json.Marshal(map[string]string{
		"title": title,
		"body":  body,
	})
	if err != nil {
		return
	}
	application.Android.Notify(string(payload))
}

// DismissPlayback is a no-op on Android as local notifications auto-cancel on tap.
func (n *AndroidNotifier) DismissPlayback() {}
