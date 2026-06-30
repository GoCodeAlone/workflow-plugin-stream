package mediamtx

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClientPathLifecycleUsesMediaMTXAPI(t *testing.T) {
	t.Parallel()

	var seen []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = append(seen, r.Method+" "+r.URL.RequestURI())
		switch r.Method + " " + r.URL.Path {
		case "POST /v3/config/paths/add/live/main":
			var body PathConfig
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode add body: %v", err)
			}
			if body.Source != "publisher" || !body.Record || !body.OverridePublisher {
				t.Fatalf("add body = %#v", body)
			}
			writeOK(w)
		case "GET /v3/paths/list":
			if r.URL.Query().Get("page") != "0" || r.URL.Query().Get("itemsPerPage") != "100" {
				t.Fatalf("list query = %q", r.URL.RawQuery)
			}
			_ = json.NewEncoder(w).Encode(pathList{
				Items: []Path{{Name: "live/main", Available: true, InboundBytes: 4096}},
			})
		case "DELETE /v3/config/paths/delete/live/main":
			writeOK(w)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.RequestURI())
		}
	}))
	defer server.Close()

	client, err := NewClient(server.URL)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	ctx := context.Background()

	if err := client.AddPath(ctx, "live/main", PathConfig{
		Source:            "publisher",
		Record:            true,
		OverridePublisher: true,
	}); err != nil {
		t.Fatalf("AddPath: %v", err)
	}
	paths, err := client.ListPaths(ctx)
	if err != nil {
		t.Fatalf("ListPaths: %v", err)
	}
	if len(paths) != 1 || paths[0].Name != "live/main" || paths[0].InboundBytes != 4096 {
		t.Fatalf("paths = %#v", paths)
	}
	if err := client.RemovePath(ctx, "live/main"); err != nil {
		t.Fatalf("RemovePath: %v", err)
	}

	want := []string{
		"POST /v3/config/paths/add/live/main",
		"GET /v3/paths/list?page=0&itemsPerPage=100",
		"DELETE /v3/config/paths/delete/live/main",
	}
	if len(seen) != len(want) {
		t.Fatalf("requests = %#v, want %#v", seen, want)
	}
	for i := range want {
		if seen[i] != want[i] {
			t.Fatalf("request[%d] = %q, want %q", i, seen[i], want[i])
		}
	}
}

func TestClientStartStopWrapPathLifecycle(t *testing.T) {
	t.Parallel()

	var seen []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = append(seen, r.Method+" "+r.URL.Path)
		writeOK(w)
	}))
	defer server.Close()

	client, err := NewClient(server.URL)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	if err := client.Start(context.Background(), "live/session", StartOptions{Record: true}); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if err := client.Stop(context.Background(), "live/session"); err != nil {
		t.Fatalf("Stop: %v", err)
	}

	want := []string{
		"POST /v3/config/paths/add/live/session",
		"DELETE /v3/config/paths/delete/live/session",
	}
	for i := range want {
		if seen[i] != want[i] {
			t.Fatalf("request[%d] = %q, want %q", i, seen[i], want[i])
		}
	}
}

func TestClientReturnsTypedAPIError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(apiErrorResponse{Status: "error", Error: "invalid path"})
	}))
	defer server.Close()

	client, err := NewClient(server.URL)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	err = client.RemovePath(context.Background(), "bad path")
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("error = %T %[1]v, want *APIError", err)
	}
	if apiErr.StatusCode != http.StatusBadRequest || apiErr.Message != "invalid path" {
		t.Fatalf("api error = %#v", apiErr)
	}
}

func writeOK(w http.ResponseWriter) {
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
