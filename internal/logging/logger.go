package logging

import (
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Level constants for convenience
const (
	LevelDebug = slog.LevelDebug
	LevelInfo  = slog.LevelInfo
	LevelWarn  = slog.LevelWarn
	LevelError = slog.LevelError
)

// Logger wraps slog with DrogonClaw-specific functionality
type Logger struct {
	logger    *slog.Logger
	level     *slog.LevelVar
	file      *os.File
	sessionID string
	mu        sync.RWMutex
}

var (
	defaultLogger *Logger
	once          sync.Once
)

// Init initializes the global logger with multi-handler (stderr + file)
// logDir is the directory for log files (e.g., "data/logs")
// level is the minimum log level (e.g., slog.LevelInfo)
func Init(logDir string, level slog.Level) *Logger {
	once.Do(func() {
		defaultLogger = newLogger(logDir, level)
		slog.SetDefault(defaultLogger.logger)
	})
	return defaultLogger
}

// InitWithSession creates a session-scoped logger
func InitWithSession(logDir, sessionID string, level slog.Level) *Logger {
	l := Init(logDir, level)
	return l.WithSession(sessionID)
}

func newLogger(logDir string, level slog.Level) *Logger {
	lvl := new(slog.LevelVar)
	lvl.Set(level)

	// Ensure log directory exists
	if err := os.MkdirAll(logDir, 0755); err != nil {
		// Fallback to stderr only
		handler := slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
			Level: lvl,
		})
		return &Logger{
			logger: slog.New(handler),
			level:  lvl,
		}
	}

	// Create log file with timestamp
	logFile := filepath.Join(logDir, "drogonclaw.log")
	f, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		// Fallback to stderr only
		handler := slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
			Level: lvl,
		})
		return &Logger{
			logger: slog.New(handler),
			level:  lvl,
		}
	}

	// Multi-writer: stderr + file
	multiWriter := io.MultiWriter(os.Stderr, f)

	handler := slog.NewJSONHandler(multiWriter, &slog.HandlerOptions{
		AddSource: false,
		Level:     lvl,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			// Redact sensitive fields
			switch a.Key {
			case "password", "token", "api_key", "authorization", "secret", "credential":
				return slog.String(a.Key, "[REDACTED]")
			}
			return a
		},
	})

	logger := slog.New(handler).With(
		slog.String("service", "drogonclaw"),
		slog.String("version", "2.0.0"),
	)

	return &Logger{
		logger: logger,
		level:  lvl,
		file:   f,
	}
}

// WithSession returns a session-scoped logger
func (l *Logger) WithSession(sessionID string) *Logger {
	if l == nil {
		return nil
	}
	child := l.logger.With(slog.String("session_id", sessionID))
	return &Logger{
		logger:    child,
		level:     l.level,
		file:      l.file,
		sessionID: sessionID,
	}
}

// SetLevel changes the log level at runtime
func (l *Logger) SetLevel(level slog.Level) {
	if l == nil || l.level == nil {
		return
	}
	l.level.Set(level)
}

// GetLevel returns the current log level
func (l *Logger) GetLevel() slog.Level {
	if l == nil || l.level == nil {
		return slog.LevelInfo
	}
	return l.level.Level()
}

// Debug logs a debug message
func (l *Logger) Debug(msg string, args ...any) {
	if l == nil || l.logger == nil {
		return
	}
	l.logger.Debug(msg, args...)
}

// Info logs an info message
func (l *Logger) Info(msg string, args ...any) {
	if l == nil || l.logger == nil {
		return
	}
	l.logger.Info(msg, args...)
}

// Warn logs a warning message
func (l *Logger) Warn(msg string, args ...any) {
	if l == nil || l.logger == nil {
		return
	}
	l.logger.Warn(msg, args...)
}

// Error logs an error message
func (l *Logger) Error(msg string, args ...any) {
	if l == nil || l.logger == nil {
		return
	}
	l.logger.Error(msg, args...)
}

// With returns a logger with additional attributes
func (l *Logger) With(args ...any) *Logger {
	if l == nil || l.logger == nil {
		return nil
	}
	child := l.logger.With(args...)
	return &Logger{
		logger:    child,
		level:     l.level,
		file:      l.file,
		sessionID: l.sessionID,
	}
}

// Enabled reports whether the logger emits at the given level
func (l *Logger) Enabled(ctx context.Context, level slog.Level) bool {
	if l == nil || l.logger == nil {
		return false
	}
	return l.logger.Enabled(ctx, level)
}

// LogToolStart logs the start of a tool execution
func (l *Logger) LogToolStart(tool, args, sessionID string) {
	l.Info("tool_started",
		slog.String("tool", tool),
		slog.String("args", truncateString(args, 1000)),
		slog.String("session_id", sessionID),
		slog.Time("timestamp", time.Now()),
	)
}

// LogToolComplete logs the completion of a tool execution
func (l *Logger) LogToolComplete(tool, result, sessionID string, duration time.Duration, success bool) {
	l.Info("tool_completed",
		slog.String("tool", tool),
		slog.String("result", truncateString(result, 2000)),
		slog.String("session_id", sessionID),
		slog.Duration("duration", duration),
		slog.Bool("success", success),
		slog.Time("timestamp", time.Now()),
	)
}

// LogFinding logs a discovered finding (vulnerability, credential, flag, etc.)
func (l *Logger) LogFinding(findingType, description, source, sessionID string) {
	l.Info("finding_discovered",
		slog.String("finding_type", findingType),
		slog.String("description", description),
		slog.String("source", source),
		slog.String("session_id", sessionID),
		slog.Time("timestamp", time.Now()),
	)
}

// LogAgentDecision logs an agent decision
func (l *Logger) LogAgentDecision(decision, reasoning, sessionID string) {
	l.Info("agent_decision",
		slog.String("decision", decision),
		slog.String("reasoning", truncateString(reasoning, 500)),
		slog.String("session_id", sessionID),
		slog.Time("timestamp", time.Now()),
	)
}

// Close closes the log file
func (l *Logger) Close() error {
	if l == nil || l.file == nil {
		return nil
	}
	return l.file.Close()
}

// Default returns the default logger
func Default() *Logger {
	if defaultLogger == nil {
		return Init("data/logs", slog.LevelInfo)
	}
	return defaultLogger
}

// truncateString truncates a string to maxLen characters
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
