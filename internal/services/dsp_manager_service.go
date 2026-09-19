package services

import (
	"fmt"
	"sync"

	"viper4windows/internal/domain/models"
	"viper4windows/internal/domain/ports"
	"viper4windows/internal/dsp/protocol"
)

// ═══════════════════════════════════════════════════════════════════════════
// DSP Manager Service
// ═══════════════════════════════════════════════════════════════════════════
// Coordinates DSP control via both DLL dispatch and shared memory paths.
// Manages payload caching to avoid redundant dispatches.
//
// The actual PARAM_HP_*/PARAM_SPK_* command list used to be built by hand
// in this file (a ~200 line buildDLLCommands + ~115 lines of private
// centi/clamp/boolInt/etc helpers). That is now internal/dsp/protocol's
// job — BuildDispatchCommands() is the exact same logic, tested, and able
// to target Speaker as well as Headphone. This file only converts the
// result into calls on ports.DLLPort and decides which of those calls are
// worth re-sending.
//
// NOTE: this still only dispatches the Headphone namespace, matching
// today's shipped behavior exactly (see the protocol package's own notes
// on the app never having dispatched PARAM_SPK_* at all). Switching the
// namespace based on the active endpoint's real form factor is a
// behavior change, not a dedup, so it's deliberately left out of this
// pass — flip the protocol.Headphone below to protocol.Speaker (or add a
// namespace parameter here) when you're ready to make that change on
// its own.

type payloadCache struct {
	arrSize uint32
	data    []byte
}

type DSPManagerService struct {
	logger    ports.Logger
	dll       ports.DLLPort
	shm       ports.SharedMemoryPort
	paramSvc  *DSPParameterService
	dllReady  bool
	shmReady  bool
	payloads  map[int]payloadCache
	payloadMu sync.RWMutex
}

func NewDSPManagerService(
	logger ports.Logger,
	dll ports.DLLPort,
	shm ports.SharedMemoryPort,
	paramSvc *DSPParameterService,
) *DSPManagerService {
	return &DSPManagerService{
		logger:   logger.WithContext("DSPManager"),
		dll:      dll,
		shm:      shm,
		paramSvc: paramSvc,
		dllReady: dll != nil && dll.IsLoaded(),
		shmReady: shm != nil && shm.IsReady(),
		payloads: make(map[int]payloadCache),
	}
}

func (s *DSPManagerService) IsDLLReady() bool {
	return s.dllReady
}

func (s *DSPManagerService) IsSHMReady() bool {
	return s.shmReady
}

// ApplyChanges applies DSP state via all available paths (DLL + SHM)
func (s *DSPManagerService) ApplyChanges(state models.DSPState) error {
	var wg sync.WaitGroup
	errChan := make(chan error, 2)

	// Apply via DLL path if available
	if s.dllReady {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := s.applyViaDLL(state); err != nil {
				errChan <- fmt.Errorf("DLL path: %w", err)
			}
		}()
	}

	// Apply via SHM path if available
	if s.shmReady {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := s.applyViaSHM(state); err != nil {
				errChan <- fmt.Errorf("SHM path: %w", err)
			}
		}()
	}

	wg.Wait()
	close(errChan)

	// Collect errors
	var errs []error
	for err := range errChan {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return fmt.Errorf("ApplyChanges: %v", errs)
	}

	return nil
}

func (s *DSPManagerService) applyViaDLL(state models.DSPState) error {
	cmds := protocol.BuildDispatchCommands(state, protocol.Headphone)
	for _, cmd := range cmds {
		if cmd.ArrSize > 0 && cmd.Payload != nil {
			if !s.shouldDispatchPayload(cmd.Param, cmd.ArrSize, cmd.Payload) {
				continue
			}
			s.dll.DispatchPayload(cmd.Param, int(cmd.Val1), int(cmd.Val2), int(cmd.Val3), int(cmd.Val4), int(cmd.ArrSize), cmd.Payload)
		} else {
			s.dll.Dispatch(cmd.Param, int(cmd.Val1), int(cmd.Val2), int(cmd.Val3), int(cmd.Val4))
		}
	}
	return nil
}

func (s *DSPManagerService) applyViaSHM(state models.DSPState) error {
	data, err := s.paramSvc.EncodeState(state)
	if err != nil {
		return fmt.Errorf("encode state: %w", err)
	}
	return s.shm.Write(data)
}

// shouldDispatchPayload avoids re-sending a bulk payload (DDC
// coefficients, a convolver kernel path) when it hasn't changed since
// the last call — keyed by the command's own PARAM_* code, since that
// already uniquely identifies which bulk payload this is.
func (s *DSPManagerService) shouldDispatchPayload(param int, arrSize uint32, payload []byte) bool {
	s.payloadMu.RLock()
	prev, ok := s.payloads[param]
	s.payloadMu.RUnlock()

	if ok && prev.arrSize == arrSize && bytesEqual(prev.data, payload) {
		return false
	}

	s.payloadMu.Lock()
	s.payloads[param] = payloadCache{
		arrSize: arrSize,
		data:    append([]byte(nil), payload...),
	}
	s.payloadMu.Unlock()

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
