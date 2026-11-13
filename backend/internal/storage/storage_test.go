package storage

import (
	"bytes"
	"context"
	"errors"
	"net/url"
	"regexp"
	"strings"
	"testing"
	"time"
)

const testTenant = "a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11"

// Magic bytes minimal yang dikenali http.DetectContentType.
var (
	jpegHeader = []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 'J', 'F', 'I', 'F', 0x00}
	pngHeader  = []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A}
)

func TestObjectKey(t *testing.T) {
	key, err := ObjectKey(testTenant, KindSelfie, ".jpg")
	if err != nil {
		t.Fatal(err)
	}
	pattern := regexp.MustCompile(`^` + testTenant + `/selfie/[0-9a-f-]{36}\.jpg$`)
	if !pattern.MatchString(key) {
		t.Fatalf("key = %q, tidak sesuai format {tenant_id}/selfie/{uuid}.jpg", key)
	}

	other, _ := ObjectKey(testTenant, KindSelfie, ".jpg")
	if other == key {
		t.Fatal("dua key berturut-turut seharusnya berbeda")
	}
}

func TestObjectKey_Invalid(t *testing.T) {
	cases := []struct {
		tenant string
		kind   Kind
	}{
		{"", KindSelfie},
		{"../other-tenant", KindSelfie},
		{testTenant, Kind("../../etc")},
		{testTenant, Kind("")},
	}
	for _, c := range cases {
		if _, err := ObjectKey(c.tenant, c.kind, ".jpg"); !errors.Is(err, ErrInvalidKey) {
			t.Errorf("ObjectKey(%q, %q) err = %v, want ErrInvalidKey", c.tenant, c.kind, err)
		}
	}
}

func TestReadPhoto(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		wantCT  string
		wantExt string
		wantErr error
	}{
		{"jpeg", append(jpegHeader, bytes.Repeat([]byte{1}, 100)...), "image/jpeg", ".jpg", nil},
		{"png", append(pngHeader, bytes.Repeat([]byte{1}, 100)...), "image/png", ".png", nil},
		{"tepat 2 MB", append(jpegHeader, make([]byte, int(MaxPhotoBytes)-len(jpegHeader))...), "image/jpeg", ".jpg", nil},
		{"lebih dari 2 MB", append(jpegHeader, make([]byte, int(MaxPhotoBytes))...), "", "", ErrTooLarge},
		{"gif ditolak", []byte("GIF89a......"), "", "", ErrUnsupportedType},
		{"teks dengan nama .jpg ditolak", []byte("<html>bukan foto</html>"), "", "", ErrUnsupportedType},
		{"kosong", nil, "", "", ErrUnsupportedType},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := ReadPhoto(bytes.NewReader(tt.data))
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("err = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if p.ContentType != tt.wantCT || p.Ext != tt.wantExt || len(p.Data) != len(tt.data) {
				t.Fatalf("got %s %s %d byte, want %s %s %d byte",
					p.ContentType, p.Ext, len(p.Data), tt.wantCT, tt.wantExt, len(tt.data))
			}
		})
	}
}

func TestConfigFromEnv(t *testing.T) {
	t.Setenv("S3_ENDPOINT", "minio:9000")
	t.Setenv("S3_ACCESS_KEY", "ak")
	t.Setenv("S3_SECRET_KEY", "sk")
	t.Setenv("S3_BUCKET_NAME", "ontrack-bucket")
	t.Setenv("S3_USE_SSL", "false")
	t.Setenv("S3_SSE", "true")
	t.Setenv("S3_REGION", "")
	t.Setenv("S3_TIMEOUT_SECONDS", "3")

	cfg := ConfigFromEnv()
	if !cfg.Enabled() || cfg.UseSSL || !cfg.SSE || cfg.Region != "us-east-1" || cfg.Timeout != 3*time.Second {
		t.Fatalf("config tidak sesuai: %+v", cfg)
	}

	t.Setenv("S3_SECRET_KEY", "")
	if ConfigFromEnv().Enabled() {
		t.Fatal("config tanpa secret key seharusnya tidak aktif")
	}
}

func TestNewMinIOStore_NotConfigured(t *testing.T) {
	if _, err := NewMinIOStore(Config{}); !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("err = %v, want ErrNotConfigured", err)
	}
}

// Signed URL dihitung lokal (Region sudah diisi), jadi test ini tidak butuh server MinIO.
func TestPresignGet_Offline(t *testing.T) {
	s, err := NewMinIOStore(Config{
		Endpoint: "localhost:9000", AccessKey: "ak", SecretKey: "sk",
		Bucket: "ontrack-bucket", Region: "us-east-1", Timeout: time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}

	raw, err := s.PresignGet(context.Background(), testTenant+"/selfie/x.jpg", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(u.Path, "/ontrack-bucket/"+testTenant+"/selfie/x.jpg") {
		t.Fatalf("path = %q", u.Path)
	}
	q := u.Query()
	if q.Get("X-Amz-Signature") == "" {
		t.Fatal("signed URL tidak membawa tanda tangan")
	}
	// Permintaan 1 jam harus dipotong menjadi batas maksimum 15 menit (900 detik).
	if q.Get("X-Amz-Expires") != "900" {
		t.Fatalf("X-Amz-Expires = %q, want 900", q.Get("X-Amz-Expires"))
	}
}
