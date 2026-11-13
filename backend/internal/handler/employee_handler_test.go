package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/tiers-undiknas/capstone-group07-2026/backend/api"
	"github.com/tiers-undiknas/capstone-group07-2026/backend/internal/auth"
	"github.com/tiers-undiknas/capstone-group07-2026/backend/internal/repository"
)

// ---------------------------------------------------------------- mock

type mockEmployeeRepo struct {
	mu        sync.Mutex
	rows      map[string]*repository.Employee
	validRefs map[string]bool // id relasi yang dianggap ada di tenant
	lastHash  string
}

func newMockEmployeeRepo() *mockEmployeeRepo {
	return &mockEmployeeRepo{rows: map[string]*repository.Employee{}, validRefs: map[string]bool{}}
}

func (m *mockEmployeeRepo) add(e *repository.Employee) *repository.Employee {
	if e.ID == "" {
		e.ID = uuid.NewString()
	}
	if e.TenantID == "" {
		e.TenantID = attTenant
	}
	m.rows[e.ID] = e
	return e
}

func (m *mockEmployeeRepo) List(_ context.Context, tenant string, f repository.EmployeeFilter) ([]*repository.Employee, int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []*repository.Employee
	for _, e := range m.rows {
		if e.TenantID == tenant && (f.Status == "" || e.Status == f.Status) &&
			(f.Search == "" || strings.Contains(strings.ToLower(e.FullName), strings.ToLower(f.Search))) {
			cp := *e
			out = append(out, &cp)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].FullName < out[j].FullName })
	total := len(out)
	start := (f.Page - 1) * f.PerPage
	if start > total {
		start = total
	}
	end := start + f.PerPage
	if end > total {
		end = total
	}
	return out[start:end], total, nil
}

func (m *mockEmployeeRepo) FindByID(_ context.Context, tenant, id string) (*repository.Employee, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if e, ok := m.rows[id]; ok && e.TenantID == tenant {
		cp := *e
		return &cp, nil
	}
	return nil, nil
}

func (m *mockEmployeeRepo) Create(_ context.Context, e *repository.Employee, hash string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, r := range m.rows {
		if r.TenantID == e.TenantID && r.Email == e.Email {
			return repository.ErrEmailTaken
		}
	}
	e.ID = uuid.NewString()
	e.CreatedAt, e.UpdatedAt = time.Now(), time.Now()
	m.lastHash = hash
	cp := *e
	m.rows[e.ID] = &cp
	return nil
}

func (m *mockEmployeeRepo) Update(_ context.Context, e *repository.Employee) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if r, ok := m.rows[e.ID]; !ok || r.TenantID != e.TenantID {
		return repository.ErrEmployeeNotFound
	}
	cp := *e
	m.rows[e.ID] = &cp
	return nil
}

func (m *mockEmployeeRepo) MissingRefs(_ context.Context, _ string, refs repository.EmployeeRefs) ([]string, error) {
	var missing []string
	check := func(field string, id *string) {
		if id != nil && !m.validRefs[*id] {
			missing = append(missing, field)
		}
	}
	check("department_id", refs.DepartmentID)
	check("position_id", refs.PositionID)
	check("manager_id", refs.ManagerID)
	check("schedule_id", refs.ScheduleID)
	check("location_id", refs.LocationID)
	return missing, nil
}

// ---------------------------------------------------------------- fixture

type empFixture struct {
	router *gin.Engine
	jwt    *auth.JWTManager
	repo   *mockEmployeeRepo
	owner  *repository.Employee
	hrd    *repository.Employee
	staff  *repository.Employee
}

func newEmpFixture(t *testing.T) *empFixture {
	t.Helper()
	gin.SetMode(gin.TestMode)
	jwtManager, err := auth.NewJWTManager("", "")
	if err != nil {
		t.Fatal(err)
	}
	repo := newMockEmployeeRepo()
	salary := "7250000.00"
	bank := "BCA"
	f := &empFixture{
		jwt:   jwtManager,
		repo:  repo,
		owner: repo.add(&repository.Employee{FullName: "Ananda Owner", Email: "owner@ontrack.id", Role: "owner", EmploymentType: "tetap", Status: "aktif"}),
		hrd:   repo.add(&repository.Employee{FullName: "Hana HRD", Email: "hrd@ontrack.id", Role: "hrd", EmploymentType: "tetap", Status: "aktif"}),
		staff: repo.add(&repository.Employee{FullName: "Dhais Staff", Email: "staff@ontrack.id", Role: "employee", EmploymentType: "kontrak",
			Status: "aktif", BaseSalary: &salary, BankName: &bank}),
	}
	// Karyawan tenant lain, tidak boleh pernah terlihat.
	repo.add(&repository.Employee{TenantID: uuid.NewString(), FullName: "Orang Lain", Email: "x@lain.id", Role: "employee", Status: "aktif"})

	srv := NewServer(&mockUserRepo{users: map[string]*repository.User{}}, &mockLocationRepo{}, &mockScheduleRepo{},
		jwtManager, auth.NewLoginLockManager(), auth.NewPasswordResetManager())
	srv.EnableEmployees(repo)
	wita, _ := time.LoadLocation("Asia/Makassar")
	_ = srv.EnableAttendance(AttendanceDeps{Repo: newMockAttendanceRepo(), Timezone: wita,
		Now: func() time.Time { return time.Date(2026, 11, 11, 10, 0, 0, 0, wita) }})

	f.router = gin.New()
	api.RegisterHandlers(f.router, srv)
	return f
}

func (f *empFixture) tokenFor(t *testing.T, e *repository.Employee) string {
	t.Helper()
	tok, _, err := f.jwt.GenerateToken(e.ID, attTenant, e.Role)
	if err != nil {
		t.Fatal(err)
	}
	return tok
}

func (f *empFixture) tokenRole(t *testing.T, role string) string {
	t.Helper()
	tok, _, err := f.jwt.GenerateToken(uuid.NewString(), attTenant, role)
	if err != nil {
		t.Fatal(err)
	}
	return tok
}

func (f *empFixture) call(t *testing.T, method, path, token string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	switch b := body.(type) {
	case nil:
	case string:
		buf.WriteString(b)
	default:
		_ = json.NewEncoder(&buf).Encode(b)
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	f.router.ServeHTTP(rec, req)
	return rec
}

func decodeUser(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &m); err != nil {
		t.Fatalf("respons bukan JSON: %s", rec.Body.String())
	}
	return m
}

func expectFieldError(t *testing.T, rec *httptest.ResponseRecorder, field string) {
	t.Helper()
	expectProblem(t, rec, http.StatusUnprocessableEntity, "VALIDATION_ERROR")
	var p struct {
		Errors map[string]string `json:"errors"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &p)
	if p.Errors[field] == "" {
		t.Fatalf("errors tidak memuat %q: %s", field, rec.Body.String())
	}
}

// ---------------------------------------------------------------- GET

func TestGetUsers_ListAndSensitiveFields(t *testing.T) {
	f := newEmpFixture(t)

	rec := f.call(t, http.MethodGet, "/users?per_page=2&page=1", f.tokenFor(t, f.hrd), nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	var list api.UserListResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &list)
	if list.Meta.Total != 3 || list.Meta.TotalPages != 2 || len(list.Data) != 2 {
		t.Fatalf("meta/data = %+v, %d item (tenant lain tidak boleh ikut)", list.Meta, len(list.Data))
	}

	// HRD melihat gaji; Team Lead tidak.
	hrdView := decodeUser(t, f.call(t, http.MethodGet, "/users/"+f.staff.ID, f.tokenFor(t, f.hrd), nil))
	if hrdView["base_salary"] != "7250000.00" || hrdView["bank_name"] != "BCA" {
		t.Fatalf("HRD harus melihat data sensitif: %v", hrdView)
	}
	leadView := decodeUser(t, f.call(t, http.MethodGet, "/users/"+f.staff.ID, f.tokenRole(t, "team_lead"), nil))
	if leadView["base_salary"] != nil || leadView["bank_name"] != nil {
		t.Fatalf("Team Lead tidak boleh melihat data sensitif: %v", leadView)
	}
}

func TestGetUsers_AccessAndValidation(t *testing.T) {
	f := newEmpFixture(t)

	expectProblem(t, f.call(t, http.MethodGet, "/users", "", nil), http.StatusUnauthorized, "UNAUTHORIZED")
	expectProblem(t, f.call(t, http.MethodGet, "/users", f.tokenFor(t, f.staff), nil), http.StatusForbidden, "FORBIDDEN")
	expectProblem(t, f.call(t, http.MethodGet, "/users?status=hilang", f.tokenFor(t, f.hrd), nil), http.StatusBadRequest, "INVALID_QUERY")
	expectProblem(t, f.call(t, http.MethodGet, "/users?per_page=500", f.tokenFor(t, f.hrd), nil), http.StatusBadRequest, "INVALID_QUERY")

	// Karyawan tenant lain dijawab 404, bukan 403.
	var foreignID string
	for id, e := range f.repo.rows {
		if e.TenantID != attTenant {
			foreignID = id
		}
	}
	expectProblem(t, f.call(t, http.MethodGet, "/users/"+foreignID, f.tokenFor(t, f.hrd), nil), http.StatusNotFound, "NOT_FOUND")
}

// ---------------------------------------------------------------- POST

func TestPostUsers_Create(t *testing.T) {
	f := newEmpFixture(t)
	dept := uuid.NewString()
	f.repo.validRefs[dept] = true

	rec := f.call(t, http.MethodPost, "/users", f.tokenFor(t, f.hrd), map[string]any{
		"name": "  Putu Ayu  ", "email": "AYU@StudioTigapagi.com", "role": "employee",
		"employment_type": "kontrak", "department_id": dept, "joined_at": "2026-11-01",
		"base_salary": "6500000.00", "phone": "0812 3456 7890",
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	u := decodeUser(t, rec)
	if u["name"] != "Putu Ayu" || u["email"] != "ayu@studiotigapagi.com" || u["status"] != "diundang" ||
		u["joined_at"] != "2026-11-01" || u["base_salary"] != "6500000.00" {
		t.Fatalf("data tidak sesuai: %v", u)
	}
	if f.repo.lastHash != unusablePasswordHash {
		t.Fatalf("akun baru harus memakai hash yang tidak bisa dipakai login, dapat %q", f.repo.lastHash)
	}

	// Email sama (beda huruf besar/kecil) ditolak.
	dup := f.call(t, http.MethodPost, "/users", f.tokenFor(t, f.hrd), map[string]any{
		"name": "Ayu Lain", "email": "ayu@studiotigapagi.com", "role": "employee", "employment_type": "tetap",
	})
	expectProblem(t, dup, http.StatusConflict, "EMAIL_TAKEN")
}

func TestPostUsers_Validation(t *testing.T) {
	f := newEmpFixture(t)
	tok := f.tokenFor(t, f.hrd)
	base := func() map[string]any {
		return map[string]any{"name": "Nama Valid", "email": "valid@ontrack.id", "role": "employee", "employment_type": "tetap"}
	}
	cases := []struct {
		field string
		value any
	}{
		{"name", "A"},
		{"email", "bukan-email"},
		{"email", "Nama <a@b.id>"},
		{"role", "super_admin"},
		{"employment_type", "freelance"},
		{"status", "nonaktif"},
		{"joined_at", "01-11-2026"},
		{"base_salary", "-100"},
		{"base_salary", "12345678901.00"},
		{"phone", "abc"},
		{"department_id", "bukan-uuid"},
		{"location_id", uuid.NewString()}, // UUID valid tapi bukan milik tenant
	}
	for _, c := range cases {
		body := base()
		body[c.field] = c.value
		expectFieldError(t, f.call(t, http.MethodPost, "/users", tok, body), c.field)
	}

	expectProblem(t, f.call(t, http.MethodPost, "/users", tok, `{"name": 123}`), http.StatusBadRequest, "INVALID_JSON")
	expectProblem(t, f.call(t, http.MethodPost, "/users", tok, `bukan json`), http.StatusBadRequest, "INVALID_JSON")
}

func TestPostUsers_RoleEscalation(t *testing.T) {
	f := newEmpFixture(t)
	body := map[string]any{"name": "Owner Baru", "email": "owner2@ontrack.id", "role": "owner", "employment_type": "tetap"}

	expectProblem(t, f.call(t, http.MethodPost, "/users", f.tokenFor(t, f.hrd), body), http.StatusForbidden, "FORBIDDEN")
	expectProblem(t, f.call(t, http.MethodPost, "/users", f.tokenRole(t, "team_lead"), body), http.StatusForbidden, "FORBIDDEN")
	if rec := f.call(t, http.MethodPost, "/users", f.tokenFor(t, f.owner), body); rec.Code != http.StatusCreated {
		t.Fatalf("Owner boleh membuat Owner: %d %s", rec.Code, rec.Body.String())
	}
}

// ---------------------------------------------------------------- PATCH

func TestPatchUser_PartialAndNull(t *testing.T) {
	f := newEmpFixture(t)
	dept := uuid.NewString()
	f.repo.validRefs[dept] = true
	f.repo.rows[f.staff.ID].DepartmentID = &dept
	tok := f.tokenFor(t, f.hrd)

	rec := f.call(t, http.MethodPatch, "/users/"+f.staff.ID, tok, `{"phone":"0857 3900 1122","department_id":null}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	u := decodeUser(t, rec)
	if u["phone"] != "0857 3900 1122" || u["department_id"] != nil {
		t.Fatalf("patch tidak sesuai: %v", u)
	}
	// Field yang tidak dikirim tetap.
	if u["name"] != "Dhais Staff" || u["base_salary"] != "7250000.00" {
		t.Fatalf("field lain ikut berubah: %v", u)
	}

	expectFieldError(t, f.call(t, http.MethodPatch, "/users/"+f.staff.ID, tok, `{"name":null}`), "name")
	expectFieldError(t, f.call(t, http.MethodPatch, "/users/"+f.staff.ID, tok, `{"manager_id":"`+f.staff.ID+`"}`), "manager_id")
	expectFieldError(t, f.call(t, http.MethodPatch, "/users/"+f.staff.ID, tok, `{"allow_wfh_without_approval":"ya"}`), "allow_wfh_without_approval")
	expectProblem(t, f.call(t, http.MethodPatch, "/users/"+uuid.NewString(), tok, `{"phone":"0812345678"}`), http.StatusNotFound, "NOT_FOUND")
}

func TestPatchUser_RoleRules(t *testing.T) {
	f := newEmpFixture(t)

	// HRD tidak boleh menaikkan staf menjadi owner, dan tidak boleh mengubah akun owner.
	expectProblem(t, f.call(t, http.MethodPatch, "/users/"+f.staff.ID, f.tokenFor(t, f.hrd), `{"role":"owner"}`), http.StatusForbidden, "FORBIDDEN")
	expectProblem(t, f.call(t, http.MethodPatch, "/users/"+f.owner.ID, f.tokenFor(t, f.hrd), `{"phone":"0812345678"}`), http.StatusForbidden, "FORBIDDEN")

	// Tidak boleh mengubah peran sendiri.
	expectProblem(t, f.call(t, http.MethodPatch, "/users/"+f.owner.ID, f.tokenFor(t, f.owner), `{"role":"employee"}`), http.StatusForbidden, "FORBIDDEN")

	// Owner boleh mengubah peran staf.
	rec := f.call(t, http.MethodPatch, "/users/"+f.staff.ID, f.tokenFor(t, f.owner), `{"role":"team_lead"}`)
	if rec.Code != http.StatusOK || decodeUser(t, rec)["role"] != "team_lead" {
		t.Fatalf("Owner mengubah peran: %d %s", rec.Code, rec.Body.String())
	}
}

// ---------------------------------------------------------------- deactivate / reactivate

func TestDeactivateAndReactivate(t *testing.T) {
	f := newEmpFixture(t)
	tok := f.tokenFor(t, f.hrd)
	path := "/users/" + f.staff.ID

	expectFieldError(t, f.call(t, http.MethodPost, path+"/deactivate", tok, `{"reason":"ok"}`), "reason")

	rec := f.call(t, http.MethodPost, path+"/deactivate", tok, `{"reason":"Masa kontrak berakhir"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	u := decodeUser(t, rec)
	if u["status"] != "nonaktif" || u["inactive_since"] != "2026-11-11" || u["deactivation_reason"] != "Masa kontrak berakhir" {
		t.Fatalf("hasil nonaktif tidak sesuai: %v", u)
	}
	expectProblem(t, f.call(t, http.MethodPost, path+"/deactivate", tok, `{"reason":"Lagi"}`), http.StatusUnprocessableEntity, "ALREADY_INACTIVE")

	rec = f.call(t, http.MethodPost, path+"/reactivate", tok, nil)
	u = decodeUser(t, rec)
	if rec.Code != http.StatusOK || u["status"] != "aktif" || u["inactive_since"] != nil || u["deactivation_reason"] != nil {
		t.Fatalf("hasil aktif kembali tidak sesuai: %d %v", rec.Code, u)
	}
	// Idempoten.
	if rec := f.call(t, http.MethodPost, path+"/reactivate", tok, nil); rec.Code != http.StatusOK {
		t.Fatalf("reactivate kedua: %d", rec.Code)
	}
}

func TestDeactivate_Guards(t *testing.T) {
	f := newEmpFixture(t)

	expectProblem(t, f.call(t, http.MethodPost, "/users/"+f.hrd.ID+"/deactivate", f.tokenFor(t, f.hrd), `{"reason":"Coba diri sendiri"}`),
		http.StatusForbidden, "FORBIDDEN")
	expectProblem(t, f.call(t, http.MethodPost, "/users/"+f.owner.ID+"/deactivate", f.tokenFor(t, f.hrd), `{"reason":"Coba owner"}`),
		http.StatusForbidden, "FORBIDDEN")
	expectProblem(t, f.call(t, http.MethodPost, "/users/"+f.staff.ID+"/deactivate", f.tokenRole(t, "team_lead"), `{"reason":"Bukan admin"}`),
		http.StatusForbidden, "FORBIDDEN")

	invited := f.repo.add(&repository.Employee{FullName: "Baru", Email: "baru@ontrack.id", Role: "employee", EmploymentType: "magang", Status: "diundang"})
	expectProblem(t, f.call(t, http.MethodPost, "/users/"+invited.ID+"/reactivate", f.tokenFor(t, f.hrd), nil),
		http.StatusUnprocessableEntity, "INVITATION_PENDING")
}

func TestEmployees_Disabled(t *testing.T) {
	jwtManager, _ := auth.NewJWTManager("", "")
	srv := NewServer(&mockUserRepo{}, &mockLocationRepo{}, &mockScheduleRepo{}, jwtManager,
		auth.NewLoginLockManager(), auth.NewPasswordResetManager())
	r := gin.New()
	api.RegisterHandlers(r, srv)
	f := &empFixture{router: r, jwt: jwtManager}
	expectProblem(t, f.call(t, http.MethodGet, "/users", f.tokenRole(t, "hrd"), nil), http.StatusServiceUnavailable, "EMPLOYEES_DISABLED")
}
