package session

import (
	"context"
	"reflect"
	"testing"
	"time"

	core "github.com/GoCodeAlone/workflow-plugin-compute-core/protocol"
)

func TestAdapterImplementsPublicServiceSessionProvider(t *testing.T) {
	t.Parallel()

	var _ core.ServiceSessionProvider = (*Adapter)(nil)
	adapter := NewAdapter()
	if got, want := reflect.TypeOf(*adapter).PkgPath(), "github.com/GoCodeAlone/workflow-plugin-stream/session"; got != want {
		t.Fatalf("adapter package = %q, want %q", got, want)
	}
	if !adapter.CanServe(core.Task{Workload: core.WorkloadSpec{Kind: core.WorkloadVideoStream}}, core.Lease{}) {
		t.Fatal("adapter did not accept video-stream workload")
	}
}

func TestServiceSessionSuperviseRenewMutateDrainAndUrgentKill(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	adapter := NewAdapter()
	raw, err := adapter.StartServiceSession(ctx, core.ServiceRunRequest{
		Task: core.Task{
			ID:       "task-1",
			OrgID:    "org-1",
			PoolID:   "pool-1",
			Workload: core.WorkloadSpec{Kind: core.WorkloadVideoStream},
		},
		Lease: core.Lease{ID: "lease-1", WorkerID: "worker-1"},
	})
	if err != nil {
		t.Fatalf("StartServiceSession: %v", err)
	}
	session, ok := raw.(*Session)
	if !ok {
		t.Fatalf("session = %T, want *Session", raw)
	}
	generation := session.Generation()

	if err := session.Renew(context.Background(), 50*time.Millisecond); err != nil {
		t.Fatalf("Renew 1: %v", err)
	}
	firstExpiry := session.ExpiresAt()
	time.Sleep(time.Millisecond)
	if err := session.Renew(context.Background(), 100*time.Millisecond); err != nil {
		t.Fatalf("Renew 2: %v", err)
	}
	if !session.ExpiresAt().After(firstExpiry) {
		t.Fatalf("renew did not extend expiry: first=%s second=%s", firstExpiry, session.ExpiresAt())
	}

	if err := session.Mutate(context.Background(), Mutation{Op: MutationAddDestination, Destination: "rtmp://edge/live"}); err != nil {
		t.Fatalf("add destination: %v", err)
	}
	if err := session.Mutate(context.Background(), Mutation{Op: MutationRemoveDestination, Destination: "rtmp://edge/live"}); err != nil {
		t.Fatalf("remove destination: %v", err)
	}
	if session.Generation() != generation {
		t.Fatalf("mutate restarted child: generation=%d want %d", session.Generation(), generation)
	}

	result, err := session.Drain(context.Background())
	if err != nil {
		t.Fatalf("Drain: %v", err)
	}
	if result.StatusEvidence.Preview != "drained" {
		t.Fatalf("drain result = %#v", result)
	}
	if err := result.Validate(); err != nil {
		t.Fatalf("drain result invalid: %v", err)
	}

	raw, err = adapter.StartServiceSession(ctx, core.ServiceRunRequest{
		Task:  core.Task{ID: "task-2", OrgID: "org-1", PoolID: "pool-1", Workload: core.WorkloadSpec{Kind: core.WorkloadVideoStream}},
		Lease: core.Lease{ID: "lease-2", WorkerID: "worker-1"},
	})
	if err != nil {
		t.Fatalf("StartServiceSession 2: %v", err)
	}
	session = raw.(*Session)
	if err := session.UrgentKill(context.Background()); err != nil {
		t.Fatalf("UrgentKill: %v", err)
	}
	select {
	case <-session.Done():
	case <-time.After(time.Second):
		t.Fatal("urgent kill did not cancel executor context")
	}
}
