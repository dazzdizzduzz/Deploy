package auth

import (
	"crypto/rand"
	"crypto/rsa"
	"net/http"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/tiers-undiknas/capstone-group07-2026/backend/internal/models"
)

func generateTestKeyPair(t *testing.T) (*rsa.PrivateKey, *rsa.PublicKey) {
	t.Helper()
	privKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate test RSA key: %v", err)
	}
	return privKey, &privKey.PublicKey
}

func TestJWTManager_GenerateAndValidateToken(t *testing.T) {
	manager, err := NewJWTManager("", "")
	if err != nil {
		t.Fatalf("Failed to create JWTManager: %v", err)
	}

	userID := "d3eebc99-9c0b-4ef8-bb6d-6bb9bd380a44"
	tenantID := "a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11"
	role := "owner"

	tokenString, expiresIn, err := manager.GenerateToken(userID, tenantID, role)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	if tokenString == "" {
		t.Error("Expected non-empty token")
	}

	if expiresIn != int((15 * time.Minute).Seconds()) {
		t.Errorf("Expected expires_in 900, got %d", expiresIn)
	}

	claims, err := manager.ValidateToken(tokenString)
	if err != nil {
		t.Fatalf("Failed to validate token: %v", err)
	}

	if claims.Subject != userID {
		t.Errorf("Expected sub %s, got %s", userID, claims.Subject)
	}
	if claims.Tid != tenantID {
		t.Errorf("Expected tid %s, got %s", tenantID, claims.Tid)
	}
	if string(claims.Role) != role {
		t.Errorf("Expected role %s, got %s", role, claims.Role)
	}
}

func TestJWTService_Success(t *testing.T) {
	privKey, pubKey := generateTestKeyPair(t)
	svc := NewJWTServiceWithKeys(pubKey, privKey)

	claims := models.JWTClaims{
		Sub:  "user-uuid-1234",
		Tid:  "tenant-uuid-5678",
		Role: models.RoleOwner,
		Eid:  "emp-001",
		Mods: []string{"attendance", "payroll", "hr"},
	}

	tokenStr, err := svc.GenerateTokenWithClaims(claims, 15*time.Minute)
	if err != nil {
		t.Fatalf("expected token generation to succeed, got %v", err)
	}

	parsedClaims, err := svc.ValidateToken(tokenStr)
	if err != nil {
		t.Fatalf("expected token validation to succeed, got %v", err)
	}

	if parsedClaims.Sub != claims.Sub {
		t.Errorf("expected sub %s, got %s", claims.Sub, parsedClaims.Sub)
	}
	if parsedClaims.Tid != claims.Tid {
		t.Errorf("expected tid %s, got %s", claims.Tid, parsedClaims.Tid)
	}
	if parsedClaims.Role != claims.Role {
		t.Errorf("expected role %s, got %s", claims.Role, parsedClaims.Role)
	}
	if parsedClaims.Eid != claims.Eid {
		t.Errorf("expected eid %s, got %s", claims.Eid, parsedClaims.Eid)
	}
	if len(parsedClaims.Mods) != 3 {
		t.Errorf("expected 3 mods, got %d", len(parsedClaims.Mods))
	}
}

func TestJWTService_ExpiredToken(t *testing.T) {
	privKey, pubKey := generateTestKeyPair(t)
	svc := NewJWTServiceWithKeys(pubKey, privKey)

	claims := models.JWTClaims{
		Sub:  "user-expired",
		Tid:  "tenant-test",
		Role: models.RoleEmployee,
	}

	tokenStr, err := svc.GenerateTokenWithClaims(claims, -1*time.Minute)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	_, err = svc.ValidateToken(tokenStr)
	if err == nil {
		t.Fatalf("expected expired token to fail validation")
	}
}

func TestJWTService_MismatchedKey(t *testing.T) {
	privKey1, _ := generateTestKeyPair(t)
	_, pubKey2 := generateTestKeyPair(t)

	svcSigner := NewJWTServiceWithKeys(nil, privKey1)
	svcVerifier := NewJWTServiceWithKeys(pubKey2, nil)

	claims := models.JWTClaims{
		Sub:  "user-test",
		Tid:  "tenant-test",
		Role: models.RoleHRD,
	}

	tokenStr, err := svcSigner.GenerateTokenWithClaims(claims, 15*time.Minute)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	_, err = svcVerifier.ValidateToken(tokenStr)
	if err == nil {
		t.Fatalf("expected validation with wrong public key to fail")
	}
}

func TestJWTService_RejectHS256(t *testing.T) {
	_, pubKey := generateTestKeyPair(t)
	svc := NewJWTServiceWithKeys(pubKey, nil)

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": "attacker",
		"tid": "tenant-test",
	})
	tokenStr, err := token.SignedString([]byte("secret"))
	if err != nil {
		t.Fatalf("failed to sign HMAC token: %v", err)
	}

	_, err = svc.ValidateToken(tokenStr)
	if err == nil {
		t.Fatalf("expected HS256 token to be rejected by RS256 validator")
	}
}

func TestExtractBearerToken(t *testing.T) {
	req, _ := http.NewRequest("GET", "/ping", nil)

	_, err := ExtractBearerToken(req)
	if err != ErrMissingAuthHeader {
		t.Errorf("expected ErrMissingAuthHeader, got %v", err)
	}

	req.Header.Set("Authorization", "Basic 12345")
	_, err = ExtractBearerToken(req)
	if err != ErrInvalidAuthFormat {
		t.Errorf("expected ErrInvalidAuthFormat, got %v", err)
	}

	req.Header.Set("Authorization", "Bearer valid-token-xyz")
	token, err := ExtractBearerToken(req)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if token != "valid-token-xyz" {
		t.Errorf("expected 'valid-token-xyz', got '%s'", token)
	}
}
