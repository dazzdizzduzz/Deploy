package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/minio/minio-go/v7/pkg/encrypt"
)

// MaxPresignExpiry membatasi umur signed URL agar tautan foto yang bocor cepat kedaluwarsa.
const MaxPresignExpiry = 15 * time.Minute

// MinIOStore mengimplementasikan ObjectStore dengan klien minio-go (kompatibel S3).
type MinIOStore struct {
	client  *minio.Client
	bucket  string
	region  string
	sse     encrypt.ServerSide // nil bila S3_SSE=false
	timeout time.Duration
}

// NewMinIOStore membuat klien. Fungsi ini belum menghubungi server; gunakan EnsureBucket
// saat startup untuk memastikan koneksi dan bucket.
func NewMinIOStore(cfg Config) (*MinIOStore, error) {
	if !cfg.Enabled() {
		return nil, ErrNotConfigured
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 10 * time.Second
	}

	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
		// Region diisi agar minio-go tidak perlu menanyakan lokasi bucket ke server
		// (menghemat satu round-trip dan membuat presign bisa dihitung lokal).
		Region: cfg.Region,
		Transport: &http.Transport{
			Proxy:                 http.ProxyFromEnvironment,
			ResponseHeaderTimeout: cfg.Timeout,
			IdleConnTimeout:       90 * time.Second,
			MaxIdleConnsPerHost:   16,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("gagal membuat klien MinIO: %w", err)
	}

	s := &MinIOStore{client: client, bucket: cfg.Bucket, region: cfg.Region, timeout: cfg.Timeout}
	if cfg.SSE {
		s.sse = encrypt.NewSSE() // SSE-S3: kunci dikelola storage (AES-256)
	}
	return s, nil
}

// EnsureBucket membuat bucket privat bila belum ada.
func (s *MinIOStore) EnsureBucket(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	exists, err := s.client.BucketExists(ctx, s.bucket)
	if err != nil {
		return fmt.Errorf("%w: cek bucket %q: %v", ErrUnavailable, s.bucket, err)
	}
	if exists {
		return nil
	}
	if err := s.client.MakeBucket(ctx, s.bucket, minio.MakeBucketOptions{Region: s.region}); err != nil {
		return fmt.Errorf("%w: buat bucket %q: %v", ErrUnavailable, s.bucket, err)
	}
	return nil
}

// Put mengunggah objek. size wajib diketahui agar minio-go tidak memakai multipart upload.
func (s *MinIOStore) Put(ctx context.Context, key string, body io.Reader, size int64, contentType string) error {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	_, err := s.client.PutObject(ctx, s.bucket, key, body, size, minio.PutObjectOptions{
		ContentType:          contentType,
		ServerSideEncryption: s.sse,
	})
	if err != nil {
		return fmt.Errorf("%w: unggah %q: %v", ErrUnavailable, key, err)
	}
	return nil
}

// Delete menghapus objek. Objek yang sudah tidak ada tidak dianggap error.
func (s *MinIOStore) Delete(ctx context.Context, key string) error {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	if err := s.client.RemoveObject(ctx, s.bucket, key, minio.RemoveObjectOptions{}); err != nil {
		var resp minio.ErrorResponse
		if errors.As(err, &resp) && resp.Code == "NoSuchKey" {
			return nil
		}
		return fmt.Errorf("%w: hapus %q: %v", ErrUnavailable, key, err)
	}
	return nil
}

// PresignGet membuat signed URL untuk membaca objek. Umur dibatasi MaxPresignExpiry.
func (s *MinIOStore) PresignGet(ctx context.Context, key string, expiry time.Duration) (string, error) {
	if expiry <= 0 || expiry > MaxPresignExpiry {
		expiry = MaxPresignExpiry
	}
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	u, err := s.client.PresignedGetObject(ctx, s.bucket, key, expiry, url.Values{})
	if err != nil {
		return "", fmt.Errorf("%w: presign %q: %v", ErrUnavailable, key, err)
	}
	return u.String(), nil
}
