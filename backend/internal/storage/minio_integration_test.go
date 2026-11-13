package storage

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"os"
	"testing"
	"time"
)

// Test integrasi terhadap MinIO sungguhan. Dilewati bila TEST_S3_ENDPOINT tidak diisi,
// sehingga CI tanpa MinIO tetap lulus. Contoh lokal:
//
//	TEST_S3_ENDPOINT=localhost:9000 TEST_S3_ACCESS_KEY=... TEST_S3_SECRET_KEY=... go test ./internal/storage
func TestMinIOStore_Integration(t *testing.T) {
	endpoint := os.Getenv("TEST_S3_ENDPOINT")
	if endpoint == "" {
		t.Skip("TEST_S3_ENDPOINT tidak diisi; lewati test integrasi MinIO")
	}

	s, err := NewMinIOStore(Config{
		Endpoint:  endpoint,
		AccessKey: os.Getenv("TEST_S3_ACCESS_KEY"),
		SecretKey: os.Getenv("TEST_S3_SECRET_KEY"),
		Bucket:    "ontrack-test-" + time.Now().Format("20060102150405"),
		Region:    "us-east-1",
		Timeout:   5 * time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()

	if err := s.EnsureBucket(ctx); err != nil {
		t.Fatalf("EnsureBucket: %v", err)
	}
	// Panggilan kedua harus aman (bucket sudah ada).
	if err := s.EnsureBucket(ctx); err != nil {
		t.Fatalf("EnsureBucket kedua: %v", err)
	}
	t.Cleanup(func() { _ = s.client.RemoveBucket(context.Background(), s.bucket) })

	photo, err := ReadPhoto(bytes.NewReader(append(jpegHeader, bytes.Repeat([]byte{7}, 512)...)))
	if err != nil {
		t.Fatal(err)
	}
	key, err := ObjectKey(testTenant, KindSelfie, photo.Ext)
	if err != nil {
		t.Fatal(err)
	}

	if err := PutPhoto(ctx, s, key, photo); err != nil {
		t.Fatalf("Put: %v", err)
	}

	// Signed URL harus bisa dibuka tanpa kredensial dan isinya sama.
	signed, err := s.PresignGet(ctx, key, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.Get(signed) //nolint:gosec // URL berasal dari server uji
	if err != nil {
		t.Fatal(err)
	}
	got, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK || !bytes.Equal(got, photo.Data) {
		t.Fatalf("GET signed URL: status %d, %d byte", resp.StatusCode, len(got))
	}
	if ct := resp.Header.Get("Content-Type"); ct != "image/jpeg" {
		t.Fatalf("Content-Type = %q, want image/jpeg", ct)
	}

	// Bucket privat: URL tanpa tanda tangan harus ditolak.
	unsigned := "http://" + endpoint + "/" + s.bucket + "/" + key
	resp, err = http.Get(unsigned) //nolint:gosec // URL berasal dari server uji
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("akses tanpa tanda tangan: status %d, want 403", resp.StatusCode)
	}

	if err := s.Delete(ctx, key); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	// Menghapus objek yang sudah tidak ada tidak boleh error.
	if err := s.Delete(ctx, key); err != nil {
		t.Fatalf("Delete kedua: %v", err)
	}
}
