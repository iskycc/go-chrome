package logx

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Logger wraps standard log with file rotation and topic filtering.
type Logger struct {
	mu         sync.Mutex
	stdLogger  *log.Logger
	file       *os.File
	dir        string
	date       string
	retention  int
	guiWriter  io.Writer
}

var defaultLogger *Logger

// Init initializes the default logger.
func Init(dir string, retentionDays int, guiWriter io.Writer) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	l := &Logger{
		dir:       dir,
		date:      time.Now().Format("2006-01-02"),
		retention: retentionDays,
		guiWriter: guiWriter,
	}
	if err := l.rotate(); err != nil {
		return err
	}
	defaultLogger = l
	return nil
}

func (l *Logger) rotate() error {
	if l.file != nil {
		l.file.Close()
	}
	fname := filepath.Join(l.dir, l.date+".log")
	f, err := os.OpenFile(fname, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	l.file = f
	writers := []io.Writer{f}
	if l.guiWriter != nil {
		writers = append(writers, l.guiWriter)
	}
	l.stdLogger = log.New(io.MultiWriter(writers...), "", log.LstdFlags)
	return nil
}

func (l *Logger) checkRotation() {
	today := time.Now().Format("2006-01-02")
	if today != l.date {
		l.mu.Lock()
		defer l.mu.Unlock()
		if today != l.date {
			l.date = today
			l.rotate()
			l.cleanup()
		}
	}
}

func (l *Logger) cleanup() {
	if l.retention <= 0 {
		return
	}
	cutoff := time.Now().AddDate(0, 0, -l.retention)
	entries, err := os.ReadDir(l.dir)
	if err != nil {
		return
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		if info.ModTime().Before(cutoff) {
			os.Remove(filepath.Join(l.dir, e.Name()))
		}
	}
}

func (l *Logger) output(level string, msg string) {
	l.checkRotation()
	l.mu.Lock()
	defer l.mu.Unlock()
	l.stdLogger.Printf("[%s] %s", level, msg)
}

// Info logs an info message.
func Info(msg string) {
	if defaultLogger != nil {
		defaultLogger.output("INFO", msg)
	}
}

// Infof logs a formatted info message.
func Infof(format string, v ...interface{}) {
	Info(fmt.Sprintf(format, v...))
}

// Warn logs a warning.
func Warn(msg string) {
	if defaultLogger != nil {
		defaultLogger.output("WARN", msg)
	}
}

// Warnf logs a formatted warning.
func Warnf(format string, v ...interface{}) {
	Warn(fmt.Sprintf(format, v...))
}

// Error logs an error.
func Error(msg string) {
	if defaultLogger != nil {
		defaultLogger.output("ERROR", msg)
	}
}

// Errorf logs a formatted error.
func Errorf(format string, v ...interface{}) {
	Error(fmt.Sprintf(format, v...))
}

// Debug logs a debug message.
func Debug(msg string) {
	if defaultLogger != nil {
		defaultLogger.output("DEBUG", msg)
	}
}

// Debugf logs a formatted debug message.
func Debugf(format string, v ...interface{}) {
	Debug(fmt.Sprintf(format, v...))
}

// Close closes the logger file.
func Close() {
	if defaultLogger != nil && defaultLogger.file != nil {
		defaultLogger.file.Close()
	}
}
