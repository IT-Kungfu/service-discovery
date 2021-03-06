package config

type Config struct {
	InstanceName string `etcd:"/configs/service-discovery/{{SERVICE_DISCOVERY_INSTANCE}}/instance_name" default:"dev"`
	ETCDTimeout  int    `etcd:"/configs/service-discovery/{{SERVICE_DISCOVERY_INSTANCE}}/etcd_timeout" default:"10"`
	LogLevel     string `etcd:"/configs/service-discovery/{{SERVICE_DISCOVERY_INSTANCE}}/log_level,watcher" default:"debug"`
	SentryDSN    string `etcd:"/configs/service-discovery/{{SERVICE_DISCOVERY_INSTANCE}}/sentry_dsn,watcher" default:""`
}
