package telemetry

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
)

// Logger provides structured logging backed by log/slog.
type Logger struct {
	inner   *slog.Logger
	level   slog.Level
	mu      sync.Mutex
	writers []io.Writer
}

// NewLogger creates a new structured logger.
func NewLogger(verbose bool) *Logger {
	level := slog.LevelInfo
	if verbose {
		level = slog.LevelDebug
	}

	output := os.Stderr
	handlerOpts := &slog.HandlerOptions{Level: level}
	handler := slog.NewTextHandler(output, handlerOpts)

	return &Logger{
		inner:   slog.New(handler),
		level:   level,
		writers: []io.Writer{output},
	}
}

// WithFile adds file output to the logger.
func (l *Logger) WithFile(path string) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}

	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}

	l.writers = append(l.writers, file)

	multi := io.MultiWriter(l.writers...)
	handlerOpts := &slog.HandlerOptions{Level: l.level}
	handler := slog.NewTextHandler(multi, handlerOpts)
	l.inner = slog.New(handler)

	return nil
}

// WithFields returns a new logger with additional key-value fields.
func (l *Logger) WithFields(fields map[string]interface{}) *Logger {
	l.mu.Lock()
	defer l.mu.Unlock()

	args := make([]any, 0, len(fields)*2)
	for k, v := range fields {
		args = append(args, k, v)
	}

	writersCopy := make([]io.Writer, len(l.writers))
	copy(writersCopy, l.writers)

	return &Logger{
		inner:   l.inner.With(args...),
		level:   l.level,
		writers: writersCopy,
	}
}

// Close closes all file writers opened via WithFile.
func (l *Logger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	var firstErr error
	for _, w := range l.writers {
		if f, ok := w.(*os.File); ok && f != os.Stderr && f != os.Stdout {
			if err := f.Close(); err != nil && firstErr == nil {
				firstErr = err
			}
		}
	}
	return firstErr
}

// Slog returns the underlying *slog.Logger.
func (l *Logger) Slog() *slog.Logger {
	return l.inner
}

// Debug logs at debug level.
func (l *Logger) Debug(msg string, keyvals ...interface{}) {
	l.inner.Debug(msg, keyvals...)
}

// Info logs at info level.
func (l *Logger) Info(msg string, keyvals ...interface{}) {
	l.inner.Info(msg, keyvals...)
}

// Warn logs at warn level.
func (l *Logger) Warn(msg string, keyvals ...interface{}) {
	l.inner.Warn(msg, keyvals...)
}

// Error logs at error level.
func (l *Logger) Error(msg string, keyvals ...interface{}) {
	l.inner.Error(msg, keyvals...)
}
