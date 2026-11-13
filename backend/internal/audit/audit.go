package audit

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/tiers-undiknas/capstone-group07-2026/backend/internal/models"
)

// AuditEntry represents a structured audit log entry (US-11 AC-1).
type AuditEntry struct {
	ID        string    `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	TenantID  string    `json:"tenant_id"`
	UserID    string    `json:"user_id"`
	Role      string    `json:"role"`
	Action    string    `json:"action"`
	Resource  string    `json:"resource"`
	Method    string    `json:"method"`
	Status    int       `json:"status"`
	ClientIP  string    `json:"client_ip"`
	UserAgent string    `json:"user_agent"`
	Details   string    `json:"details"`
}

// Logger defines the interface for recording audit events.
type Logger interface {
	LogAccessDenied(c *gin.Context, claims *models.JWTClaims, resource string, requiredRoles []string)
	LogEvent(entry AuditEntry)
}

// ConsoleLogger outputs structured JSON audit logs to stdout/stderr.
type ConsoleLogger struct{}

func NewConsoleLogger() *ConsoleLogger {
	return &ConsoleLogger{}
}

func (l *ConsoleLogger) LogEvent(entry AuditEntry) {
	bytes, err := json.Marshal(entry)
	if err != nil {
		log.Printf("[AUDIT LOG ERROR] failed to serialize audit entry: %v", err)
		return
	}
	log.Printf("[AUDIT LOG] %s", string(bytes))
}

func (l *ConsoleLogger) LogAccessDenied(c *gin.Context, claims *models.JWTClaims, resource string, requiredRoles []string) {
	entry := BuildAccessDeniedEntry(c, claims, resource, requiredRoles)
	l.LogEvent(entry)
}

// MemoryLogger stores audit entries in memory for inspection and testing.
type MemoryLogger struct {
	mu      sync.RWMutex
	entries []AuditEntry
}

func NewMemoryLogger() *MemoryLogger {
	return &MemoryLogger{
		entries: make([]AuditEntry, 0),
	}
}

func (m *MemoryLogger) LogEvent(entry AuditEntry) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.entries = append(m.entries, entry)
}

func (m *MemoryLogger) LogAccessDenied(c *gin.Context, claims *models.JWTClaims, resource string, requiredRoles []string) {
	entry := BuildAccessDeniedEntry(c, claims, resource, requiredRoles)
	m.LogEvent(entry)
}

func (m *MemoryLogger) GetEntries() []AuditEntry {
	m.mu.RLock()
	defer m.mu.RUnlock()
	res := make([]AuditEntry, len(m.entries))
	copy(res, m.entries)
	return res
}

func (m *MemoryLogger) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.entries = make([]AuditEntry, 0)
}

// CompositeLogger broadcasts audit entries to multiple loggers.
type CompositeLogger struct {
	loggers []Logger
}

func NewCompositeLogger(loggers ...Logger) *CompositeLogger {
	return &CompositeLogger{loggers: loggers}
}

func (c *CompositeLogger) LogEvent(entry AuditEntry) {
	for _, l := range c.loggers {
		l.LogEvent(entry)
	}
}

func (c *CompositeLogger) LogAccessDenied(ctx *gin.Context, claims *models.JWTClaims, resource string, requiredRoles []string) {
	for _, l := range c.loggers {
		l.LogAccessDenied(ctx, claims, resource, requiredRoles)
	}
}

// BuildAccessDeniedEntry constructs a standardized AuditEntry for RBAC rejections (US-11 AC-1).
func BuildAccessDeniedEntry(c *gin.Context, claims *models.JWTClaims, resource string, requiredRoles []string) AuditEntry {
	var tenantID, userID, role string
	if claims != nil {
		tenantID = claims.Tid
		userID = claims.Sub
		role = string(claims.Role)
	}

	path := resource
	method := "UNKNOWN"
	clientIP := "127.0.0.1"
	userAgent := ""

	if c != nil && c.Request != nil {
		if path == "" {
			path = c.Request.URL.Path
		}
		method = c.Request.Method
		clientIP = c.ClientIP()
		userAgent = c.Request.UserAgent()
	}

	details := fmt.Sprintf("Akses ditolak: Pengguna dengan peran '%s' mencoba mengakses '%s %s'. Diperlukan salah satu dari peran [%s].",
		role, method, path, strings.Join(requiredRoles, ", "))

	return AuditEntry{
		ID:        fmt.Sprintf("audit-%d", time.Now().UnixNano()),
		Timestamp: time.Now().UTC(),
		TenantID:  tenantID,
		UserID:    userID,
		Role:      role,
		Action:    "ACCESS_DENIED_RBAC",
		Resource:  path,
		Method:    method,
		Status:    http.StatusForbidden,
		ClientIP:  clientIP,
		UserAgent: userAgent,
		Details:   details,
	}
}

