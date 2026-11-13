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

// GetLocations melayani GET /locations (Daftar lokasi kantor tenant)
func (s *Server) GetLocations(c *gin.Context) {
	tenantID, err := s.getTenantID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, api.ErrorResponse{Message: "Tenant tidak teridentifikasi atau token tidak valid"})
		return
	}

	locations, err := s.locationRepo.FindAll(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, api.ErrorResponse{Message: "Gagal mengambil data lokasi kantor"})
		return
	}

	resp := api.OfficeLocationsListResponse{
		Data: make([]api.OfficeLocationResponse, 0, len(locations)),
	}
	for _, loc := range locations {
		uID, _ := uuid.Parse(loc.ID)
		tID, _ := uuid.Parse(loc.TenantID)
		resp.Data = append(resp.Data, api.OfficeLocationResponse{
			Id:             openapi_types.UUID(uID),
			TenantId:       openapi_types.UUID(tID),
			Name:           loc.Name,
			Address:        loc.Address,
			Latitude:       loc.Latitude,
			Longitude:      loc.Longitude,
			RadiusMeters:   loc.RadiusMeters,
			SelfieRequired: loc.SelfieRequired,
			IsActive:       loc.IsActive,
			CreatedAt:      loc.CreatedAt.Format(time.RFC3339),
			UpdatedAt:      loc.UpdatedAt.Format(time.RFC3339),
		})
	}

	c.JSON(http.StatusOK, resp)
}

// PostLocations melayani POST /locations (Tambah lokasi kantor baru - FR-05)
func (s *Server) PostLocations(c *gin.Context) {
	tenantID, err := s.getTenantID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, api.ErrorResponse{Message: "Tenant tidak teridentifikasi atau token tidak valid"})
		return
	}

	var req api.CreateLocationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, api.ErrorResponse{Message: "Format data lokasi kantor tidak valid"})
		return
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		c.JSON(http.StatusBadRequest, api.ErrorResponse{Message: "Nama lokasi kantor wajib diisi"})
		return
	}

	address := strings.TrimSpace(req.Address)
	if address == "" {
		c.JSON(http.StatusBadRequest, api.ErrorResponse{Message: "Alamat lokasi kantor wajib diisi"})
		return
	}

	// Validasi rentang koordinat geografis
	if req.Latitude < -90 || req.Latitude > 90 || req.Longitude < -180 || req.Longitude > 180 {
		c.JSON(http.StatusUnprocessableEntity, api.ErrorResponse{Message: "Koordinat latitude atau longitude tidak valid"})
		return
	}

	// Validasi spesifikasi FR-05: Radius lokasi wajib antara 25 sampai 500 meter
	if req.RadiusMeters < 25 || req.RadiusMeters > 500 {
		c.JSON(http.StatusUnprocessableEntity, api.ErrorResponse{Message: "Radius lokasi wajib antara 25 sampai 500 meter"})
		return
	}

	selfieRequired := false
	if req.SelfieRequired != nil {
		selfieRequired = *req.SelfieRequired
	}

	loc := &repository.OfficeLocation{
		TenantID:       tenantID,
		Name:           name,
		Address:        address,
		Latitude:       req.Latitude,
		Longitude:      req.Longitude,
		RadiusMeters:   req.RadiusMeters,
		SelfieRequired: selfieRequired,
		IsActive:       true,
	}

	if err := s.locationRepo.Create(c.Request.Context(), loc); err != nil {
		c.JSON(http.StatusInternalServerError, api.ErrorResponse{Message: "Gagal menyimpan lokasi kantor"})
		return
	}

	uID, _ := uuid.Parse(loc.ID)
	tID, _ := uuid.Parse(loc.TenantID)

	c.JSON(http.StatusCreated, api.OfficeLocationResponse{
		Id:             openapi_types.UUID(uID),
		TenantId:       openapi_types.UUID(tID),
		Name:           loc.Name,
		Address:        loc.Address,
		Latitude:       loc.Latitude,
		Longitude:      loc.Longitude,
		RadiusMeters:   loc.RadiusMeters,
		SelfieRequired: loc.SelfieRequired,
		IsActive:       loc.IsActive,
		CreatedAt:      loc.CreatedAt.Format(time.RFC3339),
		UpdatedAt:      loc.UpdatedAt.Format(time.RFC3339),
	})
}

// GetLocationById melayani GET /locations/{id} (Detail lokasi kantor)
func (s *Server) GetLocationById(c *gin.Context, id openapi_types.UUID) {
	tenantID, err := s.getTenantID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, api.ErrorResponse{Message: "Tenant tidak teridentifikasi atau token tidak valid"})
		return
	}

	loc, err := s.locationRepo.FindByID(c.Request.Context(), tenantID, id.String())
	if err != nil {
		c.JSON(http.StatusInternalServerError, api.ErrorResponse{Message: "Gagal mengambil detail lokasi kantor"})
		return
	}
	if loc == nil {
		c.JSON(http.StatusNotFound, api.ErrorResponse{Message: "Lokasi kantor tidak ditemukan"})
		return
	}

	uID, _ := uuid.Parse(loc.ID)
	tID, _ := uuid.Parse(loc.TenantID)

	c.JSON(http.StatusOK, api.OfficeLocationResponse{
		Id:             openapi_types.UUID(uID),
		TenantId:       openapi_types.UUID(tID),
		Name:           loc.Name,
		Address:        loc.Address,
		Latitude:       loc.Latitude,
		Longitude:      loc.Longitude,
		RadiusMeters:   loc.RadiusMeters,
		SelfieRequired: loc.SelfieRequired,
		IsActive:       loc.IsActive,
		CreatedAt:      loc.CreatedAt.Format(time.RFC3339),
		UpdatedAt:      loc.UpdatedAt.Format(time.RFC3339),
	})
}

// PutLocationById melayani PUT /locations/{id} (Perbarui seluruh data lokasi kantor - FR-05)
func (s *Server) PutLocationById(c *gin.Context, id openapi_types.UUID) {
	s.handleUpdateLocation(c, id)
}

// PatchLocationById melayani PATCH /locations/{id} (Perbarui sebagian data lokasi kantor - FR-05)
func (s *Server) PatchLocationById(c *gin.Context, id openapi_types.UUID) {
	s.handleUpdateLocation(c, id)
}

func (s *Server) handleUpdateLocation(c *gin.Context, id openapi_types.UUID) {
	tenantID, err := s.getTenantID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, api.ErrorResponse{Message: "Tenant tidak teridentifikasi atau token tidak valid"})
		return
	}

	var req api.UpdateLocationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, api.ErrorResponse{Message: "Format data perbaruan tidak valid"})
		return
	}

	// Validasi radius_meters jika diberikan (spesifikasi FR-05)
	if req.RadiusMeters != nil {
		if *req.RadiusMeters < 25 || *req.RadiusMeters > 500 {
			c.JSON(http.StatusUnprocessableEntity, api.ErrorResponse{Message: "Radius lokasi wajib antara 25 sampai 500 meter"})
			return
		}
	}

	// Validasi koordinat jika diberikan
	if req.Latitude != nil && (*req.Latitude < -90 || *req.Latitude > 90) {
		c.JSON(http.StatusUnprocessableEntity, api.ErrorResponse{Message: "Koordinat latitude tidak valid"})
		return
	}
	if req.Longitude != nil && (*req.Longitude < -180 || *req.Longitude > 180) {
		c.JSON(http.StatusUnprocessableEntity, api.ErrorResponse{Message: "Koordinat longitude tidak valid"})
		return
	}

	loc, err := s.locationRepo.FindByID(c.Request.Context(), tenantID, id.String())
	if err != nil {
		c.JSON(http.StatusInternalServerError, api.ErrorResponse{Message: "Gagal memproses data lokasi kantor"})
		return
	}
	if loc == nil {
		c.JSON(http.StatusNotFound, api.ErrorResponse{Message: "Lokasi kantor tidak ditemukan"})
		return
	}

	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name != "" {
			loc.Name = name
		}
	}
	if req.Address != nil {
		loc.Address = strings.TrimSpace(*req.Address)
	}
	if req.Latitude != nil {
		loc.Latitude = *req.Latitude
	}
	if req.Longitude != nil {
		loc.Longitude = *req.Longitude
	}
	if req.RadiusMeters != nil {
		loc.RadiusMeters = *req.RadiusMeters
	}
	if req.SelfieRequired != nil {
		loc.SelfieRequired = *req.SelfieRequired
	}
	if req.IsActive != nil {
		loc.IsActive = *req.IsActive
	}

	if err := s.locationRepo.Update(c.Request.Context(), loc); err != nil {
		c.JSON(http.StatusInternalServerError, api.ErrorResponse{Message: "Gagal memperbarui data lokasi kantor"})
		return
	}

	uID, _ := uuid.Parse(loc.ID)
	tID, _ := uuid.Parse(loc.TenantID)

	c.JSON(http.StatusOK, api.OfficeLocationResponse{
		Id:             openapi_types.UUID(uID),
		TenantId:       openapi_types.UUID(tID),
		Name:           loc.Name,
		Address:        loc.Address,
		Latitude:       loc.Latitude,
		Longitude:      loc.Longitude,
		RadiusMeters:   loc.RadiusMeters,
		SelfieRequired: loc.SelfieRequired,
		IsActive:       loc.IsActive,
		CreatedAt:      loc.CreatedAt.Format(time.RFC3339),
		UpdatedAt:      loc.UpdatedAt.Format(time.RFC3339),
	})
}

// DeleteLocationById melayani DELETE /locations/{id} (Hapus lokasi kantor)
func (s *Server) DeleteLocationById(c *gin.Context, id openapi_types.UUID) {
	tenantID, err := s.getTenantID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, api.ErrorResponse{Message: "Tenant tidak teridentifikasi atau token tidak valid"})
		return
	}

	err = s.locationRepo.Delete(c.Request.Context(), tenantID, id.String())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, api.ErrorResponse{Message: "Lokasi kantor tidak ditemukan"})
			return
		}
		c.JSON(http.StatusInternalServerError, api.ErrorResponse{Message: "Gagal menghapus lokasi kantor"})
		return
	}

	c.JSON(http.StatusOK, api.MessageResponse{Message: "Lokasi kantor berhasil dihapus"})
}

