## Go-TempUpload (S3 Compatible)
A simple and robust Go application to upload temporary files to S3-compatible storage (DigitalOcean Spaces, Cloudflare R2, or AWS S3).

### How to clone

```bash
git clone https://github.com/MapIHS/go-tempupload/
cd go-tempupload
```

### Config

change .env.sample to .env

```bash
cp .env.sample .env
```

```bash
# Region (e.g., sgp1, nyc3, or auto for R2)
AWS_REGION=sgp1

# Your API Credentials
AWS_ACCESS_KEY_ID=your_access_key
AWS_SECRET_ACCESS_KEY=your_secret_key

# Bucket & Endpoint Configuration
S3_BUCKET=kotonehara
S3_ENDPOINT=https://sgp1.digitaloceanspaces.com

# Application Port
PORT=8000
```


### How To Run

```bash
go mod download
go run main.go
```
