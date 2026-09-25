package windows

import (
	"os/exec"
	"strings"
	"time"

	"viper4windows/internal/domain/models"
	"viper4windows/internal/domain/ports"
)

// ═══════════════════════════════════════════════════════════════════════════
// Audio Engine Service
// ═══════════════════════════════════════════════════════════════════════════
// Restarts the Windows Audio service to apply APO changes.
// Ported from original driveManager.go RestartAudioEngine.

type AudioEngineService struct {
	logger ports.Logger
}

func NewAudioEngineService(logger ports.Logger) *AudioEngineService {
	return &AudioEngineService{
		logger: logger.WithContext("AudioEngine"),
	}
}

// Restart stops and restarts the Windows Audio service
func (s *AudioEngineService) Restart() error {
	s.logger.Info("restarting audio engine")

	// Stop AudioEndpointBuilder
	exec.Command("net", "stop", "AudioEndpointBuilder", "/y").Run()
	time.Sleep(1 * time.Second)

	// Start services in order
	services := []string{"AudioEndpointBuilder", "AudioSrv"}
	for _, svc := range services {
		cmd := exec.Command("net", "start", svc)
		if err := cmd.Run(); err != nil {
			s.logger.Warn("service start warning", "service", svc, "error", err)
		}
	}

	s.logger.Info("audio engine restarted")
	return nil
}

// IsRunning checks if the audio service is active.
//
// FIX: `.Run()` alone only reports whether sc.exe itself failed to
// execute — sc query exits 0 for a STOPPED-but-existing service too,
// so this used to report "running" for a dead service. Now actually
// parses STATE from the output.
func (s *AudioEngineService) IsRunning() bool {
	out, err := exec.Command("sc", "query", "AudioEndpointBuilder").Output()
	if err != nil {
		return false
	}
	return strings.Contains(string(out), "RUNNING")
}

// GetStatus implements ports.AudioEnginePort. SampleRate/ProcessTime/
// BufferSize would need real telemetry the APO doesn't send back yet
// (SharedMemoryService.ReadAPOStatus() is still a placeholder — see
// its own comment) — left at zero rather than a made-up number.
func (s *AudioEngineService) GetStatus() (models.AudioEngineStatus, error) {
	return models.AudioEngineStatus{
		IsRunning: s.IsRunning(),
	}, nil
}
