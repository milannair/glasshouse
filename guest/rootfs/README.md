# Guest Rootfs

Dockerfile and build scripts for the Firecracker guest root filesystem.

## Building

With Docker:
```bash
./scripts/build-rootfs.sh
```

Without Docker:
```bash
./scripts/build-minimal-rootfs.sh
```

Output: `assets/rootfs.ext4`

## Contents

- Python 3.12 (Alpine-based)
- Busybox utilities
- Guest init binary at `/sbin/init`
- `/workspace` mount point for code execution

## Customization

Edit `guest/rootfs/Dockerfile` to add Python packages:

```dockerfile
RUN pip install --no-cache-dir numpy pandas
```
