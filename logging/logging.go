package logging

import (
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
)

func NewLogger(logPath string, logLevel string) *logrus.Logger {
	logger := logrus.New()

	// Create logs directory if it doesn't exist
	logDir := filepath.Dir(logPath)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		logger.Warnf("Failed to create log directory %s: %v", logDir, err)
		logger.SetOutput(os.Stdout)
		return logger
	}

	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		logger.Warnf("Failed to open log file %s: %v", logPath, err)
		logger.SetOutput(os.Stdout)
	} else {
		// Выводим логи и в файл, и в консоль
		logger.SetOutput(io.MultiWriter(file, os.Stdout))
	}
	logger.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})

	// loglevel to lowercase
	logLevel = strings.ToLower(logLevel)
	switch logLevel {
	case "debug":
		logger.SetLevel(logrus.DebugLevel)
	case "info":
		logger.SetLevel(logrus.InfoLevel)
	case "warn":
		logger.SetLevel(logrus.WarnLevel)
	case "error":
		logger.SetLevel(logrus.ErrorLevel)
	case "fatal":
		logger.SetLevel(logrus.FatalLevel)
	case "panic":
		logger.SetLevel(logrus.PanicLevel)
	default:
		logger.SetLevel(logrus.InfoLevel)
	}

	return logger
}
