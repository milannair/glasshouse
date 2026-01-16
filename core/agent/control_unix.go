//go:build !windows

package agent

import (
	"context"
	"encoding/json"
	"net"
	"os"
)

// ControlServer exposes a unix socket for control commands.
type ControlServer struct {
	path    string
	handler func(context.Context, ControlCommand) ControlResponse
}

// NewControlServer builds a unix socket control server.
func NewControlServer(path string, handler func(context.Context, ControlCommand) ControlResponse) *ControlServer {
	return &ControlServer{path: path, handler: handler}
}

// Run starts accepting control connections.
func (s *ControlServer) Run(ctx context.Context) error {
	if s == nil {
		return nil
	}
	_ = os.Remove(s.path)
	listener, err := net.Listen("unix", s.path)
	if err != nil {
		return err
	}
	defer func() {
		listener.Close()
		_ = os.Remove(s.path)
	}()

	go func() {
		<-ctx.Done()
		listener.Close()
	}()

	for {
		conn, err := listener.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				return nil
			default:
				return err
			}
		}
		go s.handleConn(ctx, conn)
	}
}

func (s *ControlServer) handleConn(ctx context.Context, conn net.Conn) {
	defer conn.Close()
	dec := json.NewDecoder(conn)
	enc := json.NewEncoder(conn)
	for {
		var cmd ControlCommand
		if err := dec.Decode(&cmd); err != nil {
			return
		}
		resp := s.handler(ctx, cmd)
		_ = enc.Encode(resp)
	}
}
