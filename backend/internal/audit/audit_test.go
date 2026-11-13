package audit

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/tiers-undiknas/capstone-group07-2026/backend/internal/models"
)

func TestMemoryAuditLogger(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger := NewMemoryLogger()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req, _ := http.NewRequest("POST", "/v1/employees", nil)
	req.Header.Set("User-Agent", "TestAgent/1.0")
	c.Request = req

	claims := &models.JWTClaims{
		Sub:  "user-uuid-999",
		Tid:  "tenant-uuid-111",
		Role: models.RoleEmployee,
	}

	logger.LogAccessDenied(c, claims, "/v1/employees", []string{"super_admin", "owner", "hrd"})

	entries := logger.GetEntries()
	if len(entries) != 1 {
		t.Fatalf("expected 1 audit entry, got %d", len(entries))
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
	if entry.Action != "ACCESS_DENIED_RBAC" {
		t.Errorf("expected action ACCESS_DENIED_RBAC, got %s", entry.Action)
	}
	if entry.Status != http.StatusForbidden {
		t.Errorf("expected status 403, got %d", entry.Status)
	}
	if entry.Method != "POST" {
		t.Errorf("expected method POST, got %s", entry.Method)
	}
	if entry.Resource != "/v1/employees" {
		t.Errorf("expected resource /v1/employees, got %s", entry.Resource)
	}
}

