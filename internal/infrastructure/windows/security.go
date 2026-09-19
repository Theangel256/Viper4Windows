package windows

import (
	"fmt"
	"unsafe"

	"viper4windows/internal/domain/ports"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

// ═══════════════════════════════════════════════════════════════════════════
// Windows Security Service
// ═══════════════════════════════════════════════════════════════════════════
// Handles privilege elevation checks and security descriptor creation

type TokenElevation struct {
	TokenIsElevated uint32
}

type SecurityService struct {
	logger ports.Logger
}

func NewSecurityService(logger ports.Logger) *SecurityService {
	return &SecurityService{
		logger: logger.WithContext("Security"),
	}
}

// IsElevated checks if the process has administrative privileges
func (s *SecurityService) IsElevated() bool {
	// Method 1: Check token elevation (primary method)
	var token windows.Token
	err := windows.OpenProcessToken(windows.CurrentProcess(), windows.TOKEN_QUERY, &token)
	if err == nil {
		defer token.Close()

		var elevation TokenElevation
		var returnedLen uint32
		err = windows.GetTokenInformation(
			token,
			windows.TokenElevation,
			(*byte)(unsafe.Pointer(&elevation)),
			uint32(unsafe.Sizeof(elevation)),
			&returnedLen,
		)
		if err == nil {
			isElevated := elevation.TokenIsElevated != 0
			s.logger.Debug("elevation check via token", "elevated", isElevated)
			return isElevated
		}
		s.logger.Warn("failed to get token information", "error", err)
	}

	// Method 2: Fallback - try to access protected registry key
	// If we can write to HKLM, we're likely elevated
	k, err := registry.OpenKey(
		registry.LOCAL_MACHINE,
		`SOFTWARE\Microsoft\Windows NT\CurrentVersion`,
		registry.SET_VALUE,
	)
	if err == nil {
		k.Close()
		s.logger.Debug("elevation check via registry", "elevated", true)
		return true
	}

	s.logger.Debug("elevation check failed", "elevated", false)
	return false
}

// RequireAdmin returns an error if the process doesn't have admin privileges
func (s *SecurityService) RequireAdmin() error {
	if !s.IsElevated() {
		return fmt.Errorf("ACCESS_DENIED: Administrator privileges required.\n" +
			"Right-click the application and select 'Run as Administrator'")
	}
	return nil
}

// BuildSecurityAttributes creates a security descriptor for cross-session IPC
// This allows communication between user sessions and services
func (s *SecurityService) BuildSecurityAttributes() (interface{}, error) {
	// Create a security descriptor that allows all users
	// This is necessary for shared memory that needs to be accessible
	// by both the application and the Windows Audio Service

	// For now, return nil to use default security
	// A full implementation would create a SECURITY_DESCRIPTOR with specific SIDs
	s.logger.Debug("building security attributes for IPC")

	return nil, nil
}

// ═══════════════════════════════════════════════════════════════════════════
// Helper Functions
// ═══════════════════════════════════════════════════════════════════════════

// GetProcessIntegrityLevel returns the integrity level of the current process
func (s *SecurityService) GetProcessIntegrityLevel() (string, error) {
	var token windows.Token
	err := windows.OpenProcessToken(windows.CurrentProcess(), windows.TOKEN_QUERY, &token)
	if err != nil {
		return "", fmt.Errorf("open process token: %w", err)
	}
	defer token.Close()

	// This is a simplified version - a full implementation would query
	// TOKEN_MANDATORY_LABEL to get the actual integrity level
	if s.IsElevated() {
		return "High", nil
	}
	return "Medium", nil
}
