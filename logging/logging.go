package logging

import (
	"io"
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
)

func NewLogger(logPath string) *logrus.Logger {
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
	return logger
}
