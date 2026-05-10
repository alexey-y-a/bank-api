package logger

import (
	"os"

	"github.com/sirupsen/logrus"
)

type Fields = logrus.Fields

func New(level, format string) (*logrus.Logger, error) {
	log := logrus.New()

	parsedLevel, err := logrus.ParseLevel(level)
	if err != nil {
		log.Warnf("unrecognized log level '%s', falling back to 'info'", level)
		parsedLevel = logrus.InfoLevel
	}

	log.SetLevel(parsedLevel)

	switch format {
	case "json":
		log.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: "2006-01-02T15:04:05Z07:00",
		})
	default:
		log.SetFormatter(&logrus.TextFormatter{
			ForceColors:   true,
			FullTimestamp: true,
		})
	}

	log.SetOutput(os.Stdout)

	_ = log.WithFields(logrus.Fields{
		"service": "bank-api",
		"version": "1.0.0",
	})

	log.SetReportCaller(true)

	return log, nil
}

func Info(log *logrus.Logger, msg string, fields logrus.Fields) {
	if fields != nil {
		log.WithFields(fields).Info(msg)
		return
	}
	log.Info(msg)
}

func Warn(log *logrus.Logger, msg string, err error, fields logrus.Fields) {
	if fields == nil {
		fields = logrus.Fields{}
	}

	if err != nil {
		fields["error"] = err.Error()
	}

	log.WithFields(fields).Warn(msg)
}

func Error(log *logrus.Logger, msg string, err error, fields logrus.Fields) {
	if fields == nil {
		fields = logrus.Fields{}
	}

	fields["error"] = err.Error()
	log.WithFields(fields).Error(msg)
}

func Debug(log *logrus.Logger, msg string, fields logrus.Fields) {
	if fields != nil {
		log.WithFields(fields).Debug(msg)
		return
	}
	log.Debug(msg)
}
