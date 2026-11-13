package main

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/tiers-undiknas/capstone-group07-2026/backend/api"
	"github.com/tiers-undiknas/capstone-group07-2026/backend/internal/audit"
	"github.com/tiers-undiknas/capstone-group07-2026/backend/internal/auth"
	"github.com/tiers-undiknas/capstone-group07-2026/backend/internal/config"
	"github.com/tiers-undiknas/capstone-group07-2026/backend/internal/database"
	"github.com/tiers-undiknas/capstone-group07-2026/backend/internal/extconfig"
	"github.com/tiers-undiknas/capstone-group07-2026/backend/internal/handler"
	"github.com/tiers-undiknas/capstone-group07-2026/backend/internal/middleware"
	"github.com/tiers-undiknas/capstone-group07-2026/backend/internal/repository"
)

func main() {
	cfg := config.LoadConfig()

	// 0. Validasi konfigurasi layanan eksternal Backend 2 (ISSUE-03). Konfigurasi yang
	// terisi sebagian atau salah format menghentikan aplikasi dengan pesan yang jelas.
	extCfg, err := extconfig.Load()
	if err != nil {
		log.Fatalf("Konfigurasi environment tidak valid, periksa .env (lihat .env.example):\n%v", err)
	}
	log.Printf("Layanan eksternal: %s", extCfg.Summary())

	// 1. Inisialisasi Database PostgreSQL
	db, err := database.NewPostgresDB(cfg)
	if err != nil {
		log.Printf("Peringatan: Gagal terhubung ke database: %v", err)
	} else {
		defer func() {
			_ = db.Close()
		}()
		log.Println("Berhasil terhubung ke database PostgreSQL")
	}

	// 2. Inisialisasi Repository & Manager
	userRepo := repository.NewUserRepository(db)
	locationRepo := repository.NewLocationRepository(db)
	scheduleRepo := repository.NewScheduleRepository(db)

	jwtManager, err := auth.NewJWTManager(cfg.JWTPrivateKey, cfg.JWTPublicKey)
	if err != nil {
		log.Fatalf("Gagal inisialisasi JWT Manager: %v", err)
	}

	loginLock := auth.NewLoginLockManager()
	passwordReset := auth.NewPasswordResetManager()
	auditLogger := audit.NewConsoleLogger()

	// 3. Inisialisasi Server Handler
	srv := handler.NewServer(userRepo, locationRepo, scheduleRepo, jwtManager, loginLock, passwordReset)

	// 3b. Modul absensi & object storage selfie (Backend 2, ISSUE-62/64)
	enableAttendance(srv, repository.NewAttendanceRepository(db))
	srv.EnableEmployees(repository.NewEmployeeRepository(db)) // API Karyawan (ISSUE-26)
	srv.EnableOvertime(repository.NewOvertimeRepository(db))   // API Lembur (ISSUE-66)

	// 4. Inisialisasi Gin Router
	r := gin.Default()

	// Global CORS
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, PATCH, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	})

	api.RegisterHandlers(r, srv)

	// Protected API v1 dengan Middleware Guard JWT RS256 & RBAC (FR-04, US-11 AC-1)
	v1 := r.Group("/v1")
	v1.Use(middleware.Guard(middleware.GuardOptions{
		JWTService:  jwtManager,
		AuditLogger: auditLogger,
	}))
	{
		v1.GET("/me", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"id":        c.GetString("user_id"),
				"tenant_id": c.GetString("tenant_id"),
				"role":      c.GetString("role"),
			})
		})
	}

	port := ":" + cfg.ServerPort
	log.Printf("Server OnTrack berjalan pada port %s", port)
	if err := r.Run(port); err != nil {
		log.Fatalf("Gagal menjalankan server: %v", err)
	}
}
