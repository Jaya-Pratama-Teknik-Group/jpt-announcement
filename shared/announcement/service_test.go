package announcement

import (
	"context"
	"encoding/base64"
	"sync"
	"testing"

	"github.com/Jaya-Pratama-Teknik-Group/jpt-announcement/shared/config"
	"github.com/Jaya-Pratama-Teknik-Group/jpt-announcement/shared/domain"
	"github.com/Jaya-Pratama-Teknik-Group/jpt-announcement/shared/logging"
)

type fakeAudio struct {
	mu      sync.Mutex
	playing bool
	paused  bool
	last    string
}

func (f *fakeAudio) Play(_ context.Context, filePath string, _ int, onFinish func(error)) error {
	f.mu.Lock()
	f.playing = true
	f.paused = false
	f.last = filePath
	f.mu.Unlock()
	if onFinish != nil {
		onFinish(nil)
	}
	f.mu.Lock()
	f.playing = false
	f.paused = false
	f.mu.Unlock()
	return nil
}

func (f *fakeAudio) Pause() error {
	f.mu.Lock()
	if f.playing {
		f.paused = true
	}
	f.mu.Unlock()
	return nil
}

func (f *fakeAudio) Resume() error {
	f.mu.Lock()
	if f.playing {
		f.paused = false
	}
	f.mu.Unlock()
	return nil
}

func (f *fakeAudio) Stop() error {
	f.mu.Lock()
	f.playing = false
	f.paused = false
	f.mu.Unlock()
	return nil
}

func (f *fakeAudio) IsPlaying() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.playing
}

func (f *fakeAudio) IsPaused() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.playing && f.paused
}

type fakeStartup struct {
	enabled bool
}

func (f *fakeStartup) Enable() error          { f.enabled = true; return nil }
func (f *fakeStartup) Disable() error         { f.enabled = false; return nil }
func (f *fakeStartup) IsEnabled() (bool, error) { return f.enabled, nil }

type fakeDucker struct {
	mu           sync.Mutex
	duckCalls    []int
	restoreCalls int
	active       bool
}

func (f *fakeDucker) Duck(targetPercent int) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.duckCalls = append(f.duckCalls, targetPercent)
	f.active = true
	return nil
}

func (f *fakeDucker) Restore() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.restoreCalls++
	f.active = false
	return nil
}

type asyncAudio struct {
	mu       sync.Mutex
	playing  bool
	paused   bool
	onFinish func(error)
}

func (a *asyncAudio) Play(_ context.Context, _ string, _ int, onFinish func(error)) error {
	a.mu.Lock()
	a.playing = true
	a.paused = false
	a.onFinish = onFinish
	a.mu.Unlock()
	return nil
}

func (a *asyncAudio) Pause() error {
	a.mu.Lock()
	if a.playing {
		a.paused = true
	}
	a.mu.Unlock()
	return nil
}

func (a *asyncAudio) Resume() error {
	a.mu.Lock()
	if a.playing {
		a.paused = false
	}
	a.mu.Unlock()
	return nil
}

func (a *asyncAudio) Stop() error {
	a.mu.Lock()
	a.playing = false
	a.paused = false
	a.onFinish = nil
	a.mu.Unlock()
	return nil
}

func (a *asyncAudio) IsPlaying() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.playing
}

func (a *asyncAudio) IsPaused() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.playing && a.paused
}

func (a *asyncAudio) finish() {
	a.mu.Lock()
	cb := a.onFinish
	a.playing = false
	a.paused = false
	a.onFinish = nil
	a.mu.Unlock()
	if cb != nil {
		cb(nil)
	}
}

func TestServiceLifecycleAndAutostart(t *testing.T) {
	store, err := config.NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	audio := &fakeAudio{}
	startup := &fakeStartup{}

	svc := NewService(Options{
		Store:   store,
		Logger:  logging.New(""),
		Audio:   audio,
		Startup: startup,
		Capabilities: domain.PlatformCapabilities{
			Platform:          "windows",
			IsDesktop:         true,
			SupportsAutostart: true,
		},
	})

	svc.Start()
	defer svc.Shutdown()

	enabled, err := svc.SetAutostartEnabled(true)
	if err != nil || !enabled || !startup.enabled {
		t.Fatalf("expected autostart enabled, got enabled=%v err=%v", enabled, err)
	}

	b64 := base64.StdEncoding.EncodeToString([]byte("RIFFmockwav"))
	asset, err := svc.ImportAudioBase64("bel.wav", "Bel Sekolah", b64)
	if err != nil {
		t.Fatalf("ImportAudioBase64 failed: %v", err)
	}

	sch, err := svc.SaveSchedule(domain.Schedule{
		Title:     "Bel Istirahat",
		AudioID:   asset.ID,
		TimeOfDay: "10:00",
		Enabled:   true,
		Volume:    90,
	})
	if err != nil {
		t.Fatalf("SaveSchedule failed: %v", err)
	}

	if err := svc.PlayScheduleNow(sch.ID); err != nil {
		t.Fatalf("PlayScheduleNow failed: %v", err)
	}
	if audio.last != asset.FilePath {
		t.Fatalf("expected audio path %s, got %s", asset.FilePath, audio.last)
	}

	state, err := svc.GetDashboardState()
	if err != nil {
		t.Fatalf("GetDashboardState failed: %v", err)
	}
	if state.ActiveScheduleCount != 1 || state.TotalAudioCount != 1 {
		t.Fatalf("unexpected dashboard state: %+v", state)
	}

	// Export schedules and import them back into a fresh service
	exported, err := svc.ExportSchedulesJSON()
	if err != nil {
		t.Fatalf("ExportSchedulesJSON failed: %v", err)
	}

	store2, err := config.NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewStore(2) failed: %v", err)
	}
	svc2 := NewService(Options{
		Store:  store2,
		Logger: logging.New(""),
	})
	imported, err := svc2.ImportSchedulesJSON(exported)
	if err != nil {
		t.Fatalf("ImportSchedulesJSON failed: %v", err)
	}
	if len(imported) != 1 {
		t.Fatalf("expected 1 imported schedule, got %d", len(imported))
	}
	if imported[0].Enabled || imported[0].AudioID != "" {
		t.Fatalf("imported schedule must be disabled with empty AudioID, got %+v", imported[0])
	}

	// Enabling imported schedule without selecting audio must fail
	if _, err := svc2.ToggleSchedule(imported[0].ID, true); err == nil {
		t.Fatal("expected ToggleSchedule(true) to fail when AudioID is empty")
	}

	// After importing/selecting an audio, schedule can be assigned and enabled
	asset2, err := svc2.ImportAudioBase64("bel2.wav", "Bel Baru", b64)
	if err != nil {
		t.Fatalf("ImportAudioBase64(2) failed: %v", err)
	}
	imported[0].AudioID = asset2.ID
	imported[0].Enabled = true
	updated, err := svc2.SaveSchedule(imported[0])
	if err != nil {
		t.Fatalf("SaveSchedule after choosing audio failed: %v", err)
	}
	if !updated.Enabled || updated.AudioID != asset2.ID {
		t.Fatalf("expected enabled schedule with AudioID %s, got %+v", asset2.ID, updated)
	}
}

func TestAudioDuckingLifecycle(t *testing.T) {
	store, err := config.NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	audio := &asyncAudio{}
	ducker := &fakeDucker{}

	svc := NewService(Options{
		Store:       store,
		Logger:      logging.New(""),
		Audio:       audio,
		AudioDucker: ducker,
		Capabilities: domain.PlatformCapabilities{
			Platform:             "windows",
			IsDesktop:            true,
			SupportsAudioDucking: true,
		},
	})
	defer svc.Shutdown()

	b64 := base64.StdEncoding.EncodeToString([]byte("RIFFmockwav"))
	asset, err := svc.ImportAudioBase64("bel.wav", "Bel", b64)
	if err != nil {
		t.Fatalf("ImportAudioBase64 failed: %v", err)
	}

	if err := svc.PlayAudio(asset.ID, 90); err != nil {
		t.Fatalf("PlayAudio failed: %v", err)
	}
	if len(ducker.duckCalls) != 1 || ducker.duckCalls[0] != 25 || !ducker.active {
		t.Fatalf("expected duck(25) active, got calls=%v active=%v", ducker.duckCalls, ducker.active)
	}

	if err := svc.PauseAudio(); err != nil {
		t.Fatalf("PauseAudio failed: %v", err)
	}
	if ducker.active || ducker.restoreCalls != 1 {
		t.Fatalf("expected restore on pause, got active=%v restores=%d", ducker.active, ducker.restoreCalls)
	}

	if err := svc.ResumeAudio(); err != nil {
		t.Fatalf("ResumeAudio failed: %v", err)
	}
	if !ducker.active || len(ducker.duckCalls) != 2 {
		t.Fatalf("expected re-duck on resume, got active=%v calls=%v", ducker.active, ducker.duckCalls)
	}

	audio.finish()
	if ducker.active || ducker.restoreCalls != 2 {
		t.Fatalf("expected restore on finish, got active=%v restores=%d", ducker.active, ducker.restoreCalls)
	}

	// Disable ducking and verify PlayAudio does not duck
	cfg, err := svc.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings failed: %v", err)
	}
	cfg.DuckOtherAudio = false
	cfg.DuckVolumePercent = 15
	if _, err := svc.UpdateSettings(cfg); err != nil {
		t.Fatalf("UpdateSettings failed: %v", err)
	}
	if err := svc.PlayAudio(asset.ID, 90); err != nil {
		t.Fatalf("PlayAudio(disabled ducking) failed: %v", err)
	}
	if len(ducker.duckCalls) != 2 {
		t.Fatalf("expected no additional duck call when disabled, got %v", ducker.duckCalls)
	}
	if err := svc.StopAudio(); err != nil {
		t.Fatalf("StopAudio failed: %v", err)
	}
}

type fakeLiveMic struct {
	mu     sync.Mutex
	active bool
	volume int
	stops  int
	onStop func(error)
}

func (f *fakeLiveMic) Start(_ context.Context, volume int, onStop func(error)) error {
	f.mu.Lock()
	f.active = true
	f.volume = volume
	f.onStop = onStop
	f.mu.Unlock()
	return nil
}

func (f *fakeLiveMic) Stop() error {
	f.mu.Lock()
	if f.active {
		f.active = false
		f.stops++
	}
	f.mu.Unlock()
	return nil
}

func (f *fakeLiveMic) IsActive() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.active
}

func TestLiveAnnouncementLifecycle(t *testing.T) {
	store, err := config.NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	audio := &asyncAudio{}
	ducker := &fakeDucker{}
	liveMic := &fakeLiveMic{}

	svc := NewService(Options{
		Store:       store,
		Logger:      logging.New(""),
		Audio:       audio,
		AudioDucker: ducker,
		LiveMic:     liveMic,
		Capabilities: domain.PlatformCapabilities{
			Platform:                 "windows",
			IsDesktop:                true,
			SupportsAudioDucking:     true,
			SupportsLiveAnnouncement: true,
		},
	})
	defer svc.Shutdown()

	if err := svc.StartLiveAnnouncement(); err != nil {
		t.Fatalf("StartLiveAnnouncement failed: %v", err)
	}
	if !liveMic.IsActive() || liveMic.volume != 85 || !ducker.active {
		t.Fatalf("expected active live mic + ducker, got active=%v vol=%d ducker=%v", liveMic.IsActive(), liveMic.volume, ducker.active)
	}
	state, err := svc.GetDashboardState()
	if err != nil {
		t.Fatalf("GetDashboardState failed: %v", err)
	}
	if !state.Playback.IsPlaying || !state.Playback.IsLive {
		t.Fatalf("expected playback IsPlaying && IsLive, got %+v", state.Playback)
	}

	// PauseAudio is a no-op during live mic streaming
	if err := svc.PauseAudio(); err != nil {
		t.Fatalf("PauseAudio failed: %v", err)
	}
	state, _ = svc.GetDashboardState()
	if state.Playback.IsPaused || !state.Playback.IsLive {
		t.Fatalf("expected live playback to stay unpaused, got %+v", state.Playback)
	}

	if err := svc.StopLiveAnnouncement(); err != nil {
		t.Fatalf("StopLiveAnnouncement failed: %v", err)
	}
	if liveMic.IsActive() || ducker.active || ducker.restoreCalls != 1 {
		t.Fatalf("expected live mic stopped and ducker restored, got active=%v ducker=%v restores=%d", liveMic.IsActive(), ducker.active, ducker.restoreCalls)
	}

	// Starting file playback stops active live mic
	b64 := base64.StdEncoding.EncodeToString([]byte("RIFFmockwav"))
	asset, err := svc.ImportAudioBase64("bel.wav", "Bel", b64)
	if err != nil {
		t.Fatalf("ImportAudioBase64 failed: %v", err)
	}
	if err := svc.StartLiveAnnouncement(); err != nil {
		t.Fatalf("StartLiveAnnouncement(2) failed: %v", err)
	}
	if err := svc.PlayAudio(asset.ID, 80); err != nil {
		t.Fatalf("PlayAudio while live failed: %v", err)
	}
	if liveMic.IsActive() {
		t.Fatal("expected PlayAudio to stop active live mic")
	}
	audio.finish()

	// Unsupported platform rejects live announcement
	androidSvc := NewService(Options{
		Store:  store,
		Logger: logging.New(""),
		Capabilities: domain.PlatformCapabilities{
			Platform:                 "android",
			SupportsLiveAnnouncement: false,
		},
	})
	if err := androidSvc.StartLiveAnnouncement(); err == nil {
		t.Fatal("expected unsupported platform error for StartLiveAnnouncement")
	}
}
