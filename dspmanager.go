//go:build windows

package main

import (
	"encoding/binary"
	"fmt"
	"math"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	sharedMemName   = "Global\\ViPER4Windows_SharedMemory"
	sharedEventName = "Global\\ViPER4Windows_Event"
	sharedMemBytes  = 1024
	paramCount      = sharedMemBytes / 4
)

const (
	paramHPConvolverEnable      = 0x10100
	paramHPConvolverSetKernel   = 0x10101
	paramHPConvolverCrossChan   = 0x10105
	paramHPDDCEnable            = 0x10110
	paramHPDDCCoefficients      = 0x10111
	paramHPEQEnable             = 0x10120
	paramHPEQBandLevel          = 0x10121
	paramHPEQBandCount          = 0x10122
	paramHPReverbEnable         = 0x10130
	paramHPReverbRoomSize       = 0x10131
	paramHPReverbWidth          = 0x10132
	paramHPReverbDampen         = 0x10133
	paramHPReverbWet            = 0x10134
	paramHPReverbDry            = 0x10135
	paramHPAGCEnable            = 0x10140
	paramHPAGCRatio             = 0x10141
	paramHPAGCVolume            = 0x10142
	paramHPAGCMaxScaler         = 0x10143
	paramHPDynamicSystemEnable  = 0x10150
	paramHPDynamicSystemXCoeffs = 0x10151
	paramHPDynamicSystemYCoeffs = 0x10152
	paramHPDynamicSystemSide    = 0x10153
	paramHPDynamicSystemPower   = 0x10154
	paramHPBassEnable           = 0x10160
	paramHPBassMode             = 0x10161
	paramHPBassFrequency        = 0x10162
	paramHPBassGain             = 0x10163
	paramHPBassMonoEnable       = 0x10164
	paramHPBassMonoMode         = 0x10165
	paramHPBassMonoFrequency    = 0x10166
	paramHPBassMonoGain         = 0x10167
	paramHPBassAntiPop          = 0x10168
	paramHPBassMonoAntiPop      = 0x10169
	paramHPClarityEnable        = 0x10170
	paramHPClarityMode          = 0x10171
	paramHPClarityGain          = 0x10172
	paramHPHeadphoneEnable      = 0x10180
	paramHPHeadphoneStrength    = 0x10181
	paramHPSpectrumEnable       = 0x10190
	paramHPSpectrumBark         = 0x10191
	paramHPSpectrumExciter      = 0x10192
	paramHPFieldEnable          = 0x101A0
	paramHPFieldWidening        = 0x101A1
	paramHPFieldMidImage        = 0x101A2
	paramHPFieldDepth           = 0x101A3
	paramHPDiffEnable           = 0x101B0
	paramHPDiffDelay            = 0x101B1
	paramHPCureEnable           = 0x101C0
	paramHPCureStrength         = 0x101C1
	paramHPTubeEnable           = 0x101D0
	paramHPAnalogXEnable        = 0x101E0
	paramHPAnalogXMode          = 0x101E1
	paramHPOutputVolume         = 0x101F0
	paramHPChannelPan           = 0x101F1
	paramHPLimiter              = 0x101F2
	paramHPFETEnable            = 0x10200
	paramHPFETThreshold         = 0x10201
	paramHPFETRatio             = 0x10202
	paramHPFETKnee              = 0x10203
	paramHPFETAutoKnee          = 0x10204
	paramHPFETGain              = 0x10205
	paramHPFETAutoGain          = 0x10206
	paramHPFETAttack            = 0x10207
	paramHPFETAutoAttack        = 0x10208
	paramHPFETRelease           = 0x10209
	paramHPFETAutoRelease       = 0x1020A
	paramHPFETKneeMulti         = 0x1020B
	paramHPFETMaxAttack         = 0x1020C
	paramHPFETMaxRelease        = 0x1020D
	paramHPFETCrest             = 0x1020E
	paramHPFETAdapt             = 0x1020F
	paramHPFETNoClip            = 0x10210
	paramHPSpeakerCorrection    = 0x10420
)

var speakerSizeToHz = [11]int{
	20, 30, 40, 55, 70, 90, 115, 150, 200, 280, 380,
}

type paramBuf [paramCount]float32

const (
	idxEnabled                  = 0
	idxPreVol                   = 1
	idxPostVol                  = 2
	idxEQEnabled                = 4
	idxEQBands                  = 5
	idxXBassEnabled             = 23
	idxXBassMode                = 24
	idxXBassSpkSize             = 25
	idxXBassGain                = 26
	idxXClarEnabled             = 27
	idxXClarMode                = 28
	idxXClarGain                = 29
	idxSurrEnabled              = 30
	idxSurrSize                 = 31
	idxRevEnabled               = 32
	idxRevRoom                  = 33
	idxRevDamp                  = 34
	idxRevMix                   = 35
	idxConvolverEnabled         = 36
	idxCureEnabled              = 37
	idxCureStrength             = 38
	idxAnalogXEnabled           = 39
	idxAnalogXMode              = 40
	idxSpeakerCorrectionEnabled = 41
	idxLimiterEnabled           = 44
	idxLimiterValue             = 45
)

type dispatchCmd struct {
	param        int
	val1         int
	val2         int
	val3         int
	val4         int
	arrSize      int
	payload      []byte
	onlyOnChange string
}

type payloadCache struct {
	arrSize int
	data    []byte
}

type DSPManager struct {
	mu sync.Mutex

	dll       windows.Handle
	viperInst uintptr

	fnCreate        uintptr
	fnDestroy       uintptr
	fnSetSampleRate uintptr
	fnReset         uintptr
	fnDispatch      uintptr
	fnProcess       uintptr
	dllReady        bool

	memHandle   windows.Handle
	memView     uintptr
	eventHandle windows.Handle
	shmReady    bool

	activeBuf uint8
	frontBuf  paramBuf
	backBuf   paramBuf
	payloads  map[string]payloadCache
}

func (dm *DSPManager) LoadDLL(dir string) error {
	return dm.LoadDLLPath(filepath.Join(dir, "ViPERDSP.dll"))
}

func (dm *DSPManager) LoadDLLPath(dllPath string) error {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	if dm.dllReady {
		return nil
	}

	if !fileExistsAndNotEmpty(dllPath) {
		return fmt.Errorf("ViPERDSP.dll missing or empty: %s", dllPath)
	}
	h, err := windows.LoadLibrary(dllPath)
	if err != nil {
		return fmt.Errorf("LoadLibrary(%s): %w", dllPath, err)
	}

	type symbol struct {
		name string
		dst  *uintptr
	}

	symbols := []symbol{
		{name: "viper_create", dst: &dm.fnCreate},
		{name: "viper_destroy", dst: &dm.fnDestroy},
		{name: "viper_set_sample_rate", dst: &dm.fnSetSampleRate},
		{name: "viper_reset", dst: &dm.fnReset},
		{name: "viper_dispatch", dst: &dm.fnDispatch},
		{name: "viper_process", dst: &dm.fnProcess},
	}

	for _, sym := range symbols {
		addr, symErr := windows.GetProcAddress(h, sym.name)
		if symErr != nil {
			windows.FreeLibrary(h)
			return fmt.Errorf("GetProcAddress(%s): %w", sym.name, symErr)
		}
		*sym.dst = addr
	}

	ret, _, _ := syscall.SyscallN(dm.fnCreate)
	if ret == 0 {
		windows.FreeLibrary(h)
		return fmt.Errorf("viper_create() returned NULL")
	}

	dm.dll = h
	dm.viperInst = ret
	dm.callSetSampleRate(44100)
	dm.dllReady = true

	logV("dll loaded: %s", dllPath)
	return nil
}

func (dm *DSPManager) SyncSharedMemory() error {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	if dm.shmReady {
		return nil
	}

	hMap, err := windows.CreateFileMapping(
		windows.InvalidHandle,
		nil,
		windows.PAGE_READWRITE,
		0,
		sharedMemBytes,
		windows.StringToUTF16Ptr(sharedMemName),
	)
	if err != nil {
		return fmt.Errorf("CreateFileMapping: %w", err)
	}

	view, err := windows.MapViewOfFile(hMap, windows.FILE_MAP_WRITE, 0, 0, sharedMemBytes)
	if err != nil {
		windows.CloseHandle(hMap)
		return fmt.Errorf("MapViewOfFile: %w", err)
	}

	hEvent, _ := windows.CreateEvent(nil, 0, 0, windows.StringToUTF16Ptr(sharedEventName))
	if hEvent == 0 {
		hEvent, _ = windows.OpenEvent(
			windows.EVENT_MODIFY_STATE,
			false,
			windows.StringToUTF16Ptr(sharedEventName),
		)
	}

	dm.memHandle = hMap
	dm.memView = view
	dm.eventHandle = hEvent
	dm.shmReady = true

	logV("shared memory ready (%d bytes)", sharedMemBytes)
	return nil
}

func (dm *DSPManager) ApplyChanges(state DSPState) error {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	if !dm.dllReady && !dm.shmReady {
		return fmt.Errorf("no DSP path available (DLL not loaded, shared memory not mapped)")
	}

	if dm.dllReady {
		dm.applyViaDLL(state)
	}

	if dm.shmReady {
		target := dm.inactiveBuffer()
		fillParamBuf(target, state)
		dm.writeSharedMemory(target)
		dm.activeBuf ^= 1
	}

	return nil
}

func (dm *DSPManager) Close() {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	if dm.viperInst != 0 && dm.fnDestroy != 0 {
		syscall.SyscallN(dm.fnDestroy, dm.viperInst)
		dm.viperInst = 0
	}
	if dm.dll != 0 {
		windows.FreeLibrary(dm.dll)
		dm.dll = 0
	}
	dm.fnCreate = 0
	dm.fnDestroy = 0
	dm.fnSetSampleRate = 0
	dm.fnReset = 0
	dm.fnDispatch = 0
	dm.fnProcess = 0
	dm.dllReady = false

	if dm.eventHandle != 0 {
		windows.CloseHandle(dm.eventHandle)
		dm.eventHandle = 0
	}
	if dm.memView != 0 {
		windows.UnmapViewOfFile(dm.memView)
		dm.memView = 0
	}
	if dm.memHandle != 0 {
		windows.CloseHandle(dm.memHandle)
		dm.memHandle = 0
	}
	dm.shmReady = false
	dm.activeBuf = 0
	dm.frontBuf = paramBuf{}
	dm.backBuf = paramBuf{}
	dm.payloads = nil

	logV("dsp resources released")
}

func (dm *DSPManager) SetSampleRate(rate uint32) {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	if dm.dllReady {
		dm.callSetSampleRate(rate)
	}
}

func (dm *DSPManager) ResetEffects() {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	if dm.dllReady && dm.fnReset != 0 && dm.viperInst != 0 {
		syscall.SyscallN(dm.fnReset, dm.viperInst)
		dm.payloads = nil
	}
}

func (dm *DSPManager) callSetSampleRate(rate uint32) {
	if dm.fnSetSampleRate == 0 || dm.viperInst == 0 {
		return
	}
	syscall.SyscallN(dm.fnSetSampleRate, dm.viperInst, uintptr(rate))
}

func (dm *DSPManager) dispatch(param, val1, val2, val3, val4 int) {
	dm.dispatchPayload(param, val1, val2, val3, val4, 0, nil)
}

func (dm *DSPManager) dispatchPayload(param, val1, val2, val3, val4, arrSize int, payload []byte) {
	if !dm.dllReady || dm.fnDispatch == 0 || dm.viperInst == 0 {
		return
	}

	var payloadPtr uintptr
	if len(payload) > 0 {
		payloadPtr = uintptr(unsafe.Pointer(&payload[0]))
	}

	syscall.SyscallN(
		dm.fnDispatch,
		dm.viperInst,
		uintptr(param),
		uintptr(int32(val1)),
		uintptr(int32(val2)),
		uintptr(int32(val3)),
		uintptr(int32(val4)),
		uintptr(arrSize),
		payloadPtr,
	)
}

func (dm *DSPManager) applyViaDLL(state DSPState) {
	for _, cmd := range buildDLLCommands(state) {
		if cmd.onlyOnChange != "" && !dm.shouldDispatchPayload(cmd.onlyOnChange, cmd.arrSize, cmd.payload) {
			continue
		}
		dm.dispatchPayload(cmd.param, cmd.val1, cmd.val2, cmd.val3, cmd.val4, cmd.arrSize, cmd.payload)
	}
}

func buildDLLCommands(state DSPState) []dispatchCmd {
	cmds := make([]dispatchCmd, 0, 96)

	postVolume := 0
	if state.Master.Power {
		postVolume = dbLinearCenti(state.Master.PostVol)
	}
	cmds = append(cmds, dispatchCmd{param: paramHPOutputVolume, val1: postVolume})
	cmds = append(cmds, dispatchCmd{
		param: paramHPChannelPan,
		val1:  centi(clamp(state.Output.Pan, -1.0, 1.0)),
	})
	cmds = append(cmds, dispatchCmd{
		param: paramHPLimiter,
		val1:  centi(clamp(state.Output.Limiter, 0.0, 1.0)),
	})

	cmds = append(cmds, dispatchCmd{param: paramHPEQEnable, val1: boolInt(state.EqOn)})
	cmds = append(cmds, dispatchCmd{param: paramHPEQBandCount, val1: 18})
	for i := 0; i < len(state.Equalizer) && i < 18; i++ {
		cmds = append(cmds, dispatchCmd{
			param: paramHPEQBandLevel,
			val1:  i,
			val2:  centi(state.Equalizer[i]),
		})
	}

	cmds = append(cmds, dispatchCmd{param: paramHPBassEnable, val1: boolInt(state.XBass.On)})
	if state.XBass.On {
		cmds = append(cmds,
			dispatchCmd{param: paramHPBassMode, val1: parseBassMode(state.XBass.Mode)},
			dispatchCmd{param: paramHPBassFrequency, val1: speakerHz(state.XBass.SpeakerSize)},
			dispatchCmd{param: paramHPBassGain, val1: centi(state.XBass.Level)},
			dispatchCmd{param: paramHPBassAntiPop, val1: 1},
		)
	}

	cmds = append(cmds, dispatchCmd{param: paramHPBassMonoEnable, val1: boolInt(state.XBassMono.On)})
	if state.XBassMono.On {
		cmds = append(cmds,
			dispatchCmd{param: paramHPBassMonoMode, val1: parseBassMode(state.XBassMono.Mode)},
			dispatchCmd{param: paramHPBassMonoFrequency, val1: speakerHz(state.XBassMono.SpeakerSize)},
			dispatchCmd{param: paramHPBassMonoGain, val1: centi(state.XBassMono.Level)},
			dispatchCmd{param: paramHPBassMonoAntiPop, val1: 1},
		)
	}

	cmds = append(cmds, dispatchCmd{param: paramHPClarityEnable, val1: boolInt(state.XClarity.On)})
	if state.XClarity.On {
		cmds = append(cmds,
			dispatchCmd{param: paramHPClarityMode, val1: parseClarityMode(state.XClarity.Mode)},
			dispatchCmd{param: paramHPClarityGain, val1: centi(state.XClarity.Level)},
		)
	}

	cmds = append(cmds, dispatchCmd{param: paramHPHeadphoneEnable, val1: boolInt(state.Surround3D.On)})
	if state.Surround3D.On {
		cmds = append(cmds, dispatchCmd{
			param: paramHPHeadphoneStrength,
			val1:  clampInt(state.Surround3D.SpaceSize*10, 0, 100),
		})
	}

	cmds = append(cmds, dispatchCmd{param: paramHPReverbEnable, val1: boolInt(state.Reverb.On)})
	if state.Reverb.On {
		wet := clampInt(int(math.Round(state.Reverb.WetMix)), 0, 100)
		cmds = append(cmds,
			dispatchCmd{param: paramHPReverbRoomSize, val1: clampInt(int(math.Round(state.Reverb.RoomSize/10.0)), 0, 100)},
			dispatchCmd{param: paramHPReverbDampen, val1: clampInt(int(math.Round(state.Reverb.Damping)), 0, 100)},
			dispatchCmd{param: paramHPReverbWet, val1: wet},
			dispatchCmd{param: paramHPReverbDry, val1: 100 - wet},
			dispatchCmd{param: paramHPReverbWidth, val1: 100},
		)
	}

	convolverEnabled := state.Convolver.On && strings.TrimSpace(state.Convolver.KernelPath) != ""
	cmds = append(cmds, dispatchCmd{param: paramHPConvolverEnable, val1: boolInt(convolverEnabled)})
	cmds = append(cmds, dispatchCmd{
		param: paramHPConvolverCrossChan,
		val1:  centi(clamp(state.Convolver.CrossChannel, 0.0, 1.0)),
	})
	if convolverEnabled {
		payload := serializeCString(state.Convolver.KernelPath)
		cmds = append(cmds, dispatchCmd{
			param:        paramHPConvolverSetKernel,
			arrSize:      len(payload),
			payload:      payload,
			onlyOnChange: "convolver_kernel",
		})
	}

	ddcPayload, ddcSize := serializeDDCPayload(state.DDC)
	ddcEnabled := state.DDC.On && ddcSize > 0
	cmds = append(cmds, dispatchCmd{param: paramHPDDCEnable, val1: boolInt(ddcEnabled)})
	if ddcEnabled {
		cmds = append(cmds, dispatchCmd{
			param:        paramHPDDCCoefficients,
			arrSize:      ddcSize,
			payload:      ddcPayload,
			onlyOnChange: "ddc_coeffs",
		})
	}

	cmds = append(cmds, dispatchCmd{param: paramHPAGCEnable, val1: boolInt(state.AGC.On)})
	if state.AGC.On {
		cmds = append(cmds,
			dispatchCmd{param: paramHPAGCRatio, val1: centi(state.AGC.Ratio)},
			dispatchCmd{param: paramHPAGCVolume, val1: centi(state.AGC.Volume)},
			dispatchCmd{param: paramHPAGCMaxScaler, val1: centi(state.AGC.MaxScaler)},
		)
	}

	cmds = append(cmds, dispatchCmd{param: paramHPDynamicSystemEnable, val1: boolInt(state.DynamicSystem.On)})
	if state.DynamicSystem.On {
		cmds = append(cmds,
			dispatchCmd{param: paramHPDynamicSystemXCoeffs, val1: state.DynamicSystem.XCoeffsLow, val2: state.DynamicSystem.XCoeffsHigh},
			dispatchCmd{param: paramHPDynamicSystemYCoeffs, val1: state.DynamicSystem.YCoeffsLow, val2: state.DynamicSystem.YCoeffsHigh},
			dispatchCmd{param: paramHPDynamicSystemSide, val1: centi(state.DynamicSystem.SideGainX), val2: centi(state.DynamicSystem.SideGainY)},
			dispatchCmd{param: paramHPDynamicSystemPower, val1: centi(state.DynamicSystem.Strength)},
		)
	}

	cmds = append(cmds, dispatchCmd{param: paramHPSpectrumEnable, val1: boolInt(state.SpectrumExtension.On)})
	if state.SpectrumExtension.On {
		cmds = append(cmds,
			dispatchCmd{param: paramHPSpectrumBark, val1: state.SpectrumExtension.ReferenceFrequency},
			dispatchCmd{param: paramHPSpectrumExciter, val1: centi(state.SpectrumExtension.Exciter)},
		)
	}

	cmds = append(cmds, dispatchCmd{param: paramHPFieldEnable, val1: boolInt(state.FieldSurround.On)})
	if state.FieldSurround.On {
		cmds = append(cmds,
			dispatchCmd{param: paramHPFieldWidening, val1: centi(state.FieldSurround.Widening)},
			dispatchCmd{param: paramHPFieldMidImage, val1: centi(state.FieldSurround.MidImage)},
			dispatchCmd{param: paramHPFieldDepth, val1: state.FieldSurround.Depth},
		)
	}

	cmds = append(cmds, dispatchCmd{param: paramHPDiffEnable, val1: boolInt(state.DiffSurround.On)})
	if state.DiffSurround.On {
		cmds = append(cmds, dispatchCmd{param: paramHPDiffDelay, val1: centi(state.DiffSurround.Delay)})
	}

	cmds = append(cmds, dispatchCmd{param: paramHPCureEnable, val1: boolInt(state.Cure.On)})
	if state.Cure.On {
		cmds = append(cmds, dispatchCmd{
			param: paramHPCureStrength,
			val1:  clampInt(state.Cure.StrengthPreset, 0, 2),
		})
	}

	cmds = append(cmds, dispatchCmd{param: paramHPTubeEnable, val1: boolInt(state.TubeSimulator.On)})
	cmds = append(cmds, dispatchCmd{param: paramHPAnalogXEnable, val1: boolInt(state.AnalogX.On)})
	if state.AnalogX.On {
		cmds = append(cmds, dispatchCmd{param: paramHPAnalogXMode, val1: state.AnalogX.Mode})
	}

	cmds = append(cmds, dispatchCmd{param: paramHPFETEnable, val1: boolCenti(state.FETCompressor.On)})
	if state.FETCompressor.On {
		cmds = append(cmds,
			dispatchCmd{param: paramHPFETThreshold, val1: centi(state.FETCompressor.Threshold)},
			dispatchCmd{param: paramHPFETRatio, val1: centi(state.FETCompressor.Ratio)},
			dispatchCmd{param: paramHPFETKnee, val1: centi(state.FETCompressor.Knee)},
			dispatchCmd{param: paramHPFETAutoKnee, val1: boolCenti(state.FETCompressor.AutoKnee)},
			dispatchCmd{param: paramHPFETGain, val1: centi(state.FETCompressor.Gain)},
			dispatchCmd{param: paramHPFETAutoGain, val1: boolCenti(state.FETCompressor.AutoGain)},
			dispatchCmd{param: paramHPFETAttack, val1: centi(state.FETCompressor.Attack)},
			dispatchCmd{param: paramHPFETAutoAttack, val1: boolCenti(state.FETCompressor.AutoAttack)},
			dispatchCmd{param: paramHPFETRelease, val1: centi(state.FETCompressor.Release)},
			dispatchCmd{param: paramHPFETAutoRelease, val1: boolCenti(state.FETCompressor.AutoRelease)},
			dispatchCmd{param: paramHPFETKneeMulti, val1: centi(state.FETCompressor.KneeMulti)},
			dispatchCmd{param: paramHPFETMaxAttack, val1: centi(state.FETCompressor.MaxAttack)},
			dispatchCmd{param: paramHPFETMaxRelease, val1: centi(state.FETCompressor.MaxRelease)},
			dispatchCmd{param: paramHPFETCrest, val1: centi(state.FETCompressor.Crest)},
			dispatchCmd{param: paramHPFETAdapt, val1: centi(state.FETCompressor.Adapt)},
			dispatchCmd{param: paramHPFETNoClip, val1: boolCenti(state.FETCompressor.NoClip)},
		)
	}

	cmds = append(cmds, dispatchCmd{param: paramHPSpeakerCorrection, val1: boolInt(state.SpeakerCorrection.On)})
	return cmds
}

func fillParamBuf(buf *paramBuf, state DSPState) {
	*buf = paramBuf{}

	if state.Master.Power {
		buf[idxEnabled] = 1
	}
	buf[idxPreVol] = float32(state.Master.PreVol)
	buf[idxPostVol] = float32(state.Master.PostVol)

	if state.EqOn {
		buf[idxEQEnabled] = 1
	}
	for i := 0; i < len(state.Equalizer) && i < 18; i++ {
		buf[idxEQBands+i] = float32(state.Equalizer[i])
	}

	if state.XBass.On {
		buf[idxXBassEnabled] = 1
		buf[idxXBassMode] = float32(parseBassMode(state.XBass.Mode))
		buf[idxXBassSpkSize] = float32(state.XBass.SpeakerSize)
		buf[idxXBassGain] = float32(state.XBass.Level)
	}

	if state.XClarity.On {
		buf[idxXClarEnabled] = 1
		buf[idxXClarMode] = float32(parseClarityMode(state.XClarity.Mode))
		buf[idxXClarGain] = float32(state.XClarity.Level)
	}

	if state.Surround3D.On {
		buf[idxSurrEnabled] = 1
		buf[idxSurrSize] = float32(state.Surround3D.SpaceSize)
	}

	if state.Reverb.On {
		buf[idxRevEnabled] = 1
		buf[idxRevRoom] = float32(state.Reverb.RoomSize)
		buf[idxRevDamp] = float32(state.Reverb.Damping)
		buf[idxRevMix] = float32(state.Reverb.WetMix)
	}

	if state.Convolver.On {
		buf[idxConvolverEnabled] = 1
	}

	if state.Cure.On {
		buf[idxCureEnabled] = 1
		buf[idxCureStrength] = float32(clampInt(state.Cure.StrengthPreset, 0, 2))
	}

	if state.AnalogX.On {
		buf[idxAnalogXEnabled] = 1
		buf[idxAnalogXMode] = float32(state.AnalogX.Mode)
	}

	if state.SpeakerCorrection.On {
		buf[idxSpeakerCorrectionEnabled] = 1
	}

	if state.Output.Limiter < 1.0 {
		buf[idxLimiterEnabled] = 1
		buf[idxLimiterValue] = float32(clamp(state.Output.Limiter, 0.0, 1.0))
	}
}

func (dm *DSPManager) inactiveBuffer() *paramBuf {
	if dm.activeBuf == 0 {
		return &dm.backBuf
	}
	return &dm.frontBuf
}

func (dm *DSPManager) writeSharedMemory(buf *paramBuf) {
	if !dm.shmReady || dm.memView == 0 {
		return
	}

	src := unsafe.Slice((*byte)(unsafe.Pointer(buf)), sharedMemBytes)
	dst := unsafe.Slice((*byte)(unsafe.Pointer(dm.memView)), sharedMemBytes)
	copy(dst, src)

	if dm.eventHandle != 0 {
		_ = windows.SetEvent(dm.eventHandle)
	}
}

func (dm *DSPManager) shouldDispatchPayload(key string, arrSize int, payload []byte) bool {
	if dm.payloads == nil {
		dm.payloads = make(map[string]payloadCache)
	}

	prev, ok := dm.payloads[key]
	if ok && prev.arrSize == arrSize && bytesEqual(prev.data, payload) {
		return false
	}

	dm.payloads[key] = payloadCache{
		arrSize: arrSize,
		data:    append([]byte(nil), payload...),
	}
	return true
}

func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func boolCenti(b bool) int {
	if b {
		return 100
	}
	return 0
}

func centi(v float64) int {
	return int(math.Round(v * 100.0))
}

func dbLinearCenti(db float64) int {
	linear := math.Pow(10.0, db/20.0)
	return int(math.Round(linear * 100.0))
}

func parseBassMode(mode string) int {
	switch mode {
	case "Pure Bass":
		return 1
	default:
		return 0
	}
}

func parseClarityMode(mode string) int {
	switch mode {
	case "OZone+":
		return 1
	case "X-HiFi":
		return 2
	default:
		return 0
	}
}

func speakerHz(size int) int {
	return speakerSizeToHz[clampInt(size, 0, len(speakerSizeToHz)-1)]
}

func serializeCString(value string) []byte {
	return []byte(strings.TrimSpace(value))
}

func serializeDDCPayload(state DDCState) ([]byte, int) {
	count44100 := (len(state.Coeffs44100) / 5) * 5
	count48000 := (len(state.Coeffs48000) / 5) * 5
	if count44100 == 0 || count48000 == 0 {
		return nil, 0
	}

	count := count44100
	if count48000 < count {
		count = count48000
	}

	values := make([]float32, 0, count*2)
	for i := 0; i < count; i++ {
		values = append(values, float32(state.Coeffs44100[i]))
	}
	for i := 0; i < count; i++ {
		values = append(values, float32(state.Coeffs48000[i]))
	}

	return serializeFloat32Payload(values), count
}

func serializeFloat32Payload(values []float32) []byte {
	payload := make([]byte, 4*len(values))
	for i, value := range values {
		binary.LittleEndian.PutUint32(payload[i*4:], math.Float32bits(value))
	}
	return payload
}
