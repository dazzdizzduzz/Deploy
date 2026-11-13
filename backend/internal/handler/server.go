package handler

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

	"github.com/tiers-undiknas/capstone-group07-2026/backend/api"
	"github.com/tiers-undiknas/capstone-group07-2026/backend/internal/auth"
	"github.com/tiers-undiknas/capstone-group07-2026/backend/internal/repository"
	"github.com/tiers-undiknas/capstone-group07-2026/backend/internal/storage"
)

type Server struct {
	userRepo      repository.UserRepository
	locationRepo  repository.LocationRepository
	scheduleRepo  repository.ScheduleRepository
	jwtManager    *auth.JWTManager
	loginLock     *auth.LoginLockManager
	passwordReset *auth.PasswordResetManager
	attendance    AttendanceDeps                // modul absensi (ISSUE-62/64), diisi lewat EnableAttendance
	employees     repository.EmployeeRepository // API Karyawan (ISSUE-26), diisi lewat EnableEmployees
	overtime      repository.OvertimeRepository // API Lembur (ISSUE-66), diisi lewat EnableOvertime
	files         storage.ObjectStore           // upload lampiran (ISSUE-69), diisi lewat EnableFileUploads
}

func NewServer(
	userRepo repository.UserRepository,
	locationRepo repository.LocationRepository,
	scheduleRepo repository.ScheduleRepository,
	jwtManager *auth.JWTManager,
	loginLock *auth.LoginLockManager,
	passwordReset *auth.PasswordResetManager,
) *Server {
	return &Server{
		userRepo:      userRepo,
		locationRepo:  locationRepo,
		scheduleRepo:  scheduleRepo,
		jwtManager:    jwtManager,
		loginLock:     loginLock,
		passwordReset: passwordReset,
	}
}

// getTenantID mengekstrak tenant_id dari context middleware, JWT Authorization Bearer header, atau X-Tenant-ID header
func (s *Server) getTenantID(c *gin.Context) (string, error) {
	// 1. Cek dari context (misal middleware guard sudah men-set 'tenant_id' atau 'tid')
	if tid := c.GetString("tenant_id"); tid != "" {
		return tid, nil
	}
	if tid := c.GetString("tid"); tid != "" {
		return tid, nil
	}

	// 2. Cek dari Authorization Bearer header
	authHeader := c.GetHeader("Authorization")
	if strings.HasPrefix(authHeader, "Bearer ") {
		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
		claims, err := s.jwtManager.ValidateToken(tokenStr)
		if err == nil && claims.Tid != "" {
			return claims.Tid, nil
		}
	}

	// 3. Fallback header X-Tenant-ID (untuk kemudahan testing / service-to-service)
	if tid := c.GetHeader("X-Tenant-ID"); tid != "" {
		return tid, nil
	}

	return "", fmt.Errorf("tenant tidak teridentifikasi")
}

// GetHealth melayani GET /health
func (s *Server) GetHealth(c *gin.Context) {
	c.JSON(http.StatusOK, api.HealthResponse{
		Status: "ok",
	})
}

// PostAuthLogin melayani POST /auth/login
// - Verifikasi email dan password hash
// - LoginLock (blokir 15 menit jika salah 5 kali) -> HTTP 423
// - Penerbitan JWT RS256 dengan klaim sub, tid, dan role (15 menit) -> HTTP 200
func (s *Server) PostAuthLogin(c *gin.Context) {
	var req api.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, api.ErrorResponse{
			Message: "Permintaan login tidak valid, periksa format email dan kata sandi",
		})
		return
	}

	email := strings.ToLower(strings.TrimSpace(string(req.Email)))
	if email == "" || req.Password == "" {
		c.JSON(http.StatusBadRequest, api.ErrorResponse{
			Message: "Email dan kata sandi wajib diisi",
		})
		return
	}

	// 1. Cek apakah akun sedang terkunci karena limit percobaan
	if locked, remaining := s.loginLock.IsLocked(email); locked {
		c.JSON(http.StatusLocked, api.ErrorResponse{
			Message: fmt.Sprintf("Akun Anda terkunci sementara karena 5 kali percobaan login gagal. Silakan coba lagi dalam %d menit.", int(remaining.Minutes())+1),
		})
		return
	}

	// 2. Cari user di database berdasarkan email
	user, err := s.userRepo.FindByEmail(email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, api.ErrorResponse{
			Message: "Terjadi kesalahan internal pada server",
		})
		return
	}

	if user == nil {
		locked, attemptsLeft := s.loginLock.RecordFailure(email)
		if locked {
			c.JSON(http.StatusLocked, api.ErrorResponse{
				Message: "Akun Anda terkunci selama 15 menit karena mencapai batas 5 kali percobaan login gagal.",
			})
			return
		}
		c.JSON(http.StatusUnauthorized, api.ErrorResponse{
			Message: fmt.Sprintf("Email atau kata sandi tidak valid. Sisa percobaan: %d", attemptsLeft),
		})
		return
	}

	if !user.IsActive {
		c.JSON(http.StatusUnauthorized, api.ErrorResponse{
			Message: "Akun Anda sedang dinonaktifkan, hubungi administrator",
		})
		return
	}

	// 3. Verifikasi hash password (bcrypt)
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		locked, attemptsLeft := s.loginLock.RecordFailure(email)
		if locked {
			c.JSON(http.StatusLocked, api.ErrorResponse{
				Message: "Akun Anda terkunci selama 15 menit karena mencapai batas 5 kali percobaan login gagal.",
			})
			return
		}
		c.JSON(http.StatusUnauthorized, api.ErrorResponse{
			Message: fmt.Sprintf("Email atau kata sandi tidak valid. Sisa percobaan: %d", attemptsLeft),
		})
		return
	}

	// 4. Reset counter percobaan saat login berhasil
	s.loginLock.RecordSuccess(email)

	// 5. Terbitkan token JWT RS256 (klaim: sub, tid, role, masa berlaku 15 menit)
	tokenString, expiresIn, err := s.jwtManager.GenerateToken(user.ID, user.TenantID, user.Role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, api.ErrorResponse{
			Message: "Gagal menerbitkan token otentikasi",
		})
		return
	}

	c.JSON(http.StatusOK, api.LoginResponse{
		AccessToken: tokenString,
		TokenType:   "Bearer",
		ExpiresIn:   expiresIn,
	})
}

// PostAuthLogout melayani POST /auth/logout
func (s *Server) PostAuthLogout(c *gin.Context) {
	c.JSON(http.StatusOK, api.MessageResponse{
		Message: "Logout berhasil",
	})
}

// PostAuthPasswordForgot melayani POST /auth/password/forgot
// Membalas status 202 Accepted tanpa memvalidasi secara publik apakah email terdaftar (anti-user enumeration).
// Jika user ditemukan di database, membuat token reset dan mencetak ke log konsol (simulasi SMTP).
func (s *Server) PostAuthPasswordForgot(c *gin.Context) {
	var req api.ForgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, api.ErrorResponse{
			Message: "Format permintaan tidak valid, periksa format email",
		})
		return
	}

	email := strings.ToLower(strings.TrimSpace(string(req.Email)))
	if email == "" {
		c.JSON(http.StatusBadRequest, api.ErrorResponse{
			Message: "Email wajib diisi",
		})
		return
	}

	// Cari user secara internal di database
	user, err := s.userRepo.FindByEmail(email)
	if err == nil && user != nil && user.IsActive {
		// Buat token reset dan simulasikan pencetakan ke konsol (Requirement 2)
		_, _ = s.passwordReset.GenerateToken(user.ID, user.Email)
	}

	// Selalu balas status 202 Accepted tanpa membocorkan apakah email terdaftar (Requirement 1)
	c.JSON(http.StatusAccepted, api.MessageResponse{
		Message: "Jika email Anda terdaftar pada sistem kami, instruksi dan token reset kata sandi telah dikirimkan.",
	})
}

// PostAuthPasswordReset melayani POST /auth/password/reset
// Memvalidasi token dari klien, memperbarui hash kata sandi di database menggunakan bcrypt (NFR-04).
func (s *Server) PostAuthPasswordReset(c *gin.Context) {
	var req api.ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, api.ErrorResponse{
			Message: "Format permintaan tidak valid",
		})
		return
	}

	token := strings.TrimSpace(req.Token)
	newPassword := req.NewPassword

	if token == "" || newPassword == "" {
		c.JSON(http.StatusBadRequest, api.ErrorResponse{
			Message: "Token reset dan kata sandi baru wajib diisi",
		})
		return
	}

	// 1. Validasi token dan konsumsi (single-use)
	userID, email, err := s.passwordReset.ValidateAndConsumeToken(token)
	if err != nil {
		c.JSON(http.StatusBadRequest, api.ErrorResponse{
			Message: "Token reset kata sandi tidak valid atau telah kedaluwarsa",
		})
		return
	}

	// 2. Hash kata sandi baru dengan algoritma yang sama (bcrypt cost 12 - NFR-04)
	newHash, err := bcrypt.GenerateFromPassword([]byte(newPassword), 12)
	if err != nil {
		c.JSON(http.StatusInternalServerError, api.ErrorResponse{
			Message: "Gagal memproses keamanan kata sandi baru",
		})
		return
	}

	// 3. Perbarui hash kata sandi di database
	if err := s.userRepo.UpdatePasswordHash(userID, string(newHash)); err != nil {
		c.JSON(http.StatusInternalServerError, api.ErrorResponse{
			Message: "Gagal memperbarui kata sandi di database",
		})
		return
	}

	// 4. Reset counter percobaan di LoginLock jika akun sebelumnya terkunci
	s.loginLock.RecordSuccess(email)

	c.JSON(http.StatusOK, api.MessageResponse{
		Message: "Kata sandi berhasil diperbarui. Silakan masuk menggunakan kata sandi baru Anda.",
	})
}
