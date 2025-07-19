package logger

import (
	"os"

	"github.com/sirupsen/logrus"
)

type Logger struct {
	*logrus.Logger
}

func New(level string) *Logger {
	logger := logrus.New()

	// Log Level setzen
	switch level {
	case "debug":
		logger.SetLevel(logrus.DebugLevel)
	case "info":
		logger.SetLevel(logrus.InfoLevel)
	case "warn":
		logger.SetLevel(logrus.WarnLevel)
	case "error":
		logger.SetLevel(logrus.ErrorLevel)
	default:
		logger.SetLevel(logrus.InfoLevel)
	}

	// JSON Format für strukturiertes Logging
	logger.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: "2006-01-02 15:04:05",
	})

	// Output auf stdout
	logger.SetOutput(os.Stdout)

	return &Logger{logger}
}

// Convenience Methoden
func (l *Logger) Info(args ...interface{}) {
	l.Logger.Info(args...)
}

func (l *Logger) Infof(format string, args ...interface{}) {
	l.Logger.Infof(format, args...)
}

func (l *Logger) Error(args ...interface{}) {
	l.Logger.Error(args...)
}

func (l *Logger) Errorf(format string, args ...interface{}) {
	l.Logger.Errorf(format, args...)
}

func (l *Logger) Debug(args ...interface{}) {
	l.Logger.Debug(args...)
}

func (l *Logger) Debugf(format string, args ...interface{}) {
	l.Logger.Debugf(format, args...)
}

func (l *Logger) Warn(args ...interface{}) {
	l.Logger.Warn(args...)
}

func (l *Logger) Warnf(format string, args ...interface{}) {
	l.Logger.Warnf(format, args...)
}

func (l *Logger) Fatal(args ...interface{}) {
	l.Logger.Fatal(args...)
}

func (l *Logger) Fatalf(format string, args ...interface{}) {
	l.Logger.Fatalf(format, args...)
}
