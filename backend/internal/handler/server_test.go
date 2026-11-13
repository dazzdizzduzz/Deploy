package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

	"github.com/tiers-undiknas/capstone-group07-2026/backend/api"
	"github.com/tiers-undiknas/capstone-group07-2026/backend/internal/auth"
	"github.com/tiers-undiknas/capstone-group07-2026/backend/internal/repository"
)

type mockUserRepo struct {
	users map[string]*repository.User
}

func (m *mockUserRepo) FindByEmail(email string) (*repository.User, error) {
	return m.users[email], nil
}

func (m *mockUserRepo) UpdatePasswordHash(userID, passwordHash string) error {
	for _, u := range m.users {
		if u.ID == userID {
			u.PasswordHash = passwordHash
			return nil
		}
	}
	return errors.New("user not found")
}

func setupTestRouter() (*gin.Engine, *auth.PasswordResetManager) {
	gin.SetMode(gin.TestMode)

	hash, _ := bcrypt.GenerateFromPassword([]byte("password123"), 12)

	userRepo := &mockUserRepo{
		users: map[string]*repository.User{
			"owner@ontrack.id": {
				ID:           "d3eebc99-9c0b-4ef8-bb6d-6bb9bd380a44",
				TenantID:     "a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11",
				Email:        "owner@ontrack.id",
				PasswordHash: string(hash),
				FullName:     "Owner OnTrack",
				Role:         "owner",
				IsActive:     true,
			},
		},
	}

	jwtManager, _ := auth.NewJWTManager("", "")
	loginLock := auth.NewLoginLockManager()
	passwordReset := auth.NewPasswordResetManager()
	locRepo := &mockLocationRepo{locations: make(map[string]*repository.OfficeLocation)}
	schedRepo := &mockScheduleRepo{schedules: make(map[string]*repository.WorkSchedule)}
	srv := NewServer(userRepo, locRepo, schedRepo, jwtManager, loginLock, passwordReset)

	r := gin.New()
	api.RegisterHandlers(r, srv)
	return r, passwordReset
}

func TestPostAuthLogin_Success(t *testing.T) {
	r, _ := setupTestRouter()

	body, _ := json.Marshal(map[string]string{
		"email":    "owner@ontrack.id",
		"password": "password123",
	})

	req, _ := http.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK, got %d. Body: %s", w.Code, w.Body.String())
	}

	var resp api.LoginResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if resp.AccessToken == "" {
		t.Error("Expected non-empty access_token")
	}
	if resp.TokenType != "Bearer" {
		t.Errorf("Expected token_type Bearer, got %s", resp.TokenType)
	}
	if resp.ExpiresIn != 900 {
		t.Errorf("Expected expires_in 900, got %d", resp.ExpiresIn)
	}
}

func TestPostAuthLogin_InvalidPassword(t *testing.T) {
	r, _ := setupTestRouter()

	body, _ := json.Marshal(map[string]string{
		"email":    "owner@ontrack.id",
		"password": "wrongpassword",
	})

	req, _ := http.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("Expected 401 Unauthorized, got %d", w.Code)
	}
}

func TestPostAuthLogin_AccountLockoutAfterFiveAttempts(t *testing.T) {
	r, _ := setupTestRouter()

	body, _ := json.Marshal(map[string]string{
		"email":    "owner@ontrack.id",
		"password": "wrongpassword",
	})

	for i := 1; i <= 5; i++ {
		req, _ := http.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if i < 5 && w.Code != http.StatusUnauthorized {
			t.Fatalf("Attempt %d: expected 401, got %d", i, w.Code)
		}
		if i == 5 && w.Code != http.StatusLocked {
			t.Fatalf("Attempt 5: expected 423 Locked, got %d", w.Code)
		}
	}

	validBody, _ := json.Marshal(map[string]string{
		"email":    "owner@ontrack.id",
		"password": "password123",
	})
	req, _ := http.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(validBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusLocked {
		t.Fatalf("Attempt after lock: expected 423 Locked, got %d", w.Code)
	}
}

func TestPostAuthLogout(t *testing.T) {
	r, _ := setupTestRouter()

	req, _ := http.NewRequest(http.MethodPost, "/auth/logout", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK, got %d", w.Code)
	}
}

func TestPostAuthPasswordForgot_Returns202(t *testing.T) {
	r, _ := setupTestRouter()

	// 1. Email terdaftar -> Harus 202 Accepted
	bodyRegistered, _ := json.Marshal(map[string]string{
		"email": "owner@ontrack.id",
	})
	req1, _ := http.NewRequest(http.MethodPost, "/auth/password/forgot", bytes.NewReader(bodyRegistered))
	req1.Header.Set("Content-Type", "application/json")
	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, req1)

	if w1.Code != http.StatusAccepted {
		t.Fatalf("Expected 202 Accepted, got %d", w1.Code)
	}

	// 2. Email TIDAK terdaftar -> Tetap harus 202 Accepted (Anti-User Enumeration)
	bodyUnregistered, _ := json.Marshal(map[string]string{
		"email": "unknown@example.com",
	})
	req2, _ := http.NewRequest(http.MethodPost, "/auth/password/forgot", bytes.NewReader(bodyUnregistered))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	if w2.Code != http.StatusAccepted {
		t.Fatalf("Expected 202 Accepted for unknown email, got %d", w2.Code)
	}
}

func TestPostAuthPasswordReset_SuccessAndSingleUse(t *testing.T) {
	r, passwordReset := setupTestRouter()

	// Buat token reset untuk owner
	token, _ := passwordReset.GenerateToken("d3eebc99-9c0b-4ef8-bb6d-6bb9bd380a44", "owner@ontrack.id")

	// Reset kata sandi baru
	resetBody, _ := json.Marshal(map[string]string{
		"token":        token,
		"new_password": "NewSecretPassword456!",
	})
	req, _ := http.NewRequest(http.MethodPost, "/auth/password/reset", bytes.NewReader(resetBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK, got %d. Body: %s", w.Code, w.Body.String())
	}

	// Coba login dengan password LAMA -> Harus 401 Unauthorized
	oldLoginBody, _ := json.Marshal(map[string]string{
		"email":    "owner@ontrack.id",
		"password": "password123",
	})
	reqOld, _ := http.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(oldLoginBody))
	reqOld.Header.Set("Content-Type", "application/json")
	wOld := httptest.NewRecorder()
	r.ServeHTTP(wOld, reqOld)

	if wOld.Code != http.StatusUnauthorized {
		t.Fatalf("Expected 401 Unauthorized with old password, got %d", wOld.Code)
	}

	// Coba login dengan password BARU -> Harus 200 OK
	newLoginBody, _ := json.Marshal(map[string]string{
		"email":    "owner@ontrack.id",
		"password": "NewSecretPassword456!",
	})
	reqNew, _ := http.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(newLoginBody))
	reqNew.Header.Set("Content-Type", "application/json")
	wNew := httptest.NewRecorder()
	r.ServeHTTP(wNew, reqNew)

	if wNew.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK with new password, got %d", wNew.Code)
	}

	// Coba pakai token yang SAMA lagi -> Harus gagal 400 (Single-use token)
	reqReuse, _ := http.NewRequest(http.MethodPost, "/auth/password/reset", bytes.NewReader(resetBody))
	reqReuse.Header.Set("Content-Type", "application/json")
	wReuse := httptest.NewRecorder()
	r.ServeHTTP(wReuse, reqReuse)

	if wReuse.Code != http.StatusBadRequest {
		t.Fatalf("Expected 400 Bad Request on token reuse, got %d", wReuse.Code)
	}
}
