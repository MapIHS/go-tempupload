package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/MapIHS/tempuploud/helpers"
	"github.com/MapIHS/tempuploud/routes"
	"github.com/MapIHS/tempuploud/types"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/subosito/gotenv"
)

func init() {
	gotenv.Load()
}
func main() {
	ctx := context.Background()

	region := os.Getenv("AWS_REGION")
	bucket := os.Getenv("S3_BUCKET")

	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))

	if err != nil {
		panic(err)
	}

	s3Client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(os.Getenv("S3_ENDPOINT"))
	})
	uploader := manager.NewUploader(s3Client, func(u *manager.Uploader) {

	})

	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))

	r.NotFound(helpers.WriteNotFound)
	r.MethodNotAllowed(helpers.WriteMethodNotAllowed)

	up := routes.NewUploadRoute(&routes.Upload{
		Bucket:    *aws.String(bucket),
		Uploader:  uploader,
		S3Client:  s3Client,
		MaxUpload: 32 << 20,
		KeyPrefix: "uploads/",
	})

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		helpers.WriteDATA(w, http.StatusOK, types.Message{Message: "OK"})
	})

	r.Post("/upload", up.HandleUpload)
	r.Get("/file/*", up.HandleGetFile)

	addr := fmt.Sprintf(":%s", os.Getenv("PORT"))
	log.Printf("listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, r))
}
