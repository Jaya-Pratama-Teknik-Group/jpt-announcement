package scheduler

import (
	"context"
	"sync"
	"time"

	"github.com/Jaya-Pratama-Teknik-Group/jpt-announcement/shared/domain"
	"github.com/Jaya-Pratama-Teknik-Group/jpt-announcement/shared/logging"
)

// TriggerFunc is invoked when a schedule becomes due.
type TriggerFunc func(schedule domain.Schedule) error

// ScheduleProvider supplies the current list of schedules and marks triggered timestamps.
type ScheduleProvider interface {
	ListSchedules() ([]domain.Schedule, error)
	MarkTriggered(id string, at time.Time) error
}

// Scheduler evaluates enabled announcement schedules against wall-clock time.
type Scheduler struct {
	provider ScheduleProvider
	logger   *logging.Logger
	onDue    TriggerFunc
	nowFn    func() time.Time

	mu      sync.Mutex
	running bool
	cancel  context.CancelFunc
	wg      sync.WaitGroup
}

// New creates a Scheduler with injectable dependencies.
func New(provider ScheduleProvider, logger *logging.Logger, onDue TriggerFunc) *Scheduler {
	return &Scheduler{
		provider: provider,
		logger:   logger,
		onDue:    onDue,
		nowFn:    time.Now,
	}
}

// Start begins the background 1-second evaluation loop if not already running.
func (s *Scheduler) Start() {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	s.running = true
	s.wg.Add(1)
	s.mu.Unlock()

	if s.logger != nil {
		s.logger.Info("scheduler", "Penjadwal pengumuman dimulai")
	}

	go func() {
		defer s.wg.Done()
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		// Evaluate immediately at startup so a schedule matching the launch minute is not missed.
		s.Evaluate(s.nowFn())

		for {
			select {
			case <-ctx.Done():
				return
			case t := <-ticker.C:
				s.Evaluate(t)
			}
		}
	}()
}

// Stop halts the background evaluation loop and waits for clean shutdown.
func (s *Scheduler) Stop() {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return
	}
	cancel := s.cancel
	s.running = false
	s.cancel = nil
	s.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	s.wg.Wait()

	if s.logger != nil {
		s.logger.Info("scheduler", "Penjadwal pengumuman dihentikan")
	}
}

// IsRunning reports whether the scheduler loop is currently active.
func (s *Scheduler) IsRunning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.running
}

// Evaluate checks all enabled schedules at timestamp now and triggers any due schedules once per minute.
func (s *Scheduler) Evaluate(now time.Time) {
	schedules, err := s.provider.ListSchedules()
	if err != nil {
		if s.logger != nil {
			s.logger.Error("scheduler", "Gagal memuat jadwal: "+err.Error())
		}
		return
	}

	minuteKey := now.Truncate(time.Minute)
	for _, sch := range schedules {
		if !sch.Enabled {
			continue
		}
		if !sch.MatchesWeekday(now.Weekday()) {
			continue
		}
		hour, minute, err := domain.ParseTimeOfDay(sch.TimeOfDay)
		if err != nil {
			continue
		}
		if now.Hour() != hour || now.Minute() != minute {
			continue
		}
		if sch.LastTriggeredAt != nil && sch.LastTriggeredAt.In(now.Location()).Truncate(time.Minute).Equal(minuteKey) {
			continue
		}

		// Record trigger time before invoking playback to prevent duplicate firing on slow audio start.
		_ = s.provider.MarkTriggered(sch.ID, now)
		if s.onDue != nil {
			if err := s.onDue(sch); err != nil && s.logger != nil {
				s.logger.Error("scheduler", "Gagal menjalankan jadwal "+sch.Title+": "+err.Error())
			}
		}
	}
}

// NextOccurrence computes the next firing timestamp for a single enabled schedule from `from`.
func NextOccurrence(sch domain.Schedule, from time.Time) (time.Time, bool) {
	if !sch.Enabled {
		return time.Time{}, false
	}
	hour, minute, err := domain.ParseTimeOfDay(sch.TimeOfDay)
	if err != nil {
		return time.Time{}, false
	}

	loc := from.Location()
	for offset := 0; offset <= 7; offset++ {
		day := from.AddDate(0, 0, offset)
		if !sch.MatchesWeekday(day.Weekday()) {
			continue
		}
		candidate := time.Date(day.Year(), day.Month(), day.Day(), hour, minute, 0, 0, loc)
		if candidate.Before(from) {
			continue
		}
		if sch.LastTriggeredAt != nil && sch.LastTriggeredAt.In(loc).Truncate(time.Minute).Equal(candidate.Truncate(time.Minute)) {
			continue
		}
		return candidate, true
	}
	return time.Time{}, false
}

// FindNextAnnouncement returns the earliest upcoming enabled schedule across all schedules.
func FindNextAnnouncement(schedules []domain.Schedule, audioNames map[string]string, from time.Time) *domain.NextAnnouncementInfo {
	var best *domain.NextAnnouncementInfo
	for _, sch := range schedules {
		at, ok := NextOccurrence(sch, from)
		if !ok {
			continue
		}
		remaining := int64(at.Sub(from).Seconds())
		if remaining < 0 {
			remaining = 0
		}
		if best == nil || at.Before(best.ScheduledAt) {
			best = &domain.NextAnnouncementInfo{
				ScheduleID:   sch.ID,
				Title:        sch.Title,
				AudioID:      sch.AudioID,
				AudioName:    audioNames[sch.AudioID],
				ScheduledAt:  at,
				RemainingSec: remaining,
			}
		}
	}
	return best
}
