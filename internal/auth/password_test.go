package auth

import (
	"strings"
	"testing"
)

func TestHashPasswordFormat(t *testing.T) {
	hash, err := HashPassword("correct horse battery staple")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	if !strings.HasPrefix(hash, "$argon2id$v=19$m=") {
		t.Fatalf("unexpected PHC prefix: %s", hash)
	}
	if strings.Contains(hash, "correct") {
		t.Fatal("password material leaked into hash string")
	}
}

func TestVerifyPasswordRoundtrip(t *testing.T) {
	hash, err := HashPassword("s3cret-pass")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	if err := VerifyPassword("s3cret-pass", hash); err != nil {
		t.Fatalf("verify correct: %v", err)
	}
	if err := VerifyPassword("wrong-pass", hash); err == nil {
		t.Fatal("expected mismatch error for wrong password")
	}
}

func TestVerifyPasswordRejectsMalformed(t *testing.T) {
	for _, bad := range []string{"", "$argon2id$...", "$bcrypt$x$x$x$x"} {
		if err := VerifyPassword("x", bad); err == nil {
			t.Errorf("expected error for %q", bad)
		}
	}
}

func TestSaltUniquePerHash(t *testing.T) {
	a, _ := HashPassword("same-password")
	b, _ := HashPassword("same-password")
	if a == b {
		t.Fatal("two hashes of the same password must differ (unique salts)")
	}
}
