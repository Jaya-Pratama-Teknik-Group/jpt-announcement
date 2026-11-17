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
	last    string
}

func (f *fakeAudio) Play(_ context.Context, filePath string, _ int, onFinish func(error)) error {
	f.mu.Lock()
	f.playing = true
	f.last = filePath
	f.mu.Unlock()
	if onFinish != nil {
		onFinish(nil)
	}
	f.mu.Lock()
	f.playing = false
	f.mu.Unlock()
	return nil
}

func (f *fakeAudio) Stop() error {
	f.mu.Lock()
	f.playing = false
	f.mu.Unlock()
	return nil
}

func (f *fakeAudio) IsPlaying() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.playing
}

type fakeStartup struct {
	enabled bool
}

func (f *fakeStartup) Enable() error          { f.enabled = true; return nil }
func (f *fakeStartup) Disable() error         { f.enabled = false; return nil }
func (f *fakeStartup) IsEnabled() (bool, error) { return f.enabled, nil }

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
