package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"glasshouse/backend/firecracker"
	"glasshouse/core/execution"
)

var (
	port       = flag.Int("port", 8080, "HTTP port")
	kernelPath = flag.String("kernel", "", "Path to vmlinux.bin")
	rootfsPath = flag.String("rootfs", "", "Path to rootfs.ext4")
	receiptDir = flag.String("receipts", "/var/lib/glasshouse/receipts", "Receipt storage directory")
)

type Server struct {
	backend    *firecracker.Backend
	receiptDir string
	mu         sync.Mutex
	execCount  int
}

func main() {
	flag.Parse()

	if *kernelPath == "" || *rootfsPath == "" {
		// Try defaults
		if *kernelPath == "" {
			*kernelPath = "./assets/vmlinux.bin"
		}
		if *rootfsPath == "" {
			*rootfsPath = "./assets/rootfs.ext4"
		}
	}

	cfg := firecracker.Config{
		KernelImagePath: *kernelPath,
		RootFSPath:      *rootfsPath,
		BinaryPath:      "firecracker",
	}

	if err := cfg.Validate(); err != nil {
		log.Fatalf("Invalid config: %v", err)
	}

	// Ensure receipt directory exists
	if err := os.MkdirAll(*receiptDir, 0755); err != nil {
		log.Fatalf("Create receipt dir: %v", err)
	}

	srv := &Server{
		backend:    firecracker.New(cfg),
		receiptDir: *receiptDir,
	}

	http.HandleFunc("/health", srv.healthHandler)
	http.HandleFunc("/run", srv.runHandler)
	http.HandleFunc("/receipts/", srv.receiptHandler)

	httpSrv := &http.Server{
		Addr:         fmt.Sprintf(":%d", *port),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 120 * time.Second, // Long timeout for execution
	}

	// Graceful shutdown
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		log.Println("Shutting down...")
		httpSrv.Shutdown(context.Background())
	}()

	log.Printf("Starting glasshouse-server on :%d", *port)
	log.Printf("  Kernel: %s", *kernelPath)
	log.Printf("  Rootfs: %s", *rootfsPath)
	log.Printf("  Receipts: %s", *receiptDir)
	log.Println()
	log.Println("Test: curl -X POST localhost:8080/run -d '{\"code\": \"print(2+2)\"}'")

	if err := httpSrv.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatalf("Server error: %v", err)
	}
}

func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

type RunRequest struct {
	Code    string `json:"code"`
	Timeout int    `json:"timeout,omitempty"` // seconds, default 60
}

type RunResponse struct {
	Stdout     string `json:"stdout"`
	Stderr     string `json:"stderr"`
	ExitCode   int    `json:"exit_code"`
	DurationMs int64  `json:"duration_ms"`
	ReceiptID  string `json:"receipt_id"`
	Error      string `json:"error,omitempty"`
}

func (s *Server) runHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}

	var req RunRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	if req.Code == "" {
		http.Error(w, "code is required", http.StatusBadRequest)
		return
	}

	timeout := 60
	if req.Timeout > 0 {
		timeout = req.Timeout
	}

	// Generate receipt ID
	s.mu.Lock()
	s.execCount++
	receiptID := fmt.Sprintf("exec-%d-%d", time.Now().Unix(), s.execCount)
	s.mu.Unlock()

	// Create execution spec
	spec := execution.ExecutionSpec{
		Args: []string{"python3", "-c", req.Code},
	}

	// Prepare backend
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer cancel()

	if err := s.backend.Prepare(ctx); err != nil {
		writeError(w, "prepare failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Start execution
	handle, err := s.backend.Start(spec)
	if err != nil {
		writeError(w, "start failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer s.backend.Cleanup(handle)

	// Wait for completion
	log.Println("Waiting for VM to complete...")
	result, err := s.backend.Wait(handle)
	log.Printf("VM completed: exit=%d, err=%v", result.ExitCode, result.Err)

	// Build response
	resp := RunResponse{
		ExitCode:   result.ExitCode,
		DurationMs: result.CompletedAt.Sub(result.StartedAt).Milliseconds(),
		ReceiptID:  receiptID,
		Stdout:     result.Stdout,
		Stderr:     result.Stderr,
	}

	// Read stdout/stderr from guest result
	if result.Err != nil {
		resp.Error = result.Err.Error()
	}

	log.Printf("Response: stdout=%q, stderr=%q, exit=%d", resp.Stdout, resp.Stderr, resp.ExitCode)

	// Save receipt
	receipt := map[string]interface{}{
		"id":          receiptID,
		"timestamp":   time.Now().Format(time.RFC3339),
		"code_hash":   hashCode(req.Code),
		"exit_code":   result.ExitCode,
		"duration_ms": resp.DurationMs,
		"stdout":      resp.Stdout,
		"stderr":      resp.Stderr,
		"error":       resp.Error,
	}
	s.saveReceipt(receiptID, receipt)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (s *Server) receiptHandler(w http.ResponseWriter, r *http.Request) {
	id := filepath.Base(r.URL.Path)
	if id == "" || id == "receipts" {
		http.Error(w, "receipt ID required", http.StatusBadRequest)
		return
	}

	path := filepath.Join(s.receiptDir, id+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			http.Error(w, "receipt not found", http.StatusNotFound)
			return
		}
		http.Error(w, "read error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}

func (s *Server) saveReceipt(id string, receipt map[string]interface{}) {
	path := filepath.Join(s.receiptDir, id+".json")
	data, err := json.MarshalIndent(receipt, "", "  ")
	if err != nil {
		log.Printf("Marshal receipt: %v", err)
		return
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		log.Printf("Write receipt: %v", err)
	}
}

func writeError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

func hashCode(code string) string {
	// Simple hash for now
	h := 0
	for _, c := range code {
		h = 31*h + int(c)
	}
	return fmt.Sprintf("%x", uint32(h))
}
