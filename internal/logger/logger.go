package logger

import (
	"github.com/evalphobia/logrus_sentry"
	"github.com/sirupsen/logrus"
)

func NewLogger(logLevel, sentryDSN, serviceName string) *logrus.Logger {
	log := logrus.New()
	l, err := logrus.ParseLevel(logLevel)
	if err != nil {
		panic(err)
	}

	log.SetLevel(l)
	log.SetReportCaller(true)
	log.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "02.01.2006 15:04:05",
	})

	if sentryDSN != "" {
		hook, err := logrus_sentry.NewAsyncWithTagsSentryHook(sentryDSN, map[string]string{"service": serviceName}, []logrus.Level{
			logrus.PanicLevel,
			logrus.FatalLevel,
			logrus.ErrorLevel,
		})
		if err != nil {
			panic(err)
		}

		hook.StacktraceConfiguration.Enable = true
		hook.StacktraceConfiguration.IncludeErrorBreadcrumb = true
		log.Hooks.Add(hook)
	}

	return log
}
