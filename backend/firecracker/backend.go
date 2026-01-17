package firecracker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"

	"glasshouse/core/execution"
	"glasshouse/core/profiling"
	"glasshouse/core/receipt"
)

// Backend runs workloads in Firecracker microVMs.
type Backend struct {
	cfg Config
}

// New creates a Firecracker backend.
func New(cfg Config) *Backend {
	return &Backend{cfg: cfg}
}

func (b *Backend) Name() string { return "firecracker" }

func (b *Backend) Prepare(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return b.cfg.Validate()
}

// vmHandle holds runtime state for a single VM execution.
type vmHandle struct {
	socketPath    string
	workspacePath string
	fcProcess     *exec.Cmd
	startTime     time.Time
}

func (b *Backend) Start(spec execution.ExecutionSpec) (execution.ExecutionHandle, error) {
	// Create unique workspace for this execution
	workDir, err := os.MkdirTemp("", "glasshouse-workspace-")
	if err != nil {
		return execution.ExecutionHandle{}, fmt.Errorf("create workspace: %w", err)
	}

	// Create pending directory and write code
	pendingDir := filepath.Join(workDir, ".pending")
	if err := os.MkdirAll(pendingDir, 0755); err != nil {
		return execution.ExecutionHandle{}, fmt.Errorf("create pending dir: %w", err)
	}

	// Extract code from spec - first arg after python3 -c
	code := extractCode(spec.Args)
	if err := os.WriteFile(filepath.Join(pendingDir, "code.py"), []byte(code), 0644); err != nil {
		return execution.ExecutionHandle{}, fmt.Errorf("write code: %w", err)
	}

	// Create workspace ext4 image
	workspaceImg := filepath.Join(workDir, "workspace.ext4")
	if err := createWorkspaceImage(workspaceImg, workDir); err != nil {
		return execution.ExecutionHandle{}, fmt.Errorf("create workspace image: %w", err)
	}

	// Create unique socket path
	socketPath := filepath.Join(workDir, "firecracker.sock")

	// Start Firecracker process
	fcBinary := b.cfg.BinaryPath
	if fcBinary == "" {
		fcBinary = "firecracker"
	}

	fcCmd := exec.Command(fcBinary, "--api-sock", socketPath)
	fcCmd.Stdout = os.Stdout
	fcCmd.Stderr = os.Stderr

	if err := fcCmd.Start(); err != nil {
		return execution.ExecutionHandle{}, fmt.Errorf("start firecracker: %w", err)
	}

	// Wait for socket to be ready
	if err := waitForSocket(socketPath, 5*time.Second); err != nil {
		fcCmd.Process.Kill()
		return execution.ExecutionHandle{}, fmt.Errorf("wait for socket: %w", err)
	}

	// Configure VM via API
	client := newUnixClient(socketPath)

	// Machine config
	if err := apiPut(client, socketPath, "/machine-config", map[string]interface{}{
		"vcpu_count":   1,
		"mem_size_mib": 256,
		"smt":          false,
	}); err != nil {
		fcCmd.Process.Kill()
		return execution.ExecutionHandle{}, fmt.Errorf("set machine config: %w", err)
	}

	// Boot source
	if err := apiPut(client, socketPath, "/boot-source", map[string]interface{}{
		"kernel_image_path": b.cfg.KernelImagePath,
		"boot_args":         "console=ttyS0 reboot=k panic=1 pci=off root=/dev/vda rw init=/sbin/init",
	}); err != nil {
		fcCmd.Process.Kill()
		return execution.ExecutionHandle{}, fmt.Errorf("set boot source: %w", err)
	}

	// Root drive (rootfs)
	if err := apiPut(client, socketPath, "/drives/rootfs", map[string]interface{}{
		"drive_id":       "rootfs",
		"path_on_host":   b.cfg.RootFSPath,
		"is_root_device": true,
		"is_read_only":   true,
	}); err != nil {
		fcCmd.Process.Kill()
		return execution.ExecutionHandle{}, fmt.Errorf("set rootfs drive: %w", err)
	}

	// Workspace drive
	if err := apiPut(client, socketPath, "/drives/workspace", map[string]interface{}{
		"drive_id":       "workspace",
		"path_on_host":   workspaceImg,
		"is_root_device": false,
		"is_read_only":   false,
	}); err != nil {
		fcCmd.Process.Kill()
		return execution.ExecutionHandle{}, fmt.Errorf("set workspace drive: %w", err)
	}

	// Start VM
	if err := apiPut(client, socketPath, "/actions", map[string]interface{}{
		"action_type": "InstanceStart",
	}); err != nil {
		fcCmd.Process.Kill()
		return execution.ExecutionHandle{}, fmt.Errorf("start instance: %w", err)
	}

	handle := &vmHandle{
		socketPath:    socketPath,
		workspacePath: workDir,
		fcProcess:     fcCmd,
		startTime:     time.Now(),
	}

	return execution.ExecutionHandle{
		ID:            fmt.Sprintf("fc-%d", fcCmd.Process.Pid),
		BackendHandle: handle,
	}, nil
}

func (b *Backend) Wait(h execution.ExecutionHandle) (execution.ExecutionResult, error) {
	vm, ok := h.BackendHandle.(*vmHandle)
	if !ok {
		return execution.ExecutionResult{Handle: h, ExitCode: 1, Err: fmt.Errorf("invalid handle")}, nil
	}

	// Wait for Firecracker process to exit (guest powers off)
	err := vm.fcProcess.Wait()
	completedAt := time.Now()

	result := execution.ExecutionResult{
		Handle:      h,
		StartedAt:   vm.startTime,
		CompletedAt: completedAt,
	}

	// Read result from workspace
	resultPath := filepath.Join(vm.workspacePath, "workspace.ext4")
	guestResult, readErr := readResultFromImage(resultPath)
	if readErr != nil {
		result.ExitCode = 1
		result.Err = fmt.Errorf("read result: %w (process err: %v)", readErr, err)
		return result, nil
	}

	result.ExitCode = guestResult.ExitCode
	if guestResult.Error != "" {
		result.Err = fmt.Errorf("guest error: %s", guestResult.Error)
	}

	return result, nil
}

func (b *Backend) Kill(h execution.ExecutionHandle) error {
	vm, ok := h.BackendHandle.(*vmHandle)
	if !ok {
		return fmt.Errorf("invalid handle")
	}
	if vm.fcProcess.Process != nil {
		return vm.fcProcess.Process.Kill()
	}
	return nil
}

func (b *Backend) Cleanup(h execution.ExecutionHandle) error {
	vm, ok := h.BackendHandle.(*vmHandle)
	if !ok {
		return nil
	}
	// Clean up workspace directory
	if vm.workspacePath != "" {
		os.RemoveAll(vm.workspacePath)
	}
	return nil
}

func (b *Backend) ProfilingInfo(h execution.ExecutionHandle) execution.BackendProfilingInfo {
	return execution.BackendProfilingInfo{
		Identity: execution.ExecutionIdentity{
			RootPID:    0,
			CgroupPath: "",
			Namespaces: map[string]string{},
		},
		SupportedModes: []profiling.Mode{
			profiling.ProfilingDisabled,
		},
		SupportsProfile: false,
	}
}

func (b *Backend) Metadata() receipt.ExecutionInfo {
	return receipt.ExecutionInfo{Backend: b.Name(), Isolation: "vm"}
}

// GuestResult matches the JSON written by guest init
type GuestResult struct {
	Stdout     string `json:"stdout"`
	Stderr     string `json:"stderr"`
	ExitCode   int    `json:"exit_code"`
	DurationMs int64  `json:"duration_ms"`
	Error      string `json:"error,omitempty"`
}

// Helper functions

func extractCode(args []string) string {
	// Look for python3 -c <code> pattern
	for i, arg := range args {
		if arg == "-c" && i+1 < len(args) {
			return args[i+1]
		}
	}
	// Fallback: join all args as code
	if len(args) > 0 {
		return args[len(args)-1]
	}
	return ""
}

func createWorkspaceImage(path string, sourceDir string) error {
	// Create 64MB ext4 image
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	if err := f.Truncate(64 * 1024 * 1024); err != nil {
		f.Close()
		return err
	}
	f.Close()

	// Format as ext4
	if out, err := exec.Command("mkfs.ext4", "-F", path).CombinedOutput(); err != nil {
		return fmt.Errorf("mkfs.ext4: %w: %s", err, out)
	}

	// Mount and copy pending directory
	mnt, err := os.MkdirTemp("", "glasshouse-mnt-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(mnt)

	if out, err := exec.Command("sudo", "mount", path, mnt).CombinedOutput(); err != nil {
		return fmt.Errorf("mount: %w: %s", err, out)
	}
	defer exec.Command("sudo", "umount", mnt).Run()

	// Copy .pending directory
	pendingSrc := filepath.Join(sourceDir, ".pending")
	pendingDst := filepath.Join(mnt, ".pending")
	if out, err := exec.Command("sudo", "cp", "-r", pendingSrc, pendingDst).CombinedOutput(); err != nil {
		return fmt.Errorf("copy pending: %w: %s", err, out)
	}

	return nil
}

func readResultFromImage(imagePath string) (*GuestResult, error) {
	mnt, err := os.MkdirTemp("", "glasshouse-mnt-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(mnt)

	if out, err := exec.Command("sudo", "mount", imagePath, mnt).CombinedOutput(); err != nil {
		return nil, fmt.Errorf("mount: %w: %s", err, out)
	}
	defer exec.Command("sudo", "umount", mnt).Run()

	resultPath := filepath.Join(mnt, ".pending", "result.json")
	data, err := os.ReadFile(resultPath)
	if err != nil {
		return nil, fmt.Errorf("read result.json: %w", err)
	}

	var result GuestResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("parse result: %w", err)
	}

	return &result, nil
}

func waitForSocket(path string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(path); err == nil {
			// Try to connect
			conn, err := net.Dial("unix", path)
			if err == nil {
				conn.Close()
				return nil
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
	return fmt.Errorf("timeout waiting for socket")
}

func newUnixClient(socketPath string) *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				return net.Dial("unix", socketPath)
			},
		},
	}
}

func apiPut(client *http.Client, socketPath, path string, body interface{}) error {
	data, err := json.Marshal(body)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("PUT", "http://localhost"+path, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error %d: %s", resp.StatusCode, body)
	}

	return nil
}

var _ execution.ExecutionBackend = (*Backend)(nil)
var _ execution.MetadataProvider = (*Backend)(nil)

// Ensure syscall is used (for shutdown detection)
var _ = syscall.SIGCHLD
