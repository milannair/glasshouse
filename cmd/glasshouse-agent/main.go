package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"glasshouse/audit"
	"glasshouse/core/agent"
	"glasshouse/core/profiling/ebpf"
)

func main() {
	if len(os.Args) < 2 || os.Args[1] != "start" {
		usage()
		os.Exit(2)
	}

	flags := flag.NewFlagSet("start", flag.ExitOnError)
	controlSocket := flags.String("control-socket", "/tmp/glasshouse-agent.sock", "Unix socket path for control commands")
	receiptDir := flags.String("receipt-dir", "", "Directory for emitted receipts (default stdout)")
	bpfDir := flags.String("bpf-dir", "", "Directory containing eBPF objects")
	if err := flags.Parse(os.Args[2:]); err != nil {
		fmt.Fprintln(os.Stderr, "glasshouse-agent:", err)
		os.Exit(2)
	}

	cfg := agent.Config{
		ReceiptDir:    *receiptDir,
		Observation:   "host",
		ControlSocket: *controlSocket,
	}

	profiler := ebpf.NewController(ebpfConfigFromEnv(*bpfDir))
	agent := agent.New(cfg, profiler)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := agent.Run(ctx); err != nil && err != context.Canceled {
		fmt.Fprintln(os.Stderr, "glasshouse-agent:", err)
		os.Exit(1)
	}
}

func ebpfConfigFromEnv(dir string) audit.Config {
	cfg := audit.Config{}
	if dir != "" {
		cfg.BPFObjectDir = dir
		return cfg
	}
	if env := os.Getenv("GLASSHOUSE_BPF_DIR"); env != "" {
		cfg.BPFObjectDir = env
	}
	return cfg
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: glasshouse-agent start [--control-socket path] [--receipt-dir dir] [--bpf-dir dir]")
}
