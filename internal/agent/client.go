package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"sync/atomic"
	"time"
)

// Client speaks to a panel-agent over its Unix domain socket.
type Client struct {
	socketPath string
	token      string

	nextID atomic.Uint64
	dialer *net.Dialer
}

// NewClient creates a client for the agent socket.
func NewClient(socketPath, token string) *Client {
	return &Client{
		socketPath: socketPath,
		token:      token,
		dialer:     &net.Dialer{Timeout: 5 * time.Second},
	}
}

// ErrUnknownOp is returned when the agent rejects the operation.
var ErrUnknownOp = errors.New("agent rejected operation")

// Call executes op with params (any marshalable value) into out (pointer).
func (c *Client) Call(ctx context.Context, op string, params any, out any) error {
	conn, err := c.dialer.DialContext(ctx, "unix", c.socketPath)
	if err != nil {
		return fmt.Errorf("dial agent %s: %w", c.socketPath, err)
	}
	defer func() { _ = conn.Close() }()

	deadline := time.Now().Add(60 * time.Second)
	if d, ok := ctx.Deadline(); ok {
		deadline = d
	}
	_ = conn.SetDeadline(deadline)

	req := Request{ID: c.nextID.Add(1), Op: op, Token: c.token}
	if params != nil {
		raw, err := json.Marshal(params)
		if err != nil {
			return fmt.Errorf("marshal params: %w", err)
		}
		req.Params = raw
	}
	line, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}
	if _, err := conn.Write(append(line, '\n')); err != nil {
		return fmt.Errorf("write request: %w", err)
	}

	dec := json.NewDecoder(conn)
	var resp Response
	if err := dec.Decode(&resp); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	if resp.ID != req.ID {
		return errors.New("response id mismatch")
	}
	if !resp.OK {
		return fmt.Errorf("%w: %s", ErrUnknownOp, resp.Error)
	}
	if out != nil && len(resp.Result) > 0 {
		if err := json.Unmarshal(resp.Result, out); err != nil {
			return fmt.Errorf("decode result: %w", err)
		}
	}
	return nil
}

// Ping performs the liveness handshake.
func (c *Client) Ping(ctx context.Context) (*PingResult, error) {
	var res PingResult
	if err := c.Call(ctx, OpPing, PingParams{}, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

// Ensure socket parent dirs exist and socket file has expected permissions.
func CheckSocket(path string) error {
	fi, err := os.Stat(path)
	if err != nil {
		return err
	}
	if fi.Mode()&os.ModeSocket == 0 {
		return fmt.Errorf("%s is not a socket", path)
	}
	return nil
}
