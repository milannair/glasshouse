package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"glasshouse/audit"
	"glasshouse/backend/process"
	"glasshouse/core/execution"
	"glasshouse/core/profiling"
	"glasshouse/core/profiling/ebpf"
	"glasshouse/core/profiling/noop"
)

func main() {
	if len(os.Args) < 2 || os.Args[1] != "run" {
		usage()
		os.Exit(2)
	}

	opts, cmdArgs, parseErr := parseRunArgs(os.Args[2:])
	if parseErr != nil {
		fmt.Fprintln(os.Stderr, "glasshouse:", parseErr)
		usage()
		os.Exit(2)
	}
	if len(cmdArgs) == 0 {
		fmt.Fprintln(os.Stderr, "glasshouse: no command provided")
		usage()
		os.Exit(2)
	}

	spec := execution.ExecutionSpec{
		Args:      cmdArgs,
		Workdir:   mustGetwd(),
		Env:       os.Environ(),
		Guest:     opts.Guest,
		Profiling: opts.Profiling,
	}

	engine := execution.Engine{
		Backend:  process.New(process.Options{Guest: opts.Guest}),
		Profiler: selectProfiler(opts.Profiling),
	}

	result, err := engine.Run(context.Background(), spec)
	if result.Receipt != nil {
		writeErr := writeReceipt(result.Receipt)
		if writeErr != nil {
			fmt.Fprintln(os.Stderr, "glasshouse:", writeErr)
		}
	}

	if err != nil {
		exitCode := result.ExitCode
		if exitCode == 0 {
			exitCode = 1
		}
		os.Exit(exitCode)
	}
}

type runOptions struct {
	Guest     bool
	Profiling profiling.Mode
}

func parseRunArgs(args []string) (runOptions, []string, error) {
	opts := runOptions{Profiling: profiling.ProfilingDisabled}
	if len(args) == 0 {
		return opts, nil, nil
	}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--":
			return opts, args[i+1:], nil
		case "--guest":
			opts.Guest = true
		case "--profile":
			if i+1 >= len(args) {
				return opts, nil, fmt.Errorf("missing profile mode")
			}
			i++
			if err := setProfilingMode(&opts, args[i]); err != nil {
				return opts, nil, err
			}
		default:
			if strings.HasPrefix(arg, "--profile=") {
				if err := setProfilingMode(&opts, strings.TrimPrefix(arg, "--profile=")); err != nil {
					return opts, nil, err
				}
				continue
			}
			if len(arg) > 0 && arg[0] == '-' {
				return opts, nil, fmt.Errorf("unknown flag: %s", arg)
			}
			return opts, args[i:], nil
		}
	}
	return opts, nil, nil
}

func setProfilingMode(opts *runOptions, mode string) error {
	switch mode {
	case "disabled":
		opts.Profiling = profiling.ProfilingDisabled
	case "host":
		opts.Profiling = profiling.ProfilingHost
	case "guest":
		opts.Profiling = profiling.ProfilingGuest
	case "combined":
		opts.Profiling = profiling.ProfilingCombined
	default:
		return fmt.Errorf("unknown profile mode: %s", mode)
	}
	return nil
}

func selectProfiler(mode profiling.Mode) profiling.Controller {
	if mode == profiling.ProfilingDisabled {
		return noop.NewController()
	}
	return ebpf.NewController(ebpfConfigFromEnv())
}

func ebpfConfigFromEnv() audit.Config {
	cfg := audit.Config{}
	if dir := os.Getenv("GLASSHOUSE_BPF_DIR"); dir != "" {
		cfg.BPFObjectDir = dir
	}
	return cfg
}

func mustGetwd() string {
	wd, err := os.Getwd()
	if err != nil {
		return ""
	}
	return wd
}

func writeReceipt(receipt interface{}) error {
	data, err := json.MarshalIndent(receipt, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal receipt: %w", err)
	}
	if err := os.WriteFile("receipt.json", data, 0644); err != nil {
		return fmt.Errorf("write receipt.json: %w", err)
	}
	return nil
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: glasshouse run [--guest] [--profile disabled|host|guest|combined] -- <command> [args...]")
}
