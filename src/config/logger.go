package config

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"
)

type LogLevel int

const (
	TRACE LogLevel = iota
	DEBUG
	INFO
	WARN
	ERROR
)

var levelNames = []string{
	"TRACE",
	"DEBUG",
	"INFO",
	"WARN",
	"ERROR",
}

func (l LogLevel) String() string {
	if l < 0 || int(l) >= len(levelNames) {
		return "UNKNOWN"
	}
	return levelNames[l]
}

func ParseLevel(level string) (LogLevel, error) {
	switch strings.ToUpper(level) {
	case "TRACE":
		return TRACE, nil
	case "DEBUG":
		return DEBUG, nil
	case "INFO":
		return INFO, nil
	case "WARN":
		return WARN, nil
	case "ERROR":
		return ERROR, nil
	default:
		return INFO, fmt.Errorf("unknown log level: %s", level)
	}
}

type Logger struct {
	name   string
	level  LogLevel
	output io.Writer
	mu     sync.Mutex
}

var defaultOutput io.Writer = os.Stdout
var defaultLevel LogLevel = INFO

func SetDefaultLevel(level LogLevel) {
	defaultLevel = level
}

func NewLogger(name string) *Logger {
	return &Logger{
		name:   name,
		level:  defaultLevel,
		output: defaultOutput,
	}
}

func (l *Logger) Named(name string) *Logger {
	return &Logger{
		name:   name,
		level:  l.level,
		output: l.output,
	}
}

func (l *Logger) SetLevel(level LogLevel) {
	l.level = level
}

func (l *Logger) log(level LogLevel, format string, args ...interface{}) {
	if level < l.level {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	// Format: <timestamp>[LEVEL][Name] Message
	prefix := fmt.Sprintf("%s[%s][%s] ", timestamp, level.String(), l.name)
	msg := fmt.Sprintf(format, args...)
	
	fmt.Fprintln(l.output, prefix+msg)
}

func (l *Logger) Trace(format string, args ...interface{}) {
	l.log(TRACE, format, args...)
}

func (l *Logger) Debug(format string, args ...interface{}) {
	l.log(DEBUG, format, args...)
}

func (l *Logger) Info(format string, args ...interface{}) {
	l.log(INFO, format, args...)
}

func (l *Logger) Warn(format string, args ...interface{}) {
	l.log(WARN, format, args...)
}

func (l *Logger) Error(format string, args ...interface{}) {
	l.log(ERROR, format, args...)
}
