package agent

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"
)

// startTestAgent boots a real server+client pair over a unix socket.
func startTestAgent(t *testing.T, token string) (*Server, *Client) {
	t.Helper()
	srv := NewServer(filepath.Join(t.TempDir(), "agent.sock"), token, nil)
	srv.Register("ping", func(_ context.Context, _ json.RawMessage) (any, error) {
		return PingResult{Pong: true, Host: "test-host"}, nil
	})
	if err := srv.Listen(); err != nil {
		t.Fatalf("listen: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- srv.Serve(ctx) }()
	t.Cleanup(func() {
		cancel()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Error("agent did not stop in time")
		}
	})
	return srv, NewClient(srv.SocketPath(), token)
}

func TestPingRoundtrip(t *testing.T) {
	_, client := startTestAgent(t, "tok")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	res, err := client.Ping(ctx)
	if err != nil {
		t.Fatalf("ping: %v", err)
	}
	if !res.Pong || res.Host != "test-host" {
		t.Fatalf("unexpected result %+v", res)
	}
}

func TestUnknownOpRejected(t *testing.T) {
	_, client := startTestAgent(t, "tok")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var out map[string]any
	err := client.Call(ctx, "rm_rf_everything", nil, &out)
	if err == nil {
		t.Fatal("expected rejection of non-whitelisted op")
	}
}

func TestWrongTokenRejected(t *testing.T) {
	srv, _ := startTestAgent(t, "correct-token")
	bad := NewClient(srv.SocketPath(), "wrong-token")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := bad.Ping(ctx); err == nil {
		t.Fatal("expected auth failure")
	}
}

func TestEmptyTokenServerRefusesAll(t *testing.T) {
	srv, client := startTestAgent(t, "")
	// Server constructed with empty token must refuse calls even if op exists.
	srv.Register("ping", func(_ context.Context, _ json.RawMessage) (any, error) {
		return PingResult{Pong: true}, nil
	})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := client.Ping(ctx); err == nil {
		t.Fatal("expected refusal when agent token is empty")
	}
}

func TestListenRejectsNonSocketFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "plain.txt")
	if err := writeFile(path, []byte("hello")); err != nil {
		t.Fatal(err)
	}
	srv := NewServer(path, "tok", nil)
	if err := srv.Listen(); err == nil {
		t.Fatal("expected error when path exists and is not a socket")
	}
}
