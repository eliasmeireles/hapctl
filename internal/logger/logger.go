package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/natefinch/lumberjack.v2"
)

const (
	DefaultLogPath    = "/var/log/hapctl/hapctl.log"
	FallbackLogPath   = ".hapctl/log/hapctl.log"
	DefaultMaxSize    = 100
	DefaultMaxBackups = 7
	DefaultMaxAge     = 7
	DefaultCompress   = true
)

type Logger struct {
	infoLogger  *log.Logger
	warnLogger  *log.Logger
	errorLogger *log.Logger
	debugLogger *log.Logger
}

var (
	defaultLogger *Logger
	debugEnabled  bool
)

func Init(logPath string) error {
	if logPath == "" {
		logPath = DefaultLogPath
	}

	logDir := filepath.Dir(logPath)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		// Try to set proper permissions if directory was created
		os.Chmod(logDir, 0755)
		homeDir, homeErr := os.UserHomeDir()
		if homeErr != nil {
			log.Printf("[WARNING] Failed to create log directory %s: %v. Could not determine home directory: %v", logDir, err, homeErr)
			return fmt.Errorf("failed to create log directory: %w", err)
		}

		fallbackPath := filepath.Join(homeDir, FallbackLogPath)
		fallbackDir := filepath.Dir(fallbackPath)

		log.Printf("[WARNING] Failed to create log directory %s: %v. Using fallback: %s", logDir, err, fallbackPath)

		if err := os.MkdirAll(fallbackDir, 0755); err != nil {
			return fmt.Errorf("failed to create fallback log directory: %w", err)
		}

		logPath = fallbackPath
	}

	logWriter := &lumberjack.Logger{
		Filename:   logPath,
		MaxSize:    DefaultMaxSize,
		MaxBackups: DefaultMaxBackups,
		MaxAge:     DefaultMaxAge,
		Compress:   DefaultCompress,
	}

	multiWriter := io.MultiWriter(os.Stdout, logWriter)

	defaultLogger = &Logger{
		infoLogger:  log.New(multiWriter, "[INFO] ", log.LstdFlags),
		warnLogger:  log.New(multiWriter, "[WARN] ", log.LstdFlags),
		errorLogger: log.New(multiWriter, "[ERROR] ", log.LstdFlags),
		debugLogger: log.New(multiWriter, "[DEBUG] ", log.LstdFlags),
	}

	return nil
}

func Info(format string, v ...interface{}) {
	if defaultLogger == nil {
		log.Printf("[INFO] "+format, v...)
		return
	}
	defaultLogger.infoLogger.Printf(format, v...)
}

func Warn(format string, args ...interface{}) {
	if defaultLogger == nil {
		log.Printf("[WARN] "+format, args...)
		return
	}
	defaultLogger.warnLogger.Printf(format, args...)
}

func Error(format string, args ...interface{}) {
	if defaultLogger == nil {
		log.Printf("[ERROR] "+format, args...)
		return
	}
	defaultLogger.errorLogger.Printf(format, args...)
}

func Debug(format string, v ...interface{}) {
	if !debugEnabled {
		return
	}
	if defaultLogger == nil {
		log.Printf("[DEBUG] "+format, v...)
		return
	}
	defaultLogger.debugLogger.Printf(format, v...)
}

func SetDebug(enabled bool) {
	debugEnabled = enabled
}

func Fatal(format string, v ...interface{}) {
	Error(format, v...)
	os.Exit(1)
}

func LogMonitoring(logPath string, report interface{}) error {
	if logPath == "" {
		logPath = "/var/log/hapctl/monitoring.log"
	}

	logDir := filepath.Dir(logPath)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		homeDir, homeErr := os.UserHomeDir()
		if homeErr != nil {
			log.Printf("[WARNING] Failed to create monitoring log directory %s: %v. Could not determine home directory: %v", logDir, err, homeErr)
			return fmt.Errorf("failed to create monitoring log directory: %w", err)
		}

		fallbackPath := filepath.Join(homeDir, ".hapctl/log/monitoring.log")
		fallbackDir := filepath.Dir(fallbackPath)

		log.Printf("[WARNING] Failed to create monitoring log directory %s: %v. Using fallback: %s", logDir, err, fallbackPath)

		if err := os.MkdirAll(fallbackDir, 0755); err != nil {
			return fmt.Errorf("failed to create fallback monitoring log directory: %w", err)
		}

		logPath = fallbackPath
	}

	logWriter := &lumberjack.Logger{
		Filename:   logPath,
		MaxSize:    DefaultMaxSize,
		MaxBackups: DefaultMaxBackups,
		MaxAge:     DefaultMaxAge,
		Compress:   DefaultCompress,
	}

	logger := log.New(logWriter, "", log.LstdFlags)
	logger.Printf("%v", report)

	return nil
}

func GetTimestamp() string {
	return time.Now().Format("2006-01-02 15:04:05")
}
