package utils

import (
	"fmt"
	"log"
	"os"
	"strings"

	"viper4windows/internal/domain/ports"
)

// stdLogger is a minimal ports.Logger backed by the standard library's
// log package. NewApp() has called utils.NewLogger(...) since the
// Clean Architecture refactor started, but no concrete implementation
// of ports.Logger existed anywhere in the tree — this was the missing
// piece.
type stdLogger struct {
	base    *log.Logger
	context string
}

// NewLogger creates a root logger. name becomes the first context
// segment (e.g. NewLogger("ViPER4Windows") then
// .WithContext("DSPManager") logs as "[ViPER4Windows.DSPManager]").
func NewLogger(name string) ports.Logger {
	return &stdLogger{
		base:    log.New(os.Stdout, "", log.LstdFlags),
		context: name,
	}
}

func (l *stdLogger) WithContext(ctx string) ports.Logger {
	next := ctx
	if l.context != "" {
		next = l.context + "." + ctx
	}
	return &stdLogger{base: l.base, context: next}
}

func (l *stdLogger) Debug(msg string, fields ...interface{}) { l.log("DEBUG", msg, fields...) }
func (l *stdLogger) Info(msg string, fields ...interface{})  { l.log("INFO", msg, fields...) }
func (l *stdLogger) Warn(msg string, fields ...interface{})  { l.log("WARN", msg, fields...) }

func (l *stdLogger) Error(msg string, err error, fields ...interface{}) {
	if err != nil {
		fields = append(fields, "error", err.Error())
	}
	l.log("ERROR", msg, fields...)
}

func (l *stdLogger) log(level, msg string, fields ...interface{}) {
	l.base.Printf("[%s] %s: %s%s", l.context, level, msg, formatFields(fields))
}

// formatFields renders "k1=v1 k2=v2 ..." from an alternating
// key/value slice, tolerating an odd trailing element rather than
// panicking on a call site that forgot a value.
func formatFields(fields []interface{}) string {
	if len(fields) == 0 {
		return ""
	}
	var b strings.Builder
	for i := 0; i < len(fields); i += 2 {
		b.WriteString(" ")
		if i+1 < len(fields) {
			fmt.Fprintf(&b, "%v=%v", fields[i], fields[i+1])
		} else {
			fmt.Fprintf(&b, "%v=?", fields[i])
		}
	}
	return b.String()
}
