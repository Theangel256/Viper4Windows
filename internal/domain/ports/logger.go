package ports

// Logger provides structured logging capabilities
type Logger interface {
	// Debug logs debug-level messages
	Debug(msg string, fields ...interface{})

	// Info logs informational messages
	Info(msg string, fields ...interface{})

	// Warn logs warning messages
	Warn(msg string, fields ...interface{})

	// Error logs error messages
	Error(msg string, err error, fields ...interface{})

	// WithContext returns a logger with added context
	WithContext(ctx string) Logger
}
