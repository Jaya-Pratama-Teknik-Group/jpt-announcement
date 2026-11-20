package audio

import (
	"sync"
	"time"

	"github.com/gopxl/beep/v2"
	"github.com/gopxl/beep/v2/speaker"
)

// TargetSampleRate is the shared speaker output sample rate used across file and live playback.
const TargetSampleRate = beep.SampleRate(44100)

var (
	speakerOnce sync.Once
	speakerErr  error
)

// EnsureSpeaker initializes the process-wide gopxl/beep speaker exactly once.
func EnsureSpeaker() error {
	speakerOnce.Do(func() {
		speakerErr = speaker.Init(TargetSampleRate, TargetSampleRate.N(time.Second/10))
	})
	return speakerErr
}

// VolumeToGain converts a 0..100 percentage into a base-2 gain value for beep effects.Volume.
func VolumeToGain(volume int) float64 {
	if volume <= 0 {
		return -8
	}
	if volume >= 100 {
		return 0
	}
	return -5.0 * (1.0 - float64(volume)/100.0)
}
