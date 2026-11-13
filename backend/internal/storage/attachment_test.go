package storage

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"
	"time"
)

// Header minimal PDF yang dikenali http.DetectContentType.
var pdfHeader = []byte("%PDF-1.7\n%\xe2\xe3\xcf\xd3\n")

func TestReadAttachment(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		wantCT  string
		wantExt string
		wantErr error
	}{
		{"pdf surat dokter", append(append([]byte{}, pdfHeader...), bytes.Repeat([]byte("x"), 200)...), "application/pdf", ".pdf", nil},
		{"jpeg", append(append([]byte{}, jpegHeader...), bytes.Repeat([]byte{1}, 100)...), "image/jpeg", ".jpg", nil},
		{"png", append(append([]byte{}, pngHeader...), bytes.Repeat([]byte{1}, 100)...), "image/png", ".png", nil},
		{"tepat 5 MB", append(append([]byte{}, pdfHeader...), make([]byte, int(MaxAttachmentBytes)-len(pdfHeader))...), "application/pdf", ".pdf", nil},
		{"lebih dari 5 MB", append(append([]byte{}, pdfHeader...), make([]byte, int(MaxAttachmentBytes))...), "", "", ErrTooLarge},
		{"gif ditolak", []byte("GIF89a......"), "", "", ErrUnsupportedType},
		{"html dengan nama .pdf ditolak", []byte("<html>bukan pdf</html>"), "", "", ErrUnsupportedType},
		{"zip ditolak", []byte("PK\x03\x04berkas-zip"), "", "", ErrUnsupportedType},
		{"kosong", nil, "", "", ErrUnsupportedType},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, err := ReadAttachment(bytes.NewReader(tt.data))
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("err = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if f.ContentType != tt.wantCT || f.Ext != tt.wantExt || len(f.Data) != len(tt.data) {
				t.Fatalf("got %s %s %d byte, want %s %s %d byte",
					f.ContentType, f.Ext, len(f.Data), tt.wantCT, tt.wantExt, len(tt.data))
			}
		})
	}
}

// Selfie tetap hanya JPEG/PNG dan maksimal 2 MB, walaupun lampiran boleh PDF 5 MB.
func TestReadPhoto_StillRejectsPDFAndLargeFiles(t *testing.T) {
	if _, err := ReadPhoto(bytes.NewReader(pdfHeader)); !errors.Is(err, ErrUnsupportedType) {
		t.Fatalf("PDF sebagai selfie: err = %v, want ErrUnsupportedType", err)
	}
	threeMB := append(append([]byte{}, jpegHeader...), make([]byte, 3<<20)...)
	if _, err := ReadPhoto(bytes.NewReader(threeMB)); !errors.Is(err, ErrTooLarge) {
		t.Fatalf("selfie 3 MB: err = %v, want ErrTooLarge", err)
	}
	if _, err := ReadAttachment(bytes.NewReader(threeMB)); err != nil {
		t.Fatalf("lampiran 3 MB seharusnya diterima: %v", err)
	}
}

type recordingStore struct {
	key, contentType string
	size             int64
	data             []byte
}

func (r *recordingStore) Put(_ context.Context, key string, body io.Reader, size int64, contentType string) error {
	r.key, r.size, r.contentType = key, size, contentType
	r.data, _ = io.ReadAll(body)
	return nil
}
func (r *recordingStore) Delete(context.Context, string) error { return nil }
func (r *recordingStore) PresignGet(context.Context, string, time.Duration) (string, error) {
	return "", nil
}

func TestPutFile_SendsDetectedContentType(t *testing.T) {
	f, err := ReadAttachment(bytes.NewReader(append(append([]byte{}, pdfHeader...), []byte("isi")...)))
	if err != nil {
		t.Fatal(err)
	}
	key, err := ObjectKey(testTenant, KindLeaveAttachment, f.Ext)
	if err != nil {
		t.Fatal(err)
	}
	store := &recordingStore{}
	if err := PutFile(context.Background(), store, key, f); err != nil {
		t.Fatal(err)
	}
	if store.key != key || store.contentType != "application/pdf" || store.size != int64(len(f.Data)) || !bytes.Equal(store.data, f.Data) {
		t.Fatalf("Put menerima %+v", store)
	}
}

func TestKeyBelongsToTenant(t *testing.T) {
	key, _ := ObjectKey(testTenant, KindLeaveAttachment, ".pdf")
	other := "11111111-1111-1111-1111-111111111111"
	cases := []struct {
		name   string
		key    string
		tenant string
		kind   Kind
		want   bool
	}{
		{"milik tenant dan jenis yang sama", key, testTenant, KindLeaveAttachment, true},
		{"tenant lain", key, other, KindLeaveAttachment, false},
		{"jenis lain", key, testTenant, KindSelfie, false},
		{"path traversal", testTenant + "/leave/../selfie/x.jpg", testTenant, KindLeaveAttachment, false},
		{"subfolder", testTenant + "/leave/a/b.pdf", testTenant, KindLeaveAttachment, false},
		{"nama kosong", testTenant + "/leave/", testTenant, KindLeaveAttachment, false},
		{"tenant kosong", key, "", KindLeaveAttachment, false},
	}
	for _, c := range cases {
		if got := KeyBelongsToTenant(c.key, c.tenant, c.kind); got != c.want {
			t.Errorf("%s: got %v, want %v", c.name, got, c.want)
		}
	}
}
