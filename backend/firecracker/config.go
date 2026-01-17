package firecracker

import "fmt"

type Config struct {
	KernelImagePath string // Path to vmlinux.bin
	RootFSPath      string // Path to rootfs.ext4
	BinaryPath      string // Path to firecracker binary (default: "firecracker")
	SocketDir       string // Directory for API sockets (default: temp)
	TimeoutSeconds  int    // Execution timeout (default: 60)
}

func (c Config) Validate() error {
	if c.KernelImagePath == "" || c.RootFSPath == "" {
		return fmt.Errorf("firecracker config: kernel and rootfs are required")
	}
	return nil
}
