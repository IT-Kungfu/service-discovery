### Service Discovery

Обнаружение сервисов и инициализация конфигурации в ETCD

```
labels:
    discovery.service.name: auth-grpc
    discovery.service.network: auth-network
    discovery.service.instance: deploy
    discovery.service.ports.grpc: 9001
    discovery.service.host.external: 192.168.0.33
```