package announcement

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/Jaya-Pratama-Teknik-Group/jpt-announcement/platform"
	"github.com/Jaya-Pratama-Teknik-Group/jpt-announcement/shared/config"
	"github.com/Jaya-Pratama-Teknik-Group/jpt-announcement/shared/domain"
	"github.com/Jaya-Pratama-Teknik-Group/jpt-announcement/shared/logging"
	"github.com/Jaya-Pratama-Teknik-Group/jpt-announcement/shared/scheduler"
)

// FilePickerFunc opens a native OS file dialog when supported and returns the chosen path.
type FilePickerFunc func() (string, error)

// Service orchestrates schedules, audio library, playback, settings, and platform integrations.
type Service struct {
	store            *config.Store
	logger           *logging.Logger
	audio            platform.AudioPlayer
	ducker           platform.AudioDucker
	liveMic          platform.LiveMicStreamer
	startup          platform.StartupManager
	tray             platform.TrayController
	notifier         platform.Notifier
	capabilities     domain.PlatformCapabilities
	pickFile         FilePickerFunc
	pickScheduleFile FilePickerFunc
	saveScheduleFile FilePickerFunc
	onChanged        func(domain.DashboardState)
	scheduler        *scheduler.Scheduler

	playMu   sync.RWMutex
	playSeq  uint64
	playback domain.PlaybackState
}

// Options wires platform-specific adapters into the shared application service.
type Options struct {
	Store        *config.Store
	Logger       *logging.Logger
	Audio        platform.AudioPlayer
	AudioDucker  platform.AudioDucker
	LiveMic      platform.LiveMicStreamer
	Startup      platform.StartupManager
	Tray         platform.TrayController
	Notifier     platform.Notifier
	Capabilities domain.PlatformCapabilities
	PickFile     FilePickerFunc
	OnChanged    func(domain.DashboardState)
}

// NewService constructs a ready-to-use Service and attaches the shared scheduler.
func NewService(opts Options) *Service {
	svc := &Service{
		store:        opts.Store,
		logger:       opts.Logger,
		audio:        opts.Audio,
		ducker:       opts.AudioDucker,
		liveMic:      opts.LiveMic,
		startup:      opts.Startup,
		tray:         opts.Tray,
		notifier:     opts.Notifier,
		capabilities: opts.Capabilities,
		pickFile:     opts.PickFile,
		onChanged:    opts.OnChanged,
	}
	svc.scheduler = scheduler.New(opts.Store, opts.Logger, svc.handleScheduledTrigger)
	return svc
}

// SetAdapters attaches platform-specific audio, startup, and tray adapters after app creation.
func (s *Service) SetAdapters(audio platform.AudioPlayer, startup platform.StartupManager, tray platform.TrayController) {
	s.audio = audio
	s.startup = startup
	s.tray = tray
	s.notifyChanged()
}

// SetTray attaches or replaces the tray controller after window initialization.
func (s *Service) SetTray(tray platform.TrayController) {
	s.tray = tray
	s.notifyChanged()
}

// SetNotifier attaches the platform notification controller after app creation.
func (s *Service) SetNotifier(notifier platform.Notifier) {
	s.notifier = notifier
}

// SetAudioDucker attaches the platform audio ducker after app creation.
func (s *Service) SetAudioDucker(ducker platform.AudioDucker) {
	s.ducker = ducker
}

// SetLiveMic attaches the platform live microphone streamer after app creation.
func (s *Service) SetLiveMic(liveMic platform.LiveMicStreamer) {
	s.liveMic = liveMic
}

// SetFilePicker sets the native file dialog callback.
func (s *Service) SetFilePicker(fn FilePickerFunc) {
	s.pickFile = fn
}

// SetScheduleFilePickers sets native open and save file dialog callbacks for schedule import/export.
func (s *Service) SetScheduleFilePickers(openFn, saveFn FilePickerFunc) {
	s.pickScheduleFile = openFn
	s.saveScheduleFile = saveFn
}

// SetOnChanged registers a listener invoked whenever dashboard state changes.
func (s *Service) SetOnChanged(fn func(domain.DashboardState)) {
	s.onChanged = fn
}

// Start initializes scheduler state according to persisted settings and syncs autostart status.
func (s *Service) Start() {
	cfg, err := s.store.LoadConfig()
	if err != nil {
		s.logger.Error("service", "Gagal memuat pengaturan awal: "+err.Error())
		cfg = domain.DefaultAppConfig()
	}

	if s.capabilities.SupportsAutostart && s.startup != nil {
		if actual, err := s.startup.IsEnabled(); err == nil && actual != cfg.AutostartEnabled {
			cfg.AutostartEnabled = actual
			_ = s.store.SaveConfig(cfg)
		}
	}

	if cfg.SchedulerEnabled {
		s.scheduler.Start()
	}
	s.logger.Info("service", "Aplikasi JPT Announce siap dijalankan pada platform "+s.capabilities.Platform)
	s.notifyChanged()
}

// Shutdown stops background scheduling and any active audio cleanly.
func (s *Service) Shutdown() {
	s.scheduler.Stop()
	if s.liveMic != nil {
		_ = s.liveMic.Stop()
	}
	if s.audio != nil {
		_ = s.audio.Stop()
	}
	s.restoreDucking()
	if s.notifier != nil {
		s.notifier.DismissPlayback()
	}
	s.logger.Info("service", "Aplikasi JPT Announce dihentikan")
}

// GetDashboardState returns aggregated status for the UI and tray.
func (s *Service) GetDashboardState() (domain.DashboardState, error) {
	schedules, err := s.store.ListSchedules()
	if err != nil {
		return domain.DashboardState{}, err
	}
	audios, err := s.store.ListAudio()
	if err != nil {
		return domain.DashboardState{}, err
	}
	cfg, err := s.store.LoadConfig()
	if err != nil {
		return domain.DashboardState{}, err
	}

	autostartEnabled := cfg.AutostartEnabled
	if s.capabilities.SupportsAutostart && s.startup != nil {
		if enabled, err := s.startup.IsEnabled(); err == nil {
			autostartEnabled = enabled
		}
	}

	audioNames := make(map[string]string, len(audios))
	for _, a := range audios {
		audioNames[a.ID] = a.Name
	}

	activeCount := 0
	for _, sch := range schedules {
		if sch.Enabled {
			activeCount++
		}
	}

	next := scheduler.FindNextAnnouncement(schedules, audioNames, time.Now())

	s.playMu.RLock()
	pb := s.playback
	s.playMu.RUnlock()

	return domain.DashboardState{
		Platform:            s.capabilities,
		SchedulerRunning:    s.scheduler.IsRunning(),
		AutostartEnabled:    autostartEnabled,
		ActiveScheduleCount: activeCount,
		TotalScheduleCount:  len(schedules),
		TotalAudioCount:     len(audios),
		NextAnnouncement:    next,
		Playback:            pb,
	}, nil
}

// ListSchedules returns all configured announcement schedules.
func (s *Service) ListSchedules() ([]domain.Schedule, error) {
	return s.store.ListSchedules()
}

// SaveSchedule creates or updates a schedule after verifying the referenced audio exists when assigned.
// Schedules without audio cannot be enabled until the user selects an audio file.
func (s *Service) SaveSchedule(item domain.Schedule) (domain.Schedule, error) {
	audioID := strings.TrimSpace(item.AudioID)
	if audioID != "" {
		if _, err := s.store.GetAudio(audioID); err != nil {
			return domain.Schedule{}, errors.New("berkas audio yang dipilih tidak ditemukan")
		}
	} else if item.Enabled {
		return domain.Schedule{}, errors.New("pilih berkas audio terlebih dahulu sebelum mengaktifkan jadwal")
	}
	item.AudioID = audioID

	saved, err := s.store.SaveSchedule(item)
	if err != nil {
		return domain.Schedule{}, err
	}
	s.logger.Info("jadwal", fmt.Sprintf("Jadwal %q (%s) disimpan", saved.Title, strings.Join(saved.TimesOfDay, ", ")))
	s.notifyChanged()
	return saved, nil
}

// ToggleSchedule enables or disables an existing schedule.
// Enabling requires a valid assigned audio asset.
func (s *Service) ToggleSchedule(id string, enabled bool) (domain.Schedule, error) {
	items, err := s.store.ListSchedules()
	if err != nil {
		return domain.Schedule{}, err
	}
	for _, item := range items {
		if item.ID == id {
			if enabled {
				if strings.TrimSpace(item.AudioID) == "" {
					return domain.Schedule{}, errors.New("pilih berkas audio terlebih dahulu sebelum mengaktifkan jadwal")
				}
				if _, err := s.store.GetAudio(item.AudioID); err != nil {
					return domain.Schedule{}, errors.New("berkas audio jadwal tidak ditemukan; pilih audio baru terlebih dahulu")
				}
			}
			item.Enabled = enabled
			saved, err := s.store.SaveSchedule(item)
			if err != nil {
				return domain.Schedule{}, err
			}
			status := "dinonaktifkan"
			if enabled {
				status = "diaktifkan"
			}
			s.logger.Info("jadwal", fmt.Sprintf("Jadwal %q %s", saved.Title, status))
			s.notifyChanged()
			return saved, nil
		}
	}
	return domain.Schedule{}, fmt.Errorf("jadwal %q tidak ditemukan", id)
}

// ExportSchedulesJSON serializes all configured schedules into a portable JSON bundle.
func (s *Service) ExportSchedulesJSON() (string, error) {
	schedules, err := s.store.ListSchedules()
	if err != nil {
		return "", err
	}
	if len(schedules) == 0 {
		return "", errors.New("belum ada jadwal pengumuman untuk diekspor")
	}
	audios, err := s.store.ListAudio()
	if err != nil {
		return "", err
	}
	audioNames := make(map[string]string, len(audios))
	for _, a := range audios {
		audioNames[a.ID] = a.Name
	}

	items := make([]domain.ScheduleExportItem, 0, len(schedules))
	for _, sch := range schedules {
		times := sch.EffectiveTimes()
		primary := sch.TimeOfDay
		if len(times) > 0 {
			primary = times[0]
		}
		items = append(items, domain.ScheduleExportItem{
			Title:       sch.Title,
			Description: sch.Description,
			AudioName:   audioNames[sch.AudioID],
			TimeOfDay:   primary,
			TimesOfDay:  times,
			Days:        sch.Days,
			Volume:      sch.Volume,
		})
	}

	bundle := domain.ScheduleExportBundle{
		Version:    1,
		ExportedAt: time.Now(),
		Schedules:  items,
	}
	data, err := json.MarshalIndent(bundle, "", "  ")
	if err != nil {
		return "", fmt.Errorf("gagal mengekspor konfigurasi jadwal: %w", err)
	}
	return string(data), nil
}

// ExportSchedulesToDialog opens a native save-file dialog on desktop and writes the JSON bundle.
func (s *Service) ExportSchedulesToDialog() (string, error) {
	if !s.capabilities.SupportsNativeFilePicker || s.saveScheduleFile == nil {
		return "", errors.New("dialog simpan berkas native tidak tersedia pada platform ini")
	}
	payload, err := s.ExportSchedulesJSON()
	if err != nil {
		return "", err
	}
	targetPath, err := s.saveScheduleFile()
	if err != nil {
		return "", err
	}
	if targetPath == "" {
		return "", nil
	}
	if err := os.WriteFile(targetPath, []byte(payload), 0o644); err != nil {
		return "", fmt.Errorf("gagal menyimpan berkas ekspor jadwal: %w", err)
	}
	s.logger.Info("jadwal", "Konfigurasi jadwal diekspor ke "+targetPath)
	return targetPath, nil
}

// ImportSchedulesJSON imports schedules from a portable JSON bundle or array.
// All imported schedules are created disabled with no audio assigned so the user
// must choose an audio file before activating each schedule.
func (s *Service) ImportSchedulesJSON(rawJSON string) ([]domain.Schedule, error) {
	trimmed := strings.TrimSpace(rawJSON)
	if trimmed == "" {
		return nil, errors.New("berkas konfigurasi jadwal kosong")
	}

	var entries []domain.ScheduleExportItem
	if strings.HasPrefix(trimmed, "[") {
		if err := json.Unmarshal([]byte(trimmed), &entries); err != nil {
			return nil, fmt.Errorf("format JSON jadwal tidak valid: %w", err)
		}
	} else {
		var bundle domain.ScheduleExportBundle
		if err := json.Unmarshal([]byte(trimmed), &bundle); err != nil {
			return nil, fmt.Errorf("format JSON konfigurasi jadwal tidak valid: %w", err)
		}
		entries = bundle.Schedules
	}
	if len(entries) == 0 {
		return nil, errors.New("tidak ada jadwal di dalam berkas impor")
	}

	cfg, err := s.store.LoadConfig()
	defaultVol := 85
	if err == nil && cfg.DefaultVolume >= 0 && cfg.DefaultVolume <= 100 {
		defaultVol = cfg.DefaultVolume
	}

	candidates := make([]domain.Schedule, 0, len(entries))
	for idx, item := range entries {
		vol := item.Volume
		if vol < 0 || vol > 100 {
			vol = defaultVol
		}
		times, err := domain.NormalizeScheduleTimes(item.TimeOfDay, item.TimesOfDay)
		if err != nil {
			return nil, fmt.Errorf("jadwal ke-%d tidak valid: %w", idx+1, err)
		}
		candidate := domain.Schedule{
			Title:       item.Title,
			Description: item.Description,
			AudioID:     "",
			TimeOfDay:   times[0],
			TimesOfDay:  times,
			Days:        item.Days,
			Enabled:     false,
			Volume:      vol,
		}
		if err := domain.ValidateSchedule(candidate); err != nil {
			return nil, fmt.Errorf("jadwal ke-%d tidak valid: %w", idx+1, err)
		}
		candidates = append(candidates, candidate)
	}

	imported := make([]domain.Schedule, 0, len(candidates))
	for _, candidate := range candidates {
		saved, err := s.store.SaveSchedule(candidate)
		if err != nil {
			return imported, err
		}
		imported = append(imported, saved)
	}

	s.logger.Info("jadwal", fmt.Sprintf("%d jadwal diimpor (nonaktif — pilih audio untuk mengaktifkan)", len(imported)))
	s.notifyChanged()
	return imported, nil
}

// PickAndImportSchedules opens a native open-file dialog on desktop and imports schedules from JSON.
func (s *Service) PickAndImportSchedules() ([]domain.Schedule, error) {
	if !s.capabilities.SupportsNativeFilePicker || s.pickScheduleFile == nil {
		return nil, errors.New("pemilih berkas native tidak tersedia pada platform ini")
	}
	path, err := s.pickScheduleFile()
	if err != nil {
		return nil, err
	}
	if path == "" {
		return nil, nil
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("gagal membaca berkas konfigurasi jadwal: %w", err)
	}
	return s.ImportSchedulesJSON(string(raw))
}

// DeleteSchedule removes a schedule by ID.
func (s *Service) DeleteSchedule(id string) error {
	if err := s.store.DeleteSchedule(id); err != nil {
		return err
	}
	s.logger.Info("jadwal", "Jadwal pengumuman dihapus ("+id+")")
	s.notifyChanged()
	return nil
}

// ListAudioFiles returns all imported local audio assets.
func (s *Service) ListAudioFiles() ([]domain.AudioAsset, error) {
	return s.store.ListAudio()
}

// ImportAudioFromPath copies a local .mp3 or .wav file into managed storage.
func (s *Service) ImportAudioFromPath(sourcePath, displayName string) (domain.AudioAsset, error) {
	asset, err := s.store.ImportAudioFile(sourcePath, displayName)
	if err != nil {
		return domain.AudioAsset{}, err
	}
	s.logger.Info("audio", fmt.Sprintf("Audio %q (%s) ditambahkan", asset.Name, asset.Format))
	s.notifyChanged()
	return asset, nil
}

// ImportAudioBase64 stores an audio file uploaded from the WebView file picker.
func (s *Service) ImportAudioBase64(fileName, displayName, base64Data string) (domain.AudioAsset, error) {
	raw, err := base64.StdEncoding.DecodeString(base64Data)
	if err != nil {
		return domain.AudioAsset{}, errors.New("data berkas audio tidak valid")
	}
	asset, err := s.store.ImportAudioBytes(fileName, displayName, raw)
	if err != nil {
		return domain.AudioAsset{}, err
	}
	s.logger.Info("audio", fmt.Sprintf("Audio %q (%s) diunggah", asset.Name, asset.Format))
	s.notifyChanged()
	return asset, nil
}

// PickAndImportAudio triggers the native desktop file picker when supported.
func (s *Service) PickAndImportAudio() (*domain.AudioAsset, error) {
	if !s.capabilities.SupportsNativeFilePicker || s.pickFile == nil {
		return nil, errors.New("pemilih berkas native tidak tersedia pada platform ini")
	}
	path, err := s.pickFile()
	if err != nil {
		return nil, err
	}
	if path == "" {
		return nil, nil
	}
	asset, err := s.ImportAudioFromPath(path, "")
	if err != nil {
		return nil, err
	}
	return &asset, nil
}

// DeleteAudioFile deletes an unused audio asset.
func (s *Service) DeleteAudioFile(id string) error {
	if err := s.store.DeleteAudio(id); err != nil {
		return err
	}
	s.logger.Info("audio", "Berkas audio dihapus ("+id+")")
	s.notifyChanged()
	return nil
}

// PlayAudio plays a registered audio file at the specified volume (or default config volume if < 0).
func (s *Service) PlayAudio(audioID string, volume int) error {
	return s.playAsset(audioID, "", "", volume)
}

// PlayScheduleNow triggers a schedule immediately for manual testing.
func (s *Service) PlayScheduleNow(scheduleID string) error {
	schedules, err := s.store.ListSchedules()
	if err != nil {
		return err
	}
	for _, sch := range schedules {
		if sch.ID == scheduleID {
			return s.playAsset(sch.AudioID, sch.ID, sch.Title, sch.Volume)
		}
	}
	return fmt.Errorf("jadwal %q tidak ditemukan", scheduleID)
}

// PlayDefaultOrFirstAudio plays the next upcoming schedule's audio or the first available audio file.
func (s *Service) PlayDefaultOrFirstAudio() error {
	state, err := s.GetDashboardState()
	if err == nil && state.NextAnnouncement != nil {
		return s.playAsset(state.NextAnnouncement.AudioID, state.NextAnnouncement.ScheduleID, state.NextAnnouncement.Title, -1)
	}
	audios, err := s.store.ListAudio()
	if err != nil {
		return err
	}
	if len(audios) == 0 {
		return errors.New("belum ada berkas audio yang tersimpan")
	}
	return s.playAsset(audios[0].ID, "", "", -1)
}

// PauseAudio temporarily pauses active audio playback.
func (s *Service) PauseAudio() error {
	s.playMu.RLock()
	isLive := s.playback.IsLive
	s.playMu.RUnlock()
	if isLive || s.audio == nil {
		return nil
	}
	if err := s.audio.Pause(); err != nil {
		return err
	}
	s.playMu.Lock()
	if s.playback.IsPlaying {
		s.playback.IsPaused = true
	}
	pb := s.playback
	s.playMu.Unlock()
	s.restoreDucking()
	if pb.IsPlaying && pb.ScheduleID != "" && s.notifier != nil {
		s.notifier.NotifyPlayback(pb.ScheduleTitle, pb.AudioName, true)
	}
	s.logger.Info("audio", "Pemutaran audio dijeda")
	s.notifyChanged()
	return nil
}

// ResumeAudio resumes paused audio playback.
func (s *Service) ResumeAudio() error {
	s.playMu.RLock()
	isLive := s.playback.IsLive
	shouldDuck := s.playback.IsPlaying && s.playback.IsPaused
	s.playMu.RUnlock()
	if isLive || s.audio == nil {
		return nil
	}
	if shouldDuck {
		cfg, err := s.store.LoadConfig()
		if err != nil {
			cfg = domain.DefaultAppConfig()
		}
		s.applyDucking(cfg)
	}
	if err := s.audio.Resume(); err != nil {
		if shouldDuck {
			s.restoreDucking()
		}
		return err
	}
	s.playMu.Lock()
	if s.playback.IsPlaying {
		s.playback.IsPaused = false
	}
	pb := s.playback
	s.playMu.Unlock()
	if pb.IsPlaying && pb.ScheduleID != "" && s.notifier != nil {
		s.notifier.NotifyPlayback(pb.ScheduleTitle, pb.AudioName, false)
	}
	s.logger.Info("audio", "Pemutaran audio dilanjutkan")
	s.notifyChanged()
	return nil
}

// TogglePauseAudio pauses active audio if playing, or resumes if paused.
func (s *Service) TogglePauseAudio() error {
	s.playMu.RLock()
	playing := s.playback.IsPlaying
	paused := s.playback.IsPaused
	isLive := s.playback.IsLive
	s.playMu.RUnlock()
	if !playing || isLive {
		return nil
	}
	if paused {
		return s.ResumeAudio()
	}
	return s.PauseAudio()
}

// StartLiveAnnouncement activates the microphone and streams live voice audio to speakers.
func (s *Service) StartLiveAnnouncement() error {
	if !s.capabilities.SupportsLiveAnnouncement || s.liveMic == nil {
		return errors.New("pengumuman langsung tidak didukung pada platform ini")
	}
	if s.liveMic.IsActive() {
		return nil
	}

	cfg, cfgErr := s.store.LoadConfig()
	if cfgErr != nil {
		cfg = domain.DefaultAppConfig()
	}

	now := time.Now()
	s.playMu.Lock()
	s.playSeq++
	token := s.playSeq
	s.playback = domain.PlaybackState{
		IsPlaying: true,
		IsPaused:  false,
		IsLive:    true,
		AudioName: "Pengumuman Langsung (Mikrofon)",
		StartedAt: &now,
	}
	s.playMu.Unlock()

	if s.audio != nil {
		_ = s.audio.Stop()
	}
	if s.notifier != nil {
		s.notifier.DismissPlayback()
	}
	s.notifyChanged()
	s.applyDucking(cfg)

	if err := s.liveMic.Start(context.Background(), cfg.DefaultVolume, func(stopErr error) {
		s.playMu.Lock()
		isCurrent := s.playSeq == token
		if isCurrent {
			s.playback = domain.PlaybackState{}
		}
		s.playMu.Unlock()
		if !isCurrent {
			return
		}
		if stopErr != nil {
			s.logger.Warn("audio", "Siaran langsung mikrofon terhenti: "+stopErr.Error())
		}
		s.restoreDucking()
		s.notifyChanged()
	}); err != nil {
		s.playMu.Lock()
		isCurrent := s.playSeq == token
		if isCurrent {
			s.playback = domain.PlaybackState{}
		}
		s.playMu.Unlock()
		if isCurrent {
			s.restoreDucking()
		}
		s.logger.Error("audio", "Gagal memulai pengumuman langsung: "+err.Error())
		s.notifyChanged()
		return err
	}

	s.logger.Info("audio", fmt.Sprintf("Pengumuman langsung dimulai (volume %d%%)", cfg.DefaultVolume))
	return nil
}

// StopLiveAnnouncement turns off active live microphone streaming and restores ducked audio.
func (s *Service) StopLiveAnnouncement() error {
	if s.liveMic == nil {
		return nil
	}
	s.playMu.Lock()
	wasLive := s.playback.IsLive
	if wasLive {
		s.playSeq++
		s.playback = domain.PlaybackState{}
	}
	s.playMu.Unlock()

	err := s.liveMic.Stop()
	if wasLive {
		s.restoreDucking()
		s.logger.Info("audio", "Pengumuman langsung dihentikan")
		s.notifyChanged()
	}
	return err
}

// StopAudio halts active playback or live microphone announcement immediately.
func (s *Service) StopAudio() error {
	if s.audio == nil && s.liveMic == nil {
		return nil
	}
	s.playMu.Lock()
	s.playSeq++
	wasLive := s.playback.IsLive
	s.playback = domain.PlaybackState{}
	s.playMu.Unlock()

	var err error
	if s.liveMic != nil {
		err = s.liveMic.Stop()
	}
	if s.audio != nil {
		if stopErr := s.audio.Stop(); err == nil {
			err = stopErr
		}
	}
	s.restoreDucking()
	if s.notifier != nil {
		s.notifier.DismissPlayback()
	}
	if wasLive {
		s.logger.Info("audio", "Pengumuman langsung dihentikan")
	} else {
		s.logger.Info("audio", "Pemutaran audio dihentikan")
	}
	s.notifyChanged()
	return err
}

// GetSettings returns current persistent settings synced with OS autostart state.
func (s *Service) GetSettings() (domain.AppConfig, error) {
	cfg, err := s.store.LoadConfig()
	if err != nil {
		return domain.AppConfig{}, err
	}
	if s.capabilities.SupportsAutostart && s.startup != nil {
		if enabled, err := s.startup.IsEnabled(); err == nil {
			cfg.AutostartEnabled = enabled
		}
	}
	return cfg, nil
}

// UpdateSettings persists settings and toggles scheduler / autostart accordingly.
func (s *Service) UpdateSettings(cfg domain.AppConfig) (domain.AppConfig, error) {
	if s.capabilities.SupportsAutostart && s.startup != nil {
		if _, err := s.SetAutostartEnabled(cfg.AutostartEnabled); err != nil {
			return domain.AppConfig{}, err
		}
	} else {
		cfg.AutostartEnabled = false
	}

	if err := s.store.SaveConfig(cfg); err != nil {
		return domain.AppConfig{}, err
	}
	if cfg.SchedulerEnabled {
		s.scheduler.Start()
	} else {
		s.scheduler.Stop()
	}
	s.logger.Info("pengaturan", "Pengaturan aplikasi diperbarui")
	s.notifyChanged()
	return cfg, nil
}

// SetAutostartEnabled toggles OS login autostart via platform StartupManager.
func (s *Service) SetAutostartEnabled(enabled bool) (bool, error) {
	if !s.capabilities.SupportsAutostart || s.startup == nil {
		return false, errors.New("fitur autostart hanya tersedia pada platform Windows")
	}
	var err error
	if enabled {
		err = s.startup.Enable()
	} else {
		err = s.startup.Disable()
	}
	if err != nil {
		s.logger.Error("autostart", "Gagal mengubah status autostart: "+err.Error())
		return false, err
	}
	actual, err := s.startup.IsEnabled()
	if err != nil {
		actual = enabled
	}
	cfg, cfgErr := s.store.LoadConfig()
	if cfgErr == nil {
		cfg.AutostartEnabled = actual
		_ = s.store.SaveConfig(cfg)
	}
	if actual {
		s.logger.Info("autostart", "Jalankan otomatis saat Windows dimulai diaktifkan")
	} else {
		s.logger.Info("autostart", "Jalankan otomatis saat Windows dimulai dinonaktifkan")
	}
	s.notifyChanged()
	return actual, nil
}

// ListLogs returns recent application logs newest-first.
func (s *Service) ListLogs() []domain.LogEntry {
	return s.logger.List()
}

func (s *Service) handleScheduledTrigger(sch domain.Schedule) error {
	s.logger.Info("scheduler", fmt.Sprintf("Memutar pengumuman terjadwal: %s (%s)", sch.Title, time.Now().Format("15:04")))
	return s.playAsset(sch.AudioID, sch.ID, sch.Title, sch.Volume)
}

func (s *Service) playAsset(audioID, scheduleID, scheduleTitle string, volume int) error {
	if s.audio == nil {
		return errors.New("pemutar audio tidak tersedia pada perangkat ini")
	}
	asset, err := s.store.GetAudio(audioID)
	if err != nil {
		return err
	}
	cfg, cfgErr := s.store.LoadConfig()
	if cfgErr != nil {
		cfg = domain.DefaultAppConfig()
	}
	if volume < 0 || volume > 100 {
		volume = cfg.DefaultVolume
	}

	now := time.Now()
	s.playMu.Lock()
	s.playSeq++
	token := s.playSeq
	s.playback = domain.PlaybackState{
		IsPlaying:     true,
		IsPaused:      false,
		AudioID:       asset.ID,
		AudioName:     asset.Name,
		ScheduleID:    scheduleID,
		ScheduleTitle: scheduleTitle,
		StartedAt:     &now,
	}
	s.playMu.Unlock()
	if s.liveMic != nil && s.liveMic.IsActive() {
		_ = s.liveMic.Stop()
	}
	s.notifyChanged()

	s.applyDucking(cfg)

	err = s.audio.Play(context.Background(), asset.FilePath, volume, func(playErr error) {
		if playErr != nil {
			s.logger.Error("audio", fmt.Sprintf("Kesalahan pemutaran %q: %v", asset.Name, playErr))
		} else {
			s.logger.Info("audio", fmt.Sprintf("Selesai memutar %q", asset.Name))
		}
		s.playMu.Lock()
		isCurrent := s.playSeq == token
		if isCurrent {
			s.playback = domain.PlaybackState{}
		}
		s.playMu.Unlock()
		if !isCurrent {
			return
		}
		s.restoreDucking()
		if s.notifier != nil {
			s.notifier.DismissPlayback()
		}
		s.notifyChanged()
	})
	if err != nil {
		s.playMu.Lock()
		isCurrent := s.playSeq == token
		if isCurrent {
			s.playback = domain.PlaybackState{}
		}
		s.playMu.Unlock()
		if isCurrent {
			s.restoreDucking()
			if s.notifier != nil {
				s.notifier.DismissPlayback()
			}
		}
		s.logger.Error("audio", fmt.Sprintf("Gagal memulai audio %q: %v", asset.Name, err))
		s.notifyChanged()
		return err
	}

	if scheduleID != "" && s.notifier != nil {
		s.notifier.NotifyPlayback(scheduleTitle, asset.Name, false)
	}
	s.logger.Info("audio", fmt.Sprintf("Sedang memutar %q (volume %d%%)", asset.Name, volume))
	return nil
}

func (s *Service) applyDucking(cfg domain.AppConfig) {
	if s.ducker == nil || !cfg.DuckOtherAudio {
		return
	}
	if err := s.ducker.Duck(cfg.DuckVolumePercent); err != nil {
		s.logger.Warn("audio", "Gagal mengecilkan volume aplikasi lain: "+err.Error())
	}
}

func (s *Service) restoreDucking() {
	if s.ducker == nil {
		return
	}
	if err := s.ducker.Restore(); err != nil {
		s.logger.Warn("audio", "Gagal mengembalikan volume aplikasi lain: "+err.Error())
	}
}

func (s *Service) notifyChanged() {
	state, err := s.GetDashboardState()
	if err != nil {
		return
	}
	if s.tray != nil {
		if state.NextAnnouncement != nil {
			s.tray.UpdateNextAnnouncement(fmt.Sprintf(
				"Pengumuman berikutnya: %s (%s)",
				state.NextAnnouncement.Title,
				state.NextAnnouncement.ScheduledAt.Format("15:04"),
			))
		} else {
			s.tray.UpdateNextAnnouncement("Pengumuman berikutnya: Tidak ada")
		}
	}
	if s.onChanged != nil {
		s.onChanged(state)
	}
}
