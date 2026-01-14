package firecracker

import "fmt"

type Config struct {
	KernelImagePath string
	RootFSPath      string
	BinaryPath      string
	SocketPath      string
}

func (c Config) Validate() error {
	if c.KernelImagePath == "" || c.RootFSPath == "" {
		return fmt.Errorf("firecracker config: kernel and rootfs are required")
	}
	return nil
}
