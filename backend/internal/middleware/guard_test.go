package middleware

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/tiers-undiknas/capstone-group07-2026/backend/internal/audit"
	"github.com/tiers-undiknas/capstone-group07-2026/backend/internal/auth"
	"github.com/tiers-undiknas/capstone-group07-2026/backend/internal/models"
)

func setupTestGuard(t *testing.T) (*gin.Engine, *auth.JWTService, *audit.MemoryLogger) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	privKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate RSA test key: %v", err)
	}

	jwtSvc := auth.NewJWTServiceWithKeys(&privKey.PublicKey, privKey)
	auditLogger := audit.NewMemoryLogger()

	r := gin.New()
	r.Use(gin.Recovery())

	// Protected v1 group with Guard
	v1 := r.Group("/v1")
	v1.Use(Guard(GuardOptions{
		JWTService:  jwtSvc,
		AuditLogger: auditLogger,
	}))
	{
		// Endpoints defined in rbac.gen.go
		v1.GET("/me", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok", "user_id": c.GetString("user_id")})
		})

		v1.GET("/employees", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok", "role": c.GetString("role")})
		})

		v1.POST("/employees", func(c *gin.Context) {
			c.JSON(http.StatusCreated, gin.H{"status": "created"})
		})

		v1.GET("/overtime-requests", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})

		v1.POST("/schedules", func(c *gin.Context) {
			c.JSON(http.StatusCreated, gin.H{"status": "schedule_created"})
		})
	}

	return r, jwtSvc, auditLogger
}

func TestGuard_MissingAuthHeader(t *testing.T) {
	router, _, _ := setupTestGuard(t)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/v1/me", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 Unauthorized, got %d", w.Code)
	}

	var prob models.ProblemDetails
	if err := json.Unmarshal(w.Body.Bytes(), &prob); err != nil {
		t.Fatalf("expected valid RFC 7807 JSON, got error: %v", err)
	}
	if prob.Code != "UNAUTHORIZED" {
		t.Errorf("expected code UNAUTHORIZED, got %s", prob.Code)
	}
}

func TestGuard_InvalidBearerFormat(t *testing.T) {
	router, _, _ := setupTestGuard(t)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/v1/me", nil)
	req.Header.Set("Authorization", "InvalidHeaderFormat")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 Unauthorized, got %d", w.Code)
	}
}

func TestGuard_ExpiredToken(t *testing.T) {
	router, jwtSvc, _ := setupTestGuard(t)

	claims := models.JWTClaims{
		Sub:  "user-1",
		Tid:  "tenant-1",
		Role: models.RoleEmployee,
	}
	expiredToken, _ := jwtSvc.GenerateTokenWithClaims(claims, -1*time.Hour)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/v1/me", nil)
	req.Header.Set("Authorization", "Bearer "+expiredToken)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 Unauthorized, got %d", w.Code)
	}
}

func TestGuard_ValidToken_AllowedRole(t *testing.T) {
	router, jwtSvc, auditLogger := setupTestGuard(t)

	// Owner accessing POST /v1/employees (Allowed: super_admin, owner, hrd)
	claims := models.JWTClaims{
		Sub:  "owner-uuid-123",
		Tid:  "tenant-uuid-456",
		Role: models.RoleOwner,
		Eid:  "EMP-OWNER",
	}
	token, _ := jwtSvc.GenerateTokenWithClaims(claims, 15*time.Minute)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/v1/employees", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected 201 Created for owner on POST /v1/employees, got %d. Body: %s", w.Code, w.Body.String())
	}

	// No audit log should be recorded for authorized access
	if len(auditLogger.GetEntries()) != 0 {
		t.Errorf("expected 0 audit logs for authorized access, got %d", len(auditLogger.GetEntries()))
	}
}

func TestGuard_ForbiddenRole_US11_AC1(t *testing.T) {
	router, jwtSvc, auditLogger := setupTestGuard(t)

	// Employee attempting POST /v1/employees (Allowed: super_admin, owner, hrd)
	claims := models.JWTClaims{
		Sub:  "emp-uuid-789",
		Tid:  "tenant-uuid-456",
		Role: models.RoleEmployee,
		Eid:  "EMP-REGULAR",
	}
	token, _ := jwtSvc.GenerateTokenWithClaims(claims, 15*time.Minute)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/v1/employees", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("User-Agent", "TestClient/1.0")
	router.ServeHTTP(w, req)

	// 1. Must return 403 Forbidden
	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403 Forbidden, got %d", w.Code)
	}

	// 2. Must return RFC 7807 problem details
	var prob models.ProblemDetails
	if err := json.Unmarshal(w.Body.Bytes(), &prob); err != nil {
		t.Fatalf("expected valid RFC 7807 JSON, got error: %v", err)
	}
	if prob.Code != "FORBIDDEN" {
		t.Errorf("expected code FORBIDDEN, got %s", prob.Code)
	}
	if prob.Status != http.StatusForbidden {
		t.Errorf("expected status 403, got %d", prob.Status)
	}

	// 3. Must record Audit Log (US-11 AC-1)
	entries := auditLogger.GetEntries()
	if len(entries) != 1 {
		t.Fatalf("expected 1 audit log entry, got %d", len(entries))
	}

	entry := entries[0]
	if entry.TenantID != claims.Tid {
		t.Errorf("expected tenant_id %s, got %s", claims.Tid, entry.TenantID)
	}
	if entry.UserID != claims.Sub {
		t.Errorf("expected user_id %s, got %s", claims.Sub, entry.UserID)
	}
	if entry.Role != string(models.RoleEmployee) {
		t.Errorf("expected role employee, got %s", entry.Role)
	}
	if entry.Status != http.StatusForbidden {
		t.Errorf("expected status 403, got %d", entry.Status)
	}
	if entry.Action != "ACCESS_DENIED_RBAC" {
		t.Errorf("expected action ACCESS_DENIED_RBAC, got %s", entry.Action)
	}
	if entry.Method != "POST" {
		t.Errorf("expected method POST, got %s", entry.Method)
	}
}

func TestGuard_TeamLeadPermissions(t *testing.T) {
	router, jwtSvc, auditLogger := setupTestGuard(t)

	claims := models.JWTClaims{
		Sub:  "teamlead-uuid-001",
		Tid:  "tenant-uuid-456",
		Role: models.RoleTeamLead,
	}
	token, _ := jwtSvc.GenerateTokenWithClaims(claims, 15*time.Minute)

	// Skenario 1: Team lead boleh melihat overtime requests
	w1 := httptest.NewRecorder()
	req1, _ := http.NewRequest("GET", "/v1/overtime-requests", nil)
	req1.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(w1, req1)

	if w1.Code != http.StatusOK {
		t.Errorf("expected 200 OK for team_lead on GET /v1/overtime-requests, got %d", w1.Code)
	}

	// Skenario 2: Team lead DITOLAK membuat jadwal baru (POST /v1/schedules)
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("POST", "/v1/schedules", nil)
	req2.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(w2, req2)

	if w2.Code != http.StatusForbidden {
		t.Errorf("expected 403 Forbidden for team_lead on POST /v1/schedules, got %d", w2.Code)
	}

	// Verifikasi audit log mencatat percobaan akses ke schedules
	entries := auditLogger.GetEntries()
	if len(entries) != 1 {
		t.Fatalf("expected 1 audit log entry, got %d", len(entries))
	}
	if entries[0].Role != string(models.RoleTeamLead) {
		t.Errorf("expected role team_lead in audit log, got %s", entries[0].Role)
	}
}

func TestGuard_SuperAdminBypass(t *testing.T) {
	router, jwtSvc, _ := setupTestGuard(t)

	// Super admin can access any protected endpoint
	claims := models.JWTClaims{
		Sub:  "superadmin-uuid",
		Tid:  "platform",
		Role: models.RoleSuperAdmin,
	}
	token, _ := jwtSvc.GenerateTokenWithClaims(claims, 15*time.Minute)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/v1/schedules", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected 201 Created for super_admin, got %d", w.Code)
	}
}

func TestRequireRoles(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	auditLogger := audit.NewMemoryLogger()

	r.GET("/admin-only", func(c *gin.Context) {
		c.Set("role", models.RoleEmployee)
	}, RequireRoles(auditLogger, models.RoleOwner, models.RoleHRD), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/admin-only", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403 Forbidden, got %d", w.Code)
	}
}

