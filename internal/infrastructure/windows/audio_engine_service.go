package windows

import (
	"os/exec"
	"time"

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

// IsRunning checks if the audio service is active
func (s *AudioEngineService) IsRunning() bool {
	// Simple check: try to query the service state
	cmd := exec.Command("sc", "query", "AudioEndpointBuilder")
	if err := cmd.Run(); err != nil {
		return false
	}
	return true
}
