package auth_test

import (
	"path/filepath"
	"testing"
	"time"

	"sipen/internal/auth"
	"sipen/internal/store"
	"sipen/internal/totp"
)

func setupTestStore(t *testing.T) (*store.Store, *auth.Service) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_auth.db")

	s, err := store.New(dbPath)
	if err != nil {
		t.Fatalf("Failed to initialize test store: %v", err)
	}

	authSvc := auth.NewService(s)
	return s, authSvc
}

func TestAuthFlowWith2FA(t *testing.T) {
	s, authSvc := setupTestStore(t)
	defer s.Close()

	// 1. First user registration (SuperAdmin)
	superAdmin, err := authSvc.RegisterFirstUser("superadmin", "secret12345")
	if err != nil {
		t.Fatalf("Failed to register first user: %v", err)
	}
	if superAdmin.Role != "superadmin" {
		t.Errorf("Expected role superadmin, got %s", superAdmin.Role)
	}

	// 2. Authenticate without 2FA
	user, err := authSvc.Authenticate("superadmin", "secret12345")
	if err != nil {
		t.Fatalf("Authenticate failed: %v", err)
	}
	if user.TwoFactorEnabled {
		t.Errorf("Expected 2FA to be initially disabled")
	}

	// 3. Enable 2FA for this user
	totpSecret, err := totp.GenerateSecret(20)
	if err != nil {
		t.Fatalf("Failed to generate TOTP secret: %v", err)
	}
	err = s.UpdateUser2FA(superAdmin.ID, totpSecret, true)
	if err != nil {
		t.Fatalf("Failed to update user 2FA in store: %v", err)
	}

	// Reload user and verify 2FA enabled
	updatedUser, err := s.GetUserByID(superAdmin.ID)
	if err != nil || !updatedUser.TwoFactorEnabled {
		t.Fatalf("Expected 2FA to be enabled on reloaded user")
	}

	// 4. Create 2FA Challenge
	challengeID, err := authSvc.Create2FAChallenge(superAdmin.ID)
	if err != nil {
		t.Fatalf("Failed to create 2FA challenge: %v", err)
	}
	if len(challengeID) < 16 {
		t.Errorf("Challenge ID too short: %s", challengeID)
	}

	// 5. Test Verify2FA with wrong code
	_, err = authSvc.Verify2FA(challengeID, "000000")
	if err == nil {
		t.Errorf("Expected error verifying with bogus code")
	}

	// 6. Test Verify2FA with valid code
	validCode, err := totp.GeneratePasscode(totpSecret, time.Now())
	if err != nil {
		t.Fatalf("Failed to generate valid passcode: %v", err)
	}

	verifiedUser, err := authSvc.Verify2FA(challengeID, validCode)
	if err != nil {
		t.Fatalf("Verify2FA failed with valid passcode: %v", err)
	}
	if verifiedUser.ID != superAdmin.ID {
		t.Errorf("Expected user ID %d, got %d", superAdmin.ID, verifiedUser.ID)
	}

	// 7. Verify challenge cannot be reused
	_, err = authSvc.Verify2FA(challengeID, validCode)
	if err == nil {
		t.Errorf("Expected challenge to be single-use only")
	}

	// 8. Admin creates a standard user
	normalUser, err := authSvc.RegisterUser("dosen1", "dosenpassword", "user", true)
	if err != nil {
		t.Fatalf("Failed to register normal user: %v", err)
	}
	if normalUser.Role != "user" {
		t.Errorf("Expected role 'user', got %s", normalUser.Role)
	}
}
