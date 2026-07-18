package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/equate/ogsd/services/snmp-collector/internal/control"
)

// runRPC forwards one NDJSON control request from stdin to a Unix socket and prints the response.
// Used by host-side setup tooling via `docker compose exec` on platforms where bind-mounted
// Unix sockets are not dialable from the host (Docker Desktop on macOS).
func runRPC(args []string) int {
	fs := flag.NewFlagSet("rpc", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	socketPath := fs.String("socket", "", "control socket path")
	timeout := fs.Duration("timeout", control.DefaultRequestTimeout, "per-request timeout")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "rpc accepts no positional arguments")
		return 2
	}
	if *socketPath == "" {
		fmt.Fprintln(os.Stderr, "rpc: -socket is required")
		return 2
	}

	scanner := bufio.NewScanner(os.Stdin)
	if !scanner.Scan() {
		fmt.Fprintln(os.Stderr, "rpc: expected one NDJSON request line on stdin")
		return 2
	}
	var req control.Request
	if err := json.Unmarshal(scanner.Bytes(), &req); err != nil {
		fmt.Fprintf(os.Stderr, "rpc: invalid request JSON: %v\n", err)
		return 2
	}

	client := control.NewClient(*socketPath)
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	resp, err := client.Call(ctx, req.ID, req.Method, req.Params)
	if err != nil {
		fmt.Fprintf(os.Stderr, "rpc: %v\n", err)
		return 1
	}
	enc := json.NewEncoder(os.Stdout)
	if err := enc.Encode(resp); err != nil {
		fmt.Fprintf(os.Stderr, "rpc: encode response: %v\n", err)
		return 1
	}
	return 0
}
