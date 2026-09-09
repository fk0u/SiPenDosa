package totp

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base32"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"net/url"
	"strings"
	"time"

	"github.com/skip2/go-qrcode"
)

const (
	// DefaultPeriod is the standard TOTP step duration in seconds
	DefaultPeriod = 30
	// DefaultDigits is the standard number of digits for TOTP
	DefaultDigits = 6
)

// GenerateSecret generates a cryptographically random Base32 encoded secret key
func GenerateSecret(byteLength int) (string, error) {
	if byteLength <= 0 {
		byteLength = 20 // 160 bits (recommended for SHA1)
	}

	bytes := make([]byte, byteLength)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}

	// Base32 without padding is standard for authenticator apps
	secret := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(bytes)
	return secret, nil
}

// GeneratePasscode computes the 6-digit TOTP code for a given secret and timestamp
func GeneratePasscode(secret string, t time.Time) (string, error) {
	cleaned := strings.ToUpper(strings.TrimSpace(secret))
	// Add padding if missing for standard base32 decoder
	if rem := len(cleaned) % 8; rem != 0 {
		cleaned += strings.Repeat("=", 8-rem)
	}

	key, err := base32.StdEncoding.DecodeString(cleaned)
	if err != nil {
		return "", fmt.Errorf("invalid base32 secret: %w", err)
	}

	counter := uint64(t.Unix() / DefaultPeriod)
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, counter)

	mac := hmac.New(sha1.New, key)
	mac.Write(buf)
	sum := mac.Sum(nil)

	// Dynamic truncation
	offset := sum[len(sum)-1] & 0x0f
	code := binary.BigEndian.Uint32(sum[offset:offset+4]) & 0x7fffffff

	mod := uint32(math.Pow10(DefaultDigits))
	passcode := code % mod

	return fmt.Sprintf("%0*d", DefaultDigits, passcode), nil
}

// ValidatePasscode validates a given passcode with time drift tolerance
// skew = 1 checks: [current - 1 interval, current, current + 1 interval] (i.e. +/- 30s)
func ValidatePasscode(secret, passcode string, skew int) bool {
	cleanedCode := strings.TrimSpace(passcode)
	if len(cleanedCode) != DefaultDigits {
		return false
	}

	if skew < 0 {
		skew = 1
	}

	now := time.Now()
	for i := -skew; i <= skew; i++ {
		t := now.Add(time.Duration(i*DefaultPeriod) * time.Second)
		expected, err := GeneratePasscode(secret, t)
		if err == nil && hmac.Equal([]byte(expected), []byte(cleanedCode)) {
			return true
		}
	}

	return false
}

// GenerateOTPAuthURL generates standard otpauth:// URL for authenticator apps
func GenerateOTPAuthURL(username, secret, issuer string) string {
	if issuer == "" {
		issuer = "SiPenDosa"
	}
	label := fmt.Sprintf("%s:%s", issuer, username)
	vals := url.Values{}
	vals.Set("secret", secret)
	vals.Set("issuer", issuer)
	vals.Set("algorithm", "SHA1")
	vals.Set("digits", fmt.Sprintf("%d", DefaultDigits))
	vals.Set("period", fmt.Sprintf("%d", DefaultPeriod))

	return fmt.Sprintf("otpauth://totp/%s?%s", url.PathEscape(label), vals.Encode())
}

// GenerateQRCodeDataURL generates a PNG data URL of the QR code for setup in web UI
func GenerateQRCodeDataURL(username, secret, issuer string) (string, error) {
	if secret == "" {
		return "", errors.New("secret cannot be empty")
	}

	authURL := GenerateOTPAuthURL(username, secret, issuer)
	pngBytes, err := qrcode.Encode(authURL, qrcode.Medium, 256)
	if err != nil {
		return "", fmt.Errorf("failed to encode QR code: %w", err)
	}

	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(pngBytes), nil
}
