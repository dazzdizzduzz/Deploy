package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/tiers-undiknas/capstone-group07-2026/backend/internal/attendance"
	"github.com/tiers-undiknas/capstone-group07-2026/backend/internal/handler"
	"github.com/tiers-undiknas/capstone-group07-2026/backend/internal/repository"
	"github.com/tiers-undiknas/capstone-group07-2026/backend/internal/storage"
)

// enableAttendance menyiapkan modul absensi: zona waktu tenant dan object storage selfie.
// Dipisah dari main.go agar wiring modul Backend 2 tidak bercampur dengan modul lain.
func enableAttendance(srv *handler.Server, repo repository.AttendanceRepository) {
	tz, err := attendance.LoadTimezone(os.Getenv("ATTENDANCE_TIMEZONE"))
	if err != nil {
		log.Fatalf("ATTENDANCE_TIMEZONE tidak valid: %v", err)
	}

	// Variabel bertipe interface dibiarkan nil bila storage tidak aktif. Jangan mengisinya
	// dengan *MinIOStore yang nil, karena interface berisi pointer nil tidak sama dengan nil.
	var store storage.ObjectStore
	cfg := storage.ConfigFromEnv()
	if cfg.Enabled() {
		minioStore, err := storage.NewMinIOStore(cfg)
		if err != nil {
			log.Printf("Peringatan: object storage tidak aktif: %v", err)
		} else {
			ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout+5*time.Second)
			if err := minioStore.EnsureBucket(ctx); err != nil {
				// Server tetap jalan; unggahan akan dicoba lagi per request dan dijawab 503 bila gagal.
				log.Printf("Peringatan: bucket %q belum siap: %v", cfg.Bucket, err)
			}
			cancel()
			store = minioStore
			log.Printf("Object storage aktif: bucket %q di %s", cfg.Bucket, cfg.Endpoint)
		}
	} else {
		log.Println("Peringatan: S3_* belum diisi; clock-in yang mewajibkan selfie akan dijawab 503")
	}

	// Storage yang sama dipakai untuk upload lampiran izin (ISSUE-69), agar hanya ada satu klien.
	srv.EnableFileUploads(store)

	if err := srv.EnableAttendance(handler.AttendanceDeps{Repo: repo, Store: store, Timezone: tz}); err != nil {
		log.Fatalf("Gagal mengaktifkan modul absensi: %v", err)
	}
	log.Printf("Modul absensi aktif (zona waktu %s)", tz)
}
