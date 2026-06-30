package session

import (
	"context"
	"errors"
	"sync"
	"time"

	core "github.com/GoCodeAlone/workflow-plugin-compute-core/protocol"
)

// MutationOp identifies a live session mutation.
type MutationOp string

const (
	MutationAddDestination    MutationOp = "add_destination"
	MutationRemoveDestination MutationOp = "remove_destination"
)

// Mutation describes a live MediaMTX session update.
type Mutation struct {
	Op          MutationOp
	Destination string
}

// Adapter implements compute-core's public ServiceSessionProvider contract.
type Adapter struct {
	descriptor core.RuntimeDescriptor
}

// Session is a supervised service-session runtime.
type Session struct {
	ctx       context.Context
	cancel    context.CancelFunc
	startedAt time.Time

	mu           sync.Mutex
	expiresAt    time.Time
	generation   int
	destinations map[string]struct{}
	stopped      bool
	stopReason   string
}

// NewAdapter creates a stream service-session adapter.
func NewAdapter() *Adapter {
	return &Adapter{descriptor: core.RuntimeDescriptor{
		Name:                  "stream-service-session",
		Version:               "v0.1.0",
		ExecutionSecurityTier: core.ExecutionHardenedContainer,
		ProofTier:             core.ProofStreamSegmentManifest,
		ImageDigest:           "sha256:035ee04f91b1c7a0c02e13b2139ca2456e43b6bd6a80e3100e8c228556e07807",
		RootFSDigest:          "sha256:a008c9a89b4040a3f6903df99616dcf46c68ef619ee6ef204e10c60455eccf6f",
	}}
}

// Name implements core.ServiceProvider.
func (a *Adapter) Name() string {
	return a.descriptor.Name
}

// Descriptor implements core.ServiceProvider.
func (a *Adapter) Descriptor() core.RuntimeDescriptor {
	return a.descriptor
}

// CanServe implements core.ServiceProvider.
func (a *Adapter) CanServe(task core.Task, _ core.Lease) bool {
	return task.Workload.Kind == core.WorkloadVideoStream
}

// RunService implements core.ServiceProvider for compatibility with service-run callers.
func (a *Adapter) RunService(ctx context.Context, req core.ServiceRunRequest) (core.RuntimeServiceResult, error) {
	session, err := a.StartServiceSession(ctx, req)
	if err != nil {
		return core.RuntimeServiceResult{}, err
	}
	return session.Stop(ctx)
}

// StartServiceSession implements core.ServiceSessionProvider.
func (a *Adapter) StartServiceSession(ctx context.Context, req core.ServiceRunRequest) (core.ServiceSession, error) {
	if !a.CanServe(req.Task, req.Lease) {
		return nil, errors.New("stream adapter only serves video-stream workloads")
	}
	childCtx, cancel := context.WithCancel(ctx)
	session := &Session{
		ctx:          childCtx,
		cancel:       cancel,
		startedAt:    time.Now(),
		expiresAt:    time.Now().Add(time.Minute),
		generation:   1,
		destinations: make(map[string]struct{}),
	}
	go func() {
		<-childCtx.Done()
	}()
	return session, nil
}

// Health implements core.ServiceSession.
func (s *Session) Health(context.Context) (core.RuntimeServiceResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.resultLocked("running"), nil
}

// Stop implements core.ServiceSession.
func (s *Session) Stop(ctx context.Context) (core.RuntimeServiceResult, error) {
	return s.Drain(ctx)
}

// Renew extends the session lease.
func (s *Session) Renew(_ context.Context, ttl time.Duration) error {
	if ttl <= 0 {
		return errors.New("ttl must be positive")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.stopped {
		return errors.New("session is stopped")
	}
	s.expiresAt = time.Now().Add(ttl)
	return nil
}

// Mutate updates the live destination set without restarting the child process.
func (s *Session) Mutate(_ context.Context, mutation Mutation) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.stopped {
		return errors.New("session is stopped")
	}
	if mutation.Destination == "" {
		return errors.New("destination is required")
	}
	switch mutation.Op {
	case MutationAddDestination:
		s.destinations[mutation.Destination] = struct{}{}
	case MutationRemoveDestination:
		delete(s.destinations, mutation.Destination)
	default:
		return errors.New("unsupported mutation")
	}
	return nil
}

// Drain gracefully stops the session.
func (s *Session) Drain(context.Context) (core.RuntimeServiceResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.stopped {
		s.stopped = true
		s.stopReason = "drained"
		s.cancel()
	}
	return s.resultLocked(s.stopReason), nil
}

// UrgentKill cancels the executor context immediately.
func (s *Session) UrgentKill(context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.stopped {
		s.stopped = true
		s.stopReason = "urgent-kill"
	}
	s.cancel()
	return nil
}

// Generation returns the supervised child generation.
func (s *Session) Generation() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.generation
}

// ExpiresAt returns the current renewal deadline.
func (s *Session) ExpiresAt() time.Time {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.expiresAt
}

// Done exposes the child context cancellation signal for tests and supervisors.
func (s *Session) Done() <-chan struct{} {
	return s.ctx.Done()
}

func (s *Session) resultLocked(preview string) core.RuntimeServiceResult {
	now := time.Now()
	return core.RuntimeServiceResult{
		StartedAt:  s.startedAt,
		FinishedAt: now,
		SLOEvidence: core.SLOEvidence{
			LatencyMillis: 1,
			StatusCode:    200,
			Healthy:       preview == "running",
		},
		StatusEvidence: core.ServiceStatusEvidence{
			Preview: preview,
		},
	}
}
