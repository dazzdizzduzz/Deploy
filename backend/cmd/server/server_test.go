package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/tiers-undiknas/capstone-group07-2026/backend/api"
	"github.com/tiers-undiknas/capstone-group07-2026/backend/internal/audit"
	"github.com/tiers-undiknas/capstone-group07-2026/backend/internal/auth"
	"github.com/tiers-undiknas/capstone-group07-2026/backend/internal/middleware"
	"github.com/tiers-undiknas/capstone-group07-2026/backend/internal/models"
)

type mockServer struct{}

func (m *mockServer) GetHealth(c *gin.Context) {
	c.JSON(http.StatusOK, api.HealthResponse{
		Status: "ok",
	})
}

func (m *mockServer) PostAuthLogin(c *gin.Context) {
	c.JSON(http.StatusOK, api.LoginResponse{
		AccessToken: "test-token",
		ExpiresIn:   900,
		TokenType:   "Bearer",
	})
}

func (m *mockServer) PostAuthLogout(c *gin.Context) {
	c.JSON(http.StatusOK, api.MessageResponse{
		Message: "Logout berhasil",
	})
}

func (m *mockServer) PostAuthPasswordForgot(c *gin.Context) {
	c.JSON(http.StatusAccepted, api.MessageResponse{
		Message: "Password reset request accepted",
	})
}

func (m *mockServer) PostAuthPasswordReset(c *gin.Context) {
	c.JSON(http.StatusOK, api.MessageResponse{
		Message: "Password reset successful",
	})
}

func (m *mockServer) GetLocations(c *gin.Context) {
	c.Status(http.StatusOK)
}

func (m *mockServer) PostLocations(c *gin.Context) {
	c.Status(http.StatusCreated)
}

func (m *mockServer) DeleteLocationById(c *gin.Context, id openapi_types.UUID) {
	c.Status(http.StatusNoContent)
}

func (m *mockServer) GetLocationById(c *gin.Context, id openapi_types.UUID) {
	c.Status(http.StatusOK)
}

func (m *mockServer) PatchLocationById(c *gin.Context, id openapi_types.UUID) {
	c.Status(http.StatusOK)
}

func (m *mockServer) PutLocationById(c *gin.Context, id openapi_types.UUID) {
	c.Status(http.StatusOK)
}

func (m *mockServer) GetSchedules(c *gin.Context) {
	c.Status(http.StatusOK)
}

func (m *mockServer) PostSchedules(c *gin.Context) {
	c.Status(http.StatusCreated)
}

func (m *mockServer) DeleteScheduleById(c *gin.Context, id openapi_types.UUID) {
	c.Status(http.StatusNoContent)
}

func (m *mockServer) GetScheduleById(c *gin.Context, id openapi_types.UUID) {
	c.Status(http.StatusOK)
}

func (m *mockServer) PatchScheduleById(c *gin.Context, id openapi_types.UUID) {
	c.Status(http.StatusOK)
}

func (m *mockServer) PutScheduleById(c *gin.Context, id openapi_types.UUID) {
	c.Status(http.StatusOK)
}

// Absensi (ISSUE-62/64)
func (m *mockServer) PostAttendanceClockIn(c *gin.Context) {
	c.Status(http.StatusCreated)
}

func (m *mockServer) PostAttendanceClockOut(c *gin.Context) {
	c.Status(http.StatusOK)
}

// Karyawan (ISSUE-26)
func (m *mockServer) GetUsers(c *gin.Context, params api.GetUsersParams) {
	c.Status(http.StatusOK)
}

func (m *mockServer) PostUsers(c *gin.Context) {
	c.Status(http.StatusCreated)
}

func (m *mockServer) GetUserById(c *gin.Context, id openapi_types.UUID) {
	c.Status(http.StatusOK)
}

func (m *mockServer) PatchUserById(c *gin.Context, id openapi_types.UUID) {
	c.Status(http.StatusOK)
}

func (m *mockServer) PostUserDeactivate(c *gin.Context, id openapi_types.UUID) {
	c.Status(http.StatusOK)
}

func (m *mockServer) PostUserReactivate(c *gin.Context, id openapi_types.UUID) {
	c.Status(http.StatusOK)
}

// Lembur (ISSUE-66)
func (m *mockServer) GetOvertimeRequests(c *gin.Context, params api.GetOvertimeRequestsParams) {
	c.Status(http.StatusOK)
}

func (m *mockServer) PostOvertimeRequest(c *gin.Context) {
	c.Status(http.StatusCreated)
}

func (m *mockServer) GetOvertimeRequestById(c *gin.Context, id openapi_types.UUID) {
	c.Status(http.StatusOK)
}

func (m *mockServer) PostOvertimeRequestApprove(c *gin.Context, id openapi_types.UUID) {
	c.Status(http.StatusOK)
}

func (m *mockServer) PostOvertimeRequestReject(c *gin.Context, id openapi_types.UUID) {
	c.Status(http.StatusOK)
}

// Lampiran izin (ISSUE-69)
func (m *mockServer) UploadLeaveAttachment(c *gin.Context) {
	c.Status(http.StatusCreated)
}

func setupTestApp(t *testing.T) (*gin.Engine, *auth.JWTService, *audit.MemoryLogger) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	jwtSvc, err := auth.NewJWTService(auth.Config{})
	if err != nil {
		t.Fatalf("failed to create test JWTService: %v", err)
	}

	auditLogger := audit.NewMemoryLogger()

	r := gin.New()
	r.Use(gin.Recovery())

	server := &mockServer{}
	api.RegisterHandlers(r, server)

	// Protected v1 routes
	v1 := r.Group("/v1")
	v1.Use(middleware.Guard(middleware.GuardOptions{
		JWTService:  jwtSvc,
		AuditLogger: auditLogger,
	}))
	{
		v1.GET("/departments", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"departments": []string{"IT", "HRD"}})
		})
		v1.POST("/departments", func(c *gin.Context) {
			c.JSON(http.StatusCreated, gin.H{"message": "created"})
		})
	}

	return r, jwtSvc, auditLogger
}

func TestEndToEnd_PublicAndProtectedRoutes(t *testing.T) {
	r, jwtSvc, auditLogger := setupTestApp(t)

	// 1. Health check is public (200 OK)
	wHealth := httptest.NewRecorder()
	reqHealth, _ := http.NewRequest(http.MethodGet, "/health", nil)
	r.ServeHTTP(wHealth, reqHealth)
	if wHealth.Code != http.StatusOK {
		t.Errorf("expected 200 for /health, got %d", wHealth.Code)
	}

	// 2. Access /v1/departments without token (401 Unauthorized)
	wNoAuth := httptest.NewRecorder()
	reqNoAuth, _ := http.NewRequest(http.MethodGet, "/v1/departments", nil)
	r.ServeHTTP(wNoAuth, reqNoAuth)
	if wNoAuth.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for /v1/departments without token, got %d", wNoAuth.Code)
	}

	// 3. Employee accessing POST /v1/departments (403 Forbidden - only super_admin, owner, hrd allowed)
	empClaims := models.JWTClaims{
		Sub:  "emp-uuid-1",
		Tid:  "tenant-1",
		Role: models.RoleEmployee,
	}
	empToken, _ := jwtSvc.GenerateTokenWithClaims(empClaims, 15*time.Minute)

	wEmp := httptest.NewRecorder()
	reqEmp, _ := http.NewRequest(http.MethodPost, "/v1/departments", bytes.NewBufferString("{}"))
	reqEmp.Header.Set("Authorization", "Bearer "+empToken)
	r.ServeHTTP(wEmp, reqEmp)

	if wEmp.Code != http.StatusForbidden {
		t.Errorf("expected 403 Forbidden for employee on POST /v1/departments, got %d", wEmp.Code)
	}

	// Check audit log recorded
	entries := auditLogger.GetEntries()
	if len(entries) != 1 {
		t.Fatalf("expected 1 audit log entry, got %d", len(entries))
	}
	if entries[0].Role != string(models.RoleEmployee) {
		t.Errorf("expected audit role employee, got %s", entries[0].Role)
	}
	if entries[0].Action != "ACCESS_DENIED_RBAC" {
		t.Errorf("expected audit action ACCESS_DENIED_RBAC, got %s", entries[0].Action)
	}

	// 4. Owner accessing POST /v1/departments (201 Created)
	ownerClaims := models.JWTClaims{
		Sub:  "owner-uuid-1",
		Tid:  "tenant-1",
		Role: models.RoleOwner,
	}
	ownerToken, _ := jwtSvc.GenerateTokenWithClaims(ownerClaims, 15*time.Minute)

	wOwner := httptest.NewRecorder()
	reqOwner, _ := http.NewRequest(http.MethodPost, "/v1/departments", bytes.NewBufferString("{}"))
	reqOwner.Header.Set("Authorization", "Bearer "+ownerToken)
	r.ServeHTTP(wOwner, reqOwner)

	if wOwner.Code != http.StatusCreated {
		t.Errorf("expected 201 Created for owner on POST /v1/departments, got %d", wOwner.Code)
	}
}

