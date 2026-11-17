//go:build windows

package notify

import (
	"fmt"
	"sync"

	"github.com/wailsapp/wails/v3/pkg/services/notifications"
)

const (
	playbackNotificationID = "jpt-playback-notification"
	playbackThreadID       = "jpt-playback"
	categoryPlaying        = "jpt_playback_playing"
	categoryPaused         = "jpt_playback_paused"
	actionPause            = "pause"
	actionResume           = "resume"
	actionStop             = "stop"
	actionOK               = "ok"
)

// Hooks defines playback control callbacks invoked from Windows Toast action buttons.
type Hooks struct {
	OnPause  func()
	OnResume func()
	OnStop   func()
}

// WindowsNotifier sends interactive Windows Toast notifications for scheduled audio playback.
type WindowsNotifier struct {
	svc   *notifications.NotificationService
	hooks Hooks

	setupOnce sync.Once
}

// NewNotifier constructs a WindowsNotifier backed by Wails' NotificationService.
func NewNotifier(svc *notifications.NotificationService, hooks Hooks) *WindowsNotifier {
	return &WindowsNotifier{
		svc:   svc,
		hooks: hooks,
	}
}

func (n *WindowsNotifier) ensureSetup() {
	if n.svc == nil {
		return
	}
	n.setupOnce.Do(func() {
		_ = n.svc.RegisterNotificationCategory(notifications.NotificationCategory{
			ID: categoryPlaying,
			Actions: []notifications.NotificationAction{
				{ID: actionPause, Title: "Jeda Audio"},
				{ID: actionOK, Title: "OK"},
			},
		})
		_ = n.svc.RegisterNotificationCategory(notifications.NotificationCategory{
			ID: categoryPaused,
			Actions: []notifications.NotificationAction{
				{ID: actionResume, Title: "Lanjutkan"},
				{ID: actionStop, Title: "Hentikan"},
			},
		})
		n.svc.OnNotificationResponse(func(result notifications.NotificationResult) {
			if result.Error != nil {
				return
			}
			switch result.Response.ActionIdentifier {
			case actionPause:
				if n.hooks.OnPause != nil {
					n.hooks.OnPause()
				}
			case actionResume:
				if n.hooks.OnResume != nil {
					n.hooks.OnResume()
				}
			case actionStop:
				if n.hooks.OnStop != nil {
					n.hooks.OnStop()
				}
			case actionOK, notifications.DefaultActionIdentifier:
				// Biarkan audio tetap berjalan sampai selesai.
			}
		})
	})
}

// NotifyPlayback displays or updates a silent Windows Toast notification with pause/OK actions.
func (n *WindowsNotifier) NotifyPlayback(scheduleTitle, audioName string, paused bool) {
	if n.svc == nil {
		return
	}
	n.ensureSetup()

	title := "Pengumuman Sedang Diputar"
	category := categoryPlaying
	body := fmt.Sprintf("Jadwal: %s\nAudio: %s", scheduleTitle, audioName)
	if scheduleTitle == "" {
		body = fmt.Sprintf("Audio: %s", audioName)
	}
	if paused {
		title = "Pengumuman Dijeda"
		category = categoryPaused
	}

	_ = n.svc.SendNotificationWithActions(notifications.NotificationOptions{
		ID:                playbackNotificationID,
		Title:             title,
		Body:              body,
		CategoryID:        category,
		ThreadID:          playbackThreadID,
		InterruptionLevel: notifications.InterruptionLevelActive,
		Sound:             &notifications.NotificationSound{Silent: true},
	})
}

// DismissPlayback removes the active playback notification once audio finishes or stops.
func (n *WindowsNotifier) DismissPlayback() {
	if n.svc == nil {
		return
	}
	_ = n.svc.RemoveNotification(playbackNotificationID)
}
