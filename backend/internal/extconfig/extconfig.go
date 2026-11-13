// Package extconfig membaca dan memvalidasi konfigurasi layanan eksternal yang dipegang
// Backend 2 (ISSUE-03): object storage, SMTP, FCM, Midtrans, dan Xendit.
//
// Aturan:
//   - Layanan yang tidak diisi sama sekali dianggap nonaktif (aplikasi tetap jalan, cocok
//     untuk pengembangan lokal).
//   - Layanan yang diisi sebagian atau formatnya salah dianggap error: aplikasi harus berhenti
//     dengan pesan yang menyebut nama variabelnya.
//   - EXTERNAL_SERVICES_REQUIRED (contoh "storage,smtp") mewajibkan layanan tertentu aktif,
//     misalnya di production.
//   - Nilai secret tidak pernah ditulis ke pesan error maupun ringkasan log, hanya nama variabel.
package extconfig

import (
	"fmt"
	"net/mail"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Nama layanan untuk EXTERNAL_SERVICES_REQUIRED dan ringkasan log.
const (
	ServiceStorage  = "storage"
	ServiceSMTP     = "smtp"
	ServiceFCM      = "fcm"
	ServiceMidtrans = "midtrans"
	ServiceXendit   = "xendit"
)

const defaultTimeout = 10 * time.Second

// Storage: object storage S3-compatible (MinIO di lokal). Nama variabel S3_* mengikuti
// docker-compose.yml.
type Storage struct {
	Endpoint  string
	AccessKey string
	SecretKey string
	Bucket    string
	Region    string
	UseSSL    bool
	SSE       bool
	Timeout   time.Duration
}

// SMTP: pengiriman email reset kata sandi dan verifikasi email Owner.
type SMTP struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
	Timeout  time.Duration
}

// FCM: push notification. Kredensial berupa path ke file JSON service account,
// bukan isi JSON-nya, agar secret tidak masuk ke environment atau log.
type FCM struct {
	ProjectID       string
	CredentialsFile string
}

// Midtrans: payment gateway (Fase 3).
type Midtrans struct {
	ServerKey    string
	ClientKey    string
	IsProduction bool
}

// Xendit: payment gateway alternatif (Fase 3).
type Xendit struct {
	SecretKey     string
	CallbackToken string
}

// Config berisi konfigurasi semua layanan eksternal Backend 2.
type Config struct {
	Storage  Storage
	SMTP     SMTP
	FCM      FCM
	Midtrans Midtrans
	Xendit   Xendit

	enabled map[string]bool
}

// Enabled bernilai true bila layanan dikonfigurasi lengkap.
func (c Config) Enabled(service string) bool {
	return c.enabled[service]
}

// Summary menghasilkan ringkasan status layanan untuk log, tanpa nilai secret.
// Contoh: "storage=aktif smtp=nonaktif fcm=nonaktif midtrans=nonaktif xendit=nonaktif".
func (c Config) Summary() string {
	parts := make([]string, 0, len(allServices))
	for _, s := range allServices {
		state := "nonaktif"
		if c.enabled[s] {
			state = "aktif"
		}
		parts = append(parts, s+"="+state)
	}
	return strings.Join(parts, " ")
}

var allServices = []string{ServiceStorage, ServiceSMTP, ServiceFCM, ServiceMidtrans, ServiceXendit}

// serviceVars mendaftar semua variabel per layanan. Layanan dianggap "diisi" bila salah satu
// variabelnya tidak kosong.
var serviceVars = map[string][]string{
	ServiceStorage:  {"S3_ENDPOINT", "S3_ACCESS_KEY", "S3_SECRET_KEY", "S3_BUCKET_NAME", "S3_USE_SSL", "S3_REGION", "S3_SSE", "S3_TIMEOUT_SECONDS"},
	ServiceSMTP:     {"SMTP_HOST", "SMTP_PORT", "SMTP_USERNAME", "SMTP_PASSWORD", "SMTP_FROM", "SMTP_TIMEOUT_SECONDS"},
	ServiceFCM:      {"FCM_PROJECT_ID", "FCM_CREDENTIALS_FILE"},
	ServiceMidtrans: {"MIDTRANS_SERVER_KEY", "MIDTRANS_CLIENT_KEY", "MIDTRANS_IS_PRODUCTION"},
	ServiceXendit:   {"XENDIT_SECRET_KEY", "XENDIT_CALLBACK_TOKEN"},
}

// ValidationError mengumpulkan semua masalah sekaligus agar developer tidak perlu
// memperbaiki satu per satu sambil restart berulang kali.
type ValidationError struct {
	Problems []string
}

func (e *ValidationError) Error() string {
	return "konfigurasi layanan eksternal tidak valid:\n- " + strings.Join(e.Problems, "\n- ")
}

// Load membaca konfigurasi dari variabel environment proses.
func Load() (Config, error) {
	return load(os.Getenv, fileExists)
}

// load menerima fungsi pembaca env dan pemeriksa file agar mudah diuji tanpa mengubah
// environment proses.
func load(getenv func(string) string, exists func(string) bool) (Config, error) {
	r := reader{getenv: getenv}
	cfg := Config{enabled: map[string]bool{}}

	if r.anySet(ServiceStorage) {
		cfg.Storage = Storage{
			Endpoint:  r.required(ServiceStorage, "S3_ENDPOINT"),
			AccessKey: r.required(ServiceStorage, "S3_ACCESS_KEY"),
			SecretKey: r.required(ServiceStorage, "S3_SECRET_KEY"),
			Bucket:    r.required(ServiceStorage, "S3_BUCKET_NAME"),
			Region:    r.withDefault("S3_REGION", "us-east-1"),
			UseSSL:    r.boolean(ServiceStorage, "S3_USE_SSL"),
			SSE:       r.boolean(ServiceStorage, "S3_SSE"),
			Timeout:   r.seconds(ServiceStorage, "S3_TIMEOUT_SECONDS"),
		}
		cfg.enabled[ServiceStorage] = true
	}

	if r.anySet(ServiceSMTP) {
		cfg.SMTP = SMTP{
			Host:     r.required(ServiceSMTP, "SMTP_HOST"),
			Port:     r.port(ServiceSMTP, "SMTP_PORT"),
			Username: r.value("SMTP_USERNAME"),
			Password: r.value("SMTP_PASSWORD"),
			From:     r.email(ServiceSMTP, "SMTP_FROM"),
			Timeout:  r.seconds(ServiceSMTP, "SMTP_TIMEOUT_SECONDS"),
		}
		// Autentikasi SMTP harus berpasangan: username tanpa password (atau sebaliknya)
		// hampir pasti salah ketik.
		if (cfg.SMTP.Username == "") != (cfg.SMTP.Password == "") {
			r.problem(ServiceSMTP, "SMTP_USERNAME dan SMTP_PASSWORD harus diisi berpasangan")
		}
		cfg.enabled[ServiceSMTP] = true
	}

	if r.anySet(ServiceFCM) {
		cfg.FCM = FCM{
			ProjectID:       r.required(ServiceFCM, "FCM_PROJECT_ID"),
			CredentialsFile: r.required(ServiceFCM, "FCM_CREDENTIALS_FILE"),
		}
		if cfg.FCM.CredentialsFile != "" && !exists(cfg.FCM.CredentialsFile) {
			r.problem(ServiceFCM, "file pada FCM_CREDENTIALS_FILE tidak ditemukan atau tidak bisa dibaca")
		}
		cfg.enabled[ServiceFCM] = true
	}

	if r.anySet(ServiceMidtrans) {
		cfg.Midtrans = Midtrans{
			ServerKey:    r.required(ServiceMidtrans, "MIDTRANS_SERVER_KEY"),
			ClientKey:    r.required(ServiceMidtrans, "MIDTRANS_CLIENT_KEY"),
			IsProduction: r.boolean(ServiceMidtrans, "MIDTRANS_IS_PRODUCTION"),
		}
		cfg.enabled[ServiceMidtrans] = true
	}

	if r.anySet(ServiceXendit) {
		cfg.Xendit = Xendit{
			SecretKey:     r.required(ServiceXendit, "XENDIT_SECRET_KEY"),
			CallbackToken: r.required(ServiceXendit, "XENDIT_CALLBACK_TOKEN"),
		}
		cfg.enabled[ServiceXendit] = true
	}

	// Layanan yang diwajibkan lewat EXTERNAL_SERVICES_REQUIRED harus aktif.
	for _, name := range splitList(r.value("EXTERNAL_SERVICES_REQUIRED")) {
		if _, known := serviceVars[name]; !known {
			r.problems = append(r.problems, fmt.Sprintf(
				"EXTERNAL_SERVICES_REQUIRED berisi layanan tidak dikenal %q (pilihan: %s)",
				name, strings.Join(allServices, ", ")))
			continue
		}
		if !cfg.enabled[name] {
			r.problem(name, "layanan diwajibkan oleh EXTERNAL_SERVICES_REQUIRED tetapi belum dikonfigurasi")
		}
	}

	if len(r.problems) > 0 {
		sort.Strings(r.problems)
		// Layanan yang bermasalah tidak boleh dianggap aktif.
		return Config{enabled: map[string]bool{}}, &ValidationError{Problems: r.problems}
	}
	return cfg, nil
}

// reader membaca variabel env dan mencatat masalah. Pesan masalah hanya memuat nama
// variabel, tidak pernah nilainya.
type reader struct {
	getenv   func(string) string
	problems []string
}

func (r *reader) value(key string) string {
	return strings.TrimSpace(r.getenv(key))
}

func (r *reader) anySet(service string) bool {
	for _, key := range serviceVars[service] {
		if r.value(key) != "" {
			return true
		}
	}
	return false
}

func (r *reader) problem(service, msg string) {
	r.problems = append(r.problems, service+": "+msg)
}

func (r *reader) required(service, key string) string {
	v := r.value(key)
	if v == "" {
		r.problem(service, key+" wajib diisi")
	}
	return v
}

func (r *reader) withDefault(key, def string) string {
	if v := r.value(key); v != "" {
		return v
	}
	return def
}

func (r *reader) boolean(service, key string) bool {
	v := r.value(key)
	if v == "" {
		return false
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		r.problem(service, key+" harus true atau false")
		return false
	}
	return b
}

func (r *reader) seconds(service, key string) time.Duration {
	v := r.value(key)
	if v == "" {
		return defaultTimeout
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 || n > 300 {
		r.problem(service, key+" harus bilangan bulat 1 sampai 300 (detik)")
		return defaultTimeout
	}
	return time.Duration(n) * time.Second
}

func (r *reader) port(service, key string) int {
	v := r.value(key)
	if v == "" {
		r.problem(service, key+" wajib diisi")
		return 0
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 1 || n > 65535 {
		r.problem(service, key+" harus nomor port 1 sampai 65535")
		return 0
	}
	return n
}

func (r *reader) email(service, key string) string {
	v := r.required(service, key)
	if v == "" {
		return ""
	}
	if _, err := mail.ParseAddress(v); err != nil {
		r.problem(service, key+" bukan alamat email yang valid")
	}
	return v
}

func splitList(s string) []string {
	var out []string
	for _, part := range strings.Split(s, ",") {
		if p := strings.ToLower(strings.TrimSpace(part)); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
