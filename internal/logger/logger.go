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
	DefaultLogPath       = "/var/log/hapctl/hapctl.log"
	DefaultMaxSize       = 100
	DefaultMaxBackups    = 7
	DefaultMaxAge        = 7
	DefaultCompress      = true
)

type Logger struct {
	infoLogger  *log.Logger
	errorLogger *log.Logger
	debugLogger *log.Logger
}

var defaultLogger *Logger

func Init(logPath string) error {
	if logPath == "" {
		logPath = DefaultLogPath
	}

	logDir := filepath.Dir(logPath)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
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

func Error(format string, v ...interface{}) {
	if defaultLogger == nil {
		log.Printf("[ERROR] "+format, v...)
		return
	}
	defaultLogger.errorLogger.Printf(format, v...)
}

func Debug(format string, v ...interface{}) {
	if defaultLogger == nil {
		log.Printf("[DEBUG] "+format, v...)
		return
	}
	defaultLogger.debugLogger.Printf(format, v...)
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
		return fmt.Errorf("failed to create monitoring log directory: %w", err)
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
