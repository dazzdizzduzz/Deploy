package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// EndpointRBACRules maps route patterns to allowed roles.
var defaultRBACMapping = map[string][]string{
	// 1. Auth & Self
	"POST /auth/logout":               {"super_admin", "owner", "hrd", "team_lead", "employee", "cashier"},
	"GET /me":                          {"super_admin", "owner", "hrd", "team_lead", "employee", "cashier"},
	"GET /me/profile":                  {"super_admin", "owner", "hrd", "team_lead", "employee", "cashier"},
	"GET /me/schedule-location":        {"super_admin", "owner", "hrd", "team_lead", "employee", "cashier"},
	"PATCH /me/settings":               {"super_admin", "owner", "hrd", "team_lead", "employee", "cashier"},

	// 2. Struktur Organisasi
	"GET /departments":                {"super_admin", "owner", "hrd", "team_lead", "employee", "cashier"},
	"POST /departments":               {"super_admin", "owner", "hrd"},
	"GET /departments/{id}":           {"super_admin", "owner", "hrd", "team_lead", "employee", "cashier"},
	"PUT /departments/{id}":           {"super_admin", "owner", "hrd"},
	"DELETE /departments/{id}":        {"super_admin", "owner", "hrd"},

	"GET /positions":                  {"super_admin", "owner", "hrd", "team_lead", "employee", "cashier"},
	"POST /positions":                 {"super_admin", "owner", "hrd"},
	"GET /positions/{id}":             {"super_admin", "owner", "hrd", "team_lead", "employee", "cashier"},
	"PUT /positions/{id}":             {"super_admin", "owner", "hrd"},
	"DELETE /positions/{id}":          {"super_admin", "owner", "hrd"},

	// 3. Manajemen User & Karyawan
	"GET /users":                      {"super_admin", "owner", "hrd", "team_lead"},
	"POST /users":                     {"super_admin", "owner", "hrd"},
	"GET /users/{id}":                 {"super_admin", "owner", "hrd", "team_lead"},
	"PATCH /users/{id}":               {"super_admin", "owner", "hrd"},
	"POST /users/{id}/deactivate":     {"super_admin", "owner", "hrd"},
	"POST /users/{id}/reactivate":     {"super_admin", "owner", "hrd"},
	"POST /users/{id}/invitation":     {"super_admin", "owner", "hrd"},

	"GET /employees":                  {"super_admin", "owner", "hrd", "team_lead"},
	"POST /employees":                 {"super_admin", "owner", "hrd"},
	"GET /employees/{id}":             {"super_admin", "owner", "hrd", "team_lead"},
	"PATCH /employees/{id}":           {"super_admin", "owner", "hrd"},
	"POST /employees/{id}/deactivate": {"super_admin", "owner", "hrd"},
	"POST /employees/{id}/reactivate": {"super_admin", "owner", "hrd"},
	"POST /employees/{id}/invitation": {"super_admin", "owner", "hrd"},

	// 4. Data Master: Jadwal & Lokasi
	"GET /schedules":                  {"super_admin", "owner", "hrd", "team_lead", "employee", "cashier"},
	"POST /schedules":                 {"super_admin", "owner", "hrd"},
	"GET /schedules/{id}":             {"super_admin", "owner", "hrd", "team_lead", "employee", "cashier"},
	"PATCH /schedules/{id}":           {"super_admin", "owner", "hrd"},
	"DELETE /schedules/{id}":          {"super_admin", "owner", "hrd"},

	"GET /locations":                  {"super_admin", "owner", "hrd", "team_lead", "employee", "cashier"},
	"POST /locations":                 {"super_admin", "owner", "hrd"},
	"GET /locations/{id}":             {"super_admin", "owner", "hrd", "team_lead", "employee", "cashier"},
	"PATCH /locations/{id}":           {"super_admin", "owner", "hrd"},
	"DELETE /locations/{id}":          {"super_admin", "owner", "hrd"},

	// 5. Absensi Harian
	"GET /attendance/today":           {"super_admin", "owner", "hrd", "team_lead", "employee", "cashier"},
	"POST /attendance/clock-in":       {"super_admin", "owner", "hrd", "team_lead", "employee", "cashier"},
	"POST /attendance/clock-out":      {"super_admin", "owner", "hrd", "team_lead", "employee", "cashier"},
	"POST /attendances/clock-in":      {"super_admin", "owner", "hrd", "team_lead", "employee", "cashier"},
	"POST /attendances/clock-out":     {"super_admin", "owner", "hrd", "team_lead", "employee", "cashier"},

	// 6. Lembur & Persetujuan
	"GET /overtime-requests":               {"super_admin", "owner", "hrd", "team_lead"},
	"POST /overtime-requests":              {"super_admin", "owner", "hrd", "team_lead", "employee", "cashier"},
	"GET /overtime-requests/{id}":          {"super_admin", "owner", "hrd", "team_lead"},
	"POST /overtime-requests/{id}/approve": {"super_admin", "owner", "hrd", "team_lead"},
	"POST /overtime-requests/{id}/reject":  {"super_admin", "owner", "hrd", "team_lead"},

	// 7. Cuti & Izin
	"GET /me/leave-quota":             {"super_admin", "owner", "hrd", "team_lead", "employee", "cashier"},
	"GET /me/leave-requests":          {"super_admin", "owner", "hrd", "team_lead", "employee", "cashier"},
	"POST /me/leave-requests":         {"super_admin", "owner", "hrd", "team_lead", "employee", "cashier"},
	"GET /me/leave-requests/{id}":     {"super_admin", "owner", "hrd", "team_lead", "employee", "cashier"},
	"POST /me/leave-requests/preview":  {"super_admin", "owner", "hrd", "team_lead", "employee", "cashier"},

	// 8. Rekapitulasi & Ekspor
	"GET /attendance/recap":              {"super_admin", "owner", "hrd", "team_lead"},
	"GET /attendance/recap/exports/{id}": {"super_admin", "owner", "hrd"},
	"POST /attendance/recap/exports":     {"super_admin", "owner", "hrd"},

	// 9. Role & Permission Management
	"GET /roles/permissions": {"super_admin", "owner", "hrd"},
	"PUT /roles/permissions": {"super_admin", "owner"},
}

func main() {
	openapiCandidates := []string{
		"api/openapi.yaml",
		"backend/api/openapi.yaml",
		"../api/openapi.yaml",
		"../api/openapi-phase1.yaml",
		"../../api/openapi-phase1.yaml",
		"api/openapi-phase1.yaml",
		"backend/api/openapi-phase1.yaml",
		"../backend/api/openapi-phase1.yaml",
	}

	var openapiFile string
	for _, p := range openapiCandidates {
		if _, err := os.Stat(p); err == nil {
			openapiFile = p
			break
		}
	}

	if openapiFile != "" {
		fmt.Printf("Menemukan berkas OpenAPI di: %s\n", openapiFile)
		data, err := os.ReadFile(openapiFile)
		if err == nil {
			var root yaml.Node
			if err := yaml.Unmarshal(data, &root); err == nil {
				pathsNode := findChildByKey(root.Content[0], "paths")
				if pathsNode != nil {
					for i := 0; i < len(pathsNode.Content)-1; i += 2 {
						pathKey := pathsNode.Content[i].Value
						pathItem := pathsNode.Content[i+1]
						for j := 0; j < len(pathItem.Content)-1; j += 2 {
							method := strings.ToUpper(pathItem.Content[j].Value)
							if !isHTTPMethod(method) {
								continue
							}
							opNode := pathItem.Content[j+1]
							xRBAC := findChildByKey(opNode, "x-rbac")
							if xRBAC != nil && xRBAC.Kind == yaml.SequenceNode {
								var roles []string
								for _, item := range xRBAC.Content {
									roles = append(roles, item.Value)
								}
								defaultRBACMapping[method+" "+pathKey] = roles
							}
						}
					}
				}
			}
		}
	}

	// Tentukan lokasi output rbac.gen.go
	outputCandidates := []string{
		filepath.Join("internal", "rbac", "rbac.gen.go"),
		filepath.Join("backend", "internal", "rbac", "rbac.gen.go"),
		filepath.Join("..", "internal", "rbac", "rbac.gen.go"),
	}

	var outputGenFile string
	for _, candidate := range outputCandidates {
		dir := filepath.Dir(candidate)
		if _, err := os.Stat(dir); err == nil {
			outputGenFile = candidate
			break
		}
	}
	if outputGenFile == "" {
		outputGenFile = filepath.Join("internal", "rbac", "rbac.gen.go")
	}

	_ = os.MkdirAll(filepath.Dir(outputGenFile), 0755)
	generateRBACGoFile(outputGenFile, defaultRBACMapping)
}

func isHTTPMethod(m string) bool {
	switch m {
	case "GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS", "HEAD":
		return true
	default:
		return false
	}
}

func findChildByKey(node *yaml.Node, key string) *yaml.Node {
	if node == nil || node.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i < len(node.Content)-1; i += 2 {
		if node.Content[i].Value == key {
			return node.Content[i+1]
		}
	}
	return nil
}

func generateRBACGoFile(destPath string, mapping map[string][]string) {
	dedupMap := make(map[string][]string)

	for k, roles := range mapping {
		parts := strings.SplitN(k, " ", 2)
		method := parts[0]
		path := parts[1]

		// Variant 1: Original (e.g. GET /employees/{id})
		dedupMap[k] = roles

		// Variant 2: Prefixed with /v1 (e.g. GET /v1/employees/{id})
		dedupMap[method+" /v1"+path] = roles

		// Variant 3: Gin path param (e.g. GET /employees/:id)
		ginPath := toGinPath(path)
		if ginPath != path {
			dedupMap[method+" "+ginPath] = roles
			dedupMap[method+" /v1"+ginPath] = roles
		}
	}

	var keys []string
	for k := range dedupMap {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var sb strings.Builder
	sb.WriteString("// Code generated by scripts/generate_rbac.go; DO NOT EDIT.\n")
	sb.WriteString("// Source: OpenAPI Specification (vendor extension: x-rbac)\n\n")
	sb.WriteString("package rbac\n\n")

	sb.WriteString("// RouteRule holds the allowed roles for a given method and path pattern.\n")
	sb.WriteString("type RouteRule struct {\n")
	sb.WriteString("\tMethod       string   `json:\"method\"`\n")
	sb.WriteString("\tPath         string   `json:\"path\"`\n")
	sb.WriteString("\tAllowedRoles []string `json:\"allowed_roles\"`\n")
	sb.WriteString("}\n\n")

	sb.WriteString("// RBACRules maps 'METHOD PATH' (both OpenAPI style /users/{id} and Gin style /v1/users/:id) to allowed roles.\n")
	sb.WriteString("var RBACRules = map[string][]string{\n")

	for _, k := range keys {
		roles := dedupMap[k]
		quotedRoles := make([]string, len(roles))
		for i, r := range roles {
			quotedRoles[i] = fmt.Sprintf("%q", r)
		}
		fmt.Fprintf(&sb, "\t%q: {%s},\n", k, strings.Join(quotedRoles, ", "))
	}

	sb.WriteString("}\n")

	if err := os.WriteFile(destPath, []byte(sb.String()), 0644); err != nil {
		log.Fatalf("Gagal menulis berkas generated RBAC ke %s: %v", destPath, err)
	}
	fmt.Printf("✅ rbac.gen.go berhasil di-generate di %s\n", destPath)
}

func toGinPath(openAPIPath string) string {
	res := openAPIPath
	for {
		start := strings.Index(res, "{")
		if start == -1 {
			break
		}
		end := strings.Index(res[start:], "}")
		if end == -1 {
			break
		}
		param := res[start+1 : start+end]
		res = res[:start] + ":" + param + res[start+end+1:]
	}
	return res
}

