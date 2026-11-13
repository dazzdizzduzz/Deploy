package handler

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/tiers-undiknas/capstone-group07-2026/backend/api"
	"github.com/tiers-undiknas/capstone-group07-2026/backend/internal/auth"
	"github.com/tiers-undiknas/capstone-group07-2026/backend/internal/repository"
)

type mockLocationRepo struct {
	locations map[string]*repository.OfficeLocation
}

func (m *mockLocationRepo) Create(ctx context.Context, loc *repository.OfficeLocation) error {
	if loc.ID == "" {
		loc.ID = uuid.NewString()
	}
	loc.CreatedAt = time.Now()
	loc.UpdatedAt = time.Now()
	m.locations[loc.ID] = loc
	return nil
}

func (m *mockLocationRepo) Update(ctx context.Context, loc *repository.OfficeLocation) error {
	existing, ok := m.locations[loc.ID]
	if !ok || existing.TenantID != loc.TenantID {
		return sql.ErrNoRows
	}
	loc.UpdatedAt = time.Now()
	m.locations[loc.ID] = loc
	return nil
}

func (m *mockLocationRepo) FindByID(ctx context.Context, tenantID, id string) (*repository.OfficeLocation, error) {
	loc, ok := m.locations[id]
	if !ok || loc.TenantID != tenantID {
		return nil, nil
	}
	return loc, nil
}

func (m *mockLocationRepo) FindAll(ctx context.Context, tenantID string) ([]*repository.OfficeLocation, error) {
	var list []*repository.OfficeLocation
	for _, l := range m.locations {
		if l.TenantID == tenantID {
			list = append(list, l)
		}
	}
	return list, nil
}

func (m *mockLocationRepo) Delete(ctx context.Context, tenantID, id string) error {
	loc, ok := m.locations[id]
	if !ok || loc.TenantID != tenantID {
		return sql.ErrNoRows
	}
	delete(m.locations, id)
	return nil
}

type mockScheduleRepo struct {
	schedules map[string]*repository.WorkSchedule
}

func (m *mockScheduleRepo) Create(ctx context.Context, sched *repository.WorkSchedule) error {
	if sched.ID == "" {
		sched.ID = uuid.NewString()
	}
	sched.CreatedAt = time.Now()
	sched.UpdatedAt = time.Now()
	m.schedules[sched.ID] = sched
	return nil
}

func (m *mockScheduleRepo) Update(ctx context.Context, sched *repository.WorkSchedule) error {
	existing, ok := m.schedules[sched.ID]
	if !ok || existing.TenantID != sched.TenantID {
		return sql.ErrNoRows
	}
	sched.UpdatedAt = time.Now()
	m.schedules[sched.ID] = sched
	return nil
}

func (m *mockScheduleRepo) FindByID(ctx context.Context, tenantID, id string) (*repository.WorkSchedule, error) {
	sched, ok := m.schedules[id]
	if !ok || sched.TenantID != tenantID {
		return nil, nil
	}
	return sched, nil
}

func (m *mockScheduleRepo) FindAll(ctx context.Context, tenantID string) ([]*repository.WorkSchedule, error) {
	var list []*repository.WorkSchedule
	for _, s := range m.schedules {
		if s.TenantID == tenantID {
			list = append(list, s)
		}
	}
	return list, nil
}

func (m *mockScheduleRepo) Delete(ctx context.Context, tenantID, id string) error {
	sched, ok := m.schedules[id]
	if !ok || sched.TenantID != tenantID {
		return sql.ErrNoRows
	}
	delete(m.schedules, id)
	return nil
}

func setupLocationScheduleRouter() (*gin.Engine, *mockLocationRepo, *mockScheduleRepo, string, string) {
	gin.SetMode(gin.TestMode)

	userRepo := &mockUserRepo{users: make(map[string]*repository.User)}
	locRepo := &mockLocationRepo{locations: make(map[string]*repository.OfficeLocation)}
	schedRepo := &mockScheduleRepo{schedules: make(map[string]*repository.WorkSchedule)}

	jwtManager, _ := auth.NewJWTManager("", "")
	loginLock := auth.NewLoginLockManager()
	passwordReset := auth.NewPasswordResetManager()

	srv := NewServer(userRepo, locRepo, schedRepo, jwtManager, loginLock, passwordReset)

	r := gin.New()
	api.RegisterHandlers(r, srv)

	tenantID := "a0000000-0000-0000-0000-000000000001"
	token, _, _ := jwtManager.GenerateToken("user-1", tenantID, "owner")

	return r, locRepo, schedRepo, tenantID, token
}

// -----------------------------------------------------------------------------
// LOKASI KANTOR (FR-05) TESTS
// -----------------------------------------------------------------------------

func TestPostLocations_Success(t *testing.T) {
	r, _, _, _, token := setupLocationScheduleRouter()

	body, _ := json.Marshal(map[string]interface{}{
		"name":            "Kantor Sanur",
		"address":         "Jl. Danau Tamblingan No. 226, Sanur",
		"latitude":        -8.6923,
		"longitude":       115.2621,
		"radius_meters":   100,
		"selfie_required": true,
	})

	req, _ := http.NewRequest(http.MethodPost, "/locations", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp api.OfficeLocationResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Name != "Kantor Sanur" {
		t.Errorf("expected name 'Kantor Sanur', got '%s'", resp.Name)
	}
	if resp.RadiusMeters != 100 {
		t.Errorf("expected radius 100, got %d", resp.RadiusMeters)
	}
}

func TestPostLocations_ValidationRadiusMeters_UnderLimit(t *testing.T) {
	r, _, _, _, token := setupLocationScheduleRouter()

	// Radius < 25 meter harus ditolak HTTP 422
	body, _ := json.Marshal(map[string]interface{}{
		"name":          "Kantor Kecil",
		"address":       "Jl. Kecil",
		"latitude":      -8.69,
		"longitude":     115.26,
		"radius_meters": 10,
	})

	req, _ := http.NewRequest(http.MethodPost, "/locations", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected status 422 for radius < 25, got %d: %s", w.Code, w.Body.String())
	}
}

func TestPostLocations_ValidationRadiusMeters_OverLimit(t *testing.T) {
	r, _, _, _, token := setupLocationScheduleRouter()

	// Radius > 500 meter harus ditolak HTTP 422
	body, _ := json.Marshal(map[string]interface{}{
		"name":          "Kantor Luas",
		"address":       "Jl. Luas",
		"latitude":      -8.69,
		"longitude":     115.26,
		"radius_meters": 600,
	})

	req, _ := http.NewRequest(http.MethodPost, "/locations", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected status 422 for radius > 500, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUpdateLocation_SuccessAndValidation(t *testing.T) {
	r, locRepo, _, tenantID, token := setupLocationScheduleRouter()

	locID := uuid.NewString()
	locRepo.locations[locID] = &repository.OfficeLocation{
		ID:             locID,
		TenantID:       tenantID,
		Name:           "Kantor Lama",
		Address:        "Alamat Lama",
		Latitude:       -8.1,
		Longitude:      115.1,
		RadiusMeters:   100,
		SelfieRequired: false,
		IsActive:       true,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	// 1. Update sukses via PUT /locations/{id}
	putBody, _ := json.Marshal(map[string]interface{}{
		"name":          "Kantor Diperbarui",
		"radius_meters": 200,
	})
	req, _ := http.NewRequest(http.MethodPut, "/locations/"+locID, bytes.NewReader(putBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200 on update, got %d: %s", w.Code, w.Body.String())
	}

	var resp api.OfficeLocationResponse
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Name != "Kantor Diperbarui" || resp.RadiusMeters != 200 {
		t.Errorf("update didn't reflect properly: %+v", resp)
	}

	// 2. Update gagal karena radius > 500
	invalidBody, _ := json.Marshal(map[string]interface{}{
		"radius_meters": 800,
	})
	reqInvalid, _ := http.NewRequest(http.MethodPatch, "/locations/"+locID, bytes.NewReader(invalidBody))
	reqInvalid.Header.Set("Content-Type", "application/json")
	reqInvalid.Header.Set("Authorization", "Bearer "+token)

	wInvalid := httptest.NewRecorder()
	r.ServeHTTP(wInvalid, reqInvalid)

	if wInvalid.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected status 422 for radius > 500 on patch, got %d: %s", wInvalid.Code, wInvalid.Body.String())
	}

	// 3. Update 404 untuk ID yang tidak ada
	nonExistentID := uuid.NewString()
	req404, _ := http.NewRequest(http.MethodPut, "/locations/"+nonExistentID, bytes.NewReader(putBody))
	req404.Header.Set("Content-Type", "application/json")
	req404.Header.Set("Authorization", "Bearer "+token)

	w404 := httptest.NewRecorder()
	r.ServeHTTP(w404, req404)

	if w404.Code != http.StatusNotFound {
		t.Fatalf("expected status 404 for unknown location id, got %d", w404.Code)
	}
}

func TestGetLocations_And_Delete(t *testing.T) {
	r, locRepo, _, tenantID, token := setupLocationScheduleRouter()

	locID := uuid.NewString()
	locRepo.locations[locID] = &repository.OfficeLocation{
		ID:             locID,
		TenantID:       tenantID,
		Name:           "Kantor Pusat",
		Address:        "Alamat",
		Latitude:       -8.1,
		Longitude:      115.1,
		RadiusMeters:   100,
		SelfieRequired: false,
		IsActive:       true,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	// List locations
	reqList, _ := http.NewRequest(http.MethodGet, "/locations", nil)
	reqList.Header.Set("Authorization", "Bearer "+token)
	wList := httptest.NewRecorder()
	r.ServeHTTP(wList, reqList)

	if wList.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", wList.Code)
	}

	var listResp api.OfficeLocationsListResponse
	_ = json.Unmarshal(wList.Body.Bytes(), &listResp)
	if len(listResp.Data) != 1 {
		t.Errorf("expected 1 location in list, got %d", len(listResp.Data))
	}

	// Detail location
	reqDetail, _ := http.NewRequest(http.MethodGet, "/locations/"+locID, nil)
	reqDetail.Header.Set("Authorization", "Bearer "+token)
	wDetail := httptest.NewRecorder()
	r.ServeHTTP(wDetail, reqDetail)
	if wDetail.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", wDetail.Code)
	}

	// Delete location
	reqDel, _ := http.NewRequest(http.MethodDelete, "/locations/"+locID, nil)
	reqDel.Header.Set("Authorization", "Bearer "+token)
	wDel := httptest.NewRecorder()
	r.ServeHTTP(wDel, reqDel)
	if wDel.Code != http.StatusOK {
		t.Fatalf("expected 200 on delete, got %d", wDel.Code)
	}
}

// -----------------------------------------------------------------------------
// JADWAL KERJA TESTS
// -----------------------------------------------------------------------------

func TestPostSchedules_Success(t *testing.T) {
	r, _, _, _, token := setupLocationScheduleRouter()

	body, _ := json.Marshal(map[string]interface{}{
		"name":                   "Shift Pagi Reguler",
		"start_time":             "08:00",
		"end_time":               "17:00",
		"tolerance_minutes":      15,
		"overtime_after_minutes": 60,
		"max_overtime_minutes":   180,
		"working_days":           "Sen,Sel,Rab,Kam,Jum",
		"allow_remote":           false,
	})

	req, _ := http.NewRequest(http.MethodPost, "/schedules", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp api.WorkScheduleResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Name != "Shift Pagi Reguler" {
		t.Errorf("expected name 'Shift Pagi Reguler', got '%s'", resp.Name)
	}
	if resp.StartTime != "08:00" || resp.EndTime != "17:00" {
		t.Errorf("expected 08:00 - 17:00, got %s - %s", resp.StartTime, resp.EndTime)
	}
}

func TestPostSchedules_ValidationTimeFormat(t *testing.T) {
	r, _, _, _, token := setupLocationScheduleRouter()

	// Format jam tidak valid (bukan HH:MM)
	body, _ := json.Marshal(map[string]interface{}{
		"name":       "Shift Rusak",
		"start_time": "jam 8 pagi",
		"end_time":   "17:00",
	})

	req, _ := http.NewRequest(http.MethodPost, "/schedules", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected status 422 for invalid start_time, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUpdateSchedule_SuccessAndValidation(t *testing.T) {
	r, _, schedRepo, tenantID, token := setupLocationScheduleRouter()

	schedID := uuid.NewString()
	schedRepo.schedules[schedID] = &repository.WorkSchedule{
		ID:                   schedID,
		TenantID:             tenantID,
		Name:                 "Shift Pagi",
		StartTime:            "08:00",
		EndTime:              "17:00",
		ToleranceMinutes:     15,
		OvertimeAfterMinutes: 60,
		MaxOvertimeMinutes:   180,
		WorkingDays:          "Sen,Sel,Rab,Kam,Jum",
		AllowRemote:          false,
		IsActive:             true,
		CreatedAt:            time.Now(),
		UpdatedAt:            time.Now(),
	}

	// Update sukses
	putBody, _ := json.Marshal(map[string]interface{}{
		"name":       "Shift Siang",
		"start_time": "12:00",
		"end_time":   "21:00",
	})
	req, _ := http.NewRequest(http.MethodPut, "/schedules/"+schedID, bytes.NewReader(putBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp api.WorkScheduleResponse
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Name != "Shift Siang" || resp.StartTime != "12:00" {
		t.Errorf("update didn't reflect properly: %+v", resp)
	}

	// Update invalid time
	invalidBody, _ := json.Marshal(map[string]interface{}{
		"end_time": "invalid-time",
	})
	reqInvalid, _ := http.NewRequest(http.MethodPatch, "/schedules/"+schedID, bytes.NewReader(invalidBody))
	reqInvalid.Header.Set("Content-Type", "application/json")
	reqInvalid.Header.Set("Authorization", "Bearer "+token)

	wInvalid := httptest.NewRecorder()
	r.ServeHTTP(wInvalid, reqInvalid)

	if wInvalid.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected status 422 for invalid end_time on patch, got %d", wInvalid.Code)
	}
}

