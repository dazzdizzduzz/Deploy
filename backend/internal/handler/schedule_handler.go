package handler

import (
	"database/sql"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/tiers-undiknas/capstone-group07-2026/backend/api"
	"github.com/tiers-undiknas/capstone-group07-2026/backend/internal/repository"
)

func isValidTimeFormat(t string) bool {
	t = strings.TrimSpace(t)
	if _, err := time.Parse("15:04", t); err == nil {
		return true
	}
	if _, err := time.Parse("15:04:05", t); err == nil {
		return true
	}
	return false
}

// GetSchedules melayani GET /schedules (Daftar jadwal kerja tenant)
func (s *Server) GetSchedules(c *gin.Context) {
	tenantID, err := s.getTenantID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, api.ErrorResponse{Message: "Tenant tidak teridentifikasi atau token tidak valid"})
		return
	}

	schedules, err := s.scheduleRepo.FindAll(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, api.ErrorResponse{Message: "Gagal mengambil data jadwal kerja"})
		return
	}

	resp := api.WorkSchedulesListResponse{
		Data: make([]api.WorkScheduleResponse, 0, len(schedules)),
	}
	for _, sched := range schedules {
		uID, _ := uuid.Parse(sched.ID)
		tID, _ := uuid.Parse(sched.TenantID)
		resp.Data = append(resp.Data, api.WorkScheduleResponse{
			Id:                   openapi_types.UUID(uID),
			TenantId:             openapi_types.UUID(tID),
			Name:                 sched.Name,
			StartTime:            sched.StartTime,
			EndTime:              sched.EndTime,
			ToleranceMinutes:     sched.ToleranceMinutes,
			OvertimeAfterMinutes: sched.OvertimeAfterMinutes,
			MaxOvertimeMinutes:   sched.MaxOvertimeMinutes,
			WorkingDays:          sched.WorkingDays,
			AllowRemote:          sched.AllowRemote,
			IsActive:             sched.IsActive,
			CreatedAt:            sched.CreatedAt.Format(time.RFC3339),
			UpdatedAt:            sched.UpdatedAt.Format(time.RFC3339),
		})
	}

	c.JSON(http.StatusOK, resp)
}

// PostSchedules melayani POST /schedules (Tambah jadwal kerja baru)
func (s *Server) PostSchedules(c *gin.Context) {
	tenantID, err := s.getTenantID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, api.ErrorResponse{Message: "Tenant tidak teridentifikasi atau token tidak valid"})
		return
	}

	var req api.CreateScheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, api.ErrorResponse{Message: "Format data jadwal kerja tidak valid"})
		return
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		c.JSON(http.StatusBadRequest, api.ErrorResponse{Message: "Nama jadwal kerja wajib diisi"})
		return
	}

	startTime := strings.TrimSpace(req.StartTime)
	if !isValidTimeFormat(startTime) {
		c.JSON(http.StatusUnprocessableEntity, api.ErrorResponse{Message: "Format jam mulai (start_time) tidak valid (gunakan format HH:MM, contoh: 08:00)"})
		return
	}

	endTime := strings.TrimSpace(req.EndTime)
	if !isValidTimeFormat(endTime) {
		c.JSON(http.StatusUnprocessableEntity, api.ErrorResponse{Message: "Format jam selesai (end_time) tidak valid (gunakan format HH:MM, contoh: 17:00)"})
		return
	}

	tolerance := 15
	if req.ToleranceMinutes != nil && *req.ToleranceMinutes > 0 {
		tolerance = *req.ToleranceMinutes
	}

	overtimeAfter := 60
	if req.OvertimeAfterMinutes != nil && *req.OvertimeAfterMinutes > 0 {
		overtimeAfter = *req.OvertimeAfterMinutes
	}

	maxOvertime := 180
	if req.MaxOvertimeMinutes != nil && *req.MaxOvertimeMinutes > 0 {
		maxOvertime = *req.MaxOvertimeMinutes
	}

	workingDays := "Sen,Sel,Rab,Kam,Jum"
	if req.WorkingDays != nil && strings.TrimSpace(*req.WorkingDays) != "" {
		workingDays = strings.TrimSpace(*req.WorkingDays)
	}

	allowRemote := false
	if req.AllowRemote != nil {
		allowRemote = *req.AllowRemote
	}

	sched := &repository.WorkSchedule{
		TenantID:             tenantID,
		Name:                 name,
		StartTime:            startTime,
		EndTime:              endTime,
		ToleranceMinutes:     tolerance,
		OvertimeAfterMinutes: overtimeAfter,
		MaxOvertimeMinutes:   maxOvertime,
		WorkingDays:          workingDays,
		AllowRemote:          allowRemote,
		IsActive:             true,
	}

	if err := s.scheduleRepo.Create(c.Request.Context(), sched); err != nil {
		c.JSON(http.StatusInternalServerError, api.ErrorResponse{Message: "Gagal menyimpan data jadwal kerja"})
		return
	}

	uID, _ := uuid.Parse(sched.ID)
	tID, _ := uuid.Parse(sched.TenantID)

	c.JSON(http.StatusCreated, api.WorkScheduleResponse{
		Id:                   openapi_types.UUID(uID),
		TenantId:             openapi_types.UUID(tID),
		Name:                 sched.Name,
		StartTime:            sched.StartTime,
		EndTime:              sched.EndTime,
		ToleranceMinutes:     sched.ToleranceMinutes,
		OvertimeAfterMinutes: sched.OvertimeAfterMinutes,
		MaxOvertimeMinutes:   sched.MaxOvertimeMinutes,
		WorkingDays:          sched.WorkingDays,
		AllowRemote:          sched.AllowRemote,
		IsActive:             sched.IsActive,
		CreatedAt:            sched.CreatedAt.Format(time.RFC3339),
		UpdatedAt:            sched.UpdatedAt.Format(time.RFC3339),
	})
}

// GetScheduleById melayani GET /schedules/{id} (Detail jadwal kerja)
func (s *Server) GetScheduleById(c *gin.Context, id openapi_types.UUID) {
	tenantID, err := s.getTenantID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, api.ErrorResponse{Message: "Tenant tidak teridentifikasi atau token tidak valid"})
		return
	}

	sched, err := s.scheduleRepo.FindByID(c.Request.Context(), tenantID, id.String())
	if err != nil {
		c.JSON(http.StatusInternalServerError, api.ErrorResponse{Message: "Gagal mengambil detail jadwal kerja"})
		return
	}
	if sched == nil {
		c.JSON(http.StatusNotFound, api.ErrorResponse{Message: "Jadwal kerja tidak ditemukan"})
		return
	}

	uID, _ := uuid.Parse(sched.ID)
	tID, _ := uuid.Parse(sched.TenantID)

	c.JSON(http.StatusOK, api.WorkScheduleResponse{
		Id:                   openapi_types.UUID(uID),
		TenantId:             openapi_types.UUID(tID),
		Name:                 sched.Name,
		StartTime:            sched.StartTime,
		EndTime:              sched.EndTime,
		ToleranceMinutes:     sched.ToleranceMinutes,
		OvertimeAfterMinutes: sched.OvertimeAfterMinutes,
		MaxOvertimeMinutes:   sched.MaxOvertimeMinutes,
		WorkingDays:          sched.WorkingDays,
		AllowRemote:          sched.AllowRemote,
		IsActive:             sched.IsActive,
		CreatedAt:            sched.CreatedAt.Format(time.RFC3339),
		UpdatedAt:            sched.UpdatedAt.Format(time.RFC3339),
	})
}

// PutScheduleById melayani PUT /schedules/{id} (Perbarui seluruh data jadwal kerja)
func (s *Server) PutScheduleById(c *gin.Context, id openapi_types.UUID) {
	s.handleUpdateSchedule(c, id)
}

// PatchScheduleById melayani PATCH /schedules/{id} (Perbarui sebagian data jadwal kerja)
func (s *Server) PatchScheduleById(c *gin.Context, id openapi_types.UUID) {
	s.handleUpdateSchedule(c, id)
}

func (s *Server) handleUpdateSchedule(c *gin.Context, id openapi_types.UUID) {
	tenantID, err := s.getTenantID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, api.ErrorResponse{Message: "Tenant tidak teridentifikasi atau token tidak valid"})
		return
	}

	var req api.UpdateScheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, api.ErrorResponse{Message: "Format data perbaruan jadwal tidak valid"})
		return
	}

	if req.StartTime != nil {
		if !isValidTimeFormat(*req.StartTime) {
			c.JSON(http.StatusUnprocessableEntity, api.ErrorResponse{Message: "Format jam mulai (start_time) tidak valid (gunakan format HH:MM, contoh: 08:00)"})
			return
		}
	}

	if req.EndTime != nil {
		if !isValidTimeFormat(*req.EndTime) {
			c.JSON(http.StatusUnprocessableEntity, api.ErrorResponse{Message: "Format jam selesai (end_time) tidak valid (gunakan format HH:MM, contoh: 17:00)"})
			return
		}
	}

	sched, err := s.scheduleRepo.FindByID(c.Request.Context(), tenantID, id.String())
	if err != nil {
		c.JSON(http.StatusInternalServerError, api.ErrorResponse{Message: "Gagal memproses data jadwal kerja"})
		return
	}
	if sched == nil {
		c.JSON(http.StatusNotFound, api.ErrorResponse{Message: "Jadwal kerja tidak ditemukan"})
		return
	}

	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name != "" {
			sched.Name = name
		}
	}
	if req.StartTime != nil {
		sched.StartTime = strings.TrimSpace(*req.StartTime)
	}
	if req.EndTime != nil {
		sched.EndTime = strings.TrimSpace(*req.EndTime)
	}
	if req.ToleranceMinutes != nil {
		sched.ToleranceMinutes = *req.ToleranceMinutes
	}
	if req.OvertimeAfterMinutes != nil {
		sched.OvertimeAfterMinutes = *req.OvertimeAfterMinutes
	}
	if req.MaxOvertimeMinutes != nil {
		sched.MaxOvertimeMinutes = *req.MaxOvertimeMinutes
	}
	if req.WorkingDays != nil {
		sched.WorkingDays = strings.TrimSpace(*req.WorkingDays)
	}
	if req.AllowRemote != nil {
		sched.AllowRemote = *req.AllowRemote
	}
	if req.IsActive != nil {
		sched.IsActive = *req.IsActive
	}

	if err := s.scheduleRepo.Update(c.Request.Context(), sched); err != nil {
		c.JSON(http.StatusInternalServerError, api.ErrorResponse{Message: "Gagal memperbarui data jadwal kerja"})
		return
	}

	uID, _ := uuid.Parse(sched.ID)
	tID, _ := uuid.Parse(sched.TenantID)

	c.JSON(http.StatusOK, api.WorkScheduleResponse{
		Id:                   openapi_types.UUID(uID),
		TenantId:             openapi_types.UUID(tID),
		Name:                 sched.Name,
		StartTime:            sched.StartTime,
		EndTime:              sched.EndTime,
		ToleranceMinutes:     sched.ToleranceMinutes,
		OvertimeAfterMinutes: sched.OvertimeAfterMinutes,
		MaxOvertimeMinutes:   sched.MaxOvertimeMinutes,
		WorkingDays:          sched.WorkingDays,
		AllowRemote:          sched.AllowRemote,
		IsActive:             sched.IsActive,
		CreatedAt:            sched.CreatedAt.Format(time.RFC3339),
		UpdatedAt:            sched.UpdatedAt.Format(time.RFC3339),
	})
}

// DeleteScheduleById melayani DELETE /schedules/{id} (Hapus jadwal kerja)
func (s *Server) DeleteScheduleById(c *gin.Context, id openapi_types.UUID) {
	tenantID, err := s.getTenantID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, api.ErrorResponse{Message: "Tenant tidak teridentifikasi atau token tidak valid"})
		return
	}

	err = s.scheduleRepo.Delete(c.Request.Context(), tenantID, id.String())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, api.ErrorResponse{Message: "Jadwal kerja tidak ditemukan"})
			return
		}
		c.JSON(http.StatusInternalServerError, api.ErrorResponse{Message: "Gagal menghapus jadwal kerja"})
		return
	}

	c.JSON(http.StatusOK, api.MessageResponse{Message: "Jadwal kerja berhasil dihapus"})
}

