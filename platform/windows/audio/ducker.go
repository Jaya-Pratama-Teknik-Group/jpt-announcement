//go:build windows

package audio

import (
	"fmt"
	"math"
	"os"
	"runtime"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"github.com/go-ole/go-ole"
	"golang.org/x/sys/windows"
)

const (
	eRender                 uint32 = 0
	eMultimedia             uint32 = 1
	audioSessionStateActive uint32 = 1

	sOK             uintptr = 0
	sFalse          uintptr = 1
	rpcEChangedMode uintptr = 0x80010106
	eNotFound       uintptr = 0x80070490

	duckFadeSteps   = 4
	duckFadeStepDur = 25 * time.Millisecond
	restoreTol      = 0.06
)

var (
	clsidMMDeviceEnumerator  = ole.NewGUID("{BCDE0395-E52F-467C-8E3D-C4579291692E}")
	iidIMMDeviceEnumerator   = ole.NewGUID("{A95664D2-9614-4F35-A746-DE8DB63617E6}")
	iidIAudioSessionManager2 = ole.NewGUID("{77AA99A0-1BD6-484F-8BC7-2C654C9A9B6F}")
	iidIAudioSessionControl2 = ole.NewGUID("{BFB7FF88-7239-4FC9-8FA2-07C950BE9C6D}")
	iidISimpleAudioVolume    = ole.NewGUID("{87CE5498-68D6-44E5-9215-6DA47EF883D8}")
)

type duckedSession struct {
	instanceID string
	pid        uint32
	origVol    float32
	targetVol  float32
}

// WASAPIDucker attenuates other active Windows audio sessions via Core Audio / WASAPI
// during announcement playback and restores their original levels afterward.
type WASAPIDucker struct {
	mu      sync.Mutex
	ducked  map[string]duckedSession
	selfPID uint32
}

// NewDucker constructs a Windows WASAPI per-session audio ducker.
func NewDucker() *WASAPIDucker {
	return &WASAPIDucker{
		ducked:  make(map[string]duckedSession),
		selfPID: uint32(os.Getpid()),
	}
}

// Duck lowers all other active, unmuted audio sessions proportionally to targetPercent (0..100).
func (d *WASAPIDucker) Duck(targetPercent int) error {
	if targetPercent < 0 {
		targetPercent = 0
	}
	if targetPercent >= 100 {
		return d.Restore()
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	type fadeTarget struct {
		vol      *iSimpleAudioVolume
		startVol float32
		endVol   float32
	}

	return withCOMSessionEnumerator(func(enum *iAudioSessionEnumerator) error {
		count, err := enum.GetCount()
		if err != nil {
			return err
		}

		var fades []fadeTarget
		defer func() {
			for _, f := range fades {
				f.vol.Release()
			}
		}()

		for i := int32(0); i < count; i++ {
			ctrl2, simpleVol, ok := d.inspectSession(enum, i)
			if !ok {
				continue
			}

			state, err := ctrl2.GetState()
			if err != nil || state != audioSessionStateActive {
				simpleVol.Release()
				ctrl2.Release()
				continue
			}

			pid, err := ctrl2.GetProcessId()
			if err != nil || pid == 0 || pid == d.selfPID || ctrl2.IsSystemSoundsSession() {
				simpleVol.Release()
				ctrl2.Release()
				continue
			}

			instanceID, err := ctrl2.GetSessionInstanceIdentifier()
			if err != nil || instanceID == "" {
				instanceID = fmt.Sprintf("pid:%d:%d", pid, i)
			}
			ctrl2.Release()

			muted, err := simpleVol.GetMute()
			if err != nil || muted {
				simpleVol.Release()
				continue
			}

			curVol, err := simpleVol.GetMasterVolume()
			if err != nil {
				simpleVol.Release()
				continue
			}

			if existing, already := d.ducked[instanceID]; already {
				newTarget := computeDuckedVolume(existing.origVol, targetPercent)
				if newTarget < existing.origVol && math.Abs(float64(curVol-newTarget)) > 0.005 {
					existing.targetVol = newTarget
					d.ducked[instanceID] = existing
					fades = append(fades, fadeTarget{
						vol:      simpleVol,
						startVol: curVol,
						endVol:   newTarget,
					})
					continue
				}
				simpleVol.Release()
				continue
			}

			targetVol := computeDuckedVolume(curVol, targetPercent)
			if curVol <= 0.001 || targetVol >= curVol-0.001 {
				simpleVol.Release()
				continue
			}

			d.ducked[instanceID] = duckedSession{
				instanceID: instanceID,
				pid:        pid,
				origVol:    curVol,
				targetVol:  targetVol,
			}
			fades = append(fades, fadeTarget{
				vol:      simpleVol,
				startVol: curVol,
				endVol:   targetVol,
			})
		}

		if len(fades) == 0 {
			return nil
		}

		for step := 1; step <= duckFadeSteps; step++ {
			t := float32(step) / float32(duckFadeSteps)
			for _, f := range fades {
				level := f.startVol + (f.endVol-f.startVol)*t
				_ = f.vol.SetMasterVolume(clampScalar(level))
			}
			if step < duckFadeSteps {
				time.Sleep(duckFadeStepDur)
			}
		}
		return nil
	})
}

// Restore returns previously ducked audio sessions back to their original volumes.
func (d *WASAPIDucker) Restore() error {
	d.mu.Lock()
	if len(d.ducked) == 0 {
		d.mu.Unlock()
		return nil
	}
	saved := d.ducked
	d.ducked = make(map[string]duckedSession)
	d.mu.Unlock()

	type fadeTarget struct {
		vol      *iSimpleAudioVolume
		startVol float32
		endVol   float32
	}

	return withCOMSessionEnumerator(func(enum *iAudioSessionEnumerator) error {
		count, err := enum.GetCount()
		if err != nil {
			return err
		}

		byPID := make(map[uint32]duckedSession, len(saved))
		for _, entry := range saved {
			byPID[entry.pid] = entry
		}

		var fades []fadeTarget
		defer func() {
			for _, f := range fades {
				f.vol.Release()
			}
		}()

		restoredIDs := make(map[string]bool, len(saved))
		for i := int32(0); i < count; i++ {
			ctrl2, simpleVol, ok := d.inspectSession(enum, i)
			if !ok {
				continue
			}

			pid, _ := ctrl2.GetProcessId()
			instanceID, _ := ctrl2.GetSessionInstanceIdentifier()
			ctrl2.Release()

			entry, matched := saved[instanceID]
			if !matched && pid != 0 {
				if candidate, ok := byPID[pid]; ok && !restoredIDs[candidate.instanceID] {
					entry = candidate
					matched = true
				}
			}
			if !matched || restoredIDs[entry.instanceID] {
				simpleVol.Release()
				continue
			}

			curVol, err := simpleVol.GetMasterVolume()
			if err != nil || !shouldRestoreSession(curVol, entry.targetVol, entry.origVol) {
				simpleVol.Release()
				continue
			}

			restoredIDs[entry.instanceID] = true
			fades = append(fades, fadeTarget{
				vol:      simpleVol,
				startVol: curVol,
				endVol:   entry.origVol,
			})
		}

		if len(fades) == 0 {
			return nil
		}

		for step := 1; step <= duckFadeSteps; step++ {
			t := float32(step) / float32(duckFadeSteps)
			for _, f := range fades {
				level := f.startVol + (f.endVol-f.startVol)*t
				_ = f.vol.SetMasterVolume(clampScalar(level))
			}
			if step < duckFadeSteps {
				time.Sleep(duckFadeStepDur)
			}
		}
		return nil
	})
}

func (d *WASAPIDucker) inspectSession(enum *iAudioSessionEnumerator, index int32) (*iAudioSessionControl2, *iSimpleAudioVolume, bool) {
	ctrl, err := enum.GetSession(index)
	if err != nil || ctrl == nil {
		return nil, nil, false
	}

	var ctrl2 *iAudioSessionControl2
	if err := queryInterface(&ctrl.IUnknown, iidIAudioSessionControl2, unsafe.Pointer(&ctrl2)); err != nil || ctrl2 == nil {
		ctrl.Release()
		return nil, nil, false
	}

	var simpleVol *iSimpleAudioVolume
	if err := queryInterface(&ctrl.IUnknown, iidISimpleAudioVolume, unsafe.Pointer(&simpleVol)); err != nil || simpleVol == nil {
		ctrl2.Release()
		ctrl.Release()
		return nil, nil, false
	}

	ctrl.Release()
	return ctrl2, simpleVol, true
}

func computeDuckedVolume(origVol float32, targetPercent int) float32 {
	if targetPercent <= 0 {
		return 0
	}
	if targetPercent >= 100 {
		return clampScalar(origVol)
	}
	return clampScalar(origVol * (float32(targetPercent) / 100.0))
}

func shouldRestoreSession(currentVol, duckedTarget, origVol float32) bool {
	if currentVol >= origVol-0.001 {
		return false
	}
	return math.Abs(float64(currentVol-duckedTarget)) <= restoreTol
}

func clampScalar(v float32) float32 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func withCOMSessionEnumerator(fn func(enum *iAudioSessionEnumerator) error) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	shouldUninit, err := initCOMForThread()
	if err != nil {
		return err
	}
	if shouldUninit {
		defer ole.CoUninitialize()
	}

	unk, err := ole.CreateInstance(clsidMMDeviceEnumerator, iidIMMDeviceEnumerator)
	if err != nil {
		return fmt.Errorf("gagal membuat MMDeviceEnumerator: %w", err)
	}
	enumerator := (*iMMDeviceEnumerator)(unsafe.Pointer(unk))
	defer enumerator.Release()

	dev, hr, err := enumerator.GetDefaultAudioEndpoint(eRender, eMultimedia)
	if err != nil {
		if hr == eNotFound {
			return nil
		}
		return fmt.Errorf("gagal mengakses perangkat audio default: %w", err)
	}
	defer dev.Release()

	var mgr *iAudioSessionManager2
	if err := dev.Activate(iidIAudioSessionManager2, ole.CLSCTX_ALL, 0, unsafe.Pointer(&mgr)); err != nil {
		return fmt.Errorf("gagal mengaktifkan IAudioSessionManager2: %w", err)
	}
	defer mgr.Release()

	sessionEnum, err := mgr.GetSessionEnumerator()
	if err != nil {
		return fmt.Errorf("gagal mengambil enumerator sesi audio: %w", err)
	}
	defer sessionEnum.Release()

	return fn(sessionEnum)
}

func initCOMForThread() (bool, error) {
	err := ole.CoInitializeEx(0, ole.COINIT_MULTITHREADED)
	if err == nil {
		return true, nil
	}
	var oleErr *ole.OleError
	if oleErrVal, ok := err.(*ole.OleError); ok {
		oleErr = oleErrVal
		code := oleErr.Code()
		if code == sOK || code == sFalse {
			return true, nil
		}
		if code == rpcEChangedMode {
			// Thread already initialized as STA; COM calls work without CoUninitialize.
			return false, nil
		}
	}
	return false, fmt.Errorf("gagal menginisialisasi COM audio: %w", err)
}

func queryInterface(unk *ole.IUnknown, iid *ole.GUID, out unsafe.Pointer) error {
	hr, _, _ := syscall.SyscallN(
		unk.VTable().QueryInterface,
		uintptr(unsafe.Pointer(unk)),
		uintptr(unsafe.Pointer(iid)),
		uintptr(out),
	)
	if hr != 0 {
		return ole.NewError(hr)
	}
	return nil
}

type iMMDeviceEnumerator struct{ ole.IUnknown }

type iMMDeviceEnumeratorVtbl struct {
	ole.IUnknownVtbl
	EnumAudioEndpoints                     uintptr
	GetDefaultAudioEndpoint                uintptr
	GetDevice                              uintptr
	RegisterEndpointNotificationCallback   uintptr
	UnregisterEndpointNotificationCallback uintptr
}

func (v *iMMDeviceEnumerator) vtbl() *iMMDeviceEnumeratorVtbl {
	return (*iMMDeviceEnumeratorVtbl)(unsafe.Pointer(v.RawVTable))
}

func (v *iMMDeviceEnumerator) GetDefaultAudioEndpoint(dataFlow, role uint32) (*iMMDevice, uintptr, error) {
	var dev *iMMDevice
	hr, _, _ := syscall.SyscallN(
		v.vtbl().GetDefaultAudioEndpoint,
		uintptr(unsafe.Pointer(v)),
		uintptr(dataFlow),
		uintptr(role),
		uintptr(unsafe.Pointer(&dev)),
	)
	if hr != 0 {
		return nil, hr, ole.NewError(hr)
	}
	return dev, hr, nil
}

type iMMDevice struct{ ole.IUnknown }

type iMMDeviceVtbl struct {
	ole.IUnknownVtbl
	Activate          uintptr
	OpenPropertyStore uintptr
	GetId             uintptr
	GetState          uintptr
}

func (v *iMMDevice) vtbl() *iMMDeviceVtbl {
	return (*iMMDeviceVtbl)(unsafe.Pointer(v.RawVTable))
}

func (v *iMMDevice) Activate(iid *ole.GUID, clsCtx uint32, params uintptr, out unsafe.Pointer) error {
	hr, _, _ := syscall.SyscallN(
		v.vtbl().Activate,
		uintptr(unsafe.Pointer(v)),
		uintptr(unsafe.Pointer(iid)),
		uintptr(clsCtx),
		params,
		uintptr(out),
	)
	if hr != 0 {
		return ole.NewError(hr)
	}
	return nil
}

type iAudioSessionManager2 struct{ ole.IUnknown }

type iAudioSessionManager2Vtbl struct {
	ole.IUnknownVtbl
	GetAudioSessionControl        uintptr
	GetSimpleAudioVolume          uintptr
	GetSessionEnumerator          uintptr
	RegisterSessionNotification   uintptr
	UnregisterSessionNotification uintptr
	RegisterDuckNotification      uintptr
	UnregisterDuckNotification    uintptr
}

func (v *iAudioSessionManager2) vtbl() *iAudioSessionManager2Vtbl {
	return (*iAudioSessionManager2Vtbl)(unsafe.Pointer(v.RawVTable))
}

func (v *iAudioSessionManager2) GetSessionEnumerator() (*iAudioSessionEnumerator, error) {
	var enum *iAudioSessionEnumerator
	hr, _, _ := syscall.SyscallN(
		v.vtbl().GetSessionEnumerator,
		uintptr(unsafe.Pointer(v)),
		uintptr(unsafe.Pointer(&enum)),
	)
	if hr != 0 {
		return nil, ole.NewError(hr)
	}
	return enum, nil
}

type iAudioSessionEnumerator struct{ ole.IUnknown }

type iAudioSessionEnumeratorVtbl struct {
	ole.IUnknownVtbl
	GetCount   uintptr
	GetSession uintptr
}

func (v *iAudioSessionEnumerator) vtbl() *iAudioSessionEnumeratorVtbl {
	return (*iAudioSessionEnumeratorVtbl)(unsafe.Pointer(v.RawVTable))
}

func (v *iAudioSessionEnumerator) GetCount() (int32, error) {
	var count int32
	hr, _, _ := syscall.SyscallN(
		v.vtbl().GetCount,
		uintptr(unsafe.Pointer(v)),
		uintptr(unsafe.Pointer(&count)),
	)
	if hr != 0 {
		return 0, ole.NewError(hr)
	}
	return count, nil
}

func (v *iAudioSessionEnumerator) GetSession(index int32) (*iAudioSessionControl, error) {
	var ctrl *iAudioSessionControl
	hr, _, _ := syscall.SyscallN(
		v.vtbl().GetSession,
		uintptr(unsafe.Pointer(v)),
		uintptr(index),
		uintptr(unsafe.Pointer(&ctrl)),
	)
	if hr != 0 {
		return nil, ole.NewError(hr)
	}
	return ctrl, nil
}

type iAudioSessionControl struct{ ole.IUnknown }

type iAudioSessionControlVtbl struct {
	ole.IUnknownVtbl
	GetState                           uintptr
	GetDisplayName                     uintptr
	SetDisplayName                     uintptr
	GetIconPath                        uintptr
	SetIconPath                        uintptr
	GetGroupingParam                   uintptr
	SetGroupingParam                   uintptr
	RegisterAudioSessionNotification   uintptr
	UnregisterAudioSessionNotification uintptr
}

type iAudioSessionControl2 struct{ ole.IUnknown }

type iAudioSessionControl2Vtbl struct {
	iAudioSessionControlVtbl
	GetSessionIdentifier         uintptr
	GetSessionInstanceIdentifier uintptr
	GetProcessId                 uintptr
	IsSystemSoundsSession        uintptr
	SetDuckingPreference         uintptr
}

func (v *iAudioSessionControl2) vtbl() *iAudioSessionControl2Vtbl {
	return (*iAudioSessionControl2Vtbl)(unsafe.Pointer(v.RawVTable))
}

func (v *iAudioSessionControl2) GetState() (uint32, error) {
	var state uint32
	hr, _, _ := syscall.SyscallN(
		v.vtbl().GetState,
		uintptr(unsafe.Pointer(v)),
		uintptr(unsafe.Pointer(&state)),
	)
	if hr != 0 {
		return 0, ole.NewError(hr)
	}
	return state, nil
}

func (v *iAudioSessionControl2) GetSessionInstanceIdentifier() (string, error) {
	var ptr *uint16
	hr, _, _ := syscall.SyscallN(
		v.vtbl().GetSessionInstanceIdentifier,
		uintptr(unsafe.Pointer(v)),
		uintptr(unsafe.Pointer(&ptr)),
	)
	if ptr != nil {
		defer ole.CoTaskMemFree(uintptr(unsafe.Pointer(ptr)))
	}
	if hr != 0 {
		return "", ole.NewError(hr)
	}
	if ptr == nil {
		return "", nil
	}
	return windows.UTF16PtrToString(ptr), nil
}

func (v *iAudioSessionControl2) GetProcessId() (uint32, error) {
	var pid uint32
	hr, _, _ := syscall.SyscallN(
		v.vtbl().GetProcessId,
		uintptr(unsafe.Pointer(v)),
		uintptr(unsafe.Pointer(&pid)),
	)
	if hr != 0 {
		return 0, ole.NewError(hr)
	}
	return pid, nil
}

func (v *iAudioSessionControl2) IsSystemSoundsSession() bool {
	hr, _, _ := syscall.SyscallN(
		v.vtbl().IsSystemSoundsSession,
		uintptr(unsafe.Pointer(v)),
	)
	return hr == sOK
}

type iSimpleAudioVolume struct{ ole.IUnknown }

type iSimpleAudioVolumeVtbl struct {
	ole.IUnknownVtbl
	SetMasterVolume uintptr
	GetMasterVolume uintptr
	SetMute         uintptr
	GetMute         uintptr
}

func (v *iSimpleAudioVolume) vtbl() *iSimpleAudioVolumeVtbl {
	return (*iSimpleAudioVolumeVtbl)(unsafe.Pointer(v.RawVTable))
}

func (v *iSimpleAudioVolume) SetMasterVolume(level float32) error {
	hr, _, _ := syscall.SyscallN(
		v.vtbl().SetMasterVolume,
		uintptr(unsafe.Pointer(v)),
		uintptr(math.Float32bits(level)),
		0,
	)
	if hr != 0 {
		return ole.NewError(hr)
	}
	return nil
}

func (v *iSimpleAudioVolume) GetMasterVolume() (float32, error) {
	var bits uint32
	hr, _, _ := syscall.SyscallN(
		v.vtbl().GetMasterVolume,
		uintptr(unsafe.Pointer(v)),
		uintptr(unsafe.Pointer(&bits)),
	)
	if hr != 0 {
		return 0, ole.NewError(hr)
	}
	return math.Float32frombits(bits), nil
}

func (v *iSimpleAudioVolume) GetMute() (bool, error) {
	var muted int32
	hr, _, _ := syscall.SyscallN(
		v.vtbl().GetMute,
		uintptr(unsafe.Pointer(v)),
		uintptr(unsafe.Pointer(&muted)),
	)
	if hr != 0 {
		return false, ole.NewError(hr)
	}
	return muted != 0, nil
}
