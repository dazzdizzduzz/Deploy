package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/mail"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/tiers-undiknas/capstone-group07-2026/backend/api"
	"github.com/tiers-undiknas/capstone-group07-2026/backend/internal/attendance"
	"github.com/tiers-undiknas/capstone-group07-2026/backend/internal/models"
	"github.com/tiers-undiknas/capstone-group07-2026/backend/internal/repository"
)

// API Karyawan (ISSUE-26, FR-03). Acuan kontrak: api/openapi-phase1.yaml bagian
// "Karyawan (Users - FR-03)".

// Hak akses per operasi (x-rbac di openapi.yaml).
var (
	employeeReadRoles = map[models.UserRole]bool{
		models.RoleSuperAdmin: true, models.RoleOwner: true, models.RoleHRD: true, models.RoleTeamLead: true,
	}
	employeeWriteRoles = map[models.UserRole]bool{
		models.RoleSuperAdmin: true, models.RoleOwner: true, models.RoleHRD: true,
	}
	// Gaji pokok dan rekening bank hanya untuk owner & hrd (kontrak Fase 1).
	sensitiveViewerRoles = map[models.UserRole]bool{models.RoleOwner: true, models.RoleHRD: true}
)

var (
	assignableRoles = map[string]bool{"owner": true, "hrd": true, "team_lead": true, "employee": true, "cashier": true}
	employmentTypes = map[string]bool{"tetap": true, "kontrak": true, "magang": true}
	phonePattern    = regexp.MustCompile(`^[0-9+()\-\s]{6,30}$`)
	// DECIMAL(12,2): maksimal 10 digit sebelum koma.
	salaryPattern = regexp.MustCompile(`^\d{1,10}(\.\d{2})?$`)
)

// unusablePasswordHash disimpan untuk akun baru. Nilai ini bukan hash bcrypt, sehingga
// bcrypt.CompareHashAndPassword selalu gagal: akun belum bisa login sampai karyawan
// menyetel kata sandi lewat alur reset/undangan. Kolom password_hash tetap NOT NULL.
const unusablePasswordHash = "!"

// maxJSONBody membatasi ukuran badan JSON agar request raksasa tidak menghabiskan memori.
const maxJSONBody = 64 << 10

// EnableEmployees mengaktifkan endpoint /users.
func (s *Server) EnableEmployees(repo repository.EmployeeRepository) {
	s.employees = repo
}

// ---------------------------------------------------------------------------
// GET /users
// ---------------------------------------------------------------------------

func (s *Server) GetUsers(c *gin.Context, params api.GetUsersParams) {
	who, ok := s.employeeCaller(c, employeeReadRoles)
	if !ok {
		return
	}

	f := repository.EmployeeFilter{Page: 1, PerPage: 20}
	errs := map[string]string{}
	if params.Status != nil {
		switch v := string(*params.Status); v {
		case repository.EmployeeStatusActive, repository.EmployeeStatusInactive, repository.EmployeeStatusInvited:
			f.Status = v
		default:
			errs["status"] = "harus aktif, nonaktif, atau diundang"
		}
	}
	if params.Role != nil {
		if !assignableRoles[string(*params.Role)] {
			errs["role"] = "peran tidak dikenal"
		}
		f.Role = string(*params.Role)
	}
	if params.DepartmentId != nil {
		f.DepartmentID = params.DepartmentId.String()
	}
	if params.Search != nil {
		if utf8.RuneCountInString(*params.Search) > 100 {
			errs["search"] = "maksimal 100 karakter"
		}
		f.Search = *params.Search
	}
	if params.Page != nil {
		if *params.Page < 1 {
			errs["page"] = "minimal 1"
		}
		f.Page = *params.Page
	}
	if params.PerPage != nil {
		if *params.PerPage < 1 || *params.PerPage > 100 {
			errs["per_page"] = "antara 1 dan 100"
		}
		f.PerPage = *params.PerPage
	}
	if len(errs) > 0 {
		writeProblem(c, http.StatusBadRequest, "INVALID_QUERY", "Parameter tidak valid",
			"Periksa parameter pencarian.", errs)
		return
	}

	list, total, err := s.employees.List(c.Request.Context(), who.TenantID, f)
	if err != nil {
		writeInternal(c, "daftar karyawan", err)
		return
	}

	showSensitive := sensitiveViewerRoles[who.Role]
	resp := api.UserListResponse{
		Data: make([]api.UserResponse, 0, len(list)),
		Meta: api.PaginationMeta{
			Page:       f.Page,
			PerPage:    f.PerPage,
			Total:      total,
			TotalPages: (total + f.PerPage - 1) / f.PerPage,
		},
	}
	for _, e := range list {
		resp.Data = append(resp.Data, toUserResponse(e, showSensitive))
	}
	c.JSON(http.StatusOK, resp)
}

// ---------------------------------------------------------------------------
// GET /users/{id}
// ---------------------------------------------------------------------------

func (s *Server) GetUserById(c *gin.Context, id openapi_types.UUID) {
	who, ok := s.employeeCaller(c, employeeReadRoles)
	if !ok {
		return
	}
	e, ok := s.loadEmployee(c, who.TenantID, id)
	if !ok {
		return
	}
	c.JSON(http.StatusOK, toUserResponse(e, sensitiveViewerRoles[who.Role]))
}

// ---------------------------------------------------------------------------
// POST /users
// ---------------------------------------------------------------------------

type createUserInput struct {
	Name              string  `json:"name"`
	Email             string  `json:"email"`
	Phone             *string `json:"phone"`
	DepartmentID      *string `json:"department_id"`
	PositionID        *string `json:"position_id"`
	Role              string  `json:"role"`
	EmploymentType    string  `json:"employment_type"`
	Status            *string `json:"status"`
	JoinedAt          *string `json:"joined_at"`
	ManagerID         *string `json:"manager_id"`
	ScheduleID        *string `json:"schedule_id"`
	LocationID        *string `json:"location_id"`
	AllowWFH          *bool   `json:"allow_wfh_without_approval"`
	BaseSalary        *string `json:"base_salary"`
	BankName          *string `json:"bank_name"`
	BankAccountNumber *string `json:"bank_account_number"`
	BankAccountHolder *string `json:"bank_account_holder"`
}

func (s *Server) PostUsers(c *gin.Context) {
	who, ok := s.employeeCaller(c, employeeWriteRoles)
	if !ok {
		return
	}
	var in createUserInput
	if !decodeJSONBody(c, &in) {
		return
	}

	v := newValidator()
	e := &repository.Employee{
		TenantID:          who.TenantID,
		FullName:          v.name("name", in.Name),
		Email:             v.email("email", in.Email),
		Phone:             v.phone("phone", in.Phone),
		DepartmentID:      v.uuidRef("department_id", in.DepartmentID),
		PositionID:        v.uuidRef("position_id", in.PositionID),
		Role:              v.role("role", in.Role),
		EmploymentType:    v.employmentType("employment_type", in.EmploymentType),
		Status:            repository.EmployeeStatusInvited,
		JoinedAt:          v.date("joined_at", in.JoinedAt),
		ManagerID:         v.uuidRef("manager_id", in.ManagerID),
		ScheduleID:        v.uuidRef("schedule_id", in.ScheduleID),
		LocationID:        v.uuidRef("location_id", in.LocationID),
		AllowWFH:          in.AllowWFH != nil && *in.AllowWFH,
		BaseSalary:        v.salary("base_salary", in.BaseSalary),
		BankName:          v.text("bank_name", in.BankName, 100),
		BankAccountNumber: v.text("bank_account_number", in.BankAccountNumber, 50),
		BankAccountHolder: v.text("bank_account_holder", in.BankAccountHolder, 100),
	}
	if in.Status != nil {
		switch *in.Status {
		case repository.EmployeeStatusActive, repository.EmployeeStatusInvited:
			e.Status = *in.Status
		default:
			v.errs["status"] = "harus aktif atau diundang"
		}
	}
	if !s.finishValidation(c, v, who.TenantID, e) {
		return
	}

	// Mencegah eskalasi hak akses: HRD tidak boleh membuat akun Owner.
	if e.Role == string(models.RoleOwner) && who.Role != models.RoleOwner {
		writeProblem(c, http.StatusForbidden, "FORBIDDEN", "Akses ditolak",
			"Hanya Owner yang dapat membuat akun berperan owner.", nil)
		return
	}

	if err := s.employees.Create(c.Request.Context(), e, unusablePasswordHash); err != nil {
		if errors.Is(err, repository.ErrEmailTaken) {
			writeProblem(c, http.StatusConflict, "EMAIL_TAKEN", "Email sudah terdaftar",
				"Email ini sudah dipakai karyawan lain di tenant ini.", map[string]string{"email": "sudah terdaftar"})
			return
		}
		writeInternal(c, "tambah karyawan", err)
		return
	}
	log.Printf("AUDIT employee.create tenant=%s actor=%s target=%s role=%s", who.TenantID, who.UserID, e.ID, e.Role)
	c.JSON(http.StatusCreated, toUserResponse(e, sensitiveViewerRoles[who.Role]))
}

// ---------------------------------------------------------------------------
// PATCH /users/{id}
// ---------------------------------------------------------------------------

func (s *Server) PatchUserById(c *gin.Context, id openapi_types.UUID) {
	who, ok := s.employeeCaller(c, employeeWriteRoles)
	if !ok {
		return
	}
	// Badan dibaca sebagai map agar bisa membedakan field yang tidak dikirim (tidak diubah)
	// dengan field yang dikirim null (dikosongkan).
	var raw map[string]json.RawMessage
	if !decodeJSONBody(c, &raw) {
		return
	}
	e, ok := s.loadEmployee(c, who.TenantID, id)
	if !ok {
		return
	}
	if !canManage(who, e) {
		return403Owner(c)
		return
	}
	originalRole := e.Role

	v := newValidator()
	p := patch{raw: raw, v: v}
	if val, set := p.str("name", false); set {
		e.FullName = v.name("name", deref(val))
	}
	if val, set := p.str("phone", true); set {
		e.Phone = v.phone("phone", val)
	}
	if val, set := p.str("department_id", true); set {
		e.DepartmentID = v.uuidRef("department_id", val)
	}
	if val, set := p.str("position_id", true); set {
		e.PositionID = v.uuidRef("position_id", val)
	}
	if val, set := p.str("role", false); set {
		e.Role = v.role("role", deref(val))
	}
	if val, set := p.str("employment_type", false); set {
		e.EmploymentType = v.employmentType("employment_type", deref(val))
	}
	if val, set := p.str("joined_at", true); set {
		e.JoinedAt = v.date("joined_at", val)
	}
	if val, set := p.str("manager_id", true); set {
		e.ManagerID = v.uuidRef("manager_id", val)
	}
	if val, set := p.str("schedule_id", true); set {
		e.ScheduleID = v.uuidRef("schedule_id", val)
	}
	if val, set := p.str("location_id", true); set {
		e.LocationID = v.uuidRef("location_id", val)
	}
	if b, set := p.boolean("allow_wfh_without_approval"); set {
		e.AllowWFH = b
	}
	if val, set := p.str("base_salary", true); set {
		e.BaseSalary = v.salary("base_salary", val)
	}
	if val, set := p.str("bank_name", true); set {
		e.BankName = v.text("bank_name", val, 100)
	}
	if val, set := p.str("bank_account_number", true); set {
		e.BankAccountNumber = v.text("bank_account_number", val, 50)
	}
	if val, set := p.str("bank_account_holder", true); set {
		e.BankAccountHolder = v.text("bank_account_holder", val, 100)
	}
	if e.ManagerID != nil && *e.ManagerID == e.ID {
		v.errs["manager_id"] = "tidak boleh diri sendiri"
	}
	if !s.finishValidation(c, v, who.TenantID, e) {
		return
	}

	if e.Role != originalRole {
		// Pengguna tidak boleh mengubah perannya sendiri (mencegah tenant kehilangan Owner
		// atau naik hak akses sendiri), dan hanya Owner yang boleh menetapkan peran owner.
		if e.ID == who.UserID {
			writeProblem(c, http.StatusForbidden, "FORBIDDEN", "Akses ditolak",
				"Anda tidak dapat mengubah peran akun Anda sendiri.", nil)
			return
		}
		if e.Role == string(models.RoleOwner) && who.Role != models.RoleOwner {
			return403Owner(c)
			return
		}
	}

	if err := s.employees.Update(c.Request.Context(), e); err != nil {
		s.writeUpdateError(c, err)
		return
	}
	log.Printf("AUDIT employee.update tenant=%s actor=%s target=%s fields=%s",
		who.TenantID, who.UserID, e.ID, strings.Join(mapKeys(raw), ","))
	c.JSON(http.StatusOK, toUserResponse(e, sensitiveViewerRoles[who.Role]))
}

// ---------------------------------------------------------------------------
// POST /users/{id}/deactivate dan /reactivate
// ---------------------------------------------------------------------------

func (s *Server) PostUserDeactivate(c *gin.Context, id openapi_types.UUID) {
	who, ok := s.employeeCaller(c, employeeWriteRoles)
	if !ok {
		return
	}
	var in struct {
		Reason string `json:"reason"`
	}
	if !decodeJSONBody(c, &in) {
		return
	}
	reason := strings.TrimSpace(in.Reason)
	if n := utf8.RuneCountInString(reason); n < 3 || n > 200 {
		writeValidation(c, map[string]string{"reason": "wajib diisi, 3 sampai 200 karakter"})
		return
	}

	e, ok := s.loadEmployee(c, who.TenantID, id)
	if !ok {
		return
	}
	if e.ID == who.UserID {
		writeProblem(c, http.StatusForbidden, "FORBIDDEN", "Akses ditolak",
			"Anda tidak dapat menonaktifkan akun Anda sendiri.", nil)
		return
	}
	if !canManage(who, e) {
		return403Owner(c)
		return
	}
	if e.Status == repository.EmployeeStatusInactive {
		writeProblem(c, http.StatusUnprocessableEntity, "ALREADY_INACTIVE", "Karyawan sudah nonaktif",
			"Karyawan ini sudah berstatus nonaktif.", nil)
		return
	}

	today, err := s.tenantToday()
	if err != nil {
		writeInternal(c, "tanggal tenant", err)
		return
	}
	e.Status = repository.EmployeeStatusInactive
	e.InactiveSince = &today
	e.DeactivationReason = &reason
	if err := s.employees.Update(c.Request.Context(), e); err != nil {
		s.writeUpdateError(c, err)
		return
	}
	// Kontrak: alasan penonaktifan dicatat dalam audit log sistem.
	log.Printf("AUDIT employee.deactivate tenant=%s actor=%s target=%s reason=%q", who.TenantID, who.UserID, e.ID, reason)
	c.JSON(http.StatusOK, toUserResponse(e, sensitiveViewerRoles[who.Role]))
}

func (s *Server) PostUserReactivate(c *gin.Context, id openapi_types.UUID) {
	who, ok := s.employeeCaller(c, employeeWriteRoles)
	if !ok {
		return
	}
	e, ok := s.loadEmployee(c, who.TenantID, id)
	if !ok {
		return
	}
	if !canManage(who, e) {
		return403Owner(c)
		return
	}

	switch e.Status {
	case repository.EmployeeStatusActive:
		// Idempoten: memanggil ulang tidak mengubah apa pun.
		c.JSON(http.StatusOK, toUserResponse(e, sensitiveViewerRoles[who.Role]))
		return
	case repository.EmployeeStatusInvited:
		writeProblem(c, http.StatusUnprocessableEntity, "INVITATION_PENDING", "Undangan belum diterima",
			"Karyawan berstatus diundang menjadi aktif setelah menerima undangan.", nil)
		return
	}

	e.Status = repository.EmployeeStatusActive
	e.InactiveSince = nil
	e.DeactivationReason = nil
	if err := s.employees.Update(c.Request.Context(), e); err != nil {
		s.writeUpdateError(c, err)
		return
	}
	log.Printf("AUDIT employee.reactivate tenant=%s actor=%s target=%s", who.TenantID, who.UserID, e.ID)
	c.JSON(http.StatusOK, toUserResponse(e, sensitiveViewerRoles[who.Role]))
}

// ---------------------------------------------------------------------------
// Helper
// ---------------------------------------------------------------------------

func (s *Server) employeeCaller(c *gin.Context, allowed map[models.UserRole]bool) (caller, bool) {
	who, ok := s.authenticate(c, allowed)
	if !ok {
		return caller{}, false
	}
	if s.employees == nil {
		writeProblem(c, http.StatusServiceUnavailable, "EMPLOYEES_DISABLED", "Modul karyawan belum aktif",
			"Server belum mengaktifkan API karyawan.", nil)
		return caller{}, false
	}
	return who, true
}

// loadEmployee mengambil karyawan di tenant pemanggil. Karyawan milik tenant lain dijawab
// 404 (bukan 403) agar keberadaannya tidak bocor (kontrak API, FR-34).
func (s *Server) loadEmployee(c *gin.Context, tenantID string, id openapi_types.UUID) (*repository.Employee, bool) {
	e, err := s.employees.FindByID(c.Request.Context(), tenantID, id.String())
	if err != nil {
		writeInternal(c, "detail karyawan", err)
		return nil, false
	}
	if e == nil {
		writeProblem(c, http.StatusNotFound, "NOT_FOUND", "Karyawan tidak ditemukan",
			"Karyawan tidak ditemukan di tenant ini.", nil)
		return nil, false
	}
	return e, true
}

// canManage: hanya Owner yang boleh mengubah, menonaktifkan, atau mengaktifkan akun Owner.
func canManage(who caller, target *repository.Employee) bool {
	return target.Role != string(models.RoleOwner) || who.Role == models.RoleOwner
}

func return403Owner(c *gin.Context) {
	writeProblem(c, http.StatusForbidden, "FORBIDDEN", "Akses ditolak",
		"Hanya Owner yang dapat mengelola akun atau peran owner.", nil)
}

// finishValidation menjalankan pemeriksaan relasi lintas tenant lalu menulis 422 bila ada
// error. Mengembalikan true bila data valid.
func (s *Server) finishValidation(c *gin.Context, v *validator, tenantID string, e *repository.Employee) bool {
	if len(v.errs) == 0 {
		missing, err := s.employees.MissingRefs(c.Request.Context(), tenantID, repository.EmployeeRefs{
			DepartmentID: e.DepartmentID, PositionID: e.PositionID, ManagerID: e.ManagerID,
			ScheduleID: e.ScheduleID, LocationID: e.LocationID,
		})
		if err != nil {
			writeInternal(c, "cek relasi karyawan", err)
			return false
		}
		for _, field := range missing {
			v.errs[field] = "tidak ditemukan di tenant ini"
		}
	}
	if len(v.errs) > 0 {
		writeValidation(c, v.errs)
		return false
	}
	return true
}

func (s *Server) writeUpdateError(c *gin.Context, err error) {
	if errors.Is(err, repository.ErrEmployeeNotFound) {
		writeProblem(c, http.StatusNotFound, "NOT_FOUND", "Karyawan tidak ditemukan",
			"Karyawan tidak ditemukan di tenant ini.", nil)
		return
	}
	writeInternal(c, "ubah karyawan", err)
}

// tenantToday mengembalikan tanggal hari ini di zona waktu tenant (dipakai inactive_since).
func (s *Server) tenantToday() (time.Time, error) {
	loc := s.attendance.Timezone
	if loc == nil {
		var err error
		if loc, err = attendance.LoadTimezone(""); err != nil {
			return time.Time{}, err
		}
	}
	now := time.Now
	if s.attendance.Now != nil {
		now = s.attendance.Now
	}
	return attendance.WorkDate(now(), loc)
}

func writeValidation(c *gin.Context, errs map[string]string) {
	writeProblem(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "Data yang dimasukkan tidak valid",
		"Periksa kembali field yang ditandai.", errs)
}

// decodeJSONBody membaca badan JSON dengan batas ukuran. Bila gagal, 400 sudah ditulis.
func decodeJSONBody(c *gin.Context, dst any) bool {
	body, err := io.ReadAll(io.LimitReader(c.Request.Body, maxJSONBody+1))
	if err == nil && len(body) > maxJSONBody {
		err = errors.New("badan permintaan terlalu besar")
	}
	if err == nil {
		err = json.Unmarshal(body, dst)
	}
	if err != nil {
		writeProblem(c, http.StatusBadRequest, "INVALID_JSON", "Permintaan tidak valid",
			"Badan permintaan harus JSON yang valid dengan tipe field yang benar.", nil)
		return false
	}
	return true
}

func toUserResponse(e *repository.Employee, showSensitive bool) api.UserResponse {
	r := api.UserResponse{
		Id:                      parseUUID(e.ID),
		TenantId:                parseUUID(e.TenantID),
		Name:                    e.FullName,
		Email:                   e.Email,
		Phone:                   e.Phone,
		DepartmentId:            parseUUIDPtr(e.DepartmentID),
		DepartmentName:          e.DepartmentName,
		PositionId:              parseUUIDPtr(e.PositionID),
		PositionName:            e.PositionName,
		Role:                    api.UserResponseRole(e.Role),
		EmploymentType:          api.UserResponseEmploymentType(e.EmploymentType),
		Status:                  api.UserResponseStatus(e.Status),
		JoinedAt:                datePtr(e.JoinedAt),
		ManagerId:               parseUUIDPtr(e.ManagerID),
		ManagerName:             e.ManagerName,
		ScheduleId:              parseUUIDPtr(e.ScheduleID),
		ScheduleName:            e.ScheduleName,
		LocationId:              parseUUIDPtr(e.LocationID),
		LocationName:            e.LocationName,
		AllowWfhWithoutApproval: e.AllowWFH,
		InactiveSince:           datePtr(e.InactiveSince),
		DeactivationReason:      e.DeactivationReason,
		CreatedAt:               e.CreatedAt.UTC(),
		UpdatedAt:               e.UpdatedAt.UTC(),
	}
	if showSensitive {
		r.BaseSalary = e.BaseSalary
		r.BankName = e.BankName
		r.BankAccountNumber = e.BankAccountNumber
		r.BankAccountHolder = e.BankAccountHolder
	}
	return r
}

func datePtr(t *time.Time) *openapi_types.Date {
	if t == nil {
		return nil
	}
	return &openapi_types.Date{Time: *t}
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func mapKeys(m map[string]json.RawMessage) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// ---------------------------------------------------------------------------
// Validasi field. Setiap method mengembalikan nilai yang sudah dibersihkan dan mencatat
// pesan error per field ke v.errs (format errors pada ProblemDetails kontrak).
// ---------------------------------------------------------------------------

type validator struct {
	errs map[string]string
}

func newValidator() *validator { return &validator{errs: map[string]string{}} }

func (v *validator) name(field, s string) string {
	s = strings.TrimSpace(s)
	if n := utf8.RuneCountInString(s); n < 2 || n > 100 {
		v.errs[field] = "wajib diisi, 2 sampai 100 karakter"
	}
	return s
}

func (v *validator) email(field, s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	addr, err := mail.ParseAddress(s)
	// Tolak bentuk "Nama <email>": yang diterima hanya alamatnya saja.
	if err != nil || addr.Address != s || len(s) > 255 {
		v.errs[field] = "format email tidak valid"
	}
	return s
}

func (v *validator) phone(field string, s *string) *string {
	s = trimOrNil(s)
	if s != nil && !phonePattern.MatchString(*s) {
		v.errs[field] = "6 sampai 30 karakter: angka, spasi, +, -, ( )"
	}
	return s
}

func (v *validator) uuidRef(field string, s *string) *string {
	s = trimOrNil(s)
	if s != nil {
		if _, err := uuid.Parse(*s); err != nil {
			v.errs[field] = "harus UUID"
		}
	}
	return s
}

func (v *validator) role(field, s string) string {
	if !assignableRoles[s] {
		v.errs[field] = "harus owner, hrd, team_lead, employee, atau cashier"
	}
	return s
}

func (v *validator) employmentType(field, s string) string {
	if !employmentTypes[s] {
		v.errs[field] = "harus tetap, kontrak, atau magang"
	}
	return s
}

func (v *validator) date(field string, s *string) *time.Time {
	s = trimOrNil(s)
	if s == nil {
		return nil
	}
	t, err := time.Parse(time.DateOnly, *s)
	if err != nil {
		v.errs[field] = "format tanggal YYYY-MM-DD"
		return nil
	}
	return &t
}

func (v *validator) salary(field string, s *string) *string {
	s = trimOrNil(s)
	if s != nil && !salaryPattern.MatchString(*s) {
		v.errs[field] = `string desimal tidak negatif, contoh "6500000.00"`
	}
	return s
}

func (v *validator) text(field string, s *string, max int) *string {
	s = trimOrNil(s)
	if s != nil && utf8.RuneCountInString(*s) > max {
		v.errs[field] = fmt.Sprintf("maksimal %d karakter", max)
	}
	return s
}

// trimOrNil merapikan spasi dan menganggap string kosong sebagai tidak diisi (NULL).
func trimOrNil(s *string) *string {
	if s == nil {
		return nil
	}
	t := strings.TrimSpace(*s)
	if t == "" {
		return nil
	}
	return &t
}

// patch membaca field PATCH dari map JSON mentah.
type patch struct {
	raw map[string]json.RawMessage
	v   *validator
}

// str mengembalikan (nilai, dikirim). null hanya diterima untuk field yang nullable.
func (p patch) str(key string, nullable bool) (*string, bool) {
	msg, ok := p.raw[key]
	if !ok {
		return nil, false
	}
	if string(msg) == "null" {
		if !nullable {
			p.v.errs[key] = "tidak boleh null"
		}
		return nil, true
	}
	var s string
	if err := json.Unmarshal(msg, &s); err != nil {
		p.v.errs[key] = "harus berupa teks"
		return nil, true
	}
	return &s, true
}

func (p patch) boolean(key string) (bool, bool) {
	msg, ok := p.raw[key]
	if !ok {
		return false, false
	}
	var b bool
	if err := json.Unmarshal(msg, &b); err != nil {
		p.v.errs[key] = "harus true atau false"
		return false, false
	}
	return b, true
}
