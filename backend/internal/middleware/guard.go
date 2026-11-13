// Package middleware menyediakan middleware HTTP untuk otentikasi JWT RS256 dan penegakan RBAC OpenAPI.
package middleware

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/tiers-undiknas/capstone-group07-2026/backend/internal/audit"
	"github.com/tiers-undiknas/capstone-group07-2026/backend/internal/auth"
	"github.com/tiers-undiknas/capstone-group07-2026/backend/internal/models"
	"github.com/tiers-undiknas/capstone-group07-2026/backend/internal/rbac"
)

// GuardOptions configures the behavior of GuardMiddleware.
type GuardOptions struct {
	JWTService  *auth.JWTService
	AuditLogger audit.Logger
}

// Guard creates the Gin Middleware Guard that enforces:
// 1. JWT RS256 signature and claims validity.
// 2. Route matching against rbac.gen.go (OpenAPI x-rbac extension).
// 3. HTTP 403 Forbidden and Audit Logging for unauthorized access attempts (US-11 AC-1).
func Guard(opts GuardOptions) gin.HandlerFunc {
	if opts.AuditLogger == nil {
		opts.AuditLogger = audit.NewConsoleLogger()
	}

	return func(c *gin.Context) {
		if opts.JWTService == nil {
			c.Header("Content-Type", "application/problem+json")
			c.AbortWithStatusJSON(http.StatusInternalServerError, models.NewProblem(
				http.StatusInternalServerError,
				"Internal Server Error",
				"Layanan otentikasi JWT belum diinisialisasi.",
				"INTERNAL_ERROR",
				nil,
			))
			return
		}

		// 1. Validasi Token JWT RS256
		claims, err := opts.JWTService.ValidateRequest(c.Request)
		if err != nil {
			c.Header("Content-Type", "application/problem+json")
			detail := "Token otentikasi tidak valid atau sudah kedaluwarsa."
			if errors.Is(err, auth.ErrMissingAuthHeader) {
				detail = "Header Authorization: Bearer <token> wajib disertakan."
			} else if errors.Is(err, auth.ErrInvalidAuthFormat) {
				detail = "Format otentikasi harus Bearer <token>."
			}

			c.AbortWithStatusJSON(http.StatusUnauthorized, models.NewProblem(
				http.StatusUnauthorized,
				"Tidak Terautentikasi",
				detail,
				"UNAUTHORIZED",
				nil,
			))
			return
		}

		// Inject user claims into Gin context
		c.Set("user_id", claims.Sub)
		c.Set("tenant_id", claims.Tid)
		c.Set("role", string(claims.Role))
		c.Set("user_role", claims.Role)
		c.Set("employee_id", claims.Eid)
		c.Set("mods", claims.Mods)
		c.Set("claims", claims)

		// 2. Evaluasi RBAC berbasis rbac.gen.go (OpenAPI x-rbac)
		routePath := c.FullPath()
		if routePath == "" {
			routePath = c.Request.URL.Path
		}

		allowedRoles, protected := rbac.GetAllowedRoles(c.Request.Method, routePath)
		if !protected && routePath != c.Request.URL.Path {
			// Fallback check with raw request path
			allowedRoles, protected = rbac.GetAllowedRoles(c.Request.Method, c.Request.URL.Path)
		}

		if protected && len(allowedRoles) > 0 {
			userRoleStr := string(claims.Role)
			if !rbac.IsRoleAllowed(userRoleStr, allowedRoles) {
				// 3. Catat di Audit Log (US-11 AC-1)
				opts.AuditLogger.LogAccessDenied(c, claims, routePath, allowedRoles)

				// Tolak akses dengan kode status 403 (Forbidden)
				c.Header("Content-Type", "application/problem+json")
				c.AbortWithStatusJSON(http.StatusForbidden, models.NewProblem(
					http.StatusForbidden,
					"Akses Ditolak",
					fmt.Sprintf("Peran '%s' tidak memiliki izin untuk mengakses resource '%s %s'.",
						userRoleStr, c.Request.Method, routePath),
					"FORBIDDEN",
					gin.H{
						"required_roles": allowedRoles,
						"your_role":      userRoleStr,
					},
				))
				return
			}
		}

		c.Next()
	}
}

// RequireRoles returns a middleware handler for inline role enforcement on specific routes.
func RequireRoles(auditLogger audit.Logger, allowedRoles ...models.UserRole) gin.HandlerFunc {
	allowedStrs := make([]string, len(allowedRoles))
	for i, r := range allowedRoles {
		allowedStrs[i] = string(r)
	}

	return func(c *gin.Context) {
		roleVal, exists := c.Get("role")
		if !exists {
			c.Header("Content-Type", "application/problem+json")
			c.AbortWithStatusJSON(http.StatusUnauthorized, models.NewProblem(
				http.StatusUnauthorized,
				"Tidak Terautentikasi",
				"Sesi tidak valid.",
				"UNAUTHORIZED",
				nil,
			))
			return
		}

		userRole := ""
		switch v := roleVal.(type) {
		case string:
			userRole = v
		case models.UserRole:
			userRole = string(v)
		}

		if !rbac.IsRoleAllowed(userRole, allowedStrs) {
			var claims *models.JWTClaims
			if val, ok := c.Get("claims"); ok {
				claims = val.(*models.JWTClaims)
			}

			if auditLogger != nil {
				auditLogger.LogAccessDenied(c, claims, c.FullPath(), allowedStrs)
			}

			c.Header("Content-Type", "application/problem+json")
			c.AbortWithStatusJSON(http.StatusForbidden, models.NewProblem(
				http.StatusForbidden,
				"Akses Ditolak",
				fmt.Sprintf("Peran '%s' tidak memiliki izin untuk tindakan ini.", userRole),
				"FORBIDDEN",
				gin.H{
					"required_roles": allowedStrs,
					"your_role":      userRole,
				},
			))
			return
		}

		c.Next()
	}
}

