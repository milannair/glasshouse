//go:build linux

package audit

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/link"
	"github.com/cilium/ebpf/ringbuf"
)

const (
	defaultObjDir = "ebpf/objects"
	eventSize     = 308
)

type ebpfCollector struct {
	mu      sync.Mutex
	started bool
	events  chan Event
	errs    chan error
	readers []*ringbuf.Reader
	links   []link.Link
	objs    []*ebpf.Collection
	wg      sync.WaitGroup
	closed  chan struct{}
}

func NewCollector(cfg Config) (Collector, error) {
	dir := cfg.BPFObjectDir
	if dir == "" {
		if env := os.Getenv("GLASSHOUSE_BPF_DIR"); env != "" {
			dir = env
		} else {
			dir = defaultObjDir
		}
	}

	execPaths := []string{filepath.Join(dir, "exec.o")}
	if captureArgvEnabled() {
		execPaths = []string{
			filepath.Join(dir, "exec-argv.o"),
			filepath.Join(dir, "exec.o"),
		}
	}
	otherPaths := []string{
		filepath.Join(dir, "fs.o"),
		filepath.Join(dir, "net.o"),
	}

	collector := &ebpfCollector{
		events: make(chan Event, 1024),
		errs:   make(chan error, 16),
		closed: make(chan struct{}),
	}

	loaded := 0
	var loadErrors []error
	cwd, _ := os.Getwd()
	fmt.Fprintf(os.Stderr, "glasshouse: loading eBPF objects from dir=%s (cwd=%s)\n", dir, cwd)
	loadPath := func(path string) bool {
		// Debug: check if file exists
		absPath, _ := filepath.Abs(path)
		if _, err := os.Stat(path); err != nil {
			fmt.Fprintf(os.Stderr, "glasshouse: file not found: %s (abs: %s) - %v\n", path, absPath, err)
			loadErrors = append(loadErrors, fmt.Errorf("file not found: %s (%w)", path, err))
			collector.errs <- fmt.Errorf("eBPF object missing: %s", path)
			return false
		}
		fmt.Fprintf(os.Stderr, "glasshouse: attempting to load: %s\n", path)
		coll, readers, links, err := loadObject(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "glasshouse: load failed for %s: %v\n", path, err)
			loadErrors = append(loadErrors, err)
			collector.errs <- err
			return false
		}
		fmt.Fprintf(os.Stderr, "glasshouse: successfully loaded: %s\n", path)
		collector.objs = append(collector.objs, coll)
		collector.readers = append(collector.readers, readers...)
		collector.links = append(collector.links, links...)
		loaded++
		return true
	}

	execLoaded := false
	for _, path := range execPaths {
		if loadPath(path) {
			execLoaded = true
			break
		}
	}
	if !execLoaded {
		fmt.Fprintln(os.Stderr, "glasshouse: exec eBPF program not loaded; exec events will be missing")
	}
	for _, path := range otherPaths {
		_ = loadPath(path)
	}

	if loaded == 0 {
		if len(loadErrors) > 0 {
			// Print detailed errors to stderr for debugging
			fmt.Fprintf(os.Stderr, "glasshouse: eBPF load errors:\n")
			for _, err := range loadErrors {
				fmt.Fprintf(os.Stderr, "  - %s\n", err.Error())
			}
			errMsg := fmt.Sprintf("no eBPF objects found in %s", dir)
			for _, err := range loadErrors {
				errMsg += "; " + err.Error()
			}
			return nil, fmt.Errorf(errMsg)
		}
		return nil, fmt.Errorf("no eBPF objects found in %s (run scripts/build-ebpf.sh)", dir)
	}

	return collector, nil
}

func (c *ebpfCollector) Start(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.started {
		return nil
	}
	c.started = true

	for _, reader := range c.readers {
		c.wg.Add(1)
		go c.readLoop(ctx, reader)
	}

	return nil
}

func (c *ebpfCollector) Events() <-chan Event {
	return c.events
}

func (c *ebpfCollector) Errors() <-chan error {
	return c.errs
}

func (c *ebpfCollector) Close() error {
	c.mu.Lock()
	if !c.started {
		c.mu.Unlock()
		return nil
	}
	c.mu.Unlock()

	close(c.closed)
	for _, reader := range c.readers {
		_ = reader.Close()
	}
	c.wg.Wait()

	for _, l := range c.links {
		_ = l.Close()
	}
	for _, obj := range c.objs {
		obj.Close()
	}

	close(c.events)
	close(c.errs)
	return nil
}

func (c *ebpfCollector) readLoop(ctx context.Context, reader *ringbuf.Reader) {
	defer c.wg.Done()
	for {
		record, err := reader.Read()
		if err != nil {
			if errors.Is(err, ringbuf.ErrClosed) {
				return
			}
			select {
			case c.errs <- err:
			case <-c.closed:
			}
			continue
		}
		ev, err := parseEvent(record.RawSample)
		if err != nil {
			select {
			case c.errs <- err:
			case <-c.closed:
			}
			continue
		}
		select {
		case c.events <- ev:
		case <-c.closed:
			return
		case <-ctx.Done():
			return
		}
	}
}

func parseEvent(data []byte) (Event, error) {
	if len(data) < eventSize {
		return Event{}, fmt.Errorf("short event: %d", len(data))
	}

	ev := Event{}
	ev.Type = EventType(binary.LittleEndian.Uint32(data[0:4]))
	ev.PID = binary.LittleEndian.Uint32(data[4:8])
	ev.PPID = binary.LittleEndian.Uint32(data[8:12])
	ev.Flags = binary.LittleEndian.Uint32(data[12:16])
	ev.Port = binary.LittleEndian.Uint16(data[16:18])
	ev.AddrFamily = data[18]
	ev.Proto = data[19]
	copy(ev.Addr[:], data[20:36])
	ev.Comm = trimNull(data[36:52])
	ev.Path = trimNull(data[52:308])

	return ev, nil
}

func trimNull(b []byte) string {
	idx := bytes.IndexByte(b, 0)
	if idx == -1 {
		idx = len(b)
	}
	return string(b[:idx])
}

func loadObject(path string) (*ebpf.Collection, []*ringbuf.Reader, []link.Link, error) {
	if _, err := os.Stat(path); err != nil {
		return nil, nil, nil, fmt.Errorf("eBPF object missing: %s", path)
	}

	spec, err := ebpf.LoadCollectionSpec(path)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("load eBPF spec %s: %w", path, err)
	}

	coll, err := ebpf.NewCollection(spec)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("load eBPF collection %s: %w", path, err)
	}

	readers := []*ringbuf.Reader{}
	if eventsMap := coll.Maps["events"]; eventsMap != nil {
		r, err := ringbuf.NewReader(eventsMap)
		if err != nil {
			coll.Close()
			return nil, nil, nil, fmt.Errorf("open ringbuf %s: %w", path, err)
		}
		readers = append(readers, r)
	}

	links := []link.Link{}
	switch filepath.Base(path) {
	case "exec.o":
		link1, err := attachTracepoint(coll, "trace_execve", "syscalls", "sys_enter_execve")
		if err != nil {
			coll.Close()
			return nil, nil, nil, err
		}
		links = append(links, link1)

		link2, err := attachTracepoint(coll, "trace_execveat", "syscalls", "sys_enter_execveat")
		if err != nil {
			coll.Close()
			return nil, nil, nil, err
		}
		links = append(links, link2)
	case "exec-argv.o":
		link1, err := attachTracepoint(coll, "trace_execve", "syscalls", "sys_enter_execve")
		if err != nil {
			coll.Close()
			return nil, nil, nil, err
		}
		links = append(links, link1)

		link2, err := attachTracepoint(coll, "trace_execveat", "syscalls", "sys_enter_execveat")
		if err != nil {
			coll.Close()
			return nil, nil, nil, err
		}
		links = append(links, link2)
	case "fs.o":
		link1, err := attachTracepoint(coll, "trace_openat", "syscalls", "sys_enter_openat")
		if err != nil {
			coll.Close()
			return nil, nil, nil, err
		}
		links = append(links, link1)

		link2, err := attachTracepoint(coll, "trace_open", "syscalls", "sys_enter_open")
		if err != nil {
			coll.Close()
			return nil, nil, nil, err
		}
		links = append(links, link2)
	case "net.o":
		link1, err := attachTracepoint(coll, "trace_connect", "syscalls", "sys_enter_connect")
		if err != nil {
			coll.Close()
			return nil, nil, nil, err
		}
		links = append(links, link1)

		link2, err := attachTracepoint(coll, "trace_socket_enter", "syscalls", "sys_enter_socket")
		if err != nil {
			coll.Close()
			return nil, nil, nil, err
		}
		links = append(links, link2)

		link3, err := attachTracepoint(coll, "trace_socket_exit", "syscalls", "sys_exit_socket")
		if err != nil {
			coll.Close()
			return nil, nil, nil, err
		}
		links = append(links, link3)
	}

	return coll, readers, links, nil
}

func captureArgvEnabled() bool {
	value := strings.ToLower(strings.TrimSpace(os.Getenv("GLASSHOUSE_CAPTURE_ARGV")))
	if value == "" || value == "0" || value == "false" || value == "no" {
		return false
	}

	if isWSL() && value != "force" && !isTruthy(os.Getenv("GLASSHOUSE_CAPTURE_ARGV_FORCE")) {
		fmt.Fprintln(os.Stderr, "glasshouse: argv capture disabled on WSL; set GLASSHOUSE_CAPTURE_ARGV=force to override")
		return false
	}

	return true
}

func isTruthy(value string) bool {
	value = strings.ToLower(strings.TrimSpace(value))
	switch value {
	case "1", "true", "yes", "on", "force":
		return true
	default:
		return false
	}
}

func isWSL() bool {
	if data, err := os.ReadFile("/proc/version"); err == nil {
		if strings.Contains(strings.ToLower(string(data)), "microsoft") {
			return true
		}
	}
	if data, err := os.ReadFile("/proc/sys/kernel/osrelease"); err == nil {
		if strings.Contains(strings.ToLower(string(data)), "microsoft") {
			return true
		}
	}
	return false
}

func attachTracepoint(coll *ebpf.Collection, progName, category, name string) (link.Link, error) {
	prog := coll.Programs[progName]
	if prog == nil {
		return nil, fmt.Errorf("program %s not found", progName)
	}
	l, err := link.Tracepoint(category, name, prog, nil)
	if err != nil {
		return nil, fmt.Errorf("attach %s/%s: %w", category, name, err)
	}
	return l, nil
}
