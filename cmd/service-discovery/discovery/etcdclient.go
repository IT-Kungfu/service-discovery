package discovery

import (
	"context"
	"fmt"
	"go.etcd.io/etcd/clientv3"
	"os"
	"strings"
	"time"
)

func (d *Discovery) initEtcdClient() error {
	etcdAddr := os.Getenv("ETCD_ADDR")

	if len(etcdAddr) == 0 {
		return fmt.Errorf("etcd server address is not specified")
	}

	etcdConfig := clientv3.Config{
		Endpoints:   strings.Split(etcdAddr, ","),
		DialTimeout: 10 * time.Second,
	}

	if os.Getenv("ETCD_USERNAME") != "" && os.Getenv("ETCD_PASSWORD") != "" {
		etcdConfig.Username = os.Getenv("ETCD_USERNAME")
		etcdConfig.Password = os.Getenv("ETCD_PASSWORD")
	}

	var err error
	d.etcdClient, err = clientv3.New(etcdConfig)
	return err
}

func (d *Discovery) etcdPut(key, value string) error {
	ctx, _ := context.WithTimeout(context.Background(), time.Duration(d.cfg.ETCDTimeout)*time.Second)
	_, err := d.etcdClient.Put(ctx, key, value)
	return err
}

func (d *Discovery) etcdDelete(key string) error {
	ctx, _ := context.WithTimeout(context.Background(), time.Duration(d.cfg.ETCDTimeout)*time.Second)
	_, err := d.etcdClient.Delete(ctx, key)
	return err
}
