package main

import (
	"context"
	"github.com/IT-Kungfu/etcdconfig"
	"github.com/IT-Kungfu/logger"
	"github.com/IT-Kungfu/service-discovery/cmd/service-discovery/config"
	"github.com/IT-Kungfu/service-discovery/cmd/service-discovery/discovery"
	"os"
	"os/signal"
	"runtime"
	"syscall"
)

var (
	etcd = &etcdconfig.ETCDConfig{}
	cfg  = &config.Config{}
	log  *logger.Logger
)

func init() {
	var err error
	etcd, err = etcdconfig.GetConfig(cfg)
	if err != nil {
		panic(err)
	}

	if log, err = logger.New(&logger.Config{
		LogLevel:     cfg.LogLevel,
		SentryDSN:    "",
		LogstashAddr: "",
		ServiceName:  "service-discovery",
		InstanceName: "dev",
	}); err != nil {
		panic(err)
	}
	etcd.AddObserver(log)

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
