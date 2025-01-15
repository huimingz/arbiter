package arbiter

import (
	"context"
	"fmt"
	"log"
	"os"
)

// Logger is the interface that wraps the basic logging methods.
type Logger interface {
	// Debug logs a debug message.
	Debug(ctx context.Context, msg string, args ...any)
	// Info logs an info message.
	Info(ctx context.Context, msg string, args ...any)
	// Warn logs a warning message.
	Warn(ctx context.Context, msg string, args ...any)
	// Error logs an error message.
	Error(ctx context.Context, msg string, args ...any)
}

// defaultLogger is the default implementation of Logger interface.
type defaultLogger struct {
	debug *log.Logger
	info  *log.Logger
	warn  *log.Logger
	error *log.Logger
}

// newDefaultLogger creates a new default logger.
func newDefaultLogger() *defaultLogger {
	return &defaultLogger{
		debug: log.New(os.Stdout, "[DEBUG] ", log.LstdFlags),
		info:  log.New(os.Stdout, "[INFO] ", log.LstdFlags),
		warn:  log.New(os.Stdout, "[WARN] ", log.LstdFlags),
		error: log.New(os.Stderr, "[ERROR] ", log.LstdFlags),
	}
}

func (l *defaultLogger) Debug(ctx context.Context, msg string, args ...any) {
	if len(args) > 0 {
		msg = fmt.Sprintf(msg, args...)
	}
	l.debug.Println(msg)
}

func (l *defaultLogger) Info(ctx context.Context, msg string, args ...any) {
	if len(args) > 0 {
		msg = fmt.Sprintf(msg, args...)
	}
	l.info.Println(msg)
}

func (l *defaultLogger) Warn(ctx context.Context, msg string, args ...any) {
	if len(args) > 0 {
		msg = fmt.Sprintf(msg, args...)
	}
	l.warn.Println(msg)
}

func (l *defaultLogger) Error(ctx context.Context, msg string, args ...any) {
	if len(args) > 0 {
		msg = fmt.Sprintf(msg, args...)
	}
	l.error.Println(msg)
}

// NoopLogger is a logger that does nothing.
type NoopLogger struct{}

func (l *NoopLogger) Debug(ctx context.Context, msg string, args ...any) {}
func (l *NoopLogger) Info(ctx context.Context, msg string, args ...any)  {}
func (l *NoopLogger) Warn(ctx context.Context, msg string, args ...any)  {}
func (l *NoopLogger) Error(ctx context.Context, msg string, args ...any) {}
