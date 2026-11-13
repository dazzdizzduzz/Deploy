package extconfig

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// env membuat fungsi getenv palsu dari map, agar test tidak bergantung pada environment mesin.
func env(m map[string]string) func(string) string {
	return func(k string) string { return m[k] }
}

func noFile(string) bool  { return false }
func anyFile(string) bool { return true }

func validStorage() map[string]string {
	return map[string]string{
		"S3_ENDPOINT": "minio:9000", "S3_ACCESS_KEY": "ak", "S3_SECRET_KEY": "sk", "S3_BUCKET_NAME": "ontrack-bucket",
	}
}

func problemsOf(t *testing.T, err error) string {
	t.Helper()
	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("err = %v, want *ValidationError", err)
	}
	return strings.Join(ve.Problems, "\n")
}

func TestLoad_EmptyEnvAllInactive(t *testing.T) {
	cfg, err := load(env(nil), noFile)
	if err != nil {
		t.Fatalf("env kosong seharusnya valid: %v", err)
	}
	for _, s := range allServices {
		if cfg.Enabled(s) {
			t.Fatalf("%s seharusnya nonaktif", s)
		}
	}
	want := "storage=nonaktif smtp=nonaktif fcm=nonaktif midtrans=nonaktif xendit=nonaktif"
	if cfg.Summary() != want {
		t.Fatalf("Summary = %q, want %q", cfg.Summary(), want)
	}
}

func TestLoad_StorageDefaultsAndValues(t *testing.T) {
	m := validStorage()
	m["S3_USE_SSL"] = "true"
	cfg, err := load(env(m), noFile)
	if err != nil {
		t.Fatal(err)
	}
	s := cfg.Storage
	if !cfg.Enabled(ServiceStorage) || s.Endpoint != "minio:9000" || !s.UseSSL || s.SSE ||
		s.Region != "us-east-1" || s.Timeout != defaultTimeout {
		t.Fatalf("storage = %+v", s)
	}
}

func TestLoad_PartialStorageListsMissingVars(t *testing.T) {
	_, err := load(env(map[string]string{"S3_ENDPOINT": "minio:9000"}), noFile)
	p := problemsOf(t, err)
	for _, key := range []string{"S3_ACCESS_KEY", "S3_SECRET_KEY", "S3_BUCKET_NAME"} {
		if !strings.Contains(p, "storage: "+key+" wajib diisi") {
			t.Fatalf("masalah tidak menyebut %s:\n%s", key, p)
		}
	}
	if strings.Contains(p, "S3_ENDPOINT") {
		t.Fatalf("S3_ENDPOINT sudah diisi, tidak boleh dilaporkan:\n%s", p)
	}
}

func TestLoad_InvalidFormats(t *testing.T) {
	cases := []struct {
		name  string
		extra map[string]string
		want  string
	}{
		{"bool storage", map[string]string{"S3_USE_SSL": "ya"}, "S3_USE_SSL harus true atau false"},
		{"timeout nol", map[string]string{"S3_TIMEOUT_SECONDS": "0"}, "S3_TIMEOUT_SECONDS harus bilangan bulat"},
		{"timeout teks", map[string]string{"S3_TIMEOUT_SECONDS": "lama"}, "S3_TIMEOUT_SECONDS harus bilangan bulat"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			m := validStorage()
			for k, v := range c.extra {
				m[k] = v
			}
			_, err := load(env(m), noFile)
			if p := problemsOf(t, err); !strings.Contains(p, c.want) {
				t.Fatalf("masalah tidak memuat %q:\n%s", c.want, p)
			}
		})
	}
}

func TestLoad_SMTP(t *testing.T) {
	base := map[string]string{
		"SMTP_HOST": "smtp.example.com", "SMTP_PORT": "587", "SMTP_FROM": "On-Track <no-reply@ontrack.id>",
		"SMTP_USERNAME": "user", "SMTP_PASSWORD": "pass", "SMTP_TIMEOUT_SECONDS": "5",
	}
	cfg, err := load(env(base), noFile)
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.Enabled(ServiceSMTP) || cfg.SMTP.Port != 587 || cfg.SMTP.Timeout != 5*time.Second {
		t.Fatalf("smtp = %+v", cfg.SMTP)
	}

	cases := []struct {
		key, value, want string
	}{
		{"SMTP_PORT", "70000", "SMTP_PORT harus nomor port"},
		{"SMTP_PORT", "", "SMTP_PORT wajib diisi"},
		{"SMTP_FROM", "bukan-email", "SMTP_FROM bukan alamat email"},
		{"SMTP_PASSWORD", "", "harus diisi berpasangan"},
		{"SMTP_HOST", "", "SMTP_HOST wajib diisi"},
	}
	for _, c := range cases {
		m := map[string]string{}
		for k, v := range base {
			m[k] = v
		}
		m[c.key] = c.value
		_, err := load(env(m), noFile)
		if p := problemsOf(t, err); !strings.Contains(p, c.want) {
			t.Errorf("%s=%q: masalah tidak memuat %q:\n%s", c.key, c.value, c.want, p)
		}
	}

	// Tanpa autentikasi (relay internal) tetap valid.
	noAuth := map[string]string{"SMTP_HOST": "mailhog", "SMTP_PORT": "1025", "SMTP_FROM": "dev@ontrack.id"}
	if _, err := load(env(noAuth), noFile); err != nil {
		t.Fatalf("SMTP tanpa autentikasi seharusnya valid: %v", err)
	}
}

func TestLoad_FCMCredentialsFile(t *testing.T) {
	m := map[string]string{"FCM_PROJECT_ID": "ontrack-dev", "FCM_CREDENTIALS_FILE": "/rahasia/fcm.json"}
	if p := problemsOf(t, mustErr(load(env(m), noFile))); !strings.Contains(p, "FCM_CREDENTIALS_FILE tidak ditemukan") {
		t.Fatalf("file hilang tidak dilaporkan:\n%s", p)
	}
	cfg, err := load(env(m), anyFile)
	if err != nil || !cfg.Enabled(ServiceFCM) {
		t.Fatalf("FCM dengan file ada seharusnya aktif: %v", err)
	}
}

func TestLoad_PaymentGateways(t *testing.T) {
	_, err := load(env(map[string]string{"MIDTRANS_SERVER_KEY": "x"}), noFile)
	if p := problemsOf(t, err); !strings.Contains(p, "midtrans: MIDTRANS_CLIENT_KEY wajib diisi") {
		t.Fatalf("midtrans parsial:\n%s", p)
	}
	_, err = load(env(map[string]string{"XENDIT_CALLBACK_TOKEN": "x"}), noFile)
	if p := problemsOf(t, err); !strings.Contains(p, "xendit: XENDIT_SECRET_KEY wajib diisi") {
		t.Fatalf("xendit parsial:\n%s", p)
	}
	cfg, err := load(env(map[string]string{
		"MIDTRANS_SERVER_KEY": "s", "MIDTRANS_CLIENT_KEY": "c", "MIDTRANS_IS_PRODUCTION": "false",
		"XENDIT_SECRET_KEY": "x", "XENDIT_CALLBACK_TOKEN": "t",
	}), noFile)
	if err != nil || !cfg.Enabled(ServiceMidtrans) || !cfg.Enabled(ServiceXendit) || cfg.Midtrans.IsProduction {
		t.Fatalf("payment gateway lengkap: %+v %v", cfg, err)
	}
}

func TestLoad_RequiredServices(t *testing.T) {
	m := validStorage()
	m["EXTERNAL_SERVICES_REQUIRED"] = " Storage , smtp "
	p := problemsOf(t, mustErr(load(env(m), noFile)))
	if !strings.Contains(p, "smtp: layanan diwajibkan") || strings.Contains(p, "storage:") {
		t.Fatalf("hanya smtp yang seharusnya dilaporkan:\n%s", p)
	}

	m["EXTERNAL_SERVICES_REQUIRED"] = "storage,email"
	p = problemsOf(t, mustErr(load(env(m), noFile)))
	if !strings.Contains(p, `layanan tidak dikenal "email"`) {
		t.Fatalf("nama layanan salah tidak dilaporkan:\n%s", p)
	}

	m["EXTERNAL_SERVICES_REQUIRED"] = "storage"
	if _, err := load(env(m), noFile); err != nil {
		t.Fatalf("storage lengkap dan diwajibkan seharusnya valid: %v", err)
	}
}

// Nilai secret tidak boleh bocor ke pesan error atau ringkasan log.
func TestLoad_NeverLeaksSecretValues(t *testing.T) {
	const secret = "NILAI-RAHASIA-JANGAN-BOCOR"
	_, err := load(env(map[string]string{
		"S3_SECRET_KEY": secret, "SMTP_PASSWORD": secret, "MIDTRANS_SERVER_KEY": secret, "XENDIT_SECRET_KEY": secret,
	}), noFile)
	if err == nil || strings.Contains(err.Error(), secret) {
		t.Fatalf("pesan error membocorkan secret atau tidak error: %v", err)
	}

	m := validStorage()
	m["S3_SECRET_KEY"] = secret
	cfg, err := load(env(m), noFile)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(cfg.Summary(), secret) {
		t.Fatal("Summary membocorkan secret")
	}
}

func TestLoad_ErrorDisablesEverything(t *testing.T) {
	m := validStorage()
	m["SMTP_HOST"] = "smtp.example.com" // smtp tidak lengkap
	cfg, err := load(env(m), noFile)
	if err == nil || cfg.Enabled(ServiceStorage) {
		t.Fatalf("saat ada error, tidak boleh ada layanan yang dianggap aktif: %v", err)
	}
}

// Load memakai environment proses dan file sungguhan.
func TestLoad_FromProcessEnv(t *testing.T) {
	cred := filepath.Join(t.TempDir(), "fcm.json")
	if err := os.WriteFile(cred, []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, keys := range serviceVars {
		for _, k := range keys {
			t.Setenv(k, "")
		}
	}
	t.Setenv("EXTERNAL_SERVICES_REQUIRED", "fcm")
	t.Setenv("FCM_PROJECT_ID", "ontrack-dev")
	t.Setenv("FCM_CREDENTIALS_FILE", cred)

	cfg, err := Load()
	if err != nil || !cfg.Enabled(ServiceFCM) {
		t.Fatalf("Load dari env proses: %v", err)
	}
}

func mustErr(_ Config, err error) error { return err }
