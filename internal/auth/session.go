package auth

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"time"
)

// Session lifetime constants.
const (
	DefaultSessionTTL = 12 * time.Hour
	sessionIDBytes    = 32
	csrfTokenBytes    = 24
)

var ErrSessionNotFound = errors.New("auth: session not found")

// Session is an authenticated browser session.
type Session struct {
	ID        string // opaque random identifier; stored hashed? (see note in store)
	UserID    int64
	CSRFToken string
	CreatedAt time.Time
	ExpiresAt time.Time
	IP        string
	UserAgent string
}

// SessionStore persists sessions.
type SessionStore interface {
	CreateSession(ctx context.Context, s *Session) error
	GetSession(ctx context.Context, id string) (*Session, error)
	DeleteSession(ctx context.Context, id string) error
	DeleteExpiredSessions(ctx context.Context) (int64, error)
}

// SessionManager issues and validates sessions.
type SessionManager struct {
	store SessionStore
	ttl   time.Duration
	now   func() time.Time
}

// NewSessionManager constructs a manager over the given store.
func NewSessionManager(store SessionStore, ttl time.Duration) *SessionManager {
	if ttl <= 0 {
		ttl = DefaultSessionTTL
	}
	return &SessionManager{store: store, ttl: ttl, now: time.Now}
}

// randomToken returns url-safe random bytes encoded to base64.
func randomToken(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic(fmt.Sprintf("auth: entropy unavailable: %v", err)) // crypto/rand failure is unrecoverable
	}
	return base64.RawURLEncoding.EncodeToString(b)
}

// Issue creates a fresh session for the user with client metadata.
func (m *SessionManager) Issue(ctx context.Context, userID int64, ip, userAgent string) (*Session, error) {
	s := &Session{
		ID:        randomToken(sessionIDBytes),
		UserID:    userID,
		CSRFToken: randomToken(csrfTokenBytes),
		CreatedAt: m.now(),
		ExpiresAt: m.now().Add(m.ttl),
		IP:        ip,
		UserAgent: userAgent,
	}
	if err := m.store.CreateSession(ctx, s); err != nil {
		return nil, fmt.Errorf("auth: create session: %w", err)
	}
	return s, nil
}

// Validate returns the session when it exists and is unexpired.
func (m *SessionManager) Validate(ctx context.Context, id string) (*Session, error) {
	if id == "" {
		return nil, ErrSessionNotFound
	}
	s, err := m.store.GetSession(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrSessionNotFound
		}
		return nil, err
	}
	if m.now().After(s.ExpiresAt) {
		_ = m.store.DeleteSession(ctx, id)
		return nil, ErrSessionNotFound
	}
	return s, nil
}

// Revoke deletes the session (logout).
func (m *SessionManager) Revoke(ctx context.Context, id string) error {
	return m.store.DeleteSession(ctx, id)
}

// Sweep removes expired sessions; returns rows removed.
func (m *SessionManager) Sweep(ctx context.Context) (int64, error) {
	return m.store.DeleteExpiredSessions(ctx)
}
