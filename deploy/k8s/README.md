# Kubernetes Deployment

## Architecture

Run `glasshouse-server` as a Deployment with:
- Nodes that have KVM access (nested virt or bare metal)
- Kernel and rootfs as ConfigMaps or PVs
- Receipt storage via PV or log streaming

## Example Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: glasshouse-server
spec:
  replicas: 1
  selector:
    matchLabels:
      app: glasshouse
  template:
    metadata:
      labels:
        app: glasshouse
    spec:
      containers:
      - name: server
        image: glasshouse-server:latest
        ports:
        - containerPort: 8080
        volumeMounts:
        - name: kvm
          mountPath: /dev/kvm
        - name: assets
          mountPath: /assets
        - name: receipts
          mountPath: /var/lib/glasshouse/receipts
        securityContext:
          privileged: true  # Required for Firecracker
      volumes:
      - name: kvm
        hostPath:
          path: /dev/kvm
      - name: assets
        configMap:
          name: glasshouse-assets
      - name: receipts
        persistentVolumeClaim:
          claimName: glasshouse-receipts
```

## Requirements

- Nodes with `/dev/kvm` access
- `privileged: true` for Firecracker (or specific capabilities)
- Kernel and rootfs distributed via ConfigMap or init container

## Notes

- Each pod can handle multiple executions (VMs are ephemeral)
- Consider HPA based on execution queue depth
- Stream receipts to a log aggregator for analysis
