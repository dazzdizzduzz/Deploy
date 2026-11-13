package handler

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/tiers-undiknas/capstone-group07-2026/backend/api"
	"github.com/tiers-undiknas/capstone-group07-2026/backend/internal/models"
	"github.com/tiers-undiknas/capstone-group07-2026/backend/internal/repository"
)

// Hak akses modul pengajuan & persetujuan lembur (ISSUE-66 / FR-09).
var (
	overtimeReadRoles = map[models.UserRole]bool{
		models.RoleSuperAdmin: true,
		models.RoleOwner:      true,
		models.RoleHRD:        true,
		models.RoleTeamLead:   true,
	}
	overtimeDetailRoles = map[models.UserRole]bool{
		models.RoleSuperAdmin: true,
		models.RoleOwner:      true,
		models.RoleHRD:        true,
		models.RoleTeamLead:   true,
		models.RoleEmployee:   true,
		models.RoleCashier:    true,
	}
	overtimeCreateRoles = map[models.UserRole]bool{
		models.RoleOwner:    true,
		models.RoleHRD:      true,
		models.RoleTeamLead: true,
		models.RoleEmployee: true,
		models.RoleCashier:  true,
	}
	overtimeDecisionRoles = map[models.UserRole]bool{
		models.RoleSuperAdmin: true,
		models.RoleOwner:      true,
		models.RoleHRD:        true,
		models.RoleTeamLead:   true,
	}
)

// EnableOvertime mengaktifkan repository pengajuan & persetujuan lembur.
func (s *Server) EnableOvertime(repo repository.OvertimeRepository) {
	s.overtime = repo
}

// ---------------------------------------------------------------------------
// GET /overtime-requests
// ---------------------------------------------------------------------------

func (s *Server) GetOvertimeRequests(c *gin.Context, params api.GetOvertimeRequestsParams) {
	who, ok := s.authenticate(c, overtimeReadRoles)
	if !ok {
		return
	}
	if s.overtime == nil {
		writeProblem(c, http.StatusServiceUnavailable, "OVERTIME_DISABLED", "Modul lembur belum aktif",
			"Server belum mengaktifkan API lembur.", nil)
		return
	}

	page := 1
	perPage := 20
	if params.Page != nil && *params.Page > 0 {
		page = *params.Page
	}
	if params.PerPage != nil && *params.PerPage > 0 {
		perPage = *params.PerPage
	}
	if perPage > 100 {
		perPage = 100
	}

	filter := repository.OvertimeFilter{
		Page:    page,
		PerPage: perPage,
	}

	// Pembatasan hak akses: Team Lead hanya melihat pengajuan lembur anggota timnya
	if who.Role == models.RoleTeamLead {
		filter.ManagerID = who.UserID
	}

	if params.Status != nil {
		st := string(*params.Status)
		if st != repository.OvertimeStatusPending && st != repository.OvertimeStatusApproved && st != repository.OvertimeStatusRejected {
			writeProblem(c, http.StatusBadRequest, "INVALID_STATUS", "Status tidak valid",
				"Status harus berupa menunggu, disetujui, atau ditolak.", nil)
			return
		}
		filter.Status = st
	}

	if params.DepartmentId != nil {
		filter.DepartmentID = params.DepartmentId.String()
	}

	if params.Date != nil {
		filter.Date = &params.Date.Time
	}

	list, total, counts, err := s.overtime.List(c.Request.Context(), who.TenantID, filter)
	if err != nil {
		writeInternal(c, "daftar lembur", err)
		return
	}

	resp := api.OvertimeListResponse{
		Data: make([]api.OvertimeResponse, 0, len(list)),
	}
	resp.Meta.Page = filter.Page
	resp.Meta.PerPage = filter.PerPage
	resp.Meta.Total = total
	resp.Meta.TotalPages = (total + filter.PerPage - 1) / filter.PerPage
	if total == 0 {
		resp.Meta.TotalPages = 1
	}

	resp.Counts.Menunggu = counts[repository.OvertimeStatusPending]
	resp.Counts.Disetujui = counts[repository.OvertimeStatusApproved]
	resp.Counts.Ditolak = counts[repository.OvertimeStatusRejected]

	for _, o := range list {
		resp.Data = append(resp.Data, toOvertimeResponse(o))
	}

	c.JSON(http.StatusOK, resp)
}

// ---------------------------------------------------------------------------
// POST /overtime-requests
// ---------------------------------------------------------------------------

func (s *Server) PostOvertimeRequest(c *gin.Context) {
	who, ok := s.authenticate(c, overtimeCreateRoles)
	if !ok {
		return
	}
	if s.overtime == nil {
		writeProblem(c, http.StatusServiceUnavailable, "OVERTIME_DISABLED", "Modul lembur belum aktif",
			"Server belum mengaktifkan API lembur.", nil)
		return
	}

	var req api.CreateOvertimeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeProblem(c, http.StatusBadRequest, "INVALID_BODY", "Format data tidak valid",
			"Format data pengajuan lembur tidak valid.", nil)
		return
	}

	reason := strings.TrimSpace(req.Reason)
	if reason == "" {
		writeProblem(c, http.StatusBadRequest, "VALIDATION_ERROR", "Data tidak lengkap",
			"Alasan lembur wajib diisi.", gin.H{"reason": "wajib diisi"})
		return
	}

	startTime := strings.TrimSpace(req.StartTime)
	endTime := strings.TrimSpace(req.EndTime)
	if startTime == "" || endTime == "" {
		writeProblem(c, http.StatusBadRequest, "VALIDATION_ERROR", "Data tidak lengkap",
			"Jam mulai dan jam selesai lembur wajib diisi.", gin.H{"start_time": "wajib diisi", "end_time": "wajib diisi"})
		return
	}

	const timeLayout = "15:04"
	tStart, err1 := time.Parse(timeLayout, startTime)
	tEnd, err2 := time.Parse(timeLayout, endTime)
	if err1 != nil || err2 != nil {
		writeProblem(c, http.StatusBadRequest, "VALIDATION_ERROR", "Format jam tidak valid",
			"Format jam harus berupa HH:MM (contoh: 17:30).", nil)
		return
	}

	duration := int(tEnd.Sub(tStart) / time.Minute)
	if duration <= 0 {
		writeProblem(c, http.StatusBadRequest, "VALIDATION_ERROR", "Rentang waktu tidak valid",
			"Jam selesai lembur harus setelah jam mulai.", gin.H{"end_time": "harus setelah jam mulai"})
		return
	}

	ot := &repository.OvertimeRequest{
		TenantID:        who.TenantID,
		UserID:          who.UserID,
		Date:            req.Date.Time,
		StartTime:       startTime,
		EndTime:         endTime,
		DurationMinutes: duration,
		Reason:          reason,
		Status:          repository.OvertimeStatusPending,
	}

	if err := s.overtime.Create(c.Request.Context(), ot); err != nil {
		writeInternal(c, "buat pengajuan lembur", err)
		return
	}

	created, err := s.overtime.FindByID(c.Request.Context(), who.TenantID, ot.ID)
	if err != nil || created == nil {
		c.JSON(http.StatusCreated, toOvertimeResponse(ot))
		return
	}

	c.JSON(http.StatusCreated, toOvertimeResponse(created))
}

// ---------------------------------------------------------------------------
// GET /overtime-requests/{id}
// ---------------------------------------------------------------------------

func (s *Server) GetOvertimeRequestById(c *gin.Context, id openapi_types.UUID) {
	who, ok := s.authenticate(c, overtimeDetailRoles)
	if !ok {
		return
	}
	if s.overtime == nil {
		writeProblem(c, http.StatusServiceUnavailable, "OVERTIME_DISABLED", "Modul lembur belum aktif",
			"Server belum mengaktifkan API lembur.", nil)
		return
	}

	ot, err := s.overtime.FindByID(c.Request.Context(), who.TenantID, id.String())
	if err != nil {
		if errors.Is(err, repository.ErrOvertimeNotFound) {
			writeProblem(c, http.StatusNotFound, "NOT_FOUND", "Pengajuan lembur tidak ditemukan",
				"Pengajuan lembur tidak ditemukan pada tenant ini.", nil)
			return
		}
		writeInternal(c, "detail pengajuan lembur", err)
		return
	}

	// Employee / Cashier hanya boleh melihat pengajuan lembur miliknya sendiri
	if (who.Role == models.RoleEmployee || who.Role == models.RoleCashier) && ot.UserID != who.UserID {
		writeProblem(c, http.StatusForbidden, "FORBIDDEN", "Akses ditolak",
			"Anda hanya berhak melihat pengajuan lembur milik sendiri.", nil)
		return
	}

	c.JSON(http.StatusOK, toOvertimeResponse(ot))
}

// ---------------------------------------------------------------------------
// POST /overtime-requests/{id}/approve
// ---------------------------------------------------------------------------

func (s *Server) PostOvertimeRequestApprove(c *gin.Context, id openapi_types.UUID) {
	who, ok := s.authenticate(c, overtimeDecisionRoles)
	if !ok {
		return
	}
	if s.overtime == nil {
		writeProblem(c, http.StatusServiceUnavailable, "OVERTIME_DISABLED", "Modul lembur belum aktif",
			"Server belum mengaktifkan API lembur.", nil)
		return
	}

	ctx := c.Request.Context()

	// Jika pemanggil adalah team_lead, verifikasi bahwa pemohon adalah bawahannya
	if who.Role == models.RoleTeamLead && s.employees != nil {
		existing, err := s.overtime.FindByID(ctx, who.TenantID, id.String())
		if err != nil {
			if errors.Is(err, repository.ErrOvertimeNotFound) {
				writeProblem(c, http.StatusNotFound, "NOT_FOUND", "Pengajuan lembur tidak ditemukan",
					"Pengajuan lembur tidak ditemukan.", nil)
				return
			}
			writeInternal(c, "cek pemohon lembur", err)
			return
		}
		if existing.UserID == who.UserID {
			writeProblem(c, http.StatusForbidden, "SELF_APPROVAL_FORBIDDEN", "Akses ditolak",
				"Karyawan tidak boleh menyetujui pengajuannya sendiri.", nil)
			return
		}
		emp, err := s.employees.FindByID(ctx, who.TenantID, existing.UserID)
		if err != nil {
			writeInternal(c, "cek relasi manager pemohon", err)
			return
		}
		if emp == nil || emp.ManagerID == nil || *emp.ManagerID != who.UserID {
			writeProblem(c, http.StatusForbidden, "FORBIDDEN", "Akses ditolak",
				"Team Lead hanya dapat menyetujui lembur bawahan langsungnya.", nil)
			return
		}
	}

	updated, err := s.overtime.Approve(ctx, who.TenantID, id.String(), who.UserID)
	if err != nil {
		if errors.Is(err, repository.ErrOvertimeNotFound) {
			writeProblem(c, http.StatusNotFound, "NOT_FOUND", "Pengajuan lembur tidak ditemukan",
				"Pengajuan lembur tidak ditemukan.", nil)
			return
		}
		if errors.Is(err, repository.ErrOvertimeSelfApproval) {
			writeProblem(c, http.StatusForbidden, "SELF_APPROVAL_FORBIDDEN", "Akses ditolak",
				"Karyawan tidak boleh menyetujui pengajuannya sendiri.", nil)
			return
		}
		if errors.Is(err, repository.ErrOvertimeAlreadyProcessed) {
			writeProblem(c, http.StatusConflict, "ALREADY_PROCESSED", "Sudah diputuskan",
				"Pengajuan lembur ini sudah disetujui atau ditolak sebelumnya.", nil)
			return
		}
		writeInternal(c, "setujui lembur", err)
		return
	}

	c.JSON(http.StatusOK, toOvertimeResponse(updated))
}

// ---------------------------------------------------------------------------
// POST /overtime-requests/{id}/reject
// ---------------------------------------------------------------------------

func (s *Server) PostOvertimeRequestReject(c *gin.Context, id openapi_types.UUID) {
	who, ok := s.authenticate(c, overtimeDecisionRoles)
	if !ok {
		return
	}
	if s.overtime == nil {
		writeProblem(c, http.StatusServiceUnavailable, "OVERTIME_DISABLED", "Modul lembur belum aktif",
			"Server belum mengaktifkan API lembur.", nil)
		return
	}

	var req api.PostOvertimeRequestRejectJSONRequestBody
	if err := c.ShouldBindJSON(&req); err != nil {
		writeProblem(c, http.StatusBadRequest, "INVALID_BODY", "Format data tidak valid",
			"Format data penolakan tidak valid.", nil)
		return
	}

	reason := strings.TrimSpace(req.Reason)
	if reason == "" {
		writeProblem(c, http.StatusBadRequest, "VALIDATION_ERROR", "Alasan wajib diisi",
			"Alasan penolakan lembur wajib diisi.", gin.H{"reason": "wajib diisi"})
		return
	}

	ctx := c.Request.Context()

	// Jika pemanggil adalah team_lead, verifikasi bahwa pemohon adalah bawahannya
	if who.Role == models.RoleTeamLead && s.employees != nil {
		existing, err := s.overtime.FindByID(ctx, who.TenantID, id.String())
		if err != nil {
			if errors.Is(err, repository.ErrOvertimeNotFound) {
				writeProblem(c, http.StatusNotFound, "NOT_FOUND", "Pengajuan lembur tidak ditemukan",
					"Pengajuan lembur tidak ditemukan.", nil)
				return
			}
			writeInternal(c, "cek pemohon lembur", err)
			return
		}
		if existing.UserID == who.UserID {
			writeProblem(c, http.StatusForbidden, "SELF_APPROVAL_FORBIDDEN", "Akses ditolak",
				"Karyawan tidak boleh menolak pengajuannya sendiri.", nil)
			return
		}
		emp, err := s.employees.FindByID(ctx, who.TenantID, existing.UserID)
		if err != nil {
			writeInternal(c, "cek relasi manager pemohon", err)
			return
		}
		if emp == nil || emp.ManagerID == nil || *emp.ManagerID != who.UserID {
			writeProblem(c, http.StatusForbidden, "FORBIDDEN", "Akses ditolak",
				"Team Lead hanya dapat menolak lembur bawahan langsungnya.", nil)
			return
		}
	}

	updated, err := s.overtime.Reject(ctx, who.TenantID, id.String(), who.UserID, reason)
	if err != nil {
		if errors.Is(err, repository.ErrOvertimeNotFound) {
			writeProblem(c, http.StatusNotFound, "NOT_FOUND", "Pengajuan lembur tidak ditemukan",
				"Pengajuan lembur tidak ditemukan.", nil)
			return
		}
		if errors.Is(err, repository.ErrOvertimeSelfApproval) {
			writeProblem(c, http.StatusForbidden, "SELF_APPROVAL_FORBIDDEN", "Akses ditolak",
				"Karyawan tidak boleh menolak pengajuannya sendiri.", nil)
			return
		}
		if errors.Is(err, repository.ErrOvertimeAlreadyProcessed) {
			writeProblem(c, http.StatusConflict, "ALREADY_PROCESSED", "Sudah diputuskan",
				"Pengajuan lembur ini sudah disetujui atau ditolak sebelumnya.", nil)
			return
		}
		writeInternal(c, "tolak lembur", err)
		return
	}

	c.JSON(http.StatusOK, toOvertimeResponse(updated))
}

// ---------------------------------------------------------------------------
// Mapper helper
// ---------------------------------------------------------------------------

func toOvertimeResponse(o *repository.OvertimeRequest) api.OvertimeResponse {
	resp := api.OvertimeResponse{
		Id:              parseUUID(o.ID),
		UserId:          parseUUID(o.UserID),
		UserName:        o.UserName,
		DepartmentName:  o.DepartmentName,
		Date:            openapi_types.Date{Time: o.Date},
		StartTime:       o.StartTime,
		EndTime:         o.EndTime,
		DurationMinutes: o.DurationMinutes,
		Reason:          o.Reason,
		Status:          api.OvertimeResponseStatus(o.Status),
		ReviewedBy:      parseUUIDPtr(o.ReviewedBy),
		ReviewerName:    o.ReviewerName,
		ReviewedAt:      utcPtr(o.ReviewedAt),
		RejectionReason: o.RejectionReason,
		CreatedAt:       o.CreatedAt.UTC(),
	}
	if !o.UpdatedAt.IsZero() {
		u := o.UpdatedAt.UTC()
		resp.UpdatedAt = &u
	}
	return resp
}
