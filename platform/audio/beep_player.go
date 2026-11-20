package audio

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/gopxl/beep/v2"
	"github.com/gopxl/beep/v2/effects"
	"github.com/gopxl/beep/v2/mp3"
	"github.com/gopxl/beep/v2/speaker"
	"github.com/gopxl/beep/v2/wav"
)

const targetSampleRate = TargetSampleRate

// BeepPlayer implements platform.AudioPlayer using gopxl/beep/v2 (pure Go on Windows, Oboe on Android).
type BeepPlayer struct {
	mu      sync.Mutex
	playing bool
	paused  bool
	current beep.StreamCloser
	ctrl    *beep.Ctrl
	cancel  context.CancelFunc
}

// NewBeepPlayer constructs a reusable cross-platform beep audio player.
func NewBeepPlayer() *BeepPlayer {
	return &BeepPlayer{}
}

func (p *BeepPlayer) ensureSpeaker() error {
	return EnsureSpeaker()
}

// Play decodes an MP3 or WAV file and plays it asynchronously, replacing any active playback.
func (p *BeepPlayer) Play(ctx context.Context, filePath string, volume int, onFinish func(error)) error {
	if err := p.ensureSpeaker(); err != nil {
		return fmt.Errorf("gagal menginisialisasi perangkat audio: %w", err)
	}

	_ = p.Stop()

	f, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("gagal membuka berkas audio: %w", err)
	}

	var streamSource io.ReadCloser = f
	var streamer beep.StreamCloser
	var format beep.Format
	switch strings.ToLower(filepath.Ext(filePath)) {
	case ".mp3", ".mpeg", ".mpga":
		streamSource, err = prepareMPEGAudioStream(f)
		if err != nil {
			_ = f.Close()
			return fmt.Errorf("gagal menyiapkan berkas audio MPEG: %w", err)
		}
		streamer, format, err = mp3.Decode(streamSource)
	case ".wav":
		streamer, format, err = wav.Decode(f)
	default:
		_ = f.Close()
		return fmt.Errorf("format audio %q tidak didukung", filepath.Ext(filePath))
	}
	if err != nil {
		_ = streamSource.Close()
		return fmt.Errorf("gagal mendekode berkas audio (pastikan berkas .mp3/.mpeg berformat MPEG Layer III atau .wav): %w", err)
	}

	var stream beep.Streamer = streamer
	if format.SampleRate != targetSampleRate {
		stream = beep.Resample(3, format.SampleRate, targetSampleRate, streamer)
	}

	volEffect := &effects.Volume{
		Streamer: stream,
		Base:     2,
		Volume:   volumeToGain(volume),
		Silent:   volume <= 0,
	}
	ctrl := &beep.Ctrl{
		Streamer: volEffect,
		Paused:   false,
	}

	playCtx, cancel := context.WithCancel(ctx)

	p.mu.Lock()
	p.playing = true
	p.paused = false
	p.current = streamer
	p.ctrl = ctrl
	p.cancel = cancel
	p.mu.Unlock()

	done := make(chan struct{})
	speaker.Play(beep.Seq(ctrl, beep.Callback(func() {
		close(done)
	})))

	go func() {
		defer func() {
			_ = streamer.Close()
			_ = streamSource.Close()
		}()

		select {
		case <-done:
		case <-playCtx.Done():
			speaker.Clear()
		}

		p.mu.Lock()
		wasPlaying := p.playing
		p.playing = false
		p.paused = false
		if p.current == streamer {
			p.current = nil
		}
		if p.ctrl == ctrl {
			p.ctrl = nil
		}
		p.cancel = nil
		p.mu.Unlock()

		if wasPlaying && onFinish != nil {
			onFinish(nil)
		}
	}()

	return nil
}

// Pause temporarily pauses active audio playback without clearing the stream.
func (p *BeepPlayer) Pause() error {
	p.mu.Lock()
	ctrl := p.ctrl
	playing := p.playing
	if playing && ctrl != nil {
		p.paused = true
	}
	p.mu.Unlock()

	if !playing || ctrl == nil {
		return nil
	}
	speaker.Lock()
	ctrl.Paused = true
	speaker.Unlock()
	return nil
}

// Resume continues paused audio playback from the current position.
func (p *BeepPlayer) Resume() error {
	p.mu.Lock()
	ctrl := p.ctrl
	playing := p.playing
	if playing && ctrl != nil {
		p.paused = false
	}
	p.mu.Unlock()

	if !playing || ctrl == nil {
		return nil
	}
	speaker.Lock()
	ctrl.Paused = false
	speaker.Unlock()
	return nil
}

// Stop halts any active audio stream cleanly.
func (p *BeepPlayer) Stop() error {
	p.mu.Lock()
	cancel := p.cancel
	p.playing = false
	p.paused = false
	p.ctrl = nil
	p.cancel = nil
	p.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	speaker.Clear()
	return nil
}

// IsPlaying reports whether audio playback is currently active.
func (p *BeepPlayer) IsPlaying() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.playing
}

// IsPaused reports whether active audio playback is currently paused.
func (p *BeepPlayer) IsPaused() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.playing && p.paused
}

func volumeToGain(volume int) float64 {
	return VolumeToGain(volume)
}
