package stream

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	core "github.com/GoCodeAlone/workflow-plugin-compute-core/protocol"
)

func TestBuildIngestDescriptorReturnsEndpointsAndScopedTokenRef(t *testing.T) {
	t.Parallel()

	descriptor, err := BuildIngestDescriptor(core.StreamSpec{
		IngestProtocols: []string{"rtmp", "srt", "whip"},
		ViewerEgress:    core.ViewerEgressConfig{HLS: true, WHEP: true},
	}, LeaseScope{
		OrgID:           "org-1",
		PoolID:          "pool-1",
		SessionID:       "session-1",
		PathName:        "org-1/pool-1/session-1",
		IngestHost:      "ingest.example.test",
		PublishTokenRef: "secret://streams/session-1/publish",
		MaxConnections:  2,
		Codecs:          []string{"h264", "aac"},
	})
	if err != nil {
		t.Fatalf("BuildIngestDescriptor: %v", err)
	}

	if descriptor.PublishTokenRef != "secret://streams/session-1/publish" {
		t.Fatalf("publish token ref = %q", descriptor.PublishTokenRef)
	}
	if descriptor.PathName != "org-1/pool-1/session-1" {
		t.Fatalf("path name = %q", descriptor.PathName)
	}
	for _, protocol := range []string{"rtmp", "srt", "whip"} {
		if !slices.ContainsFunc(descriptor.Endpoints, func(endpoint IngestEndpoint) bool {
			return endpoint.Protocol == protocol && endpoint.URL != ""
		}) {
			t.Fatalf("endpoint for %q missing: %#v", protocol, descriptor.Endpoints)
		}
	}
	if strings.Contains(formatDescriptorForTest(descriptor), "raw-secret") {
		t.Fatalf("descriptor leaked raw token: %#v", descriptor)
	}
}

func TestAuthHookDecisionAcceptsHostTokenBoundToLease(t *testing.T) {
	t.Parallel()

	hook := NewAuthHook(LeaseScope{
		OrgID:          "org-1",
		PoolID:         "pool-1",
		PathName:       "org-1/pool-1/session-1",
		MaxConnections: 2,
	}, fakeValidator{claims: TokenClaims{
		OrgID:     "org-1",
		PoolID:    "pool-1",
		ExpiresAt: time.Now().Add(time.Hour),
	}})

	decision := hook.Decide(context.Background(), AuthHookRequest{
		Token:    "raw-secret-token",
		Action:   "publish",
		Path:     "org-1/pool-1/session-1",
		Protocol: "rtmp",
		ID:       "conn-1",
	})
	if !decision.Allowed {
		t.Fatalf("decision = %#v", decision)
	}
}

func TestAuthHookDecisionDeniesExpiredMismatchedAndOverCapWithoutLeakingToken(t *testing.T) {
	t.Parallel()

	hook := NewAuthHook(LeaseScope{
		OrgID:          "org-1",
		PoolID:         "pool-1",
		PathName:       "org-1/pool-1/session-1",
		MaxConnections: 1,
	}, fakeValidator{claims: TokenClaims{
		OrgID:     "org-1",
		PoolID:    "wrong-pool",
		ExpiresAt: time.Now().Add(time.Hour),
	}})

	decision := hook.Decide(context.Background(), AuthHookRequest{
		Token:    "raw-secret-token",
		Action:   "publish",
		Path:     "org-1/pool-1/session-1",
		Protocol: "rtmp",
		ID:       "conn-1",
	})
	if decision.Allowed {
		t.Fatalf("mismatched token was allowed: %#v", decision)
	}
	if strings.Contains(decision.Reason, "raw-secret-token") || !strings.Contains(decision.Reason, "(creds redacted)") {
		t.Fatalf("decision reason did not redact credentials: %q", decision.Reason)
	}

	hook = NewAuthHook(LeaseScope{
		OrgID:          "org-1",
		PoolID:         "pool-1",
		PathName:       "org-1/pool-1/session-1",
		MaxConnections: 1,
	}, fakeValidator{claims: TokenClaims{
		OrgID:     "org-1",
		PoolID:    "pool-1",
		ExpiresAt: time.Now().Add(time.Hour),
	}})
	if !hook.Decide(context.Background(), AuthHookRequest{
		Token:    "raw-secret-token",
		Action:   "publish",
		Path:     "org-1/pool-1/session-1",
		Protocol: "rtmp",
		ID:       "conn-1",
	}).Allowed {
		t.Fatal("first connection denied")
	}
	overCap := hook.Decide(context.Background(), AuthHookRequest{
		Token:    "raw-secret-token",
		Action:   "publish",
		Path:     "org-1/pool-1/session-1",
		Protocol: "rtmp",
		ID:       "conn-2",
	})
	if overCap.Allowed || !strings.Contains(overCap.Reason, "connection cap") {
		t.Fatalf("over-cap decision = %#v", overCap)
	}

	anonymousHook := NewAuthHook(LeaseScope{
		OrgID:          "org-1",
		PoolID:         "pool-1",
		PathName:       "org-1/pool-1/session-1",
		MaxConnections: 1,
	}, fakeValidator{claims: TokenClaims{OrgID: "org-1", PoolID: "pool-1", ExpiresAt: time.Now().Add(time.Hour)}})
	for i := 0; i < 2; i++ {
		decision := anonymousHook.Decide(context.Background(), AuthHookRequest{
			Token:    "raw-secret-token",
			Action:   "publish",
			Path:     "org-1/pool-1/session-1",
			Protocol: "rtmp",
		})
		if i == 0 && !decision.Allowed {
			t.Fatalf("anonymous first connection denied: %#v", decision)
		}
		if i == 1 && (decision.Allowed || !strings.Contains(decision.Reason, "connection cap")) {
			t.Fatalf("anonymous over-cap decision = %#v", decision)
		}
	}

	expired := NewAuthHook(LeaseScope{OrgID: "org-1", PoolID: "pool-1", PathName: "org-1/pool-1/session-1"},
		fakeValidator{claims: TokenClaims{OrgID: "org-1", PoolID: "pool-1", ExpiresAt: time.Now().Add(-time.Minute)}}).
		Decide(context.Background(), AuthHookRequest{
			Token:    "raw-secret-token",
			Action:   "publish",
			Path:     "org-1/pool-1/session-1",
			Protocol: "rtmp",
			ID:       "conn-1",
		})
	if expired.Allowed || !strings.Contains(expired.Reason, "expired") {
		t.Fatalf("expired decision = %#v", expired)
	}
}

func TestAuthHookDecisionRedactsValidatorErrors(t *testing.T) {
	t.Parallel()

	hook := NewAuthHook(LeaseScope{OrgID: "org-1", PoolID: "pool-1", PathName: "live"},
		fakeValidator{err: errors.New("invalid raw-secret-token")})

	decision := hook.Decide(context.Background(), AuthHookRequest{
		Token:    "raw-secret-token",
		Action:   "publish",
		Path:     "live",
		Protocol: "rtmp",
		ID:       "conn-1",
	})
	if decision.Allowed {
		t.Fatalf("decision = %#v", decision)
	}
	if strings.Contains(decision.Reason, "raw-secret-token") || !strings.Contains(decision.Reason, "(creds redacted)") {
		t.Fatalf("reason leaked credentials: %q", decision.Reason)
	}
}

type fakeValidator struct {
	claims TokenClaims
	err    error
}

func (v fakeValidator) ValidatePublishToken(context.Context, string) (TokenClaims, error) {
	return v.claims, v.err
}

func formatDescriptorForTest(descriptor IngestDescriptor) string {
	var b strings.Builder
	b.WriteString(descriptor.PublishTokenRef)
	for _, endpoint := range descriptor.Endpoints {
		b.WriteString(endpoint.URL)
	}
	return b.String()
}
