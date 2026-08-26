package agent

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"sync"
)

// HandlerFunc executes one whitelisted operation. params is already-validated
// JSON produced by the caller; implementations unmarshal into concrete types.
type HandlerFunc func(ctx context.Context, params json.RawMessage) (any, error)

// Server serves whitelisted operations over a Unix domain socket.
type Server struct {
	socketPath string
	token      string
	log        *slog.Logger

	mu       sync.RWMutex
	handlers map[string]HandlerFunc

	ln net.Listener
	wg sync.WaitGroup
}

// NewServer constructs an agent server bound to socketPath requiring token.
func NewServer(socketPath, token string, log *slog.Logger) *Server {
	if log == nil {
		log = slog.Default()
	}
	return &Server{
		socketPath: socketPath,
		token:      token,
		log:        log,
		handlers:   make(map[string]HandlerFunc),
	}
}

// Register adds op to the whitelist. Registration is idempotent per op.
func (s *Server) Register(op string, h HandlerFunc) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.handlers[op] = h
}

// Listen removes any stale socket file and starts listening.
func (s *Server) Listen() error {
	if err := os.MkdirAll(filepath.Dir(s.socketPath), 0o755); err != nil {
		return fmt.Errorf("create socket dir: %w", err)
	}
	if fi, err := os.Lstat(s.socketPath); err == nil {
		if fi.Mode()&os.ModeSocket == 0 {
			return fmt.Errorf("%s exists and is not a socket", s.socketPath)
		}
		if err := os.Remove(s.socketPath); err != nil {
			return fmt.Errorf("remove stale socket: %w", err)
		}
	}
	ln, err := net.Listen("unix", s.socketPath)
	if err != nil {
		return fmt.Errorf("listen unix: %w", err)
	}
	// Restrict to owner/group; panel user joins the owning group.
	if err := os.Chmod(s.socketPath, 0o660); err != nil {
		_ = ln.Close()
		return fmt.Errorf("chmod socket: %w", err)
	}
	s.ln = ln
	return nil
}

// Serve accepts connections until ctx is cancelled or the listener closes.
func (s *Server) Serve(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	go func() {
		<-ctx.Done()
		_ = s.ln.Close()
	}()
	for {
		conn, err := s.ln.Accept()
		if err != nil {
			if ctx.Err() != nil {
				s.wg.Wait()
				return nil
			}
			return fmt.Errorf("accept: %w", err)
		}
		s.wg.Add(1)
		go func(c net.Conn) {
			defer s.wg.Done()
			defer func() { _ = c.Close() }()
			s.handleConn(ctx, c)
		}(conn)
	}
}

// SocketPath reports the bound socket location (valid after Listen).
func (s *Server) SocketPath() string { return s.socketPath }

func (s *Server) handleConn(ctx context.Context, conn net.Conn) {
	peer := describePeer(conn)
	scanner := bufio.NewScanner(conn)
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	writer := bufio.NewWriter(conn)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		resp := s.dispatch(ctx, line)
		respBytes, err := json.Marshal(resp)
		if err != nil {
			s.log.Error("marshal response", slog.String("peer", peer), slog.Any("err", err))
			continue
		}
		if _, err := writer.Write(append(respBytes, '\n')); err != nil {
			return
		}
		if err := writer.Flush(); err != nil {
			return
		}
	}
}

func (s *Server) dispatch(ctx context.Context, line []byte) Response {
	var req Request
	if err := json.Unmarshal(line, &req); err != nil {
		return Response{OK: false, Error: "malformed request"}
	}
	fail := func(msg string) Response { return Response{ID: req.ID, OK: false, Error: msg} }

	s.mu.RLock()
	h, known := s.handlers[req.Op]
	s.mu.RUnlock()
	if !known {
		return fail(fmt.Sprintf("unknown or non-whitelisted op %q", req.Op))
	}
	if s.token == "" {
		return fail("agent has no token configured; refusing all calls")
	}
	if req.Token != s.token {
		s.log.Warn("auth failure", slog.String("op", req.Op))
		return fail("authentication failed")
	}
	result, err := safeCall(ctx, h, req.Params)
	if err != nil {
		return fail(err.Error())
	}
	raw, err := json.Marshal(result)
	if err != nil {
		return fail("encode result: " + err.Error())
	}
	return Response{ID: req.ID, OK: true, Result: raw}
}

// safeCall converts panics into errors so a buggy handler cannot kill the agent.
func safeCall(ctx context.Context, h HandlerFunc, params json.RawMessage) (result any, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = errors.New("internal handler panic")
		}
	}()
	return h(ctx, params)
}
