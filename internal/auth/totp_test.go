package auth

import (
	"strings"
	"testing"
	"time"
)

func TestTOTPRoundtrip(t *testing.T) {
	secret := "GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ"
	now := time.Now()
	code, err := GenerateCode(secret, now)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if len(code) != 6 {
		t.Fatalf("expected 6-digit code, got %q (len=%d)", code, len(code))
	}
	if err := ValidateTOTPAt(secret, code, now); err != nil {
		t.Fatalf("validate current code: %v", err)
	}
}

func TestTOTPClockDriftAcceptance(t *testing.T) {
	secret := "GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ"
	now := time.Date(2025, 1, 1, 0, 0, 30, 0, time.UTC)
	code, err := GenerateCode(secret, now)
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateTOTPAt(secret, code, now.Add(30*time.Second)); err != nil {
		t.Fatalf("expected skew tolerance, got: %v", err)
	}
}

func TestTOTPWrongCodeRejected(t *testing.T) {
	secret := "GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ"
	now := time.Now()
	code, _ := GenerateCode(secret, now)
	wrong := flipDigit(code)
	if err := ValidateTOTPAt(secret, wrong, now); err == nil {
		t.Error("expected invalid code error")
	}
}

func TestValidateTOTPRejectsBadInput(t *testing.T) {
	secret := "GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ"
	now := time.Now()
	for _, code := range []string{"", "abcdef", "12345", "1234567", "12 456"} {
		if err := ValidateTOTPAt(secret, code, now); err == nil {
			t.Errorf("expected rejection for %q", code)
		}
	}
}

func TestEnrollProducesWorkingSecret(t *testing.T) {
	enr, err := EnrollTOTP("admin@panel.test")
	if err != nil {
		t.Fatalf("enroll: %v", err)
	}
	if len(enr.Secret) < 16 {
		t.Fatalf("secret too short: %q", enr.Secret)
	}
	if !strings.Contains(enr.ProvisioningURI, "otpauth://totp/") {
		t.Fatalf("unexpected provisioning URI: %s", enr.ProvisioningURI)
	}
	code, err := GenerateCode(enr.Secret, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateTOTP(enr.Secret, code); err != nil {
		t.Fatalf("freshly enrolled secret rejected valid code: %v", err)
	}
}

func flipDigit(code string) string {
	b := []byte(code)
	if b[0] == '0' {
		b[0] = '1'
	} else {
		b[0] = '0'
	}
	return string(b)
}
