package dap

import (
	strFmt "fmt"

	"github.com/open-policy-agent/opa/v1/logging"
)

type DebugLogger struct {
	Local           logging.Logger
	ProtocolManager *ProtocolManager

	level         logging.Level
	remoteEnabled bool
}

func NewDebugLogger(localLogger logging.Logger, level logging.Level) *DebugLogger {
	return &DebugLogger{Local: localLogger, level: level}
}

func NoOpLogger() *DebugLogger {
	return &DebugLogger{Local: logging.NewNoOpLogger(), level: logging.Error}
}

func (l *DebugLogger) Debug(fmt string, a ...any) {
	if l == nil {
		return
	}

	l.Local.Debug(fmt, a...)

	l.send(logging.Debug, fmt, a...)
}

func (l *DebugLogger) Info(fmt string, a ...any) {
	if l == nil {
		return
	}

	l.Local.Info(fmt, a...)

	l.send(logging.Info, fmt, a...)
}

func (l *DebugLogger) Error(fmt string, a ...any) {
	if l == nil {
		return
	}

	l.Local.Error(fmt, a...)

	l.send(logging.Error, fmt, a...)
}

func (l *DebugLogger) Warn(fmt string, a ...any) {
	if l == nil {
		return
	}

	l.Local.Warn(fmt, a...)

	l.send(logging.Warn, fmt, a...)
}

func (l *DebugLogger) WithFields(map[string]any) logging.Logger {
	if l == nil {
		return nil
	}

	return l
}

func (l *DebugLogger) GetLevel() logging.Level {
	if l == nil {
		return 0
	}

	if l.Local.GetLevel() > l.level {
		return l.level
	}

	return l.level
}

func (l *DebugLogger) SetRemoteEnabled(enabled bool) {
	if l == nil {
		return
	}

	l.remoteEnabled = enabled
}

func (l *DebugLogger) SetLevel(level logging.Level) {
	if l == nil {
		return
	}

	l.level = level
}

func (l *DebugLogger) SetLevelFromString(level string) {
	if l == nil {
		return
	}

	l.remoteEnabled = true

	switch level {
	case "error":
		l.level = logging.Error
	case "warn":
		l.level = logging.Warn
	case "info":
		l.level = logging.Info
	case "debug":
		l.level = logging.Debug
	}
}

func (l *DebugLogger) send(level logging.Level, fmt string, a ...any) {
	if l == nil || l.ProtocolManager == nil || !l.remoteEnabled || level > l.level {
		return
	}

	var levelStr string

	switch level {
	case logging.Error:
		levelStr = "ERROR"
	case logging.Warn:
		levelStr = "WARN"
	case logging.Info:
		levelStr = "INFO"
	case logging.Debug:
		levelStr = "DEBUG"
	default:
		levelStr = "UNKNOWN"
	}

	message := strFmt.Sprintf(fmt, a...)
	output := strFmt.Sprintf("%s: %s\n", levelStr, message)

	l.ProtocolManager.SendEvent(NewOutputEvent("console", output))
}
