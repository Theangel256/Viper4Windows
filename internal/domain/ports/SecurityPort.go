package ports

// SecurityPort handles Windows security and privilege checks
type SecurityPort interface {
	// IsElevated checks if process has admin privileges
	IsElevated() bool

	// RequireAdmin returns error if not running as admin
	RequireAdmin() error

	// BuildSecurityAttributes creates security descriptors for IPC
	BuildSecurityAttributes() (interface{}, error)
}
