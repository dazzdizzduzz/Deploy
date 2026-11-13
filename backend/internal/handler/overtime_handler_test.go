package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/tiers-undiknas/capstone-group07-2026/backend/api"
	"github.com/tiers-undiknas/capstone-group07-2026/backend/internal/auth"
	"github.com/tiers-undiknas/capstone-group07-2026/backend/internal/models"
	"github.com/tiers-undiknas/capstone-group07-2026/backend/internal/repository"
)

// ---------------------------------------------------------------- mock repository

type mockOvertimeRepo struct {
	mu   sync.Mutex
	rows map[string]*repository.OvertimeRequest
}

func newMockOvertimeRepo() *mockOvertimeRepo {
	return &mockOvertimeRepo{rows: map[string]*repository.OvertimeRequest{}}
}

func (m *mockOvertimeRepo) add(o *repository.OvertimeRequest) *repository.OvertimeRequest {
	m.mu.Lock()
	defer m.mu.Unlock()
	if o.ID == "" {
		o.ID = uuid.NewString()
	}
	if o.TenantID == "" {
		o.TenantID = attTenant
	}
	if o.CreatedAt.IsZero() {
		o.CreatedAt = time.Now()
	}
	if o.UpdatedAt.IsZero() {
		o.UpdatedAt = time.Now()
	}
	cp := *o
	m.rows[o.ID] = &cp
	return o
}

func (m *mockOvertimeRepo) Create(_ context.Context, req *repository.OvertimeRequest) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if req.ID == "" {
		req.ID = uuid.NewString()
	}
	req.CreatedAt = time.Now()
	req.UpdatedAt = time.Now()
	cp := *req
	m.rows[req.ID] = &cp
	return nil
}

func (m *mockOvertimeRepo) FindByID(_ context.Context, tenantID, id string) (*repository.OvertimeRequest, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if r, ok := m.rows[id]; ok && r.TenantID == tenantID {
		cp := *r
		return &cp, nil
	}
	return nil, repository.ErrOvertimeNotFound
}

func (m *mockOvertimeRepo) FindByAttendanceID(_ context.Context, tenantID, attendanceID string) (*repository.OvertimeRequest, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, r := range m.rows {
		if r.TenantID == tenantID && r.AttendanceID != nil && *r.AttendanceID == attendanceID {
			cp := *r
			return &cp, nil
		}
	}
	return nil, repository.ErrOvertimeNotFound
}

func (m *mockOvertimeRepo) List(_ context.Context, tenantID string, f repository.OvertimeFilter) ([]*repository.OvertimeRequest, int, map[string]int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	counts := map[string]int{
		repository.OvertimeStatusPending:  0,
		repository.OvertimeStatusApproved: 0,
		repository.OvertimeStatusRejected: 0,
	}
	var filtered []*repository.OvertimeRequest
	for _, r := range m.rows {
		if r.TenantID != tenantID {
			continue
		}
		// In mock, tab counts reflect matching tenant items
		counts[r.Status]++

		if f.Status != "" && r.Status != f.Status {
			continue
		}
		cp := *r
		filtered = append(filtered, &cp)
	}
	total := len(filtered)
	start := (f.Page - 1) * f.PerPage
	if start > total {
		start = total
	}
	end := start + f.PerPage
	if end > total {
		end = total
	}
	return filtered[start:end], total, counts, nil
}

func (m *mockOvertimeRepo) Approve(_ context.Context, tenantID, id, reviewerID string) (*repository.OvertimeRequest, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	r, ok := m.rows[id]
	if !ok || r.TenantID != tenantID {
		return nil, repository.ErrOvertimeNotFound
	}
	if r.UserID == reviewerID {
		return nil, repository.ErrOvertimeSelfApproval
	}
	if r.Status != repository.OvertimeStatusPending {
		return nil, repository.ErrOvertimeAlreadyProcessed
	}
	r.Status = repository.OvertimeStatusApproved
	r.ReviewedBy = &reviewerID
	now := time.Now()
	r.ReviewedAt = &now
	r.UpdatedAt = now
	cp := *r
	return &cp, nil
}

func (m *mockOvertimeRepo) Reject(_ context.Context, tenantID, id, reviewerID, reason string) (*repository.OvertimeRequest, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if strings.TrimSpace(reason) == "" {
		return nil, errors.New("alasan penolakan wajib diisi")
	}
	r, ok := m.rows[id]
	if !ok || r.TenantID != tenantID {
		return nil, repository.ErrOvertimeNotFound
	}
	if r.UserID == reviewerID {
		return nil, repository.ErrOvertimeSelfApproval
	}
	if r.Status != repository.OvertimeStatusPending {
		return nil, repository.ErrOvertimeAlreadyProcessed
	}
	r.Status = repository.OvertimeStatusRejected
	r.ReviewedBy = &reviewerID
	now := time.Now()
	r.ReviewedAt = &now
	r.RejectionReason = &reason
	r.UpdatedAt = now
	cp := *r
	return &cp, nil
}

// ---------------------------------------------------------------- fixture

type overtimeFixture struct {
	router       *gin.Engine
	jwt          *auth.JWTManager
	otRepo       *mockOvertimeRepo
	empRepo      *mockEmployeeRepo
	ownerID      string
	leadID       string
	staffID      string
	otherStaffID string
}

func newOvertimeFixture(t *testing.T) *overtimeFixture {
	t.Helper()
	gin.SetMode(gin.TestMode)
	jwtManager, err := auth.NewJWTManager("", "")
	if err != nil {
		t.Fatal(err)
	}

	otRepo := newMockOvertimeRepo()
	empRepo := newMockEmployeeRepo()

	ownerID := uuid.NewString()
	leadID := uuid.NewString()
	staffID := uuid.NewString()
	otherStaffID := uuid.NewString()

	empRepo.add(&repository.Employee{ID: ownerID, TenantID: attTenant, FullName: "Owner Boss", Email: "owner@ontrack.id", Role: "owner", Status: "aktif"})
	empRepo.add(&repository.Employee{ID: leadID, TenantID: attTenant, FullName: "Lead Bob", Email: "lead@ontrack.id", Role: "team_lead", Status: "aktif"})
	// staffID has manager leadID
	empRepo.add(&repository.Employee{ID: staffID, TenantID: attTenant, FullName: "Staff Alice", Email: "alice@ontrack.id", Role: "employee", Status: "aktif", ManagerID: &leadID})
	// otherStaffID has no manager (or different manager)
	empRepo.add(&repository.Employee{ID: otherStaffID, TenantID: attTenant, FullName: "Other Charlie", Email: "charlie@ontrack.id", Role: "employee", Status: "aktif"})

	srv := NewServer(
		&mockUserRepo{users: map[string]*repository.User{}},
		&mockLocationRepo{},
		&mockScheduleRepo{},
		jwtManager,
		auth.NewLoginLockManager(),
		auth.NewPasswordResetManager(),
	)
	srv.EnableEmployees(empRepo)
	srv.EnableOvertime(otRepo)

	r := gin.New()
	r.Use(gin.Recovery())
	api.RegisterHandlers(r, srv)

	return &overtimeFixture{
		router:       r,
		jwt:          jwtManager,
		otRepo:       otRepo,
		empRepo:      empRepo,
		ownerID:      ownerID,
		leadID:       leadID,
		staffID:      staffID,
		otherStaffID: otherStaffID,
	}
}

func (f *overtimeFixture) token(t *testing.T, userID string, role models.UserRole) string {
	t.Helper()
	tok, _, err := f.jwt.GenerateToken(userID, attTenant, string(role))
	if err != nil {
		t.Fatal(err)
	}
	return tok
}

func (f *overtimeFixture) call(t *testing.T, method, path, token string, body string) *httptest.ResponseRecorder {
	t.Helper()
	var rdr *bytes.Reader
	if body != "" {
		rdr = bytes.NewReader([]byte(body))
	} else {
		rdr = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, path, rdr)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	f.router.ServeHTTP(w, req)
	return w
}

// ---------------------------------------------------------------- tests

func TestGetOvertimeRequests_RBAC(t *testing.T) {
	f := newOvertimeFixture(t)

	// 1. Tanpa token -> 401
	w := f.call(t, http.MethodGet, "/overtime-requests", "", "")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for unauthorized, got %d", w.Code)
	}

	// 2. Peran employee -> 403
	empToken := f.token(t, f.staffID, models.RoleEmployee)
	w = f.call(t, http.MethodGet, "/overtime-requests", empToken, "")
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for employee, got %d", w.Code)
	}

	// 3. Peran owner -> 200
	ownerToken := f.token(t, f.ownerID, models.RoleOwner)
	w = f.call(t, http.MethodGet, "/overtime-requests", ownerToken, "")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for owner, got %d: %s", w.Code, w.Body.String())
	}

	var resp api.OvertimeListResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Meta.Page != 1 || resp.Meta.PerPage != 20 {
		t.Errorf("unexpected pagination: %+v", resp.Meta)
	}

	// 4. Peran team_lead -> 200
	leadToken := f.token(t, f.leadID, models.RoleTeamLead)
	w = f.call(t, http.MethodGet, "/overtime-requests?status=menunggu", leadToken, "")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for team lead, got %d: %s", w.Code, w.Body.String())
	}

	// 5. Invalid status -> 400
	w = f.call(t, http.MethodGet, "/overtime-requests?status=invalid_status", leadToken, "")
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid status, got %d", w.Code)
	}
}

func TestPostOvertimeRequest(t *testing.T) {
	f := newOvertimeFixture(t)
	empToken := f.token(t, f.staffID, models.RoleEmployee)

	// 1. Validasi: reason kosong -> 400
	bodyInvalid := `{"date":"2026-11-13","start_time":"17:30","end_time":"19:30","reason":""}`
	w := f.call(t, http.MethodPost, "/overtime-requests", empToken, bodyInvalid)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for empty reason, got %d", w.Code)
	}

	// 2. Validasi: format jam salah -> 400
	bodyBadTime := `{"date":"2026-11-13","start_time":"5pm","end_time":"7pm","reason":"Selesai tugas"}`
	w = f.call(t, http.MethodPost, "/overtime-requests", empToken, bodyBadTime)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid time format, got %d", w.Code)
	}

	// 3. Validasi: end_time <= start_time -> 400
	bodyBackwards := `{"date":"2026-11-13","start_time":"19:00","end_time":"17:00","reason":"Selesai tugas"}`
	w = f.call(t, http.MethodPost, "/overtime-requests", empToken, bodyBackwards)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 when end_time <= start_time, got %d", w.Code)
	}

	// 4. Sukses: 201 Created
	bodyValid := `{"date":"2026-11-13","start_time":"17:30","end_time":"19:30","reason":"Menyelesaikan revisi branding klien"}`
	w = f.call(t, http.MethodPost, "/overtime-requests", empToken, bodyValid)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d: %s", w.Code, w.Body.String())
	}

	var res api.OvertimeResponse
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if res.DurationMinutes != 120 {
		t.Errorf("expected duration 120 minutes, got %d", res.DurationMinutes)
	}
	if string(res.Status) != repository.OvertimeStatusPending {
		t.Errorf("expected status menunggu, got %s", res.Status)
	}
	if res.Reason != "Menyelesaikan revisi branding klien" {
		t.Errorf("expected reason match, got %s", res.Reason)
	}
}

func TestGetOvertimeRequestById(t *testing.T) {
	f := newOvertimeFixture(t)
	empToken := f.token(t, f.staffID, models.RoleEmployee)
	otherToken := f.token(t, f.otherStaffID, models.RoleEmployee)
	ownerToken := f.token(t, f.ownerID, models.RoleOwner)

	// Simpan satu lembur milik staffID
	req := f.otRepo.add(&repository.OvertimeRequest{
		UserID:          f.staffID,
		Date:            time.Date(2026, 11, 13, 0, 0, 0, 0, time.UTC),
		StartTime:       "17:30",
		EndTime:         "19:30",
		DurationMinutes: 120,
		Reason:          "Lempar tugas",
		Status:          repository.OvertimeStatusPending,
	})

	// 1. Staff melihat miliknya sendiri -> 200
	w := f.call(t, http.MethodGet, "/overtime-requests/"+req.ID, empToken, "")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for owner of overtime, got %d", w.Code)
	}

	// 2. Other staff melihat milik staff -> 403 Forbidden
	w = f.call(t, http.MethodGet, "/overtime-requests/"+req.ID, otherToken, "")
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for other employee viewing request, got %d", w.Code)
	}

	// 3. Owner melihat milik staff -> 200
	w = f.call(t, http.MethodGet, "/overtime-requests/"+req.ID, ownerToken, "")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for owner viewing request, got %d", w.Code)
	}

	// 4. ID tidak ada -> 404
	randomUUID := uuid.NewString()
	w = f.call(t, http.MethodGet, "/overtime-requests/"+randomUUID, ownerToken, "")
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for non-existent id, got %d", w.Code)
	}
}

func TestPostOvertimeRequestApprove(t *testing.T) {
	f := newOvertimeFixture(t)
	empToken := f.token(t, f.staffID, models.RoleEmployee)
	leadToken := f.token(t, f.leadID, models.RoleTeamLead)
	ownerToken := f.token(t, f.ownerID, models.RoleOwner)

	// Lembur milik staff (bawahan lead)
	staffOt := f.otRepo.add(&repository.OvertimeRequest{
		UserID:          f.staffID,
		Date:            time.Date(2026, 11, 13, 0, 0, 0, 0, time.UTC),
		StartTime:       "17:30",
		EndTime:         "19:30",
		DurationMinutes: 120,
		Reason:          "Tugas lembur staff",
		Status:          repository.OvertimeStatusPending,
	})

	// Lembur milik otherStaff (bukan bawahan lead)
	otherOt := f.otRepo.add(&repository.OvertimeRequest{
		UserID:          f.otherStaffID,
		Date:            time.Date(2026, 11, 13, 0, 0, 0, 0, time.UTC),
		StartTime:       "17:30",
		EndTime:         "19:30",
		DurationMinutes: 120,
		Reason:          "Tugas lembur other staff",
		Status:          repository.OvertimeStatusPending,
	})

	// Lembur milik lead (untuk tes self-approval)
	leadOt := f.otRepo.add(&repository.OvertimeRequest{
		UserID:          f.leadID,
		Date:            time.Date(2026, 11, 13, 0, 0, 0, 0, time.UTC),
		StartTime:       "17:30",
		EndTime:         "19:30",
		DurationMinutes: 120,
		Reason:          "Tugas lembur lead",
		Status:          repository.OvertimeStatusPending,
	})

	// 1. Employee mencoba approve -> 403
	w := f.call(t, http.MethodPost, "/overtime-requests/"+staffOt.ID+"/approve", empToken, "")
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for employee approve, got %d", w.Code)
	}

	// 2. Lead mencoba self-approval lembur miliknya sendiri -> 403
	w = f.call(t, http.MethodPost, "/overtime-requests/"+leadOt.ID+"/approve", leadToken, "")
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for self-approval, got %d", w.Code)
	}

	// 3. Lead mencoba approve karyawan bukan bawahannya -> 403
	w = f.call(t, http.MethodPost, "/overtime-requests/"+otherOt.ID+"/approve", leadToken, "")
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for approving non-subordinate, got %d", w.Code)
	}

	// 4. Lead approve bawahan langsungnya -> 200 OK
	w = f.call(t, http.MethodPost, "/overtime-requests/"+staffOt.ID+"/approve", leadToken, "")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", w.Code, w.Body.String())
	}

	var res api.OvertimeResponse
	_ = json.Unmarshal(w.Body.Bytes(), &res)
	if string(res.Status) != repository.OvertimeStatusApproved {
		t.Errorf("expected status disetujui, got %s", res.Status)
	}

	// 5. Approve kembali (duplicate decision) -> 409 Conflict
	w = f.call(t, http.MethodPost, "/overtime-requests/"+staffOt.ID+"/approve", ownerToken, "")
	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409 Conflict for already decided overtime, got %d", w.Code)
	}
}

func TestPostOvertimeRequestReject(t *testing.T) {
	f := newOvertimeFixture(t)
	leadToken := f.token(t, f.leadID, models.RoleTeamLead)
	ownerToken := f.token(t, f.ownerID, models.RoleOwner)

	staffOt := f.otRepo.add(&repository.OvertimeRequest{
		UserID:          f.staffID,
		Date:            time.Date(2026, 11, 13, 0, 0, 0, 0, time.UTC),
		StartTime:       "17:30",
		EndTime:         "19:30",
		DurationMinutes: 120,
		Reason:          "Tugas lembur staff",
		Status:          repository.OvertimeStatusPending,
	})

	// 1. Alasan penolakan kosong -> 400
	w := f.call(t, http.MethodPost, "/overtime-requests/"+staffOt.ID+"/reject", leadToken, `{"reason":""}`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for empty rejection reason, got %d", w.Code)
	}

	// 2. Reject berhasil dengan alasan -> 200 OK
	w = f.call(t, http.MethodPost, "/overtime-requests/"+staffOt.ID+"/reject", leadToken, `{"reason":"Pekerjaan belum mendesak"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for rejection, got %d: %s", w.Code, w.Body.String())
	}

	var res api.OvertimeResponse
	_ = json.Unmarshal(w.Body.Bytes(), &res)
	if string(res.Status) != repository.OvertimeStatusRejected {
		t.Errorf("expected status ditolak, got %s", res.Status)
	}
	if res.RejectionReason == nil || *res.RejectionReason != "Pekerjaan belum mendesak" {
		t.Errorf("expected rejection reason match, got %v", res.RejectionReason)
	}

	// 3. Reject kembali -> 409 Conflict
	w = f.call(t, http.MethodPost, "/overtime-requests/"+staffOt.ID+"/reject", ownerToken, `{"reason":"Sudah ditolak"}`)
	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409 Conflict, got %d", w.Code)
	}
}

func TestClockOut_AutoCreateOvertime(t *testing.T) {
	f := newAttFixture(t)
	otRepo := newMockOvertimeRepo()
	f.srv.EnableOvertime(otRepo)

	// Clock in jam 07:58 WITA
	clockInAt(t, f, time.Date(2026, 11, 11, 7, 58, 0, 0, f.wita))

	// Clock out jam 18:30 WITA (lembur 90 menit dari jadwal selesai 17:00)
	f.now = time.Date(2026, 11, 11, 18, 30, 0, 0, f.wita)
	rec := f.do(t, clockOutPath, f.token(t, "employee", attTenant), pointForm(near100m))
	if rec.Code != http.StatusOK {
		t.Fatalf("clock-out failed: %d %s", rec.Code, rec.Body.String())
	}

	resp := decodeAttendance(t, rec)
	if resp.OvertimeMinutes != 90 {
		t.Fatalf("expected 90 minutes overtime, got %d", resp.OvertimeMinutes)
	}

	otRepo.mu.Lock()
	defer otRepo.mu.Unlock()
	if len(otRepo.rows) != 1 {
		t.Fatalf("expected 1 auto created overtime request, got %d", len(otRepo.rows))
	}
	for _, ot := range otRepo.rows {
		if ot.UserID != attUser {
			t.Errorf("expected user %s, got %s", attUser, ot.UserID)
		}
		if ot.DurationMinutes != 90 {
			t.Errorf("expected 90 minutes overtime, got %d", ot.DurationMinutes)
		}
		if ot.Status != repository.OvertimeStatusPending {
			t.Errorf("expected status menunggu, got %s", ot.Status)
		}
		if ot.AttendanceID == nil {
			t.Errorf("expected attendance ID to be set")
		}
	}
}
