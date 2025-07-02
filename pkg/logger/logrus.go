package logger

import (
	"os"

	"github.com/sirupsen/logrus"
)

var base = logrus.New()

func init() {
	base.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})
	base.SetOutput(os.Stdout)
	base.SetLevel(logrus.DebugLevel)
}

func Info(args ...interface{}) {
	base.Info(args...)
}

func Error(args ...interface{}) {
	base.Error(args...)
}

func Warn(args ...interface{}) {
	base.Warn(args...)
}

func WithField(key string, value interface{}) *logrus.Entry {
	return base.WithField(key, value)
}

func WithFields(fields logrus.Fields) *logrus.Entry {
	return base.WithFields(fields)
}

func WithError(err error) *logrus.Entry {
	return base.WithError(err)
}

