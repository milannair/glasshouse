package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"glasshouse/audit"
	"glasshouse/backend/firecracker"
	"glasshouse/backend/process"
	"glasshouse/core/profiling"
	"glasshouse/core/profiling/ebpf"
	"glasshouse/core/profiling/noop"
	"glasshouse/node/config"
	"glasshouse/node/manager"
	"glasshouse/node/registry"
)

func main() {
	var (
		backendFlag    string
		profileFlag    string
		concurrency    int
		firecrackerCfg string
	)
	flag.StringVar(&backendFlag, "backend", "process", "backend name (process|firecracker)")
	flag.StringVar(&profileFlag, "profile", "disabled", "profiling mode: disabled|host|guest|combined")
	flag.IntVar(&concurrency, "concurrency", 1, "max concurrent executions")
	flag.StringVar(&firecrackerCfg, "firecracker-cfg", "", "path to firecracker config (unused placeholder)")
	flag.Parse()

	cmdArgs := flag.Args()
	if len(cmdArgs) == 0 {
		usage()
		os.Exit(2)
	}

	profileMode, err := parseProfilingMode(profileFlag)
	if err != nil {
		fmt.Fprintln(os.Stderr, "node-agent:", err)
		os.Exit(2)
	}

	cfg := config.Defaults()
	cfg.DefaultBackend = backendFlag
	cfg.ProfilingMode = profileMode
	cfg.Concurrency = concurrency

	reg := registry.New()
	reg.Register("process", process.New(process.Options{}))
	reg.Register("firecracker", firecracker.New(firecracker.Config{}))
	_ = firecrackerCfg // placeholder for future loading

	mgr, err := manager.New(cfg, reg)
	if err != nil {
		fmt.Fprintln(os.Stderr, "node-agent:", err)
		os.Exit(1)
	}
	mgr.Profiler = selectProfiler(profileMode)

	resp, runErr := mgr.Run(context.Background(), manager.Request{
		Args:      cmdArgs,
		Workdir:   mustGetwd(),
		Env:       os.Environ(),
		Backend:   backendFlag,
		Profiling: profileMode,
	})

	if resp.Result.Receipt != nil {
		_ = writeReceipt(resp.Result.Receipt)
	}

	exitCode := resp.Result.ExitCode
	if exitCode == 0 && runErr != nil {
		exitCode = 1
	}
	if runErr != nil {
		fmt.Fprintln(os.Stderr, "node-agent:", runErr)
	}
	if exitCode != 0 {
		os.Exit(exitCode)
	}
}

func parseProfilingMode(value string) (profiling.Mode, error) {
	switch strings.ToLower(value) {
	case "disabled", "":
		return profiling.ProfilingDisabled, nil
	case "host":
		return profiling.ProfilingHost, nil
	case "guest":
		return profiling.ProfilingGuest, nil
	case "combined":
		return profiling.ProfilingCombined, nil
	default:
		return profiling.ProfilingDisabled, fmt.Errorf("unknown profile mode: %s", value)
	}
}

func writeReceipt(receipt interface{}) error {
	data, err := json.MarshalIndent(receipt, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile("receipt.json", data, 0644)
}

func mustGetwd() string {
	wd, err := os.Getwd()
	if err != nil {
		return ""
	}
	return wd
}

func selectProfiler(mode profiling.Mode) profiling.Controller {
	if mode == profiling.ProfilingDisabled {
		return noop.NewController()
	}
	return ebpf.NewController(auditConfigFromEnv())
}

func auditConfigFromEnv() audit.Config {
	cfg := audit.Config{}
	if dir := os.Getenv("GLASSHOUSE_BPF_DIR"); dir != "" {
		cfg.BPFObjectDir = dir
	}
	return cfg
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: node-agent [--backend name] [--profile disabled|host|guest|combined] [--concurrency N] -- <command> [args]")
}
