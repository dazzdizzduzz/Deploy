// Package storage menangani penyimpanan berkas di object storage S3-compatible (MinIO di lokal):
// selfie absensi dan lampiran izin/task. Aturan dari tugas Backend 2:
//   - path per tenant: {tenant_id}/{jenis}/{nama}
//   - foto maksimal 2 MB (JPEG/PNG)
//   - akses baca lewat signed URL berumur pendek, bucket tidak publik
//   - enkripsi di sisi storage (SAD NFR-06), diaktifkan lewat S3_SSE=true
//
// Setiap panggilan ke storage memakai timeout agar request API tidak menggantung
// bila MinIO lambat atau mati.
package storage

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

// MaxPhotoBytes adalah batas ukuran foto selfie (2 MB, kontrak API: 413 bila lebih).
const MaxPhotoBytes int64 = 2 << 20

// MaxAttachmentBytes adalah batas lampiran izin/task (5 MB, draf kontrak
// api/drafts/leave-requests.openapi.yaml: 413 ATTACHMENT_TOO_LARGE bila lebih).
const MaxAttachmentBytes int64 = 5 << 20

// Tipe berkas yang diterima, ditentukan dari isi berkas (magic bytes), dipetakan ke ekstensi.
var (
	photoTypes      = map[string]string{"image/jpeg": ".jpg", "image/png": ".png"}
	attachmentTypes = map[string]string{"image/jpeg": ".jpg", "image/png": ".png", "application/pdf": ".pdf"}
)

// Kind adalah jenis berkas, menjadi segmen kedua path objek.
type Kind string

const (
	KindSelfie          Kind = "selfie"
	KindLeaveAttachment Kind = "leave"
	KindTaskAttachment  Kind = "task"
)

var (
	// ErrNotConfigured: variabel lingkungan S3_* belum diisi.
	ErrNotConfigured = errors.New("object storage belum dikonfigurasi")
	// ErrUnavailable: storage tidak merespons, timeout, atau menolak permintaan.
	ErrUnavailable = errors.New("object storage tidak dapat diakses")
	// ErrTooLarge: berkas melebihi batas ukuran.
	ErrTooLarge = errors.New("ukuran berkas melebihi batas")
	// ErrUnsupportedType: format berkas tidak diizinkan (selfie: JPEG/PNG; lampiran: JPEG/PNG/PDF).
	ErrUnsupportedType = errors.New("format berkas tidak didukung")
	// ErrInvalidKey: tenant_id atau jenis berkas tidak valid untuk membentuk path.
	ErrInvalidKey = errors.New("path objek tidak valid")
)

// ObjectStore adalah kontrak yang dipakai handler. Handler bergantung pada interface ini,
// bukan pada MinIO langsung, sehingga bisa diuji dengan implementasi palsu.
type ObjectStore interface {
	Put(ctx context.Context, key string, body io.Reader, size int64, contentType string) error
	Delete(ctx context.Context, key string) error
	PresignGet(ctx context.Context, key string, expiry time.Duration) (string, error)
}

// Config dibaca dari variabel lingkungan yang sudah ada di docker-compose.yml dan .env.example.
type Config struct {
	Endpoint  string // S3_ENDPOINT, contoh minio:9000 (tanpa skema http://)
	AccessKey string // S3_ACCESS_KEY
	SecretKey string // S3_SECRET_KEY
	Bucket    string // S3_BUCKET_NAME
	UseSSL    bool   // S3_USE_SSL
	Region    string // S3_REGION, default us-east-1
	SSE       bool   // S3_SSE: enkripsi sisi server (MinIO butuh KMS aktif)
	Timeout   time.Duration
}

// ConfigFromEnv membaca konfigurasi storage. Tidak ada nilai default untuk kredensial:
// secret tidak boleh di-hardcode (Panduan Capstone).
func ConfigFromEnv() Config {
	timeout := 10 * time.Second
	if v, err := strconv.Atoi(os.Getenv("S3_TIMEOUT_SECONDS")); err == nil && v > 0 {
		timeout = time.Duration(v) * time.Second
	}
	region := os.Getenv("S3_REGION")
	if region == "" {
		region = "us-east-1"
	}
	return Config{
		Endpoint:  os.Getenv("S3_ENDPOINT"),
		AccessKey: os.Getenv("S3_ACCESS_KEY"),
		SecretKey: os.Getenv("S3_SECRET_KEY"),
		Bucket:    os.Getenv("S3_BUCKET_NAME"),
		UseSSL:    parseBool(os.Getenv("S3_USE_SSL")),
		Region:    region,
		SSE:       parseBool(os.Getenv("S3_SSE")),
		Timeout:   timeout,
	}
}

// Enabled bernilai true bila semua variabel wajib terisi.
func (c Config) Enabled() bool {
	return c.Endpoint != "" && c.AccessKey != "" && c.SecretKey != "" && c.Bucket != ""
}

// ObjectKey membentuk path {tenant_id}/{jenis}/{uuid}{ext}. Nama berkas dari klien tidak
// dipakai sama sekali, sehingga tidak ada risiko path traversal ("../") atau nama bentrok.
func ObjectKey(tenantID string, kind Kind, ext string) (string, error) {
	if _, err := uuid.Parse(tenantID); err != nil {
		return "", fmt.Errorf("%w: tenant_id bukan UUID", ErrInvalidKey)
	}
	switch kind {
	case KindSelfie, KindLeaveAttachment, KindTaskAttachment:
	default:
		return "", fmt.Errorf("%w: jenis %q tidak dikenal", ErrInvalidKey, kind)
	}
	return fmt.Sprintf("%s/%s/%s%s", tenantID, kind, uuid.NewString(), ext), nil
}

// File adalah berkas yang sudah dibaca utuh ke memori dan lolos validasi.
type File struct {
	Data        []byte
	ContentType string // hasil deteksi isi berkas, contoh image/jpeg atau application/pdf
	Ext         string // ekstensi sesuai tipe, contoh .jpg atau .pdf
}

// Photo dipertahankan agar kode selfie absensi yang sudah ada tidak berubah.
type Photo = File

// ReadPhoto membaca foto selfie: JPEG/PNG, maksimal MaxPhotoBytes.
func ReadPhoto(r io.Reader) (Photo, error) {
	return readFile(r, MaxPhotoBytes, photoTypes)
}

// ReadAttachment membaca lampiran izin/task (contoh surat dokter): JPEG/PNG/PDF,
// maksimal MaxAttachmentBytes.
func ReadAttachment(r io.Reader) (File, error) {
	return readFile(r, MaxAttachmentBytes, attachmentTypes)
}

// readFile membaca berkas dengan batas ukuran, lalu menentukan tipenya dari isi berkas
// (magic bytes). Header Content-Type dan nama berkas dari klien tidak dipercaya karena
// mudah dipalsukan.
func readFile(r io.Reader, maxBytes int64, allowed map[string]string) (File, error) {
	// Baca maksimal satu byte lebih dari batas agar berkas yang terlalu besar terdeteksi
	// tanpa harus membaca seluruhnya ke memori.
	data, err := io.ReadAll(io.LimitReader(r, maxBytes+1))
	if err != nil {
		return File{}, fmt.Errorf("gagal membaca berkas: %w", err)
	}
	if int64(len(data)) > maxBytes {
		return File{}, ErrTooLarge
	}
	if len(data) == 0 {
		return File{}, ErrUnsupportedType
	}

	ct := http.DetectContentType(data)
	ext, ok := allowed[ct]
	if !ok {
		return File{}, ErrUnsupportedType
	}
	return File{Data: data, ContentType: ct, Ext: ext}, nil
}

// PutFile menyimpan File ke key tertentu.
func PutFile(ctx context.Context, store ObjectStore, key string, f File) error {
	return store.Put(ctx, key, bytes.NewReader(f.Data), int64(len(f.Data)), f.ContentType)
}

// PutPhoto adalah jalan pintas untuk menyimpan foto selfie.
func PutPhoto(ctx context.Context, store ObjectStore, key string, p Photo) error {
	return PutFile(ctx, store, key, p)
}

func parseBool(v string) bool {
	b, err := strconv.ParseBool(strings.TrimSpace(v))
	return err == nil && b
}

// KeyBelongsToTenant memastikan key objek dibuat untuk tenant dan jenis berkas tertentu.
// Dipakai saat klien mengirim balik attachment_url (misalnya POST /me/leave-requests) agar
// karyawan tidak bisa mencantumkan berkas milik tenant lain atau jenis lain.
func KeyBelongsToTenant(key, tenantID string, kind Kind) bool {
	prefix := tenantID + "/" + string(kind) + "/"
	if tenantID == "" || !strings.HasPrefix(key, prefix) {
		return false
	}
	name := strings.TrimPrefix(key, prefix)
	// Nama objek harus satu segmen (tanpa "/" atau "..") hasil ObjectKey.
	return name != "" && !strings.Contains(name, "/") && !strings.Contains(name, "..")
}
