package receipt

import (
	"crypto/sha256"
	"encoding/hex"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// Meta bundles execution context used to enrich receipts after aggregation.
type Meta struct {
	Start       time.Time
	RootPID     uint32
	Args        []string
	Workdir     string
	Stdout      []byte
	Stderr      []byte
	RunErr      error
	ExtraErrors []string
	Resources   Resources
	Backend     ExecutionInfo
	Provenance  string
	RedactPaths []string
}

func PopulateMetadata(r *Receipt, meta Meta) {
	r.ExecutionID = executionID(meta.Start, meta.RootPID, meta.Args)
	r.Timestamp = meta.Start.UTC().Format(time.RFC3339Nano)
	r.Provenance = meta.Provenance

	exitCode := r.ExitCode
	errStr := errorString(meta.RunErr)
	if len(meta.ExtraErrors) > 0 {
		extra := strings.Join(meta.ExtraErrors, "; ")
		if errStr == nil {
			errStr = &extra
		} else {
			combined := *errStr + "; " + extra
			errStr = &combined
		}
	}
	r.Outcome = &Outcome{
		ExitCode: exitCode,
		Signal:   signalForError(meta.RunErr),
		Error:    errStr,
	}

	r.Timing = &Timing{
		DurationMs: r.DurationMs,
		CPUTimeMs:  meta.Resources.CPUTimeMs,
	}

	rootExe := resolveExe(meta.Args)
	r.ProcessTree = buildProcessTree(r.Processes, meta.RootPID, rootExe, meta.Args, meta.Workdir)

	r.Environment = &Environment{
		Runtime: runtimeName(meta.Args),
		OS:      runtime.GOOS,
		Arch:    runtime.GOARCH,
		Sandbox: Sandbox{Network: "enabled"},
	}

	r.Execution = &ExecutionInfo{
		Backend:   meta.Backend.Backend,
		Isolation: meta.Backend.Isolation,
	}

	r.Artifacts = &Artifacts{
		StdoutHash: hashBytes(meta.Stdout),
		StderrHash: hashBytes(meta.Stderr),
	}

	if meta.Resources.CPUTimeMs > 0 || meta.Resources.MaxRSSKB > 0 {
		resCopy := meta.Resources
		r.Resources = &resCopy
	}

	if len(meta.RedactPaths) > 0 {
		r.MaskPaths(meta.RedactPaths)
	}
}

func (r *Receipt) MaskPaths(prefixes []string) {
	if r.Filesystem == nil {
		return
	}
	if r.Redactions == nil {
		r.Redactions = []string{}
	}
	r.Filesystem.Reads = redactList(r.Filesystem.Reads, prefixes, &r.Redactions)
	r.Filesystem.Writes = redactList(r.Filesystem.Writes, prefixes, &r.Redactions)
	r.Filesystem.Deletes = redactList(r.Filesystem.Deletes, prefixes, &r.Redactions)
}

func redactList(values []string, prefixes []string, redactions *[]string) []string {
	out := values[:0]
	for _, v := range values {
		if hasPrefix(v, prefixes) {
			*redactions = append(*redactions, v)
			continue
		}
		out = append(out, v)
	}
	return out
}

func hasPrefix(value string, prefixes []string) bool {
	for _, p := range prefixes {
		if strings.HasPrefix(value, p) {
			return true
		}
	}
	return false
}

func executionID(start time.Time, pid uint32, cmdArgs []string) string {
	base := []byte(strings.Join([]string{
		strconv.FormatInt(start.UnixNano(), 10),
		strconv.FormatInt(int64(pid), 10),
		strings.Join(cmdArgs, " "),
	}, ":"))
	sum := sha256.Sum256(base)
	return hex.EncodeToString(sum[:])
}

func hashBytes(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func resolveExe(cmdArgs []string) string {
	if len(cmdArgs) == 0 {
		return ""
	}
	exe, err := exec.LookPath(cmdArgs[0])
	if err != nil {
		return cmdArgs[0]
	}
	return exe
}

func buildProcessTree(processes []ProcessEntry, rootPID uint32, rootExe string, rootArgv []string, workingDir string) []ProcessV2 {
	if len(processes) == 0 {
		return []ProcessV2{}
	}
	out := make([]ProcessV2, 0, len(processes))
	for _, proc := range processes {
		argv := argvFromCmd(proc.Cmd)
		exe := ""
		if len(argv) > 0 {
			exe = argv[0]
		}
		wd := ""
		if proc.PID == rootPID {
			if len(rootExe) > 0 {
				exe = rootExe
			}
			if len(rootArgv) > 0 {
				argv = append([]string(nil), rootArgv...)
			}
			wd = workingDir
		}
		out = append(out, ProcessV2{
			PID:        proc.PID,
			PPID:       proc.PPID,
			Exe:        exe,
			Argv:       argv,
			WorkingDir: wd,
		})
	}
	return out
}

func argvFromCmd(cmd string) []string {
	if strings.TrimSpace(cmd) == "" {
		return []string{}
	}
	return strings.Fields(cmd)
}

func runtimeName(cmdArgs []string) string {
	if len(cmdArgs) == 0 {
		return "unknown"
	}
	base := filepath.Base(cmdArgs[0])
	if strings.HasPrefix(base, "python3") {
		return "python3.x"
	}
	if strings.HasPrefix(base, "python") {
		return "pythonx"
	}
	return base
}

func signalForError(err error) *string {
	if err == nil {
		return nil
	}
	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		return nil
	}
	status, ok := exitErr.Sys().(syscall.WaitStatus)
	if !ok || !status.Signaled() {
		return nil
	}
	sig := status.Signal().String()
	return &sig
}

func errorString(err error) *string {
	if err == nil {
		return nil
	}
	msg := err.Error()
	return &msg
}
