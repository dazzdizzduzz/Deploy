package auth

import (
	"testing"
)

func TestPasswordResetManager_GenerateAndConsume(t *testing.T) {
	mgr := NewPasswordResetManager()
	userID := "user-123"
	email := "test@ontrack.id"

	token, err := mgr.GenerateToken(userID, email)
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}

	if len(token) != 64 {
		t.Errorf("Expected 64-character hex token, got length %d", len(token))
	}

	// Konsumsi token pertama kali -> harus berhasil
	consumedUserID, consumedEmail, err := mgr.ValidateAndConsumeToken(token)
	if err != nil {
		t.Fatalf("ValidateAndConsumeToken failed: %v", err)
	}

	if consumedUserID != userID {
		t.Errorf("Expected userID %s, got %s", userID, consumedUserID)
	}
	if consumedEmail != email {
		t.Errorf("Expected email %s, got %s", email, consumedEmail)
	}

	// Konsumsi kedua kali (token yang sama) -> harus gagal (single-use)
	_, _, err = mgr.ValidateAndConsumeToken(token)
	if err == nil {
		t.Error("Expected error when consuming already used token, got nil")
	}

	// Validasi token non-existent -> harus gagal
	_, _, err = mgr.ValidateAndConsumeToken("random-invalid-token")
	if err == nil {
		t.Error("Expected error for non-existent token, got nil")
	}
}

