package auth

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
)

// TOTP configuration (RFC 6238, SHA-1, 6 digits, 30s period).
const (
	totpPeriod    = uint(30)
	totpDigits    = otp.DigitsSix
	totpSkew      = uint(1) // accept ±30s clock drift
	totpSecretLen = 20
	totpIssuer    = "ServerPanel"
)

var ErrTOTPCodeInvalid = errors.New("auth: invalid TOTP code")

// TOTPEnrollment holds a freshly generated secret awaiting confirmation.
type TOTPEnrollment struct {
	Secret          string `json:"secret"`
	ProvisioningURI string `json:"provisioningUri"`
}

// EnrollTOTP generates a new secret for the user. The secret is stored but the
// factor is not enabled until VerifyTOTPEnrollment succeeds.
func EnrollTOTP(accountName string) (*TOTPEnrollment, error) {
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      totpIssuer,
		AccountName: accountName,
		Period:      totpPeriod,
		Digits:      totpDigits,
		SecretSize:  totpSecretLen,
	})
	if err != nil {
		return nil, fmt.Errorf("auth: generate totp: %w", err)
	}
	return &TOTPEnrollment{
		Secret:          key.Secret(),
		ProvisioningURI: key.URL(),
	}, nil
}

// ValidateTOTP checks a 6-digit code against the secret with skew tolerance.
func ValidateTOTP(secret, code string) error {
	return ValidateTOTPAt(secret, code, time.Now())
}

// ValidateTOTPAt validates a code at a specific time (used by tests with known vectors).
func ValidateTOTPAt(secret, code string, at time.Time) error {
	code = strings.TrimSpace(code)
	if len(code) != 6 || !isAllDigits(code) {
		return ErrTOTPCodeInvalid
	}
	ok, err := totp.ValidateCustom(code, secret, at, totp.ValidateOpts{
		Period:    totpPeriod,
		Skew:      totpSkew,
		Digits:    totpDigits,
		Algorithm: otp.AlgorithmSHA1,
	})
	if err != nil || !ok {
		return ErrTOTPCodeInvalid
	}
	return nil
}

func isAllDigits(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// GenerateCode produces the current valid code for a secret — used only by
// tests and integration harnesses.
func GenerateCode(secret string, at time.Time) (string, error) {
	return totp.GenerateCodeCustom(secret, at, totp.ValidateOpts{
		Period:    totpPeriod,
		Skew:      0,
		Digits:    totpDigits,
		Algorithm: otp.AlgorithmSHA1,
	})
}
