package routes

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/MapIHS/tempuploud/helpers"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
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

func (u *Upload) buildObjectKey(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	if ext == "" {
		ext = ".bin"
	}

	prefix := u.KeyPrefix
	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}

	return fmt.Sprintf("%s%s%s", prefix, helpers.RandomHex(16), ext)
}

func (u *Upload) HandleUpload(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, u.MaxUpload)

	ctx := r.Context()

	reader, err := r.MultipartReader()
	if err != nil {
		helpers.WriteError(w, http.StatusBadRequest, "Invalid multipart form")
		return
	}

	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}

		if err != nil {
			helpers.WriteError(w, http.StatusInternalServerError, "Failed Read")
			return
		}

		if part.FormName() != "file" {
			part.Close()
			continue
		}

		filename := strings.TrimSpace(part.FileName())
		if filename == "" {
			helpers.WriteError(w, http.StatusBadRequest, "Field 'file' not exits")
			return
		}

		buf := make([]byte, 512)
		n, _ := io.ReadFull(part, buf)
		sniffed := http.DetectContentType(buf[:n])
		contentType := sniffed
		if contentType == "application/octet-stream" {
			if ct := strings.TrimSpace(part.Header.Get("Content-Type")); ct != "" {
				contentType = ct
			}
		}

		key := u.buildObjectKey(filename)
		base := filepath.Base(filename)
		body := io.MultiReader(bytes.NewReader(buf[:n]), part)

		input := &s3.PutObjectInput{
			Bucket:      aws.String(u.Bucket),
			Key:         aws.String(key),
			Body:        body,
			ContentType: aws.String(contentType),
			Metadata: map[string]string{
				"original-filename": strings.TrimSpace(base),
				"uploaded-at":       time.Now().UTC().Format(time.RFC3339),
			},
		}

		_, err = u.Uploader.Upload(ctx, input)
		if err != nil {
			helpers.WriteError(w, http.StatusInternalServerError, "Upload to s3 failed")
			return
		}

		resp := map[string]any{
			"key":          key,
			"content_type": contentType,
		}

		helpers.WriteDATA(w, http.StatusOK, resp)
		return
	}

	helpers.WriteError(w, http.StatusBadRequest, "Missing field 'file'")
}

func (u *Upload) HandleGetFile(w http.ResponseWriter, r *http.Request) {
	key := chi.URLParam(r, "*")
	if key == "" {
		helpers.WriteError(w, http.StatusBadRequest, "Missing key")
		return
	}

	if u.KeyPrefix != "" {
		p := u.KeyPrefix
		if !strings.HasSuffix(p, "/") {
			p += "/"
		}
		if !strings.HasPrefix(key, p) {
			helpers.WriteError(w, http.StatusForbidden, "Not Access")
			return
		}
	}

	ctx := r.Context()
	out, err := u.S3Client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(u.Bucket),
		Key:    aws.String(key),
	})

	if err != nil {
		var nsk *types.NoSuchKey
		if errors.As(err, &nsk) {
			helpers.WriteError(w, http.StatusNotFound, "File Not Found")
			return
		}
		helpers.WriteError(w, http.StatusInternalServerError, "Failed to fetch file")
		return
	}
	defer out.Body.Close()

	if out.ContentType != nil && *out.ContentType != "" {
		w.Header().Set("Content-Type", *out.ContentType)
	} else {
		w.Header().Set("Content-Type", "application/octet-stream")
	}

	if out.ETag != nil {
		w.Header().Set("ETag", *out.ETag)
	}
	if out.LastModified != nil {
		w.Header().Set("Last-Modified", out.LastModified.UTC().Format(http.TimeFormat))
	}

	if _, err := io.Copy(w, out.Body); err != nil {
		log.Printf("stream error: %v", err)
		return
	}
}
