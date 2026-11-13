package handler

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/tiers-undiknas/capstone-group07-2026/backend/api"
	"github.com/tiers-undiknas/capstone-group07-2026/backend/internal/attendance"
	"github.com/tiers-undiknas/capstone-group07-2026/backend/internal/models"
	"github.com/tiers-undiknas/capstone-group07-2026/backend/internal/repository"
	"github.com/tiers-undiknas/capstone-group07-2026/backend/internal/storage"
)

// maxAttendanceBody membatasi total badan request multipart: foto 2 MB + ruang untuk field
// teks dan header multipart. Request yang lebih besar langsung dipotong (413) sebelum dibaca.
const maxAttendanceBody = storage.MaxPhotoBytes + 512<<10

// cleanupTimeout dipakai untuk menghapus selfie yatim bila penyimpanan presensi gagal.
const cleanupTimeout = 5 * time.Second

// attendanceRoles adalah peran yang boleh clock-in/out (x-rbac di openapi.yaml).
// super_admin tidak termasuk karena tidak terikat ke tenant.
var attendanceRoles = map[models.UserRole]bool{
	models.RoleOwner:    true,
	models.RoleHRD:      true,
	models.RoleTeamLead: true,
	models.RoleEmployee: true,
	models.RoleCashier:  true,
}

// AttendanceDeps adalah dependensi modul absensi (ISSUE-62/64). Dipasang lewat
// EnableAttendance agar konstruktor NewServer milik modul lain tidak perlu diubah.
type AttendanceDeps struct {
	Repo     repository.AttendanceRepository
	Store    storage.ObjectStore // boleh nil; selfie yang wajib akan dijawab 503
	Timezone *time.Location      // zona waktu tenant; nil berarti attendance.DefaultTimezone
	Now      func() time.Time    // diganti di test; nil berarti time.Now
}

// EnableAttendance mengaktifkan endpoint clock-in/clock-out.
func (s *Server) EnableAttendance(d AttendanceDeps) error {
	if d.Repo == nil {
		return errors.New("attendance repository wajib diisi")
	}
	if d.Timezone == nil {
		loc, err := attendance.LoadTimezone("")
		if err != nil {
			return err
		}
		d.Timezone = loc
	}
	if d.Now == nil {
		d.Now = time.Now
	}
	s.attendance = d
	return nil
}

// ---------------------------------------------------------------------------
// POST /attendances/clock-in (ISSUE-62: FR-06, FR-08, FR-09)
// ---------------------------------------------------------------------------

func (s *Server) PostAttendanceClockIn(c *gin.Context) {
	tenantID, userID, ok := s.attendanceIdentity(c)
	if !ok {
		return
	}
	if !s.attendanceReady(c) {
		return
	}
	form, ok := parseAttendanceForm(c)
	if !ok {
		return
	}
	ctx := c.Request.Context()
	deps := s.attendance

	// 1. Mode kerja (FR-09). Default office.
	mode := strings.TrimSpace(form.value("work_mode"))
	if mode == "" {
		mode = string(api.ClockInRequestWorkModeOffice)
	}
	switch api.ClockInRequestWorkMode(mode) {
	case api.ClockInRequestWorkModeOffice, api.ClockInRequestWorkModeWfh, api.ClockInRequestWorkModeBusinessTrip:
	default:
		writeProblem(c, http.StatusBadRequest, "INVALID_WORK_MODE", "Mode kerja tidak valid",
			"work_mode harus office, wfh, atau business_trip.", nil)
		return
	}

	// 2. Jadwal kerja yang berlaku.
	sched, ok := s.resolveSchedule(c, tenantID, form.value("work_schedule_id"))
	if !ok {
		return
	}

	// 3. Status on_time/late dan tanggal kerja (FR-08).
	now := deps.Now().UTC()
	eval, err := attendance.EvaluateClockIn(now, toRuleSchedule(sched), deps.Timezone)
	if err != nil {
		log.Printf("attendance: jadwal %s tidak valid: %v", sched.ID, err)
		writeProblem(c, http.StatusUnprocessableEntity, "INVALID_SCHEDULE", "Jadwal kerja tidak valid",
			"Format jam pada jadwal kerja tidak dapat dibaca. Hubungi HRD.", nil)
		return
	}

	// 4. Cek presensi ganda lebih awal agar selfie tidak diunggah sia-sia.
	//    Constraint UNIQUE di database tetap menjadi penjaga terakhir (race condition).
	existing, err := deps.Repo.FindByUserAndDate(ctx, tenantID, userID, eval.WorkDate)
	if err != nil {
		writeInternal(c, "cek presensi", err)
		return
	}
	if existing != nil {
		writeProblem(c, http.StatusConflict, "ALREADY_CLOCKED_IN", "Sudah clock-in",
			"Anda sudah clock-in pada tanggal kerja ini.", nil)
		return
	}

	// 5. Geofence (FR-06) atau izin kerja remote (FR-09).
	rec := &repository.Attendance{
		TenantID:    tenantID,
		UserID:      userID,
		WorkDate:    eval.WorkDate,
		WorkMode:    mode,
		ClockIn:     now,
		LatitudeIn:  form.point.Lat,
		LongitudeIn: form.point.Lon,
		Status:      string(eval.Status),
		LateMinutes: eval.LateMinutes,
	}
	schedID := sched.ID
	rec.WorkScheduleID = &schedID

	selfieRequired := true // mode remote selalu wajib selfie sebagai bukti kehadiran
	if mode == string(api.ClockInRequestWorkModeOffice) {
		loc, dist, ok := s.nearestLocationInside(c, tenantID, form.point)
		if !ok {
			return
		}
		locID := loc.ID
		rec.OfficeLocationID = &locID
		rec.DistanceInMeters = &dist
		selfieRequired = loc.SelfieRequired
	} else if !sched.AllowRemote {
		writeProblem(c, http.StatusUnprocessableEntity, "REMOTE_NOT_ALLOWED", "Kerja remote tidak diizinkan",
			"Jadwal kerja Anda tidak mengizinkan clock-in WFH atau dinas luar.", nil)
		return
	}

	// 6. Selfie (Backend 2: object storage).
	key, ok := s.storeSelfie(c, tenantID, form.selfie, selfieRequired)
	if !ok {
		return
	}
	rec.SelfieInKey = key

	// 7. Simpan.
	if err := deps.Repo.CreateClockIn(ctx, rec); err != nil {
		s.discardSelfie(key)
		if errors.Is(err, repository.ErrAlreadyClockedIn) {
			writeProblem(c, http.StatusConflict, "ALREADY_CLOCKED_IN", "Sudah clock-in",
				"Anda sudah clock-in pada tanggal kerja ini.", nil)
			return
		}
		writeInternal(c, "simpan clock-in", err)
		return
	}

	c.JSON(http.StatusCreated, toAttendanceResponse(rec))
}

// ---------------------------------------------------------------------------
// POST /attendances/clock-out (ISSUE-64: FR-07, FR-08)
// ---------------------------------------------------------------------------

func (s *Server) PostAttendanceClockOut(c *gin.Context) {
	tenantID, userID, ok := s.attendanceIdentity(c)
	if !ok {
		return
	}
	if !s.attendanceReady(c) {
		return
	}
	form, ok := parseAttendanceForm(c)
	if !ok {
		return
	}
	ctx := c.Request.Context()
	deps := s.attendance
	now := deps.Now().UTC()

	// 1. Cari presensi yang masih terbuka: hari ini, atau kemarin untuk shift malam
	//    yang clock-out-nya lewat tengah malam.
	rec, err := s.findOpenAttendance(ctx, tenantID, userID, now)
	if err != nil {
		writeInternal(c, "cari presensi", err)
		return
	}
	if rec == nil {
		writeProblem(c, http.StatusNotFound, "NOT_CLOCKED_IN", "Belum clock-in",
			"Tidak ada presensi yang bisa di-clock-out. Lakukan clock-in terlebih dahulu.", nil)
		return
	}
	if rec.ClockOut != nil {
		writeProblem(c, http.StatusConflict, "ALREADY_CLOCKED_OUT", "Sudah clock-out",
			"Anda sudah clock-out pada tanggal kerja ini.", nil)
		return
	}
	if now.Before(rec.ClockIn) {
		writeProblem(c, http.StatusBadRequest, "INVALID_CLOCK_OUT_TIME", "Waktu tidak valid",
			"Waktu clock-out tidak boleh mendahului waktu clock-in.", nil)
		return
	}

	update := repository.ClockOutUpdate{
		TenantID:     tenantID,
		AttendanceID: rec.ID,
		ClockOut:     now,
		LatitudeOut:  form.point.Lat,
		LongitudeOut: form.point.Lon,
	}

	// 2. Geofence ke lokasi yang sama dengan clock-in (mode office).
	selfieRequired := true
	if rec.WorkMode == string(api.ClockInRequestWorkModeOffice) {
		selfieRequired = false
		if rec.OfficeLocationID != nil {
			loc, err := s.locationRepo.FindByID(ctx, tenantID, *rec.OfficeLocationID)
			if err != nil {
				writeInternal(c, "ambil lokasi kantor", err)
				return
			}
			// Lokasi bisa saja sudah dihapus HRD setelah clock-in; clock-out tetap diizinkan
			// tanpa pemeriksaan radius agar karyawan tidak terkunci.
			if loc != nil {
				geo, err := attendance.CheckGeofence(form.point,
					attendance.Point{Lat: loc.Latitude, Lon: loc.Longitude}, loc.RadiusMeters)
				if err != nil {
					writeInternal(c, "geofence clock-out", err)
					return
				}
				if !geo.Inside {
					writeOutsideGeofence(c, loc, geo.DistanceMeters)
					return
				}
				d := geo.DistanceMeters
				update.DistanceOutMeters = &d
				selfieRequired = loc.SelfieRequired
			}
		}
	}

	// 3. Pulang cepat / lembur dari jadwal saat clock-in.
	var sched *repository.WorkSchedule
	if rec.WorkScheduleID != nil {
		sSched, err := s.scheduleRepo.FindByID(ctx, tenantID, *rec.WorkScheduleID)
		if err != nil {
			writeInternal(c, "ambil jadwal kerja", err)
			return
		}
		sched = sSched
		// Bila jadwal sudah dihapus, menit pulang cepat/lembur dibiarkan 0.
		if sched != nil {
			out, err := attendance.EvaluateClockOut(rec.ClockIn, now, toRuleSchedule(sched), deps.Timezone)
			if err != nil {
				log.Printf("attendance: evaluasi clock-out %s: %v", rec.ID, err)
			} else {
				update.EarlyLeaveMinutes = out.EarlyLeaveMinutes
				update.OvertimeMinutes = out.OvertimeMinutes
			}
		}
	}

	// 4. Selfie.
	key, ok := s.storeSelfie(c, tenantID, form.selfie, selfieRequired)
	if !ok {
		return
	}
	update.SelfieOutKey = key

	// 5. Simpan. Repository hanya mengubah baris yang clock_out-nya masih NULL.
	updated, err := deps.Repo.UpdateClockOut(ctx, update)
	if err != nil {
		s.discardSelfie(key)
		if errors.Is(err, repository.ErrAttendanceNotOpen) {
			writeProblem(c, http.StatusConflict, "ALREADY_CLOCKED_OUT", "Sudah clock-out",
				"Anda sudah clock-out pada tanggal kerja ini.", nil)
			return
		}
		writeInternal(c, "simpan clock-out", err)
		return
	}

	// 6. Catat lembur otomatis jika ada kelebihan jam kerja (FR-09 / ISSUE-66).
	if s.overtime != nil && updated.OvertimeMinutes > 0 {
		startTimeStr := "17:00"
		if sched != nil && len(sched.EndTime) >= 5 {
			startTimeStr = sched.EndTime[:5]
		}
		endTimeStr := "17:00"
		if updated.ClockOut != nil {
			endTimeStr = updated.ClockOut.In(deps.Timezone).Format("15:04")
		}
		otReq := &repository.OvertimeRequest{
			TenantID:        tenantID,
			AttendanceID:    &updated.ID,
			UserID:          updated.UserID,
			Date:            updated.WorkDate,
			StartTime:       startTimeStr,
			EndTime:         endTimeStr,
			DurationMinutes: updated.OvertimeMinutes,
			Reason:          "Lembur tercatat otomatis saat clock-out",
			Status:          repository.OvertimeStatusPending,
		}
		if err := s.overtime.Create(ctx, otReq); err != nil {
			log.Printf("attendance: auto create overtime for %s failed: %v", updated.ID, err)
		}
	}

	c.JSON(http.StatusOK, toAttendanceResponse(updated))
}

// ---------------------------------------------------------------------------
// Helper
// ---------------------------------------------------------------------------

// attendanceIdentity membaca tenant dan user dari JWT untuk endpoint absensi.
func (s *Server) attendanceIdentity(c *gin.Context) (tenantID, userID string, ok bool) {
	who, ok := s.authenticate(c, attendanceRoles)
	return who.TenantID, who.UserID, ok
}

// caller adalah identitas pemanggil yang sudah divalidasi dari JWT.
type caller struct {
	TenantID string
	UserID   string
	Role     models.UserRole
}

// authenticate memvalidasi JWT, peran (x-rbac), dan tenant. Sengaja tidak menerima header
// X-Tenant-ID: tenant_id hanya boleh berasal dari klaim tid (konvensi kontrak API).
// Bila ok == false, respons error sudah ditulis.
func (s *Server) authenticate(c *gin.Context, allowed map[models.UserRole]bool) (caller, bool) {
	var claims *models.JWTClaims
	if v, exists := c.Get("claims"); exists {
		claims, _ = v.(*models.JWTClaims) // sudah divalidasi middleware Guard
	}
	if claims == nil {
		if s.jwtManager == nil {
			writeProblem(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Kesalahan server",
				"Layanan otentikasi belum diinisialisasi.", nil)
			return caller{}, false
		}
		parsed, err := s.jwtManager.ValidateRequest(c.Request)
		if err != nil {
			writeProblem(c, http.StatusUnauthorized, "UNAUTHORIZED", "Tidak terautentikasi",
				"Token otentikasi tidak ada, tidak valid, atau sudah kedaluwarsa.", nil)
			return caller{}, false
		}
		claims = parsed
	}

	userID := claims.Sub
	if userID == "" {
		userID = claims.Subject
	}
	if _, err := uuid.Parse(userID); err != nil {
		writeProblem(c, http.StatusUnauthorized, "UNAUTHORIZED", "Tidak terautentikasi",
			"Token tidak membawa identitas pengguna yang valid.", nil)
		return caller{}, false
	}
	if !allowed[claims.Role] {
		writeProblem(c, http.StatusForbidden, "FORBIDDEN", "Akses ditolak",
			fmt.Sprintf("Peran '%s' tidak berwenang untuk tindakan ini.", claims.Role), nil)
		return caller{}, false
	}
	if _, err := uuid.Parse(claims.Tid); err != nil {
		writeProblem(c, http.StatusForbidden, "TENANT_REQUIRED", "Akses ditolak",
			"Akun ini tidak terikat ke tenant mana pun.", nil)
		return caller{}, false
	}
	return caller{TenantID: claims.Tid, UserID: userID, Role: claims.Role}, true
}

func (s *Server) attendanceReady(c *gin.Context) bool {
	if s.attendance.Repo == nil {
		writeProblem(c, http.StatusServiceUnavailable, "ATTENDANCE_DISABLED", "Modul absensi belum aktif",
			"Server belum mengaktifkan modul absensi.", nil)
		return false
	}
	return true
}

type attendanceForm struct {
	point  attendance.Point
	values map[string][]string
	selfie *multipart.FileHeader
}

func (f attendanceForm) value(name string) string {
	if v := f.values[name]; len(v) > 0 {
		return v[0]
	}
	return ""
}

// parseAttendanceForm membaca multipart/form-data dengan batas ukuran total, lalu
// memvalidasi koordinat. Semua kegagalan sudah dijawab ke klien bila ok == false.
func parseAttendanceForm(c *gin.Context) (attendanceForm, bool) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxAttendanceBody)
	if err := c.Request.ParseMultipartForm(maxAttendanceBody); err != nil {
		var tooLarge *http.MaxBytesError
		switch {
		case errors.As(err, &tooLarge):
			writeProblem(c, http.StatusRequestEntityTooLarge, "PHOTO_TOO_LARGE", "Foto terlalu besar",
				"Ukuran foto selfie maksimal 2 MB.", nil)
		default:
			writeProblem(c, http.StatusBadRequest, "INVALID_FORM", "Permintaan tidak valid",
				"Kirim data sebagai multipart/form-data.", nil)
		}
		return attendanceForm{}, false
	}

	form := attendanceForm{values: c.Request.MultipartForm.Value}
	if files := c.Request.MultipartForm.File["selfie"]; len(files) > 0 {
		form.selfie = files[0]
	}

	lat, errLat := strconv.ParseFloat(strings.TrimSpace(form.value("latitude")), 64)
	lon, errLon := strconv.ParseFloat(strings.TrimSpace(form.value("longitude")), 64)
	form.point = attendance.Point{Lat: lat, Lon: lon}
	if errLat != nil || errLon != nil || form.point.Validate() != nil {
		writeProblem(c, http.StatusBadRequest, "INVALID_COORDINATE", "Koordinat tidak valid",
			"latitude (-90..90) dan longitude (-180..180) wajib diisi dalam derajat desimal.", nil)
		return attendanceForm{}, false
	}
	return form, true
}

// resolveSchedule memilih jadwal kerja. Selama tabel penugasan jadwal per karyawan belum ada,
// klien boleh mengirim work_schedule_id; bila tidak, dipakai satu-satunya jadwal aktif tenant.
func (s *Server) resolveSchedule(c *gin.Context, tenantID, rawID string) (*repository.WorkSchedule, bool) {
	ctx := c.Request.Context()
	rawID = strings.TrimSpace(rawID)

	if rawID != "" {
		if _, err := uuid.Parse(rawID); err != nil {
			writeProblem(c, http.StatusBadRequest, "INVALID_SCHEDULE_ID", "Jadwal tidak valid",
				"work_schedule_id harus UUID.", nil)
			return nil, false
		}
		sched, err := s.scheduleRepo.FindByID(ctx, tenantID, rawID)
		if err != nil {
			writeInternal(c, "ambil jadwal kerja", err)
			return nil, false
		}
		if sched == nil || !sched.IsActive {
			writeProblem(c, http.StatusBadRequest, "SCHEDULE_NOT_FOUND", "Jadwal tidak ditemukan",
				"Jadwal kerja tidak ditemukan atau tidak aktif.", nil)
			return nil, false
		}
		return sched, true
	}

	all, err := s.scheduleRepo.FindAll(ctx, tenantID)
	if err != nil {
		writeInternal(c, "daftar jadwal kerja", err)
		return nil, false
	}
	var active []*repository.WorkSchedule
	for _, sc := range all {
		if sc.IsActive {
			active = append(active, sc)
		}
	}
	switch len(active) {
	case 0:
		writeProblem(c, http.StatusUnprocessableEntity, "SCHEDULE_NOT_SET", "Jadwal kerja belum diatur",
			"Tenant belum memiliki jadwal kerja aktif. Hubungi HRD.", nil)
		return nil, false
	case 1:
		return active[0], true
	default:
		writeProblem(c, http.StatusUnprocessableEntity, "SCHEDULE_AMBIGUOUS", "Pilih jadwal kerja",
			"Tenant memiliki lebih dari satu jadwal aktif; kirim work_schedule_id.", nil)
		return nil, false
	}
}

// nearestLocationInside memilih lokasi kantor aktif terdekat dan memastikan karyawan
// berada di dalam radiusnya.
func (s *Server) nearestLocationInside(c *gin.Context, tenantID string, p attendance.Point) (*repository.OfficeLocation, int, bool) {
	all, err := s.locationRepo.FindAll(c.Request.Context(), tenantID)
	if err != nil {
		writeInternal(c, "daftar lokasi kantor", err)
		return nil, 0, false
	}

	var nearest *repository.OfficeLocation
	best := math.Inf(1)
	for _, loc := range all {
		if !loc.IsActive {
			continue
		}
		d := attendance.DistanceMeters(p, attendance.Point{Lat: loc.Latitude, Lon: loc.Longitude})
		if d < best {
			best, nearest = d, loc
		}
	}
	if nearest == nil {
		writeProblem(c, http.StatusUnprocessableEntity, "LOCATION_NOT_SET", "Lokasi kantor belum diatur",
			"Tenant belum memiliki lokasi kantor aktif. Hubungi HRD.", nil)
		return nil, 0, false
	}

	geo, err := attendance.CheckGeofence(p, attendance.Point{Lat: nearest.Latitude, Lon: nearest.Longitude}, nearest.RadiusMeters)
	if err != nil {
		writeInternal(c, "geofence clock-in", err)
		return nil, 0, false
	}
	if !geo.Inside {
		writeOutsideGeofence(c, nearest, geo.DistanceMeters)
		return nil, 0, false
	}
	return nearest, geo.DistanceMeters, true
}

// storeSelfie memvalidasi dan mengunggah selfie. Mengembalikan (nil, true) bila selfie
// tidak dikirim dan tidak wajib.
func (s *Server) storeSelfie(c *gin.Context, tenantID string, fh *multipart.FileHeader, required bool) (*string, bool) {
	if fh == nil {
		if required {
			writeProblem(c, http.StatusUnprocessableEntity, "SELFIE_REQUIRED", "Selfie wajib",
				"Lokasi atau mode kerja ini mewajibkan foto selfie.", nil)
			return nil, false
		}
		return nil, true
	}

	f, err := fh.Open()
	if err != nil {
		writeProblem(c, http.StatusBadRequest, "INVALID_FORM", "Permintaan tidak valid",
			"Berkas selfie tidak dapat dibaca.", nil)
		return nil, false
	}
	defer func() {
		_ = f.Close()
	}()

	photo, err := storage.ReadPhoto(f)
	switch {
	case errors.Is(err, storage.ErrTooLarge):
		writeProblem(c, http.StatusRequestEntityTooLarge, "PHOTO_TOO_LARGE", "Foto terlalu besar",
			"Ukuran foto selfie maksimal 2 MB.", nil)
		return nil, false
	case errors.Is(err, storage.ErrUnsupportedType):
		writeProblem(c, http.StatusUnsupportedMediaType, "PHOTO_UNSUPPORTED", "Format foto tidak didukung",
			"Foto selfie harus JPEG atau PNG.", nil)
		return nil, false
	case err != nil:
		writeProblem(c, http.StatusBadRequest, "INVALID_FORM", "Permintaan tidak valid",
			"Berkas selfie tidak dapat dibaca.", nil)
		return nil, false
	}

	if s.attendance.Store == nil {
		writeProblem(c, http.StatusServiceUnavailable, "STORAGE_UNAVAILABLE", "Penyimpanan foto tidak tersedia",
			"Object storage belum dikonfigurasi. Coba lagi nanti.", nil)
		return nil, false
	}
	key, err := storage.ObjectKey(tenantID, storage.KindSelfie, photo.Ext)
	if err != nil {
		writeInternal(c, "bentuk path selfie", err)
		return nil, false
	}
	if err := storage.PutPhoto(c.Request.Context(), s.attendance.Store, key, photo); err != nil {
		log.Printf("attendance: unggah selfie gagal: %v", err)
		writeProblem(c, http.StatusServiceUnavailable, "STORAGE_UNAVAILABLE", "Penyimpanan foto tidak tersedia",
			"Foto selfie gagal disimpan. Coba lagi beberapa saat lagi.", nil)
		return nil, false
	}
	return &key, true
}

// discardSelfie menghapus selfie yang terlanjur diunggah bila presensi gagal disimpan,
// agar tidak ada foto yatim di storage (foto yang gagal tidak disimpan, kontrak API).
// Memakai context baru karena context request bisa saja sudah dibatalkan.
func (s *Server) discardSelfie(key *string) {
	if key == nil || s.attendance.Store == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), cleanupTimeout)
	defer cancel()
	if err := s.attendance.Store.Delete(ctx, *key); err != nil {
		log.Printf("attendance: gagal menghapus selfie yatim %s: %v", *key, err)
	}
}

func (s *Server) findOpenAttendance(ctx context.Context, tenantID, userID string, now time.Time) (*repository.Attendance, error) {
	today, err := attendance.WorkDate(now, s.attendance.Timezone)
	if err != nil {
		return nil, err
	}
	rec, err := s.attendance.Repo.FindByUserAndDate(ctx, tenantID, userID, today)
	if err != nil || rec != nil {
		return rec, err
	}
	prev, err := s.attendance.Repo.FindByUserAndDate(ctx, tenantID, userID, today.AddDate(0, 0, -1))
	if err != nil {
		return nil, err
	}
	if prev != nil && prev.ClockOut == nil {
		return prev, nil
	}
	return nil, nil
}

func toRuleSchedule(ws *repository.WorkSchedule) attendance.Schedule {
	return attendance.Schedule{
		StartTime:            ws.StartTime,
		EndTime:              ws.EndTime,
		ToleranceMinutes:     ws.ToleranceMinutes,
		OvertimeAfterMinutes: ws.OvertimeAfterMinutes,
		MaxOvertimeMinutes:   ws.MaxOvertimeMinutes,
	}
}

func toAttendanceResponse(a *repository.Attendance) api.AttendanceResponse {
	return api.AttendanceResponse{
		Id:                parseUUID(a.ID),
		OfficeLocationId:  parseUUIDPtr(a.OfficeLocationID),
		WorkScheduleId:    parseUUIDPtr(a.WorkScheduleID),
		WorkDate:          openapi_types.Date{Time: a.WorkDate},
		WorkMode:          api.AttendanceResponseWorkMode(a.WorkMode),
		ClockIn:           a.ClockIn.UTC(),
		ClockOut:          utcPtr(a.ClockOut),
		LatitudeIn:        a.LatitudeIn,
		LongitudeIn:       a.LongitudeIn,
		LatitudeOut:       a.LatitudeOut,
		LongitudeOut:      a.LongitudeOut,
		DistanceInMeters:  a.DistanceInMeters,
		DistanceOutMeters: a.DistanceOutMeters,
		Status:            api.AttendanceResponseStatus(a.Status),
		LateMinutes:       a.LateMinutes,
		EarlyLeaveMinutes: a.EarlyLeaveMinutes,
		OvertimeMinutes:   a.OvertimeMinutes,
		HasSelfieIn:       a.SelfieInKey != nil,
		HasSelfieOut:      a.SelfieOutKey != nil,
	}
}

func writeOutsideGeofence(c *gin.Context, loc *repository.OfficeLocation, distance int) {
	writeProblem(c, http.StatusUnprocessableEntity, "OUTSIDE_GEOFENCE", "Di luar area kantor",
		fmt.Sprintf("Posisi Anda %d meter dari %s, batas radius %d meter.", distance, loc.Name, loc.RadiusMeters),
		gin.H{"distance_meters": distance, "radius_meters": loc.RadiusMeters, "location_name": loc.Name})
}

// writeProblem menulis error RFC 7807 memakai helper models.NewProblem yang juga dipakai Guard.
func writeProblem(c *gin.Context, status int, code, title, detail string, errs any) {
	c.Header("Content-Type", "application/problem+json")
	c.AbortWithStatusJSON(status, models.NewProblem(status, title, detail, code, errs))
}

// writeInternal mencatat detail error di log server, tetapi klien hanya menerima pesan umum
// agar informasi internal (query, nama tabel) tidak bocor.
func writeInternal(c *gin.Context, op string, err error) {
	log.Printf("attendance: %s: %v", op, err)
	writeProblem(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Kesalahan server",
		"Terjadi kesalahan pada server. Coba lagi beberapa saat lagi.", nil)
}

func parseUUID(s string) openapi_types.UUID {
	id, _ := uuid.Parse(s) // id berasal dari database (kolom UUID), selalu valid
	return id
}

func parseUUIDPtr(s *string) *openapi_types.UUID {
	if s == nil {
		return nil
	}
	id := parseUUID(*s)
	return &id
}

func utcPtr(t *time.Time) *time.Time {
	if t == nil {
		return nil
	}
	u := t.UTC()
	return &u
}
