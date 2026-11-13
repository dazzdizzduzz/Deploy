package handler

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/tiers-undiknas/capstone-group07-2026/backend/api"
	"github.com/tiers-undiknas/capstone-group07-2026/backend/internal/auth"
	"github.com/tiers-undiknas/capstone-group07-2026/backend/internal/repository"
	"github.com/tiers-undiknas/capstone-group07-2026/backend/internal/storage"
)

var pdfBytes = append([]byte("%PDF-1.7\n%\xe2\xe3\xcf\xd3\n"), bytes.Repeat([]byte("x"), 300)...)

func newUploadRouter(t *testing.T, store storage.ObjectStore) (*gin.Engine, *auth.JWTManager) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	jwtManager, err := auth.NewJWTManager("", "")
	if err != nil {
		t.Fatal(err)
	}
	srv := NewServer(&mockUserRepo{users: map[string]*repository.User{}}, &mockLocationRepo{}, &mockScheduleRepo{},
		jwtManager, auth.NewLoginLockManager(), auth.NewPasswordResetManager())
	srv.EnableFileUploads(store)
	r := gin.New()
	api.RegisterHandlers(r, srv)
	return r, jwtManager
}

func uploadRequest(t *testing.T, r *gin.Engine, token, field, filename string, data []byte) *httptest.ResponseRecorder {
	t.Helper()
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	if data != nil {
		part, _ := w.CreateFormFile(field, filename)
		_, _ = part.Write(data)
	}
	_ = w.Close()
	req := httptest.NewRequest(http.MethodPost, "/me/leave-requests/attachments", &body)
	req.Header.Set("Content-Type", w.FormDataContentType())
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec
}

func TestUploadLeaveAttachment_PDF(t *testing.T) {
	store := newFakeStore()
	r, jwtManager := newUploadRouter(t, store)
	tok, _, _ := jwtManager.GenerateToken(attUser, attTenant, "employee")

	rec := uploadRequest(t, r, tok, "file", `C:\Users\ayu\Dokumen\surat-dokter.pdf`, pdfBytes)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	var res api.LeaveAttachment
	_ = json.Unmarshal(rec.Body.Bytes(), &res)
	if !storage.KeyBelongsToTenant(res.AttachmentUrl, attTenant, storage.KindLeaveAttachment) || !strings.HasSuffix(res.AttachmentUrl, ".pdf") {
		t.Fatalf("attachment_url = %q, want {tenant}/leave/{uuid}.pdf", res.AttachmentUrl)
	}
	if res.FileName != "surat-dokter.pdf" || res.ContentType != "application/pdf" || res.SizeBytes != len(pdfBytes) {
		t.Fatalf("respons = %+v", res)
	}
	if got := store.objects[res.AttachmentUrl]; !bytes.Equal(got, pdfBytes) {
		t.Fatal("isi berkas di storage tidak sama")
	}
}

func TestUploadLeaveAttachment_Errors(t *testing.T) {
	store := newFakeStore()
	r, jwtManager := newUploadRouter(t, store)
	tok, _, _ := jwtManager.GenerateToken(attUser, attTenant, "employee")

	tooBig := append(append([]byte{}, pdfBytes...), make([]byte, storage.MaxAttachmentBytes)...)
	expectProblem(t, uploadRequest(t, r, tok, "file", "besar.pdf", tooBig), http.StatusRequestEntityTooLarge, "ATTACHMENT_TOO_LARGE")
	expectProblem(t, uploadRequest(t, r, tok, "file", "virus.pdf", []byte("MZ\x90\x00 bukan pdf")), http.StatusUnsupportedMediaType, "ATTACHMENT_UNSUPPORTED")
	expectProblem(t, uploadRequest(t, r, tok, "berkas", "salah-field.pdf", pdfBytes), http.StatusBadRequest, "INVALID_FORM")
	expectProblem(t, uploadRequest(t, r, tok, "file", "", nil), http.StatusBadRequest, "INVALID_FORM")
	expectProblem(t, uploadRequest(t, r, "", "file", "a.pdf", pdfBytes), http.StatusUnauthorized, "UNAUTHORIZED")
	superAdmin, _, _ := jwtManager.GenerateToken(attUser, "", "super_admin")
	expectProblem(t, uploadRequest(t, r, superAdmin, "file", "a.pdf", pdfBytes), http.StatusForbidden, "FORBIDDEN")

	if len(store.objects) != 0 {
		t.Fatalf("berkas tidak valid tidak boleh tersimpan, ada %d", len(store.objects))
	}

	store.failPut = true
	expectProblem(t, uploadRequest(t, r, tok, "file", "a.pdf", pdfBytes), http.StatusServiceUnavailable, "STORAGE_UNAVAILABLE")
}

func TestUploadLeaveAttachment_StorageNotConfigured(t *testing.T) {
	r, jwtManager := newUploadRouter(t, nil)
	tok, _, _ := jwtManager.GenerateToken(attUser, attTenant, "hrd")
	expectProblem(t, uploadRequest(t, r, tok, "file", "a.pdf", pdfBytes), http.StatusServiceUnavailable, "STORAGE_UNAVAILABLE")
}

func TestDisplayFileName(t *testing.T) {
	cases := map[string]string{
		"surat-dokter.pdf":                "surat-dokter.pdf",
		"/etc/passwd":                     "passwd",
		`..\..\windows\system32\evil.pdf`: "evil.pdf",
		"nama\x00dengan\x07kontrol.png":   "namadengankontrol.png",
		`kutip"ganda.pdf`:                 "kutipganda.pdf",
		"":                                "lampiran.pdf",
		"   ":                             "lampiran.pdf",
		strings.Repeat("a", 150) + ".pdf": strings.Repeat("a", 100),
	}
	for in, want := range cases {
		if got := displayFileName(in, ".pdf"); got != want {
			t.Errorf("displayFileName(%q) = %q, want %q", in, got, want)
		}
	}
}
