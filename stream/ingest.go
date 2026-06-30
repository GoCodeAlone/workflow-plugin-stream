package stream

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	core "github.com/GoCodeAlone/workflow-plugin-compute-core/protocol"
)

// LeaseScope carries host-issued stream lease boundaries.
type LeaseScope struct {
	OrgID           string
	PoolID          string
	SessionID       string
	PathName        string
	IngestHost      string
	PublishTokenRef string
	MaxConnections  int
	Codecs          []string
}

// IngestDescriptor describes how a publisher can reach a stream session.
type IngestDescriptor struct {
	PathName        string           `json:"path_name"`
	Endpoints       []IngestEndpoint `json:"endpoints"`
	PublishTokenRef string           `json:"publish_token_ref"`
	MaxConnections  int              `json:"max_connections,omitempty"`
	Codecs          []string         `json:"codecs,omitempty"`
}

// IngestEndpoint is a protocol-specific publisher endpoint.
type IngestEndpoint struct {
	Protocol string `json:"protocol"`
	URL      string `json:"url"`
}

// TokenClaims are host-issued publish token claims used by the auth hook.
type TokenClaims struct {
	OrgID     string
	PoolID    string
	ExpiresAt time.Time
}

// TokenValidator validates raw host-issued publish tokens server-side.
type TokenValidator interface {
	ValidatePublishToken(context.Context, string) (TokenClaims, error)
}

// AuthHook validates MediaMTX HTTP auth requests against a lease scope.
type AuthHook struct {
	scope     LeaseScope
	validator TokenValidator
	mu        sync.Mutex
	activeIDs map[string]struct{}
	anonymous int
}

// AuthHookRequest matches the MediaMTX HTTP auth payload fields this plugin uses.
type AuthHookRequest struct {
	User      string `json:"user,omitempty"`
	Password  string `json:"password,omitempty"`
	Token     string `json:"token,omitempty"`
	Action    string `json:"action"`
	Path      string `json:"path"`
	Protocol  string `json:"protocol"`
	ID        string `json:"id,omitempty"`
	Query     string `json:"query,omitempty"`
	UserAgent string `json:"userAgent,omitempty"`
	IP        string `json:"ip,omitempty"`
}

// AuthHookDecision is the auth-hook allow/deny result.
type AuthHookDecision struct {
	Allowed    bool
	StatusCode int
	Reason     string
}

// BuildIngestDescriptor returns publish endpoints plus the scoped credential ref.
func BuildIngestDescriptor(spec core.StreamSpec, scope LeaseScope) (IngestDescriptor, error) {
	if err := spec.Validate(); err != nil {
		return IngestDescriptor{}, fmt.Errorf("stream spec: %w", err)
	}
	if strings.TrimSpace(scope.PathName) == "" {
		return IngestDescriptor{}, errors.New("path name is required")
	}
	if strings.TrimSpace(scope.IngestHost) == "" {
		return IngestDescriptor{}, errors.New("ingest host is required")
	}
	if !strings.HasPrefix(scope.PublishTokenRef, "secret://") {
		return IngestDescriptor{}, errors.New("publish token ref must use secret://")
	}

	descriptor := IngestDescriptor{
		PathName:        scope.PathName,
		PublishTokenRef: scope.PublishTokenRef,
		MaxConnections:  scope.MaxConnections,
		Codecs:          append([]string(nil), scope.Codecs...),
	}
	escapedPath := escapePath(scope.PathName)
	for _, protocol := range spec.IngestProtocols {
		switch protocol {
		case "rtmp":
			descriptor.Endpoints = append(descriptor.Endpoints, IngestEndpoint{
				Protocol: "rtmp",
				URL:      "rtmp://" + scope.IngestHost + "/" + escapedPath,
			})
		case "srt":
			descriptor.Endpoints = append(descriptor.Endpoints, IngestEndpoint{
				Protocol: "srt",
				URL: "srt://" + scope.IngestHost +
					":8890?streamid=publish:" + url.QueryEscape(scope.PathName),
			})
		case "whip":
			descriptor.Endpoints = append(descriptor.Endpoints, IngestEndpoint{
				Protocol: "whip",
				URL:      "https://" + scope.IngestHost + "/" + escapedPath + "/whip",
			})
		default:
			return IngestDescriptor{}, fmt.Errorf("unsupported ingest protocol %q", protocol)
		}
	}
	return descriptor, nil
}

// NewAuthHook creates a MediaMTX HTTP auth-hook evaluator.
func NewAuthHook(scope LeaseScope, validator TokenValidator) *AuthHook {
	return &AuthHook{
		scope:     scope,
		validator: validator,
		activeIDs: make(map[string]struct{}),
	}
}

// Decide validates a MediaMTX HTTP auth request. Denials never echo raw credentials.
func (h *AuthHook) Decide(ctx context.Context, req AuthHookRequest) AuthHookDecision {
	if h == nil || h.validator == nil {
		return deny("auth hook unavailable")
	}
	if req.Action != "publish" {
		return deny("action not allowed")
	}
	if req.Path != h.scope.PathName {
		return deny("path outside lease scope")
	}
	if !supportedPublishProtocol(req.Protocol) {
		return deny("protocol not allowed")
	}
	token := req.Token
	if token == "" {
		token = req.Password
	}
	if token == "" {
		return deny("publish token required")
	}

	claims, err := h.validator.ValidatePublishToken(ctx, token)
	if err != nil {
		return deny("publish token rejected")
	}
	if !claims.ExpiresAt.IsZero() && !claims.ExpiresAt.After(time.Now()) {
		return deny("publish token expired")
	}
	if claims.OrgID != h.scope.OrgID || claims.PoolID != h.scope.PoolID {
		return deny("publish token outside lease scope")
	}
	if !h.reserveConnection(req.ID) {
		return deny("connection cap reached")
	}
	return AuthHookDecision{Allowed: true, StatusCode: 204}
}

func (h *AuthHook) reserveConnection(id string) bool {
	if h.scope.MaxConnections <= 0 {
		return true
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if id != "" {
		if _, ok := h.activeIDs[id]; ok {
			return true
		}
	}
	if len(h.activeIDs) >= h.scope.MaxConnections {
		return false
	}
	if id == "" {
		h.anonymous++
		id = fmt.Sprintf("anonymous-%d", h.anonymous)
	}
	h.activeIDs[id] = struct{}{}
	return true
}

func deny(reason string) AuthHookDecision {
	return AuthHookDecision{
		Allowed:    false,
		StatusCode: 403,
		Reason:     reason + " (creds redacted)",
	}
}

func supportedPublishProtocol(protocol string) bool {
	switch protocol {
	case "rtmp", "srt", "webrtc", "whip":
		return true
	default:
		return false
	}
}

func escapePath(name string) string {
	segments := strings.Split(name, "/")
	for i, segment := range segments {
		segments[i] = url.PathEscape(segment)
	}
	return strings.Join(segments, "/")
}
