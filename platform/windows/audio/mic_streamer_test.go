//go:build windows

package audio

import (
	"encoding/binary"
	"math"
	"testing"
	"unsafe"
)

func TestWaveStructSizes(t *testing.T) {
	// Win32 WAVEFORMATEX fields sit at offsets 0..16 (Go pads struct tail to 4-byte alignment = 20).
	var wfx waveFormatEx
	if got := unsafe.Offsetof(wfx.cbSize); got != 16 {
		t.Fatalf("expected waveFormatEx.cbSize offset 16, got %d", got)
	}
	if got := unsafe.Sizeof(waveHdr{}); got != 48 {
		t.Fatalf("expected waveHdr size 48, got %d", got)
	}
}

func TestLiveRingStreamerConversionOverflowAndClose(t *testing.T) {
	ring := newLiveRingStreamer(3)

	// Push 4 samples into capacity 3; oldest sample should be dropped.
	pcm := make([]byte, 8)
	binary.LittleEndian.PutUint16(pcm[0:2], uint16(1000))
	binary.LittleEndian.PutUint16(pcm[2:4], uint16(16384))
	neg := int16(-16384)
	binary.LittleEndian.PutUint16(pcm[4:6], uint16(neg))
	binary.LittleEndian.PutUint16(pcm[6:8], uint16(0))
	ring.pushPCM16Mono(pcm)

	out := make([][2]float64, 4)
	n, ok := ring.Stream(out)
	if !ok || n != 4 {
		t.Fatalf("expected (4, true), got (%d, %v)", n, ok)
	}
	if math.Abs(out[0][0]-0.5) > 0.001 || out[0][0] != out[0][1] {
		t.Fatalf("expected sample[0] ~0.5 stereo, got %+v", out[0])
	}
	if math.Abs(out[1][0]+0.5) > 0.001 {
		t.Fatalf("expected sample[1] ~-0.5, got %+v", out[1])
	}
	if out[2][0] != 0 || out[3][0] != 0 {
		t.Fatalf("expected trailing silence on underrun, got %+v and %+v", out[2], out[3])
	}

	ring.close()
	n, ok = ring.Stream(out)
	if ok || n != 0 {
		t.Fatalf("expected (0, false) after close, got (%d, %v)", n, ok)
	}
}
