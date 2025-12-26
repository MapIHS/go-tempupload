package routes

import (
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/MapIHS/tempuploud/helpers"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/go-chi/chi/v5"
)

type Upload struct {
	Bucket    string
	Uploader  *manager.Uploader
	S3Client  *s3.Client
	MaxUpload int64
	KeyPrefix string
}

func NewUploadRoute(app *Upload) *Upload {
	return &Upload{
		Bucket:    app.Bucket,
		Uploader:  app.Uploader,
		S3Client:  app.S3Client,
		MaxUpload: app.MaxUpload,
		KeyPrefix: app.KeyPrefix,
	}
}

func (u *Upload) buildObjectKey(h *multipart.FileHeader) string {
	ext := strings.ToLower(filepath.Ext(h.Filename))
	if ext == "" {
		ext = ".bin"
	}
	return fmt.Sprintf("%s%s%s", u.KeyPrefix, helpers.RandomHex(16), ext)
}

func (u *Upload) HandleUpload(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, u.MaxUpload)

	if err := r.ParseMultipartForm(u.MaxUpload); err != nil {
		helpers.WriteError(w, http.StatusBadRequest, "Invalid Multipart form")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		helpers.WriteError(w, http.StatusBadRequest, "Missing field 'file'")
		return
	}

	defer file.Close()

	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	key := u.buildObjectKey(header)
	base := filepath.Base(header.Filename)

	input := &s3.PutObjectInput{
		Bucket:      aws.String(u.Bucket),
		Key:         aws.String(key),
		Body:        file,
		ContentType: aws.String(contentType),
		Metadata: map[string]string{
			"original-filename": strings.TrimSpace(base),
			"uploaded-at":       time.Now().UTC().Format(time.RFC3339),
		},
	}

	ctx := r.Context()

	_, err = u.Uploader.Upload(ctx, input)
	if err != nil {
		helpers.WriteError(w, http.StatusInternalServerError, "Upload to s3 failed")
		return
	}

	resp := map[string]any{
		"key":          key,
		"content_type": contentType,
		"size":         header.Size,
	}

	helpers.WriteDATA(w, http.StatusOK, resp)

}

func (u *Upload) HandleGetFile(w http.ResponseWriter, r *http.Request) {
	key := chi.URLParam(r, "*")
	if key == "" {
		helpers.WriteError(w, http.StatusBadRequest, "Missing key")
		return
	}

	ctx := r.Context()
	out, err := u.S3Client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(u.Bucket),
		Key:    aws.String(key),
	})

	if err != nil {
		helpers.WriteError(w, http.StatusNotFound, "File Not Found")
		return
	}

	defer out.Body.Close()

	if out.ContentType != nil {
		w.Header().Set("Content-Type", *out.ContentType)
	} else {
		w.Header().Set("Content-Type", "application/octet-stream")
	}

	if _, err := io.Copy(w, out.Body); err != nil {
		log.Printf("stream error: %v", err)
		return
	}
}
