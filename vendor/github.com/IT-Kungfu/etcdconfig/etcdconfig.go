package etcdconfig

import (
	"context"
	"fmt"
	"go.etcd.io/etcd/clientv3"
	"log"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"
)

const (
	DefaultETCDAddr              = "localhost:2379"
	TagDefault                   = "default"
	TagETCD                      = "etcd"
	TagConnection                = "connection"
	TagETCDWatcher               = "watcher"
	EnvInstanceSuffix            = "_INSTANCE"
	EnvExternalSuffix            = "_EXTERNAL"
	ETCDExternalSuffix           = "/external"
	ETCDHostPattern              = "/services/%s/%s/host"
	ETCDExternalHostPattern      = "/services/%s/%s/host/external"
	ETCDPortsGrpcPattern         = "/services/%s/%s/ports/grpc"
	ETCDExternalPortsGrpcPattern = "/services/%s/%s/ports/grpc/external"
)

type ETCDObserver interface {
	ETCDValueChanged(key string, value []byte, cfg interface{})
}

type ETCDConfig struct {
	config    interface{}
	observers []ETCDObserver
}

func GetConfig(c interface{}) (*ETCDConfig, error) {
	cfg := &ETCDConfig{
		config: c,
	}

	etcdAddr := os.Getenv("ETCD_ADDR")
	if len(etcdAddr) == 0 {
		etcdAddr = DefaultETCDAddr
	}

	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)

	etcdConfig := clientv3.Config{
		Endpoints:   strings.Split(etcdAddr, ","),
		DialTimeout: 10 * time.Second,
	}

	if os.Getenv("ETCD_USERNAME") != "" && os.Getenv("ETCD_PASSWORD") != "" {
		etcdConfig.Username = os.Getenv("ETCD_USERNAME")
		etcdConfig.Password = os.Getenv("ETCD_PASSWORD")
	}

	cli, err := clientv3.New(etcdConfig)
	if err != nil {
		return nil, err
	}

	ref := reflect.Indirect(reflect.ValueOf(c))
	for i := 0; i < ref.Type().NumField(); i++ {
		f := ref.Type().Field(i)

		keyName, isWatch := cfg.parseEtcdTag(ref.Type().Field(i).Tag.Get(TagETCD), TagETCDWatcher)
		if keyName == "" {
			continue
		}

		keyName = cfg.prepareKey(keyName, f.Tag.Get(TagConnection) == "true")

		v, err := cli.Get(ctx, keyName)
		if err != nil {
			return nil, err
		}

		value := ""
		if len(v.Kvs) > 0 {
			value = string(v.Kvs[0].Value)
		} else {
			value = f.Tag.Get(TagDefault)
		}

		if _, ok := f.Tag.Lookup(TagDefault); !ok && len(value) == 0 {
			return nil, fmt.Errorf("required configuration parameter is not specified - %s", keyName)
		}

		configField := ref.Field(i)
		configKind := ref.Type().Field(i).Type.Kind()
		switch configKind {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			iVal, _ := strconv.Atoi(value)
			configField.SetInt(int64(iVal))
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			iVal, _ := strconv.ParseUint(value, 10, 64)
			configField.SetUint(iVal)
		case reflect.String:
			configField.SetString(value)
		case reflect.Bool:
			configField.SetBool(value == "true")
		}

		if isWatch {
			cfg.addWatcher(cli, keyName, configField, configKind)
		}
	}

	return cfg, nil
}

func (cfg *ETCDConfig) prepareKey(key string, isConnection bool) string {
	for {
		in := strings.Index(key, "{{")
		if in == -1 {
			break
		}
		out := strings.Index(key, "}}")
		if out == -1 {
			break
		}

		env := key[in+2 : out]
		value := os.Getenv(env)
		key = strings.Replace(key, key[in:out+2], value, -1)

		if isConnection && strings.HasSuffix(env, EnvInstanceSuffix) {
			serviceName := strings.ReplaceAll(env, EnvInstanceSuffix, "")
			if os.Getenv(serviceName+EnvExternalSuffix) == "true" {
				key += ETCDExternalSuffix
			}
		}
	}
	return key
}

func (cfg *ETCDConfig) addWatcher(etcd *clientv3.Client, keyName string, configField reflect.Value, configKind reflect.Kind) {
	log.Printf("Add watcher: %s", keyName)
	go func() {
		rch := etcd.Watch(context.Background(), keyName)
		for wresp := range rch {
			for _, ev := range wresp.Events {
				log.Printf("Watched config value changed: %s %q", ev.Kv.Key, ev.Kv.Value)
				switch configKind {
				case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
					iVal, _ := strconv.Atoi(string(ev.Kv.Value))
					configField.SetInt(int64(iVal))
				case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
					iVal, _ := strconv.ParseUint(string(ev.Kv.Value), 10, 64)
					configField.SetUint(iVal)
				case reflect.String:
					configField.SetString(string(ev.Kv.Value))
				case reflect.Bool:
					configField.SetBool(string(ev.Kv.Value) == "true")
				}
				cfg.notifyObservers(string(ev.Kv.Key), ev)
			}
		}
	}()
}

func (cfg *ETCDConfig) parseEtcdTag(tag, etcdTag string) (string, bool) {
	params := strings.Split(tag, ",")
	if len(params) == 2 {
		return params[0], params[1] == etcdTag
	}
	return tag, false
}

func (cfg *ETCDConfig) notifyObservers(key string, event *clientv3.Event) {
	if cfg.observers == nil {
		return
	}
	for _, o := range cfg.observers {
		o.ETCDValueChanged(key, event.Kv.Value, cfg.config)
	}
}

func (cfg *ETCDConfig) AddObserver(o ETCDObserver) {
	if cfg.observers == nil {
		cfg.observers = make([]ETCDObserver, 0, 10)
	}
	cfg.observers = append(cfg.observers, o)
}
