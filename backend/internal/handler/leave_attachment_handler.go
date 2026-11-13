package handler

import (
	"errors"
	"log"
	"net/http"
	"path/filepath"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/gin-gonic/gin"

	"github.com/tiers-undiknas/capstone-group07-2026/backend/api"
	"github.com/tiers-undiknas/capstone-group07-2026/backend/internal/storage"
)

// Upload lampiran pengajuan izin (ISSUE-69, FR-10). Hanya bagian integrasi object storage;
// pembuatan pengajuan izin (POST /me/leave-requests) ada di modul izin.

// maxLeaveAttachmentBody: lampiran 5 MB + ruang untuk header multipart.
const maxLeaveAttachmentBody = storage.MaxAttachmentBytes + 512<<10

// EnableFileUploads memasang object storage untuk upload lampiran. Boleh nil: endpoint akan
// menjawab 503 sampai storage dikonfigurasi.
func (s *Server) EnableFileUploads(store storage.ObjectStore) {
	s.files = store
}

// POST /me/leave-requests/attachments
func (s *Server) UploadLeaveAttachment(c *gin.Context) {
	who, ok := s.authenticate(c, attendanceRoles)
	if !ok {
		return
	}

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxLeaveAttachmentBody)
	if err := c.Request.ParseMultipartForm(maxLeaveAttachmentBody); err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			writeAttachmentTooLarge(c)
			return
		}
		writeProblem(c, http.StatusBadRequest, "INVALID_FORM", "Permintaan tidak valid",
			"Kirim berkas sebagai multipart/form-data dengan field file.", nil)
		return
	}
	files := c.Request.MultipartForm.File["file"]
	if len(files) == 0 {
		writeProblem(c, http.StatusBadRequest, "INVALID_FORM", "Permintaan tidak valid",
			"Field file wajib diisi.", nil)
		return
	}
	fh := files[0]

	f, err := fh.Open()
	if err != nil {
		writeProblem(c, http.StatusBadRequest, "INVALID_FORM", "Permintaan tidak valid", "Berkas tidak dapat dibaca.", nil)
		return
	}
	defer func() {
		_ = f.Close()
	}()

	file, err := storage.ReadAttachment(f)
	switch {
	case errors.Is(err, storage.ErrTooLarge):
		writeAttachmentTooLarge(c)
		return
	case errors.Is(err, storage.ErrUnsupportedType):
		writeProblem(c, http.StatusUnsupportedMediaType, "ATTACHMENT_UNSUPPORTED", "Format berkas tidak didukung",
			"Lampiran harus JPG, PNG, atau PDF.", nil)
		return
	case err != nil:
		writeProblem(c, http.StatusBadRequest, "INVALID_FORM", "Permintaan tidak valid", "Berkas tidak dapat dibaca.", nil)
		return
	}

	if s.files == nil {
		writeProblem(c, http.StatusServiceUnavailable, "STORAGE_UNAVAILABLE", "Penyimpanan berkas tidak tersedia",
			"Object storage belum dikonfigurasi. Coba lagi nanti.", nil)
		return
	}
	key, err := storage.ObjectKey(who.TenantID, storage.KindLeaveAttachment, file.Ext)
	if err != nil {
		writeInternal(c, "bentuk path lampiran", err)
		return
	}
	if err := storage.PutFile(c.Request.Context(), s.files, key, file); err != nil {
		log.Printf("leave attachment: unggah gagal: %v", err)
		writeProblem(c, http.StatusServiceUnavailable, "STORAGE_UNAVAILABLE", "Penyimpanan berkas tidak tersedia",
			"Lampiran gagal disimpan. Coba lagi beberapa saat lagi.", nil)
		return
	}

	c.JSON(http.StatusCreated, api.LeaveAttachment{
		AttachmentUrl: key,
		FileName:      displayFileName(fh.Filename, file.Ext),
		ContentType:   file.ContentType,
		SizeBytes:     len(file.Data),
	})
}

func writeAttachmentTooLarge(c *gin.Context) {
	writeProblem(c, http.StatusRequestEntityTooLarge, "ATTACHMENT_TOO_LARGE", "Berkas terlalu besar",
		"Ukuran lampiran maksimal 5 MB.", nil)
}

// displayFileName membersihkan nama berkas dari klien untuk ditampilkan saja: hanya nama
// dasar (tanpa folder), tanpa karakter kontrol, maksimal 100 karakter. Nama ini tidak pernah
// dipakai sebagai path di storage.
func displayFileName(name, ext string) string {
	name = filepath.Base(strings.ReplaceAll(name, `\`, "/"))
	name = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) || r == '"' {
			return -1
		}
		return r
	}, strings.TrimSpace(name))
	if name == "" || name == "." || name == "/" {
		return "lampiran" + ext
	}
	if utf8.RuneCountInString(name) > 100 {
		runes := []rune(name)
		name = string(runes[:100])
	}
	return name
}
