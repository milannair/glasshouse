package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"runtime"
	"time"

	"glasshouse/guest/transport"
)

func main() {
	mode := flag.String("mode", "heartbeat", "probe mode: heartbeat or info")
	flag.Parse()

	ctx := context.Background()
	tx := transport.NewLoopback(os.Stdout)
	defer tx.Close()

	ev := transport.Event{
		Type: *mode,
		Payload: map[string]interface{}{
			"timestamp": time.Now().UTC().Format(time.RFC3339Nano),
			"os":        runtime.GOOS,
			"arch":      runtime.GOARCH,
		},
	}
	if err := tx.Send(ctx, ev); err != nil {
		fmt.Fprintf(os.Stderr, "guest-probe: %v\n", err)
		os.Exit(1)
	}
}
