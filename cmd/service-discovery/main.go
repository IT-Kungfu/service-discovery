package main

import (
	"context"
	"github.com/sirupsen/logrus"
	"os"
	"os/signal"
	"runtime"
	"secret-maintenance/cmd/service-discovery/config"
	"secret-maintenance/cmd/service-discovery/discovery"
	"secret-maintenance/internal/etcdconfig"
	"secret-maintenance/internal/logger"
	"syscall"
)

var (
	cfg = &config.Config{}
	log = logrus.New()
)

func init() {
	_, err := etcdconfig.GetConfig(cfg)
	if err != nil {
		log.Fatal(err)
	}

	log = logger.NewLogger(cfg.LogLevel, cfg.SentryDSN, "service-discovery")

	runtime.GOMAXPROCS(runtime.NumCPU())
}

func main() {
	ctx := context.WithValue(context.Background(), "services", map[string]interface{}{
		"cfg": cfg,
		"log": log,
	})

	d, err := discovery.New(ctx)
	if err != nil {
		log.Fatalf("Service discovery start error: %v", err)
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	d.Stop()
}
