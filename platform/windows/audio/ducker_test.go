//go:build windows

package audio

import (
	"errors"
	"math"
	"testing"

	"github.com/go-ole/go-ole"
)

func TestComputeDuckedVolume(t *testing.T) {
	cases := []struct {
		orig    float32
		percent int
		want    float32
	}{
		{0.8, 25, 0.2},
		{1.0, 0, 0.0},
		{0.6, 100, 0.6},
		{0.5, 50, 0.25},
		{1.2, 50, 0.6},
	}
	for _, tc := range cases {
		got := computeDuckedVolume(tc.orig, tc.percent)
		if math.Abs(float64(got-tc.want)) > 0.001 {
			t.Fatalf("computeDuckedVolume(%v, %d) = %v, want %v", tc.orig, tc.percent, got, tc.want)
		}
	}
}

func TestShouldRestoreSession(t *testing.T) {
	if !shouldRestoreSession(0.20, 0.20, 0.80) {
		t.Fatal("expected exact ducked target to restore")
	}
	if !shouldRestoreSession(0.23, 0.20, 0.80) {
		t.Fatal("expected slight float drift within tolerance to restore")
	}
	if shouldRestoreSession(0.65, 0.20, 0.80) {
		t.Fatal("expected manual user volume change outside tolerance not to be overwritten")
	}
	if shouldRestoreSession(0.85, 0.20, 0.80) {
		t.Fatal("expected volume already >= original not to restore")
	}
}

func TestWASAPIDuckerSmoke(t *testing.T) {
	ducker := NewDucker()
	if err := ducker.Duck(25); err != nil {
		var oleErr *ole.OleError
		if errors.As(err, &oleErr) && (oleErr.Code() == ole.E_ACCESSDENIED || oleErr.Code() == eNotFound) {
			t.Skipf("skipping live WASAPI test in restricted or headless environment: %v", err)
		}
		t.Fatalf("Duck failed: %v", err)
	}
	if err := ducker.Restore(); err != nil {
		t.Fatalf("Restore failed: %v", err)
	}
}
