package execution

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"syscall"
	"time"

	"glasshouse/core/identity"
	"glasshouse/core/profiling"
	"glasshouse/core/receipt"
)

// Engine orchestrates execution, optional profiling, and receipt building.
type Engine struct {
	Backend  ExecutionBackend
	Profiler profiling.Controller
}

func (e Engine) Run(ctx context.Context, spec ExecutionSpec) (ExecutionResult, error) {
	result := ExecutionResult{
		ProfilingEnabled: spec.Profiling != profiling.ProfilingDisabled,
	}
	if err := e.validateSpec(spec); err != nil {
		result.Err = err
		return result, err
	}
	if e.Backend == nil {
		err := errors.New("backend required")
		result.Err = err
		return result, err
	}

	if err := e.Backend.Prepare(ctx); err != nil {
		result.Err = err
		return result, err
	}

	result.StartedAt = time.Now()
	handle, err := e.Backend.Start(spec)
	result.Handle = handle
	if err != nil {
		result.Err = err
		_ = e.Backend.Cleanup(handle)
		return result, err
	}
	profileInfo := e.Backend.ProfilingInfo(handle)
	rootPID := profileInfo.Identity.RootPID

	var (
		session        profiling.Session
		agg            *receipt.Aggregator
		execID         identity.ExecutionID
		rootStartTime  uint64
		aggWG          sync.WaitGroup
		aggErrors      []string
		profilingErr   error
		profilingReady bool
	)

	if spec.Profiling != profiling.ProfilingDisabled {
		if e.Profiler == nil {
			profilingErr = fmt.Errorf("profiling requested but profiler not configured")
		} else {
			target := profiling.Target{
				RootPID:    profileInfo.Identity.RootPID,
				CgroupPath: profileInfo.Identity.CgroupPath,
				Namespaces: profileInfo.Identity.Namespaces,
				Mode:       spec.Profiling,
			}
			session, profilingErr = e.Profiler.Start(ctx, target)
			if profilingErr == nil {
				provenance := e.provenanceFor(spec)
				agg = receipt.NewAggregator(provenance)
				rootPID := uint32(target.RootPID)
				rootStart, startErr := identity.ProcessStartTime(rootPID)
				if startErr != nil {
					aggErrors = append(aggErrors, fmt.Sprintf("resolve pid start time: %v", startErr))
				}
				rootStartTime = rootStart
				execID = agg.StartExecution(receipt.ExecutionStart{
					RootPID:         rootPID,
					RootStartTime:   rootStart,
					Command:         strings.Join(spec.Args, " "),
					StartedAt:       result.StartedAt,
					ObservationMode: observationModeForProfiling(spec.Profiling),
				})
				profilingReady = true

				aggWG.Add(1)
				go func() {
					defer aggWG.Done()
					for ev := range session.Events() {
						_ = agg.HandleEvent(ev)
					}
				}()

				aggWG.Add(1)
				go func() {
					defer aggWG.Done()
					for err := range session.Errors() {
						if err != nil {
							aggErrors = append(aggErrors, err.Error())
						}
					}
				}()
			}
		}
	}

	waitRes, waitErr := e.Backend.Wait(handle)
	result.ExitCode = waitRes.ExitCode
	result.Err = waitRes.Err
	if result.Err == nil {
		result.Err = waitErr
	}
	result.CompletedAt = time.Now()
	result.ProfilingAttached = profilingReady
	result.ProfilingError = profilingErr

	if session != nil {
		_ = session.Close()
	}
	aggWG.Wait()

	extraErrors := aggErrors
	if profErr := result.ProfilingError; profErr != nil {
		extraErrors = append(extraErrors, profErr.Error())
	}

	if extra, ok := e.Backend.(ExtraErrorProvider); ok {
		extraErrors = append(extraErrors, extra.ExtraErrors()...)
	}

	resources := ResourcesFromBackend(e.Backend)

	if agg != nil && profilingReady {
		agg.EndExecution(execID, result.CompletedAt)
		rec, ok := agg.FlushExecution(execID, result.ExitCode, result.CompletedAt.Sub(result.StartedAt))
		if !ok {
			rec = agg.Receipt(result.ExitCode, result.CompletedAt.Sub(result.StartedAt))
		}
		stdoutBytes, stderrBytes := backendOutput(e.Backend)
		backendInfo := e.metadataForBackend()
		provenance := e.provenanceFor(spec)
		meta := receipt.Meta{
			Start:           result.StartedAt,
			End:             result.CompletedAt,
			RootPID:         uint32(rootPID),
			RootStartTime:   rootStartTime,
			ExecutionID:     execID.String(),
			Args:            spec.Args,
			Workdir:         spec.Workdir,
			Stdout:          stdoutBytes,
			Stderr:          stderrBytes,
			RunErr:          result.Err,
			ExtraErrors:     extraErrors,
			Resources:       resources,
			Backend:         backendInfo,
			Provenance:      provenance,
			ObservationMode: observationModeForProfiling(spec.Profiling),
			Completeness:    "closed",
			RedactPaths:     spec.ReceiptMask,
		}
		receipt.PopulateMetadata(&rec, meta)
		result.Receipt = &rec
	}

	if cleanupErr := e.Backend.Cleanup(handle); cleanupErr != nil && result.Err == nil {
		result.Err = cleanupErr
	}

	return result, result.Err
}

func (e Engine) validateSpec(spec ExecutionSpec) error {
	if len(spec.Args) == 0 {
		return errors.New("no command provided")
	}
	return nil
}

func (e Engine) provenanceFor(spec ExecutionSpec) string {
	switch spec.Profiling {
	case profiling.ProfilingGuest:
		return "guest"
	case profiling.ProfilingCombined:
		return "host+guest"
	default:
		return "host"
	}
}

func observationModeForProfiling(mode profiling.Mode) string {
	switch mode {
	case profiling.ProfilingGuest:
		return "guest"
	case profiling.ProfilingCombined:
		return "host+guest"
	default:
		return "host"
	}
}

func (e Engine) metadataForBackend() receipt.ExecutionInfo {
	if provider, ok := e.Backend.(MetadataProvider); ok {
		return provider.Metadata()
	}
	return receipt.ExecutionInfo{
		Backend:   e.Backend.Name(),
		Isolation: "none",
	}
}

func backendOutput(b ExecutionBackend) ([]byte, []byte) {
	if b == nil {
		return nil, nil
	}
	if outputProvider, ok := b.(OutputProvider); ok {
		return outputProvider.Stdout(), outputProvider.Stderr()
	}
	return nil, nil
}

func ResourcesFromBackend(b ExecutionBackend) receipt.Resources {
	if psProvider, ok := b.(ProcessStateProvider); ok {
		if ps := psProvider.ProcessState(); ps != nil {
			cpu := ps.UserTime() + ps.SystemTime()
			resources := receipt.Resources{
				CPUTimeMs: cpu.Milliseconds(),
			}
			if usage, ok := ps.SysUsage().(*syscall.Rusage); ok {
				resources.MaxRSSKB = int64(usage.Maxrss)
			}
			return resources
		}
	}
	return receipt.Resources{}
}
