//go:build windows

package tray

import (
	"sync"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
	"github.com/wailsapp/wails/v3/pkg/icons"
)

// Actions defines callbacks invoked by the Windows system tray menu.
type Actions struct {
	OnPlayAudio    func()
	OnOpenSettings func()
	OnQuit         func()
}

// Controller manages the Windows system tray icon, dynamic menu, and close-to-tray window hook.
type Controller struct {
	app      *application.App
	window   *application.WebviewWindow
	systray  *application.SystemTray
	menu     *application.Menu
	nextItem *application.MenuItem
	mu       sync.Mutex
}

// NewController initializes the Windows system tray and attaches the WindowClosing close-to-tray hook.
func NewController(app *application.App, window *application.WebviewWindow, actions Actions) *Controller {
	c := &Controller{
		app:    app,
		window: window,
	}

	window.RegisterHook(events.Common.WindowClosing, func(e *application.WindowEvent) {
		window.Hide()
		e.Cancel()
	})

	systray := app.SystemTray.New()
	systray.SetLabel("JPT Announce")
	systray.SetIcon(icons.SystrayLight)
	systray.SetDarkModeIcon(icons.SystrayDark)

	menu := app.NewMenu()

	menu.Add("Buka JPT Announce").OnClick(func(_ *application.Context) {
		c.ShowWindow()
	})

	c.nextItem = menu.Add("Pengumuman berikutnya: Tidak ada")
	c.nextItem.SetEnabled(false)

	menu.Add("Putar audio").OnClick(func(_ *application.Context) {
		if actions.OnPlayAudio != nil {
			actions.OnPlayAudio()
		}
	})

	menu.Add("Pengaturan").OnClick(func(_ *application.Context) {
		c.ShowWindow()
		c.NavigateTo("settings")
		if actions.OnOpenSettings != nil {
			actions.OnOpenSettings()
		}
	})

	menu.AddSeparator()

	menu.Add("Keluar").OnClick(func(_ *application.Context) {
		if actions.OnQuit != nil {
			actions.OnQuit()
		}
		app.Quit()
	})

	systray.SetMenu(menu)
	systray.OnClick(func() {
		c.ShowWindow()
	})

	c.systray = systray
	c.menu = menu
	return c
}

// UpdateNextAnnouncement refreshes the "Pengumuman berikutnya" tray menu label.
func (c *Controller) UpdateNextAnnouncement(summary string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.nextItem != nil && c.menu != nil {
		c.nextItem.SetLabel(summary)
		c.menu.Update()
	}
}

// ShowWindow restores and focuses the main application window.
func (c *Controller) ShowWindow() {
	if c.window != nil {
		c.window.Show()
		c.window.Focus()
	}
}

// HideWindow hides the main application window to the system tray.
func (c *Controller) HideWindow() {
	if c.window != nil {
		c.window.Hide()
	}
}

// NavigateTo emits an event to switch the active frontend view.
func (c *Controller) NavigateTo(route string) {
	if c.app != nil {
		c.app.Event.Emit("app:navigate", route)
	}
}
