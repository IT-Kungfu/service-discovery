package logger

import (
	"github.com/evalphobia/logrus_sentry"
	"github.com/sirupsen/logrus"
	"strings"
	"time"
)

type Logger struct {
	cfg *Config
	l   *logrus.Logger
}

type Config struct {
	LogLevel     string
	SentryDSN    string
	LogstashAddr string
	ServiceName  string
	InstanceName string
}

func New(cfg *Config) (*Logger, error) {
	log := &Logger{
		cfg: cfg,
	}
	return log, log.createLogger()
}

func (log *Logger) createLogger() error {
	log.l = logrus.New()

	level, err := logrus.ParseLevel(log.cfg.LogLevel)
	if err != nil {
		return err
	}

	log.l.SetLevel(level)
	log.l.SetReportCaller(true)
	log.l.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "02.01.2006 15:04:05",
	})

	if log.cfg.SentryDSN != "" {
		sentryHook, err := logrus_sentry.NewAsyncWithTagsSentryHook(log.cfg.SentryDSN, map[string]string{"service": log.cfg.ServiceName}, []logrus.Level{
			logrus.PanicLevel,
			logrus.FatalLevel,
			logrus.ErrorLevel,
		})
		if err != nil {
			return err
		}

		sentryHook.StacktraceConfiguration.Enable = true
		sentryHook.StacktraceConfiguration.IncludeErrorBreadcrumb = true
		log.l.Hooks.Add(sentryHook)
	}

	if log.cfg.LogstashAddr != "" {
		logstashHook, err := NewAsyncHook("tcp", log.cfg.LogstashAddr, log.cfg.ServiceName)
		if err != nil {
			return err
		}

		logstashHook.ReconnectBaseDelay = time.Second
		logstashHook.ReconnectDelayMultiplier = 2
		logstashHook.MaxReconnectRetries = 10
		if log.cfg.InstanceName != "" {
			logstashHook.WithField("instance", log.cfg.InstanceName)
		}
		logstashHook.SetLevels([]logrus.Level{
			logrus.PanicLevel,
			logrus.FatalLevel,
			logrus.ErrorLevel,
			logrus.InfoLevel,
		})
		log.l.Hooks.Add(logstashHook)
	}

	log.l.Infof("Logger started level:%s service:%s sentry:%s logstash:%s", log.cfg.LogLevel, log.cfg.ServiceName, log.cfg.SentryDSN, log.cfg.LogstashAddr)

	return nil
}

func (log *Logger) Debug(args ...interface{}) {
	log.l.Debug(args...)
}

func (log *Logger) Debugf(format string, args ...interface{}) {
	log.l.Debugf(format, args...)
}

func (log *Logger) Info(args ...interface{}) {
	log.l.Info(args...)
}

func (log *Logger) Infof(format string, args ...interface{}) {
	log.l.Infof(format, args...)
}

func (log *Logger) Warn(args ...interface{}) {
	log.l.Warn(args...)
}

func (log *Logger) Warnf(format string, args ...interface{}) {
	log.l.Warnf(format, args...)
}

func (log *Logger) Error(args ...interface{}) {
	log.l.Error(args...)
}

func (log *Logger) Errorf(format string, args ...interface{}) {
	log.l.Errorf(format, args...)
}

func (log *Logger) Panic(args ...interface{}) {
	log.l.Panic(args...)
}

func (log *Logger) Panicf(format string, args ...interface{}) {
	log.l.Panicf(format, args...)
}

func (log *Logger) Fatal(args ...interface{}) {
	log.l.Fatal(args...)
}

func (log *Logger) Fatalf(format string, args ...interface{}) {
	log.l.Fatalf(format, args...)
}

func (log *Logger) ETCDValueChanged(key string, value []byte, cfg interface{}) {
	if strings.HasSuffix(key, "/log_level") || strings.HasSuffix(key, "/sentry_dsn") {
		log.Infof("logger config changed: %s %s", key, value)
		_ = log.createLogger()
	}
}
