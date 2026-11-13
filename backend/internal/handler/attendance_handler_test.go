package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/tiers-undiknas/capstone-group07-2026/backend/api"
	"github.com/tiers-undiknas/capstone-group07-2026/backend/internal/attendance"
	"github.com/tiers-undiknas/capstone-group07-2026/backend/internal/auth"
	"github.com/tiers-undiknas/capstone-group07-2026/backend/internal/repository"
	"github.com/tiers-undiknas/capstone-group07-2026/backend/internal/storage"
)

const (
	attTenant = "a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11"
	attUser   = "f5eebc99-9c0b-4ef8-bb6d-6bb9bd380a66"
)

// Kantor contoh di Sanur dan titik-titik uji di sekitarnya.
var (
	officePoint = attendance.Point{Lat: -8.6850, Lon: 115.2620}
	near100m    = attendance.Point{Lat: -8.6841, Lon: 115.2620} // sekitar 100 m
	far500m     = attendance.Point{Lat: -8.6805, Lon: 115.2620} // sekitar 500 m
)

var jpegBytes = append([]byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 'J', 'F', 'I', 'F', 0x00}, bytes.Repeat([]byte{1}, 256)...)

// ---------------------------------------------------------------- mocks

type mockAttendanceRepo struct {
	mu             sync.Mutex
	rows           map[string]*repository.Attendance // kunci: tenant|user|tanggal
	forceDuplicate bool                              // simulasi race: Find kosong, Insert kena UNIQUE
}

func newMockAttendanceRepo() *mockAttendanceRepo {
	return &mockAttendanceRepo{rows: map[string]*repository.Attendance{}}
}

func attKey(tenant, user string, d time.Time) string {
	return tenant + "|" + user + "|" + d.Format(time.DateOnly)
}

func (m *mockAttendanceRepo) CreateClockIn(_ context.Context, a *repository.Attendance) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	k := attKey(a.TenantID, a.UserID, a.WorkDate)
	if _, exists := m.rows[k]; exists || m.forceDuplicate {
		return repository.ErrAlreadyClockedIn
	}
	a.ID = uuid.NewString()
	cp := *a
	m.rows[k] = &cp
	return nil
}

func (m *mockAttendanceRepo) FindByUserAndDate(_ context.Context, tenant, user string, d time.Time) (*repository.Attendance, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if r, ok := m.rows[attKey(tenant, user, d)]; ok {
		cp := *r
		return &cp, nil
	}
	return nil, nil
}

func (m *mockAttendanceRepo) UpdateClockOut(_ context.Context, u repository.ClockOutUpdate) (*repository.Attendance, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, r := range m.rows {
		if r.ID == u.AttendanceID && r.TenantID == u.TenantID && r.ClockOut == nil {
			out := u.ClockOut
			r.ClockOut = &out
			r.LatitudeOut, r.LongitudeOut = &u.LatitudeOut, &u.LongitudeOut
			r.DistanceOutMeters = u.DistanceOutMeters
			r.SelfieOutKey = u.SelfieOutKey
			r.EarlyLeaveMinutes = u.EarlyLeaveMinutes
			r.OvertimeMinutes = u.OvertimeMinutes
			cp := *r
			return &cp, nil
		}
	}
	return nil, repository.ErrAttendanceNotOpen
}

type fakeStore struct {
	mu      sync.Mutex
	objects map[string][]byte
	deleted []string
	failPut bool
}

func newFakeStore() *fakeStore { return &fakeStore{objects: map[string][]byte{}} }

func (f *fakeStore) Put(_ context.Context, key string, body io.Reader, _ int64, _ string) error {
	if f.failPut {
		return storage.ErrUnavailable
	}
	data, _ := io.ReadAll(body)
	f.mu.Lock()
	defer f.mu.Unlock()
	f.objects[key] = data
	return nil
}

func (f *fakeStore) Delete(_ context.Context, key string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.objects, key)
	f.deleted = append(f.deleted, key)
	return nil
}

func (f *fakeStore) PresignGet(context.Context, string, time.Duration) (string, error) {
	return "", nil
}

// ---------------------------------------------------------------- fixture

type attFixture struct {
	router *gin.Engine
	jwt    *auth.JWTManager
	repo   *mockAttendanceRepo
	store  *fakeStore
	locs   *mockLocationRepo
	scheds *mockScheduleRepo
	now    time.Time
	wita   *time.Location
	srv    *Server
}

func newAttFixture(t *testing.T) *attFixture {
	t.Helper()
	gin.SetMode(gin.TestMode)

	wita, err := attendance.LoadTimezone("Asia/Makassar")
	if err != nil {
		t.Fatal(err)
	}
	jwtManager, err := auth.NewJWTManager("", "")
	if err != nil {
		t.Fatal(err)
	}

	f := &attFixture{
		jwt:    jwtManager,
		repo:   newMockAttendanceRepo(),
		store:  newFakeStore(),
		locs:   &mockLocationRepo{locations: map[string]*repository.OfficeLocation{}},
		scheds: &mockScheduleRepo{schedules: map[string]*repository.WorkSchedule{}},
		now:    time.Date(2026, 11, 11, 7, 55, 0, 0, wita),
		wita:   wita,
	}
	f.locs.locations["loc-1"] = &repository.OfficeLocation{
		ID: uuid.NewString(), TenantID: attTenant, Name: "Kantor Sanur",
		Latitude: officePoint.Lat, Longitude: officePoint.Lon, RadiusMeters: 150, IsActive: true,
	}
	// mockLocationRepo memakai ID sebagai kunci map; samakan.
	loc := f.locs.locations["loc-1"]
	delete(f.locs.locations, "loc-1")
	f.locs.locations[loc.ID] = loc

	sched := &repository.WorkSchedule{
		ID: uuid.NewString(), TenantID: attTenant, Name: "Kantor",
		StartTime: "08:00", EndTime: "17:00", ToleranceMinutes: 15,
		OvertimeAfterMinutes: 60, MaxOvertimeMinutes: 180, IsActive: true,
	}
	f.scheds.schedules[sched.ID] = sched

	srv := NewServer(&mockUserRepo{users: map[string]*repository.User{}}, f.locs, f.scheds,
		jwtManager, auth.NewLoginLockManager(), auth.NewPasswordResetManager())
	f.srv = srv
	if err := srv.EnableAttendance(AttendanceDeps{
		Repo: f.repo, Store: f.store, Timezone: wita,
		Now: func() time.Time { return f.now },
	}); err != nil {
		t.Fatal(err)
	}

	f.router = gin.New()
	api.RegisterHandlers(f.router, srv)
	return f
}

func (f *attFixture) location() *repository.OfficeLocation {
	for _, l := range f.locs.locations {
		return l
	}
	return nil
}

func (f *attFixture) schedule() *repository.WorkSchedule {
	for _, s := range f.scheds.schedules {
		return s
	}
	return nil
}

func (f *attFixture) token(t *testing.T, role, tenant string) string {
	t.Helper()
	tok, _, err := f.jwt.GenerateToken(attUser, tenant, role)
	if err != nil {
		t.Fatal(err)
	}
	return tok
}

type formInput struct {
	lat, lon string
	fields   map[string]string
	selfie   []byte
}

func pointForm(p attendance.Point) formInput {
	return formInput{
		lat: strconvFloat(p.Lat), lon: strconvFloat(p.Lon),
		fields: map[string]string{},
	}
}

func strconvFloat(v float64) string {
	b, _ := json.Marshal(v)
	return string(b)
}

func (f *attFixture) do(t *testing.T, path, token string, in formInput) *httptest.ResponseRecorder {
	t.Helper()
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	_ = w.WriteField("latitude", in.lat)
	_ = w.WriteField("longitude", in.lon)
	for k, v := range in.fields {
		_ = w.WriteField(k, v)
	}
	if in.selfie != nil {
		part, _ := w.CreateFormFile("selfie", "selfie.jpg")
		_, _ = part.Write(in.selfie)
	}
	_ = w.Close()

	req := httptest.NewRequest(http.MethodPost, path, &body)
	req.Header.Set("Content-Type", w.FormDataContentType())
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	f.router.ServeHTTP(rec, req)
	return rec
}

func decodeAttendance(t *testing.T, rec *httptest.ResponseRecorder) api.AttendanceResponse {
	t.Helper()
	var resp api.AttendanceResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("respons bukan AttendanceResponse: %v; body=%s", err, rec.Body.String())
	}
	return resp
}

func expectProblem(t *testing.T, rec *httptest.ResponseRecorder, status int, code string) {
	t.Helper()
	if rec.Code != status {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, status, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/problem+json") {
		t.Fatalf("Content-Type = %q, want application/problem+json", ct)
	}
	var p struct {
		Code   string `json:"code"`
		Status int    `json:"status"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &p); err != nil || p.Code != code || p.Status != status {
		t.Fatalf("problem = %+v (err %v), want code %s; body=%s", p, err, code, rec.Body.String())
	}
}

const (
	clockInPath  = "/attendances/clock-in"
	clockOutPath = "/attendances/clock-out"
)

// ---------------------------------------------------------------- clock-in

func TestClockIn_OnTimeInsideGeofence(t *testing.T) {
	f := newAttFixture(t)
	in := pointForm(near100m)
	in.selfie = jpegBytes

	rec := f.do(t, clockInPath, f.token(t, "employee", attTenant), in)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d; body=%s", rec.Code, rec.Body.String())
	}
	resp := decodeAttendance(t, rec)
	if resp.Status != api.OnTime || resp.LateMinutes != 0 {
		t.Fatalf("status = %s/%d, want on_time/0", resp.Status, resp.LateMinutes)
	}
	if resp.DistanceInMeters == nil || *resp.DistanceInMeters != 100 {
		t.Fatalf("distance = %v, want 100", resp.DistanceInMeters)
	}
	if !resp.HasSelfieIn || resp.WorkDate.Format(time.DateOnly) != "2026-11-11" {
		t.Fatalf("selfie/work_date tidak sesuai: %+v", resp)
	}
	if !resp.ClockIn.Equal(f.now) || resp.ClockIn.Location() != time.UTC {
		t.Fatalf("clock_in = %v, want %v dalam UTC", resp.ClockIn, f.now)
	}
	if len(f.store.objects) != 1 {
		t.Fatalf("objek tersimpan = %d, want 1", len(f.store.objects))
	}
	for key := range f.store.objects {
		if !strings.HasPrefix(key, attTenant+"/selfie/") || !strings.HasSuffix(key, ".jpg") {
			t.Fatalf("key = %q, want {tenant_id}/selfie/*.jpg", key)
		}
	}
}

func TestClockIn_Late(t *testing.T) {
	f := newAttFixture(t)
	f.now = time.Date(2026, 11, 11, 8, 40, 0, 0, f.wita)

	rec := f.do(t, clockInPath, f.token(t, "employee", attTenant), pointForm(near100m))
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d; body=%s", rec.Code, rec.Body.String())
	}
	if resp := decodeAttendance(t, rec); resp.Status != api.Late || resp.LateMinutes != 40 {
		t.Fatalf("status = %s/%d, want late/40", resp.Status, resp.LateMinutes)
	}
}

func TestClockIn_OutsideGeofence(t *testing.T) {
	f := newAttFixture(t)
	in := pointForm(far500m)
	in.selfie = jpegBytes

	rec := f.do(t, clockInPath, f.token(t, "employee", attTenant), in)
	expectProblem(t, rec, http.StatusUnprocessableEntity, "OUTSIDE_GEOFENCE")
	if len(f.store.objects) != 0 {
		t.Fatal("selfie tidak boleh diunggah bila di luar geofence")
	}
}

func TestClockIn_SelfieRequired(t *testing.T) {
	f := newAttFixture(t)
	f.location().SelfieRequired = true

	rec := f.do(t, clockInPath, f.token(t, "employee", attTenant), pointForm(near100m))
	expectProblem(t, rec, http.StatusUnprocessableEntity, "SELFIE_REQUIRED")
}

func TestClockIn_Duplicate(t *testing.T) {
	f := newAttFixture(t)
	tok := f.token(t, "employee", attTenant)
	if rec := f.do(t, clockInPath, tok, pointForm(near100m)); rec.Code != http.StatusCreated {
		t.Fatalf("clock-in pertama: %d", rec.Code)
	}
	in := pointForm(near100m)
	in.selfie = jpegBytes
	rec := f.do(t, clockInPath, tok, in)
	expectProblem(t, rec, http.StatusConflict, "ALREADY_CLOCKED_IN")
	if len(f.store.objects) != 0 {
		t.Fatal("selfie tidak boleh diunggah untuk clock-in ganda")
	}
}

func TestClockIn_RaceDuplicateDeletesUploadedSelfie(t *testing.T) {
	f := newAttFixture(t)
	f.repo.forceDuplicate = true
	in := pointForm(near100m)
	in.selfie = jpegBytes

	rec := f.do(t, clockInPath, f.token(t, "employee", attTenant), in)
	expectProblem(t, rec, http.StatusConflict, "ALREADY_CLOCKED_IN")
	if len(f.store.objects) != 0 || len(f.store.deleted) != 1 {
		t.Fatalf("selfie yatim harus dihapus: objects=%d deleted=%d", len(f.store.objects), len(f.store.deleted))
	}
}

func TestClockIn_Auth(t *testing.T) {
	f := newAttFixture(t)

	t.Run("tanpa token", func(t *testing.T) {
		expectProblem(t, f.do(t, clockInPath, "", pointForm(near100m)), http.StatusUnauthorized, "UNAUTHORIZED")
	})
	t.Run("header X-Tenant-ID tanpa token ditolak", func(t *testing.T) {
		var body bytes.Buffer
		w := multipart.NewWriter(&body)
		_ = w.WriteField("latitude", "-8.6841")
		_ = w.WriteField("longitude", "115.262")
		_ = w.Close()
		req := httptest.NewRequest(http.MethodPost, clockInPath, &body)
		req.Header.Set("Content-Type", w.FormDataContentType())
		req.Header.Set("X-Tenant-ID", attTenant)
		rec := httptest.NewRecorder()
		f.router.ServeHTTP(rec, req)
		expectProblem(t, rec, http.StatusUnauthorized, "UNAUTHORIZED")
	})
	t.Run("token palsu", func(t *testing.T) {
		expectProblem(t, f.do(t, clockInPath, "bukan.token.jwt", pointForm(near100m)), http.StatusUnauthorized, "UNAUTHORIZED")
	})
	t.Run("super_admin tidak bisa absen", func(t *testing.T) {
		expectProblem(t, f.do(t, clockInPath, f.token(t, "super_admin", ""), pointForm(near100m)), http.StatusForbidden, "FORBIDDEN")
	})
	t.Run("token tanpa tenant", func(t *testing.T) {
		expectProblem(t, f.do(t, clockInPath, f.token(t, "employee", ""), pointForm(near100m)), http.StatusForbidden, "TENANT_REQUIRED")
	})
}

func TestClockIn_InvalidInput(t *testing.T) {
	f := newAttFixture(t)
	tok := f.token(t, "employee", attTenant)

	bad := pointForm(near100m)
	bad.lat = "95"
	expectProblem(t, f.do(t, clockInPath, tok, bad), http.StatusBadRequest, "INVALID_COORDINATE")

	bad = pointForm(near100m)
	bad.lon = "abc"
	expectProblem(t, f.do(t, clockInPath, tok, bad), http.StatusBadRequest, "INVALID_COORDINATE")

	bad = pointForm(near100m)
	bad.fields["work_mode"] = "liburan"
	expectProblem(t, f.do(t, clockInPath, tok, bad), http.StatusBadRequest, "INVALID_WORK_MODE")

	bad = pointForm(near100m)
	bad.fields["work_schedule_id"] = "bukan-uuid"
	expectProblem(t, f.do(t, clockInPath, tok, bad), http.StatusBadRequest, "INVALID_SCHEDULE_ID")

	req := httptest.NewRequest(http.MethodPost, clockInPath, strings.NewReader(`{"latitude":1}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tok)
	rec := httptest.NewRecorder()
	f.router.ServeHTTP(rec, req)
	expectProblem(t, rec, http.StatusBadRequest, "INVALID_FORM")
}

func TestClockIn_PhotoValidation(t *testing.T) {
	f := newAttFixture(t)
	tok := f.token(t, "employee", attTenant)

	big := pointForm(near100m)
	big.selfie = append(append([]byte{}, jpegBytes...), make([]byte, storage.MaxPhotoBytes)...)
	expectProblem(t, f.do(t, clockInPath, tok, big), http.StatusRequestEntityTooLarge, "PHOTO_TOO_LARGE")

	notImage := pointForm(near100m)
	notImage.selfie = []byte("<html>bukan foto</html>")
	expectProblem(t, f.do(t, clockInPath, tok, notImage), http.StatusUnsupportedMediaType, "PHOTO_UNSUPPORTED")

	if len(f.store.objects) != 0 {
		t.Fatal("foto tidak valid tidak boleh diunggah")
	}
}

func TestClockIn_RemoteMode(t *testing.T) {
	f := newAttFixture(t)
	tok := f.token(t, "employee", attTenant)

	wfh := pointForm(far500m)
	wfh.fields["work_mode"] = "wfh"
	wfh.selfie = jpegBytes
	expectProblem(t, f.do(t, clockInPath, tok, wfh), http.StatusUnprocessableEntity, "REMOTE_NOT_ALLOWED")

	f.schedule().AllowRemote = true
	noSelfie := pointForm(far500m)
	noSelfie.fields["work_mode"] = "wfh"
	expectProblem(t, f.do(t, clockInPath, tok, noSelfie), http.StatusUnprocessableEntity, "SELFIE_REQUIRED")

	rec := f.do(t, clockInPath, tok, wfh)
	if rec.Code != http.StatusCreated {
		t.Fatalf("WFH diizinkan: status %d; body=%s", rec.Code, rec.Body.String())
	}
	resp := decodeAttendance(t, rec)
	if resp.WorkMode != "wfh" || resp.OfficeLocationId != nil || resp.DistanceInMeters != nil || !resp.HasSelfieIn {
		t.Fatalf("respons WFH tidak sesuai: %+v", resp)
	}
}

func TestClockIn_StorageProblems(t *testing.T) {
	t.Run("storage gagal", func(t *testing.T) {
		f := newAttFixture(t)
		f.store.failPut = true
		in := pointForm(near100m)
		in.selfie = jpegBytes
		expectProblem(t, f.do(t, clockInPath, f.token(t, "employee", attTenant), in),
			http.StatusServiceUnavailable, "STORAGE_UNAVAILABLE")
		if rec, _ := f.repo.FindByUserAndDate(context.Background(), attTenant, attUser, f.now); rec != nil {
			t.Fatal("presensi tidak boleh tersimpan bila selfie gagal diunggah")
		}
	})
	t.Run("storage tidak dikonfigurasi", func(t *testing.T) {
		f := newAttFixture(t)
		f.location().SelfieRequired = true
		srv := NewServer(&mockUserRepo{}, f.locs, f.scheds, f.jwt, auth.NewLoginLockManager(), auth.NewPasswordResetManager())
		_ = srv.EnableAttendance(AttendanceDeps{Repo: f.repo, Timezone: f.wita, Now: func() time.Time { return f.now }})
		f.router = gin.New()
		api.RegisterHandlers(f.router, srv)

		in := pointForm(near100m)
		in.selfie = jpegBytes
		expectProblem(t, f.do(t, clockInPath, f.token(t, "employee", attTenant), in),
			http.StatusServiceUnavailable, "STORAGE_UNAVAILABLE")
	})
}

func TestClockIn_ScheduleSelection(t *testing.T) {
	f := newAttFixture(t)
	tok := f.token(t, "employee", attTenant)

	second := &repository.WorkSchedule{
		ID: uuid.NewString(), TenantID: attTenant, StartTime: "09:00", EndTime: "18:00",
		ToleranceMinutes: 0, IsActive: true,
	}
	f.scheds.schedules[second.ID] = second
	expectProblem(t, f.do(t, clockInPath, tok, pointForm(near100m)), http.StatusUnprocessableEntity, "SCHEDULE_AMBIGUOUS")

	// Jadwal tenant lain tidak boleh dipakai.
	foreign := &repository.WorkSchedule{ID: uuid.NewString(), TenantID: uuid.NewString(), StartTime: "08:00", EndTime: "17:00", IsActive: true}
	f.scheds.schedules[foreign.ID] = foreign
	in := pointForm(near100m)
	in.fields["work_schedule_id"] = foreign.ID
	expectProblem(t, f.do(t, clockInPath, tok, in), http.StatusBadRequest, "SCHEDULE_NOT_FOUND")

	in.fields["work_schedule_id"] = second.ID
	rec := f.do(t, clockInPath, tok, in)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d; body=%s", rec.Code, rec.Body.String())
	}
	resp := decodeAttendance(t, rec)
	if resp.WorkScheduleId == nil || resp.WorkScheduleId.String() != second.ID || resp.Status != api.OnTime {
		t.Fatalf("jadwal terpilih salah: %+v", resp)
	}
}

func TestClockIn_NoScheduleOrLocation(t *testing.T) {
	f := newAttFixture(t)
	tok := f.token(t, "employee", attTenant)

	f.schedule().IsActive = false
	expectProblem(t, f.do(t, clockInPath, tok, pointForm(near100m)), http.StatusUnprocessableEntity, "SCHEDULE_NOT_SET")

	f.schedule().IsActive = true
	f.location().IsActive = false
	expectProblem(t, f.do(t, clockInPath, tok, pointForm(near100m)), http.StatusUnprocessableEntity, "LOCATION_NOT_SET")
}

func TestAttendance_Disabled(t *testing.T) {
	jwtManager, _ := auth.NewJWTManager("", "")
	srv := NewServer(&mockUserRepo{}, &mockLocationRepo{}, &mockScheduleRepo{}, jwtManager,
		auth.NewLoginLockManager(), auth.NewPasswordResetManager())
	r := gin.New()
	api.RegisterHandlers(r, srv)

	tok, _, _ := jwtManager.GenerateToken(attUser, attTenant, "employee")
	f := &attFixture{router: r}
	expectProblem(t, f.do(t, clockInPath, tok, pointForm(near100m)), http.StatusServiceUnavailable, "ATTENDANCE_DISABLED")

	if err := srv.EnableAttendance(AttendanceDeps{}); err == nil {
		t.Fatal("EnableAttendance tanpa repository seharusnya error")
	}
}

// ---------------------------------------------------------------- clock-out

func clockInAt(t *testing.T, f *attFixture, at time.Time) {
	t.Helper()
	f.now = at
	if rec := f.do(t, clockInPath, f.token(t, "employee", attTenant), pointForm(near100m)); rec.Code != http.StatusCreated {
		t.Fatalf("clock-in persiapan gagal: %d %s", rec.Code, rec.Body.String())
	}
}

func TestClockOut_OvertimeAndEarlyLeave(t *testing.T) {
	tests := []struct {
		name     string
		out      time.Time
		early    int
		overtime int
	}{
		{"lembur 90 menit", time.Date(2026, 11, 11, 18, 30, 0, 0, time.UTC), 0, 90},
		{"pulang cepat 60 menit", time.Date(2026, 11, 11, 16, 0, 0, 0, time.UTC), 60, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newAttFixture(t)
			clockInAt(t, f, time.Date(2026, 11, 11, 7, 58, 0, 0, f.wita))
			f.now = time.Date(tt.out.Year(), tt.out.Month(), tt.out.Day(), tt.out.Hour(), tt.out.Minute(), 0, 0, f.wita)

			rec := f.do(t, clockOutPath, f.token(t, "employee", attTenant), pointForm(near100m))
			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d; body=%s", rec.Code, rec.Body.String())
			}
			resp := decodeAttendance(t, rec)
			if resp.ClockOut == nil || resp.EarlyLeaveMinutes != tt.early || resp.OvertimeMinutes != tt.overtime {
				t.Fatalf("got clock_out=%v early=%d overtime=%d", resp.ClockOut, resp.EarlyLeaveMinutes, resp.OvertimeMinutes)
			}
			if resp.DistanceOutMeters == nil || *resp.DistanceOutMeters != 100 {
				t.Fatalf("distance_out = %v, want 100", resp.DistanceOutMeters)
			}
		})
	}
}

func TestClockOut_Errors(t *testing.T) {
	f := newAttFixture(t)
	tok := f.token(t, "employee", attTenant)

	expectProblem(t, f.do(t, clockOutPath, tok, pointForm(near100m)), http.StatusNotFound, "NOT_CLOCKED_IN")

	clockInAt(t, f, time.Date(2026, 11, 11, 8, 0, 0, 0, f.wita))
	f.now = time.Date(2026, 11, 11, 17, 5, 0, 0, f.wita)

	expectProblem(t, f.do(t, clockOutPath, tok, pointForm(far500m)), http.StatusUnprocessableEntity, "OUTSIDE_GEOFENCE")

	if rec := f.do(t, clockOutPath, tok, pointForm(near100m)); rec.Code != http.StatusOK {
		t.Fatalf("clock-out pertama: %d %s", rec.Code, rec.Body.String())
	}
	expectProblem(t, f.do(t, clockOutPath, tok, pointForm(near100m)), http.StatusConflict, "ALREADY_CLOCKED_OUT")
}

func TestClockOut_SelfieRequiredAndStored(t *testing.T) {
	f := newAttFixture(t)
	tok := f.token(t, "employee", attTenant)
	clockInAt(t, f, time.Date(2026, 11, 11, 8, 0, 0, 0, f.wita))
	f.location().SelfieRequired = true
	f.now = time.Date(2026, 11, 11, 17, 0, 0, 0, f.wita)

	expectProblem(t, f.do(t, clockOutPath, tok, pointForm(near100m)), http.StatusUnprocessableEntity, "SELFIE_REQUIRED")

	in := pointForm(near100m)
	in.selfie = jpegBytes
	rec := f.do(t, clockOutPath, tok, in)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d; body=%s", rec.Code, rec.Body.String())
	}
	if resp := decodeAttendance(t, rec); !resp.HasSelfieOut {
		t.Fatal("has_selfie_out seharusnya true")
	}
}

func TestClockOut_NightShiftAfterMidnight(t *testing.T) {
	f := newAttFixture(t)
	s := f.schedule()
	s.StartTime, s.EndTime = "22:00", "06:00"
	s.OvertimeAfterMinutes = 30

	clockInAt(t, f, time.Date(2026, 11, 11, 21, 55, 0, 0, f.wita))
	// Clock-out 07:00 keesokan harinya: presensi kemarin yang masih terbuka harus ditemukan.
	f.now = time.Date(2026, 11, 12, 7, 0, 0, 0, f.wita)

	rec := f.do(t, clockOutPath, f.token(t, "employee", attTenant), pointForm(near100m))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d; body=%s", rec.Code, rec.Body.String())
	}
	resp := decodeAttendance(t, rec)
	if resp.WorkDate.Format(time.DateOnly) != "2026-11-11" || resp.OvertimeMinutes != 60 {
		t.Fatalf("work_date=%s overtime=%d, want 2026-11-11 / 60", resp.WorkDate.Format(time.DateOnly), resp.OvertimeMinutes)
	}
}

func TestClockOut_ClockOutBeforeClockIn_Returns400(t *testing.T) {
	f := newAttFixture(t)
	tok := f.token(t, "employee", attTenant)
	clockInAt(t, f, time.Date(2026, 11, 11, 8, 0, 0, 0, f.wita))

	// Mencoba clock-out sebelum jam clock-in
	f.now = time.Date(2026, 11, 11, 7, 50, 0, 0, f.wita)
	expectProblem(t, f.do(t, clockOutPath, tok, pointForm(near100m)), http.StatusBadRequest, "INVALID_CLOCK_OUT_TIME")
}

func TestClockOut_WFH_SkipsGeofence(t *testing.T) {
	f := newAttFixture(t)
	tok := f.token(t, "employee", attTenant)
	f.schedule().AllowRemote = true

	// Clock-in mode WFH dari jarak jauh
	in := pointForm(far500m)
	in.fields["work_mode"] = "wfh"
	in.selfie = jpegBytes
	f.now = time.Date(2026, 11, 11, 8, 0, 0, 0, f.wita)
	rec := f.do(t, clockInPath, tok, in)
	if rec.Code != http.StatusCreated {
		t.Fatalf("clock-in wfh: status %d body %s", rec.Code, rec.Body.String())
	}

	// Clock-out mode WFH dari jarak jauh dengan selfie
	f.now = time.Date(2026, 11, 11, 17, 0, 0, 0, f.wita)
	out := pointForm(far500m)
	out.selfie = jpegBytes
	recOut := f.do(t, clockOutPath, tok, out)
	if recOut.Code != http.StatusOK {
		t.Fatalf("clock-out wfh: status %d body %s", recOut.Code, recOut.Body.String())
	}
	resp := decodeAttendance(t, recOut)
	if resp.ClockOut == nil || resp.WorkMode != api.AttendanceResponseWorkModeWfh {
		t.Fatalf("unexpected clock-out response: %+v", resp)
	}
}

func TestDiscardSelfie_IgnoresNil(t *testing.T) {
	s := &Server{}
	s.discardSelfie(nil) // tidak boleh panic
	key := "x"
	s.discardSelfie(&key) // store nil, tidak boleh panic
}
