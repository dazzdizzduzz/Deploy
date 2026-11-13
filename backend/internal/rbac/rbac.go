package rbac

import "strings"

// GetAllowedRoles retrieves the allowed roles for a given HTTP method and path.
// It checks direct match, /v1 prefix, or stripped prefix.
func GetAllowedRoles(method, path string) ([]string, bool) {
	method = strings.ToUpper(method)
	cleanPath := strings.TrimRight(path, "/")
	if cleanPath == "" {
		cleanPath = "/"
	}

	// 1. Direct match: METHOD PATH
	key := method + " " + cleanPath
	if roles, ok := RBACRules[key]; ok {
		return roles, true
	}

	// 2. Try with /v1 prefix if not present
	if !strings.HasPrefix(cleanPath, "/v1/") && cleanPath != "/v1" {
		if roles, ok := RBACRules[method+" /v1"+cleanPath]; ok {
			return roles, true
		}
	}

	// 3. Try removing /v1 prefix
	if strings.HasPrefix(cleanPath, "/v1/") {
		unprefixed := strings.TrimPrefix(cleanPath, "/v1")
		if roles, ok := RBACRules[method+" "+unprefixed]; ok {
			return roles, true
		}
	}

	return nil, false
}

// IsRoleAllowed checks if userRole exists in the allowedRoles slice.
func IsRoleAllowed(userRole string, allowedRoles []string) bool {
	// super_admin always has access to all protected endpoints
	if userRole == "super_admin" {
		return true
	}
	for _, r := range allowedRoles {
		if strings.EqualFold(r, userRole) {
			return true
		}
	}
	return false
}

// AllRulesSummary returns a copy of all registered RBAC rules.
func AllRulesSummary() map[string][]string {
	res := make(map[string][]string, len(RBACRules))
	for k, v := range RBACRules {
		roles := make([]string, len(v))
		copy(roles, v)
		res[k] = roles
	}
	return res
}

