package models

import "github.com/golang-jwt/jwt/v5"

// UserRole represents the hierarchical user roles in On-Track.
type UserRole string

const (
	RoleSuperAdmin UserRole = "super_admin"
	RoleOwner      UserRole = "owner"
	RoleHRD        UserRole = "hrd"
	RoleTeamLead   UserRole = "team_lead"
	RoleEmployee   UserRole = "employee"
	RoleCashier    UserRole = "cashier"
)

// ValidRoles returns the list of all valid roles.
func ValidRoles() []UserRole {
	return []UserRole{
		RoleSuperAdmin,
		RoleOwner,
		RoleHRD,
		RoleTeamLead,
		RoleEmployee,
		RoleCashier,
	}
}

// JWTClaims represents custom claims contained within an On-Track JWT RS256 token.
type JWTClaims struct {
	Sub  string   `json:"sub"`            // User ID (UUID)
	Tid  string   `json:"tid"`            // Tenant ID (UUID)
	Role UserRole `json:"role"`           // Peran (Role)
	Eid  string   `json:"eid,omitempty"`  // Employee ID
	Mods []string `json:"mods,omitempty"` // Active modules / permissions
	jwt.RegisteredClaims
}

