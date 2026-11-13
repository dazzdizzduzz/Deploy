package rbac

import "testing"

func TestGetAllowedRoles(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		path       string
		expectRule bool
		expectRole string
	}{
		{
			name:       "exact employees POST",
			method:     "POST",
			path:       "/employees",
			expectRule: true,
			expectRole: "owner",
		},
		{
			name:       "prefixed /v1 employees POST",
			method:     "POST",
			path:       "/v1/employees",
			expectRule: true,
			expectRole: "hrd",
		},
		{
			name:       "gin parameterized /v1/employees/:id GET",
			method:     "GET",
			path:       "/v1/employees/:id",
			expectRule: true,
			expectRole: "team_lead",
		},
		{
			name:       "openapi parameterized /v1/employees/{id} PATCH",
			method:     "PATCH",
			path:       "/v1/employees/{id}",
			expectRule: true,
			expectRole: "owner",
		},
		{
			name:       "unregistered route",
			method:     "GET",
			path:       "/v1/unknown-endpoint",
			expectRule: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			roles, ok := GetAllowedRoles(tt.method, tt.path)
			if ok != tt.expectRule {
				t.Fatalf("expected rule existence %v, got %v", tt.expectRule, ok)
			}
			if tt.expectRule {
				if !IsRoleAllowed(tt.expectRole, roles) {
					t.Errorf("expected role %s to be allowed in %v", tt.expectRole, roles)
				}
			}
		})
	}
}

func TestIsRoleAllowed(t *testing.T) {
	allowed := []string{"owner", "hrd"}

	// super_admin bypass
	if !IsRoleAllowed("super_admin", allowed) {
		t.Errorf("expected super_admin to always be allowed")
	}

	// owner allowed
	if !IsRoleAllowed("owner", allowed) {
		t.Errorf("expected owner to be allowed")
	}

	// hrd allowed
	if !IsRoleAllowed("hrd", allowed) {
		t.Errorf("expected hrd to be allowed")
	}

	// employee NOT allowed
	if IsRoleAllowed("employee", allowed) {
		t.Errorf("expected employee to NOT be allowed")
	}

	// cashier NOT allowed
	if IsRoleAllowed("cashier", allowed) {
		t.Errorf("expected cashier to NOT be allowed")
	}
}

