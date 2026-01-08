package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"glasshouse/runner"
)

func main() {
	if len(os.Args) < 2 || os.Args[1] != "run" {
		usage()
		os.Exit(2)
	}

	cmdArgs, parseErr := parseRunArgs(os.Args[2:])
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

	result, err := runner.Run(context.Background(), cmdArgs)
	writeErr := writeReceipt(result.Receipt)
	if writeErr != nil {
		fmt.Fprintln(os.Stderr, "glasshouse:", writeErr)
	}

	if err != nil {
		exitCode := result.ExitCode
		if exitCode == 0 {
			exitCode = 1
		}
		os.Exit(exitCode)
	}
}

func parseRunArgs(args []string) ([]string, error) {
	if len(args) == 0 {
		return nil, nil
	}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--":
			return args[i+1:], nil
		default:
			if len(arg) > 0 && arg[0] == '-' {
				return nil, fmt.Errorf("unknown flag: %s", arg)
			}
			return args[i:], nil
		}
	}
	return nil, nil
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
	fmt.Fprintln(os.Stderr, "usage: glasshouse run -- <command> [args...]")
}
