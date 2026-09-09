package totp

import (
	"strings"
	"testing"
	"time"
)

func TestTOTPFlow(t *testing.T) {
	// 1. Generate Secret
	secret, err := GenerateSecret(20)
	if err != nil {
		t.Fatalf("GenerateSecret failed: %v", err)
	}
	if len(secret) == 0 {
		t.Fatalf("Expected non-empty secret")
	}

	// 2. Generate Passcode
	now := time.Now()
	code, err := GeneratePasscode(secret, now)
	if err != nil {
		t.Fatalf("GeneratePasscode failed: %v", err)
	}
	if len(code) != 6 {
		t.Fatalf("Expected 6-digit code, got %s", code)
	}

	// 3. Validate Passcode immediately
	if !ValidatePasscode(secret, code, 1) {
		t.Fatalf("ValidatePasscode failed for current code: %s", code)
	}

	// 4. Validate with incorrect code
	if ValidatePasscode(secret, "000000", 1) && code != "000000" {
		t.Fatalf("ValidatePasscode should fail for invalid code")
	}

	// 5. Test skew tolerance (within 30s drift)
	past30s := now.Add(-25 * time.Second)
	pastCode, _ := GeneratePasscode(secret, past30s)
	if !ValidatePasscode(secret, pastCode, 1) {
		t.Fatalf("ValidatePasscode should accept code within 1 skew period")
	}

	// 6. Test beyond skew tolerance (e.g. 5 minutes ago)
	wayPast := now.Add(-300 * time.Second)
	wayPastCode, _ := GeneratePasscode(secret, wayPast)
	if ValidatePasscode(secret, wayPastCode, 1) {
		t.Fatalf("ValidatePasscode should reject code from 5 minutes ago")
	}

	// 7. Test QR Data URL generation
	qrDataURL, err := GenerateQRCodeDataURL("testadmin", secret, "SiPenDosa")
	if err != nil {
		t.Fatalf("GenerateQRCodeDataURL failed: %v", err)
	}
	if !strings.HasPrefix(qrDataURL, "data:image/png;base64,") {
		t.Fatalf("Expected base64 data URL, got prefix %s", qrDataURL[:20])
	}
}
