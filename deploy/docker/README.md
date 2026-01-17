# Docker Deployment

## Building the Server Image

```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o glasshouse-server ./cmd/glasshouse-server

FROM alpine:latest
RUN apk add --no-cache ca-certificates
COPY --from=builder /app/glasshouse-server /usr/local/bin/
COPY --from=builder /app/assets /assets
ENTRYPOINT ["glasshouse-server"]
```

## Running

The server requires:
- `/dev/kvm` access (nested virtualization)
- Kernel and rootfs in `/assets/`

```bash
docker run --rm \
  --device /dev/kvm \
  -v /path/to/kernel:/assets/vmlinux.bin \
  -v /path/to/rootfs:/assets/rootfs.ext4 \
  -v /var/lib/glasshouse/receipts:/var/lib/glasshouse/receipts \
  -p 8080:8080 \
  glasshouse-server
```

## CLI Mode (No Firecracker)

For sandbox-only execution without Firecracker:

```bash
docker run --rm -v $(pwd):/work glasshouse ./glasshouse run --profile disabled -- ls
```

## Notes

- VM backends require KVM access and nested virtualization
- Receipts should be persisted via volume mount
