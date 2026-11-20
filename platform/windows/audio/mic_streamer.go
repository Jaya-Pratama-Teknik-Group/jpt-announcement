//go:build windows

package audio

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"runtime"
	"sync"
	"unsafe"

	"github.com/gopxl/beep/v2/effects"
	"github.com/gopxl/beep/v2/speaker"
	"golang.org/x/sys/windows"

	sharedaudio "github.com/Jaya-Pratama-Teknik-Group/jpt-announcement/platform/audio"
)

const (
	waveMapper     = uintptr(0xFFFFFFFF)
	callbackEvent  = uintptr(0x00050000)
	waveFormatPCM  = uint16(1)
	whdrDone       = uint32(0x00000001)
	waitObject0    = uint32(windows.WAIT_OBJECT_0)
	waitFailed     = uint32(0xFFFFFFFF)
	micSampleRate  = 44100
	micChannels    = 1
	micBits        = 16
	micChunkFrames = 882 // 20ms at 44.1 kHz
	micChunkBytes  = micChunkFrames * (micBits / 8)
	micBufferCount = 4
	ringCapacity   = micSampleRate / 2 // 0.5s bounded jitter buffer
)

var (
	winmm                   = windows.NewLazySystemDLL("winmm.dll")
	procWaveInGetNumDevs    = winmm.NewProc("waveInGetNumDevs")
	procWaveInOpen          = winmm.NewProc("waveInOpen")
	procWaveInPrepareHeader = winmm.NewProc("waveInPrepareHeader")
	procWaveInUnprepareHdr  = winmm.NewProc("waveInUnprepareHeader")
	procWaveInAddBuffer     = winmm.NewProc("waveInAddBuffer")
	procWaveInStart         = winmm.NewProc("waveInStart")
	procWaveInStop          = winmm.NewProc("waveInStop")
	procWaveInReset         = winmm.NewProc("waveInReset")
	procWaveInClose         = winmm.NewProc("waveInClose")
)

type waveFormatEx struct {
	wFormatTag      uint16
	nChannels       uint16
	nSamplesPerSec  uint32
	nAvgBytesPerSec uint32
	nBlockAlign     uint16
	wBitsPerSample  uint16
	cbSize          uint16
}

type waveHdr struct {
	lpData          uintptr
	dwBufferLength  uint32
	dwBytesRecorded uint32
	dwUser          uintptr
	dwFlags         uint32
	dwLoops         uint32
	lpNext          uintptr
	reserved        uintptr
}

type liveRingStreamer struct {
	mu     sync.Mutex
	buf    [][2]float64
	head   int
	count  int
	closed bool
}

func newLiveRingStreamer(capacity int) *liveRingStreamer {
	return &liveRingStreamer{
		buf: make([][2]float64, capacity),
	}
}

func (r *liveRingStreamer) pushPCM16Mono(pcm []byte) {
	frames := len(pcm) / 2
	if frames == 0 {
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return
	}

	capLen := len(r.buf)
	for i := 0; i < frames; i++ {
		raw := int16(binary.LittleEndian.Uint16(pcm[i*2 : i*2+2]))
		sample := float64(raw) / 32768.0
		pair := [2]float64{sample, sample}

		if r.count == capLen {
			// Drop oldest sample on overflow to keep live latency bounded.
			r.head = (r.head + 1) % capLen
			r.count--
		}
		tail := (r.head + r.count) % capLen
		r.buf[tail] = pair
		r.count++
	}
}

func (r *liveRingStreamer) close() {
	r.mu.Lock()
	r.closed = true
	r.head = 0
	r.count = 0
	r.mu.Unlock()
}

// Stream fills samples for beep/speaker without blocking; underruns output silence while open.
func (r *liveRingStreamer) Stream(samples [][2]float64) (int, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.closed {
		return 0, false
	}

	capLen := len(r.buf)
	for i := range samples {
		if r.count > 0 {
			samples[i] = r.buf[r.head]
			r.head = (r.head + 1) % capLen
			r.count--
		} else {
			samples[i] = [2]float64{0, 0}
		}
	}
	return len(samples), true
}

func (r *liveRingStreamer) Err() error { return nil }

type micSession struct {
	ring      *liveRingStreamer
	stopEvent windows.Handle
	doneCh    chan struct{}
	cancel    context.CancelFunc
}

// WindowsMicStreamer captures live audio from the default Windows microphone via winmm.dll
// and plays it through the shared gopxl/beep speaker in pure Go (CGO_ENABLED=0).
type WindowsMicStreamer struct {
	mu      sync.Mutex
	session *micSession
}

// NewMicStreamer constructs a Windows live microphone streamer.
func NewMicStreamer() *WindowsMicStreamer {
	return &WindowsMicStreamer{}
}

// Start opens the default Windows microphone and streams live audio to speakers at volume (0..100).
func (m *WindowsMicStreamer) Start(ctx context.Context, volume int, onStop func(error)) error {
	if err := sharedaudio.EnsureSpeaker(); err != nil {
		return fmt.Errorf("gagal menginisialisasi perangkat audio: %w", err)
	}

	_ = m.Stop()

	numDevs, _, _ := procWaveInGetNumDevs.Call()
	if numDevs == 0 {
		return errors.New("mikrofon tidak ditemukan pada perangkat ini")
	}

	dataEvent, err := windows.CreateEvent(nil, 0, 0, nil)
	if err != nil {
		return fmt.Errorf("gagal menyiapkan event audio mikrofon: %w", err)
	}
	stopEvent, err := windows.CreateEvent(nil, 1, 0, nil)
	if err != nil {
		_ = windows.CloseHandle(dataEvent)
		return fmt.Errorf("gagal menyiapkan event penghenti mikrofon: %w", err)
	}

	wfx := waveFormatEx{
		wFormatTag:      waveFormatPCM,
		nChannels:       micChannels,
		nSamplesPerSec:  micSampleRate,
		nAvgBytesPerSec: micSampleRate * micChannels * (micBits / 8),
		nBlockAlign:     micChannels * (micBits / 8),
		wBitsPerSample:  micBits,
		cbSize:          0,
	}

	var hWaveIn uintptr
	mmr, _, _ := procWaveInOpen.Call(
		uintptr(unsafe.Pointer(&hWaveIn)),
		waveMapper,
		uintptr(unsafe.Pointer(&wfx)),
		uintptr(dataEvent),
		0,
		callbackEvent,
	)
	if mmr != 0 {
		_ = windows.CloseHandle(stopEvent)
		_ = windows.CloseHandle(dataEvent)
		return fmt.Errorf("gagal membuka mikrofon default (kode %d)", mmr)
	}

	var pinner runtime.Pinner
	buffers := make([][]byte, micBufferCount)
	headers := make([]waveHdr, micBufferCount)
	prepared := 0

	cleanupOpen := func() {
		_, _, _ = procWaveInReset.Call(hWaveIn)
		for i := 0; i < prepared; i++ {
			_, _, _ = procWaveInUnprepareHdr.Call(hWaveIn, uintptr(unsafe.Pointer(&headers[i])), unsafe.Sizeof(headers[i]))
		}
		_, _, _ = procWaveInClose.Call(hWaveIn)
		pinner.Unpin()
		_ = windows.CloseHandle(stopEvent)
		_ = windows.CloseHandle(dataEvent)
	}

	pinner.Pin(&headers[0])
	for i := 0; i < micBufferCount; i++ {
		buf := make([]byte, micChunkBytes)
		buffers[i] = buf
		pinner.Pin(&buf[0])
		headers[i] = waveHdr{
			lpData:         uintptr(unsafe.Pointer(&buf[0])),
			dwBufferLength: uint32(len(buf)),
		}
		if r1, _, _ := procWaveInPrepareHeader.Call(hWaveIn, uintptr(unsafe.Pointer(&headers[i])), unsafe.Sizeof(headers[i])); r1 != 0 {
			cleanupOpen()
			return fmt.Errorf("gagal menyiapkan buffer mikrofon (kode %d)", r1)
		}
		prepared++
		if r1, _, _ := procWaveInAddBuffer.Call(hWaveIn, uintptr(unsafe.Pointer(&headers[i])), unsafe.Sizeof(headers[i])); r1 != 0 {
			cleanupOpen()
			return fmt.Errorf("gagal mengantrekan buffer mikrofon (kode %d)", r1)
		}
	}

	if r1, _, _ := procWaveInStart.Call(hWaveIn); r1 != 0 {
		cleanupOpen()
		return fmt.Errorf("gagal memulai perekaman mikrofon (kode %d)", r1)
	}

	ring := newLiveRingStreamer(ringCapacity)
	volEffect := &effects.Volume{
		Streamer: ring,
		Base:     2,
		Volume:   sharedaudio.VolumeToGain(volume),
		Silent:   volume <= 0,
	}

	liveCtx, cancel := context.WithCancel(ctx)
	sess := &micSession{
		ring:      ring,
		stopEvent: stopEvent,
		doneCh:    make(chan struct{}),
		cancel:    cancel,
	}

	m.mu.Lock()
	m.session = sess
	m.mu.Unlock()

	speaker.Play(volEffect)

	go func() {
		defer close(sess.doneCh)
		defer cleanupOpen()
		defer ring.close()

		stopWatchDone := make(chan struct{})
		go func() {
			select {
			case <-liveCtx.Done():
				_ = windows.SetEvent(stopEvent)
			case <-stopWatchDone:
			}
		}()

		var captureErr error
		handles := []windows.Handle{stopEvent, dataEvent}
		for {
			eventIdx, waitErr := windows.WaitForMultipleObjects(handles, false, windows.INFINITE)
			if eventIdx == waitObject0 || waitErr != nil || eventIdx == waitFailed {
				if waitErr != nil && liveCtx.Err() == nil {
					captureErr = waitErr
				}
				break
			}

			for i := 0; i < micBufferCount; i++ {
				if headers[i].dwFlags&whdrDone == 0 {
					continue
				}
				recorded := int(headers[i].dwBytesRecorded)
				if recorded >= 2 && recorded <= len(buffers[i]) {
					ring.pushPCM16Mono(buffers[i][:recorded])
				}
				headers[i].dwBytesRecorded = 0
				headers[i].dwFlags &^= whdrDone
				if r1, _, _ := procWaveInAddBuffer.Call(hWaveIn, uintptr(unsafe.Pointer(&headers[i])), unsafe.Sizeof(headers[i])); r1 != 0 {
					if liveCtx.Err() == nil {
						captureErr = fmt.Errorf("mikrofon terputus (kode %d)", r1)
					}
					break
				}
			}
			if captureErr != nil {
				break
			}
		}

		close(stopWatchDone)
		_, _, _ = procWaveInStop.Call(hWaveIn)

		m.mu.Lock()
		isCurrent := m.session == sess
		if isCurrent {
			m.session = nil
		}
		m.mu.Unlock()

		if captureErr != nil {
			speaker.Clear()
			if onStop != nil {
				onStop(captureErr)
			}
		}
	}()

	return nil
}

// Stop halts any active live microphone capture and clears the live speaker stream.
func (m *WindowsMicStreamer) Stop() error {
	m.mu.Lock()
	sess := m.session
	m.session = nil
	m.mu.Unlock()

	if sess == nil {
		return nil
	}

	sess.ring.close()
	speaker.Clear()
	sess.cancel()
	_ = windows.SetEvent(sess.stopEvent)
	<-sess.doneCh
	return nil
}

// IsActive reports whether live microphone streaming is currently running.
func (m *WindowsMicStreamer) IsActive() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.session != nil
}
