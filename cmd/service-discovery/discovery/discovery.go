package discovery

import (
	"context"
	"fmt"
	"github.com/IT-Kungfu/logger"
	"github.com/IT-Kungfu/service-discovery/cmd/service-discovery/config"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/client"
	"go.etcd.io/etcd/clientv3"
	"strings"
)

const (
	LabelServiceName             = "discovery.service.name"
	LabelServiceNetwork          = "discovery.service.network"
	LabelServiceInstance         = "discovery.service.instance"
	LabelServicePortsGrpc        = "discovery.service.ports.grpc"
	LabelServiceHostExternal     = "discovery.service.host.external"
	ETCDHostPattern              = "/services/%s/%s/host"
	ETCDExternalHostPattern      = "/services/%s/%s/host/external"
	ETCDPortsGrpcPattern         = "/services/%s/%s/ports/grpc"
	ETCDExternalPortsGrpcPattern = "/services/%s/%s/ports/grpc/external"
)

type Discovery struct {
	cfg          *config.Config
	log          *logger.Logger
	dockerClient *client.Client
	etcdClient   *clientv3.Client
	ctx          context.Context
	ctxCancel    context.CancelFunc
}

func New(ctx context.Context) (*Discovery, error) {
	services := ctx.Value("services").(map[string]interface{})
	d := &Discovery{
		cfg: services["cfg"].(*config.Config),
		log: services["log"].(*logger.Logger),
	}

	d.ctx, d.ctxCancel = context.WithCancel(context.Background())

	if err := d.initEtcdClient(); err != nil {
		return nil, err
	}

	go d.start()

	return d, nil
}

func (d *Discovery) start() {
	d.log.Info("Service discovery started")

	var err error
	d.dockerClient, err = client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		panic(err)
	}

	msgCh, errCh := d.dockerClient.Events(d.ctx, types.EventsOptions{})

	var isBreak bool
	for !isBreak {
		select {
		case <-d.ctx.Done():
			isBreak = true
		case err := <-errCh:
			if err != nil {
				d.log.Errorf("Event error: %v", err)
			}
		case msg := <-msgCh:
			if msg.Status != "" {
				if msg.Status == "start" || msg.Status == "unpause" {
					d.serviceStart(msg)
				} else if msg.Status == "die" || msg.Status == "pause" {
					d.serviceStop(msg)
				}
			}
		}
	}
}

func (d *Discovery) serviceStart(msg events.Message) {
	inspect, err := d.dockerClient.ContainerInspect(d.ctx, msg.ID)
	if err != nil {
		d.log.Errorf("Inspect error: %v", err)
		return
	}

	if _, ok := inspect.Config.Labels[LabelServiceName]; !ok {
		return
	}

	if _, ok := inspect.Config.Labels[LabelServiceInstance]; !ok {
		return
	}

	d.log.Infof("%s started", inspect.Config.Labels[LabelServiceName])

	if _, ok := inspect.Config.Labels[LabelServiceNetwork]; !ok {
		d.log.Errorf("No network defined")
		return
	}

	if _, ok := inspect.NetworkSettings.Networks[inspect.Config.Labels[LabelServiceNetwork]]; !ok {
		d.log.Errorf("Network %s not found", inspect.Config.Labels[LabelServiceNetwork])
		return
	}

	serviceName := inspect.Config.Labels[LabelServiceName]
	serviceInstance := inspect.Config.Labels[LabelServiceInstance]
	containerIP := inspect.NetworkSettings.Networks[inspect.Config.Labels[LabelServiceNetwork]].IPAddress
	containerPorts := inspect.NetworkSettings.Ports

	etcdKv := make(map[string]string, 4)
	etcdKv[fmt.Sprintf(ETCDHostPattern, serviceName, serviceInstance)] = containerIP
	if inspect.Config.Labels[LabelServiceHostExternal] != "" {
		etcdKv[fmt.Sprintf(ETCDExternalHostPattern, serviceName, serviceInstance)] = inspect.Config.Labels[LabelServiceHostExternal]
	}

	if _, ok := inspect.Config.Labels[LabelServicePortsGrpc]; ok {
		for k, v := range containerPorts {
			if k.Port() == inspect.Config.Labels[LabelServicePortsGrpc] {
				ports := make([]string, 0, len(v))
				for _, p := range v {
					if p.HostPort != "" {
						ports = append(ports, p.HostPort)
					}
				}
				etcdKv[fmt.Sprintf(ETCDExternalPortsGrpcPattern, serviceName, serviceInstance)] = strings.Join(ports, ",")
			}
			etcdKv[fmt.Sprintf(ETCDPortsGrpcPattern, serviceName, serviceInstance)] = k.Port()
		}
	}

	for k, v := range etcdKv {
		if err := d.etcdPut(k, v); err != nil {
			d.log.Errorf("Error writing to ETCD: %v", err)
		}
	}
}

func (d *Discovery) serviceStop(msg events.Message) {
	inspect, err := d.dockerClient.ContainerInspect(d.ctx, msg.ID)
	if err != nil {
		d.log.Errorf("Inspect error: %v", err)
		return
	}

	if _, ok := inspect.Config.Labels[LabelServiceName]; !ok {
		return
	}

	if _, ok := inspect.Config.Labels[LabelServiceInstance]; !ok {
		return
	}

	d.log.Infof("%s stoped", inspect.Config.Labels[LabelServiceName])

	serviceName := inspect.Config.Labels[LabelServiceName]
	serviceInstance := inspect.Config.Labels[LabelServiceInstance]
	keys := []string{
		fmt.Sprintf(ETCDHostPattern, serviceName, serviceInstance),
		fmt.Sprintf(ETCDExternalHostPattern, serviceName, serviceInstance),
		fmt.Sprintf(ETCDPortsGrpcPattern, serviceName, serviceInstance),
		fmt.Sprintf(ETCDExternalPortsGrpcPattern, serviceName, serviceInstance),
	}

	for _, k := range keys {
		if err := d.etcdDelete(k); err != nil {
			d.log.Errorf("Error deleting from ETCD: %v", err)
		}
	}
}

func (d *Discovery) Stop() {
	d.ctxCancel()
}
