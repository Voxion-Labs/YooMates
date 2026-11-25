package hubs

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"yoomates/internal/auth"
)

const (
	// DefaultMaxUploadSize is 25 MB (standard chat upload size)
	DefaultMaxUploadSize = 25 * 1024 * 1024
)

// UploadResponse is returned to the client upon successful file upload
type UploadResponse struct {
	Status   string `json:"status"`
	FileURL  string `json:"file_url"`
	FileName string `json:"file_name"`
	FileType string `json:"file_type"`
	FileSize int64  `json:"file_size"`
}

// UploadErrorResponse is returned when an upload fails validation
type UploadErrorResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

// Uploader manages file validation, storage, and retrieval
type Uploader struct {
	uploadDir    string
	maxSizeBytes int64
	baseURL      string
}

// NewUploader creates an uploader instance and ensures the upload directory exists
func NewUploader(uploadDir string, maxSizeBytes int64, baseURL string) (*Uploader, error) {
	if uploadDir == "" {
		uploadDir = "data/uploads"
	}
	if maxSizeBytes <= 0 {
		maxSizeBytes = DefaultMaxUploadSize
	}
	if baseURL == "" {
		baseURL = "/uploads"
	}

	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create upload directory %q: %w", uploadDir, err)
	}

	return &Uploader{
		uploadDir:    uploadDir,
		maxSizeBytes: maxSizeBytes,
		baseURL:      baseURL,
	}, nil
}

// IsAllowedExtension checks if the extension is safe for chat upload
func (u *Uploader) IsAllowedExtension(ext string) bool {
	ext = strings.ToLower(ext)
	// Explicitly block executable and script formats
	blocked := map[string]bool{
		".exe": true, ".bat": true, ".cmd": true, ".sh": true, ".ps1": true,
		".vbs": true, ".js": true, ".dll": true, ".so": true, ".bin": true,
		".msi": true, ".php": true, ".jsp": true, ".asp": true, ".jar": true,
	}
	if blocked[ext] {
		return false
	}

	// Permitted safe extensions
	allowed := map[string]bool{
		".png": true, ".jpg": true, ".jpeg": true, ".gif": true, ".webp": true, ".svg": true,
		".mp3": true, ".ogg": true, ".wav": true, ".mp4": true, ".webm": true,
		".pdf": true, ".txt": true, ".md": true, ".csv": true, ".json": true, ".zip": true,
	}
	return allowed[ext]
}

// IsAllowedMimeType inspects the sniffed MIME type
func (u *Uploader) IsAllowedMimeType(mime string) bool {
	mime = strings.ToLower(mime)
	if strings.HasPrefix(mime, "image/") ||
		strings.HasPrefix(mime, "audio/") ||
		strings.HasPrefix(mime, "video/") ||
		strings.HasPrefix(mime, "text/") ||
		mime == "application/pdf" ||
		mime == "application/zip" ||
		mime == "application/json" ||
		mime == "application/octet-stream" {
		return true
	}
	return false
}

// SaveFile validates and persists the uploaded file
func (u *Uploader) SaveFile(ctx context.Context, file multipart.File, header *multipart.FileHeader) (*UploadResponse, error) {
	if header.Size > u.maxSizeBytes {
		return nil, fmt.Errorf("file size exceeds maximum allowed limit of %d MB", u.maxSizeBytes/(1024*1024))
	}

	origName := filepath.Base(header.Filename)
	ext := filepath.Ext(origName)
	if !u.IsAllowedExtension(ext) {
		return nil, fmt.Errorf("file extension %q is not permitted for upload", ext)
	}

	// Sniff MIME type using the initial 512 bytes
	buf := make([]byte, 512)
	n, err := file.Read(buf)
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("failed to read file header: %w", err)
	}

	detectedMime := http.DetectContentType(buf[:n])
	if !u.IsAllowedMimeType(detectedMime) {
		return nil, fmt.Errorf("unsupported or dangerous file content type %q", detectedMime)
	}

	// Seek back to start
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("failed to seek file: %w", err)
	}

	// Generate safe, unique storage filename
	tokenBytes := make([]byte, 8)
	_, _ = rand.Read(tokenBytes)
	uniqueName := fmt.Sprintf("att_%d_%s%s", time.Now().Unix(), hex.EncodeToString(tokenBytes), ext)

	destPath := filepath.Join(u.uploadDir, uniqueName)
	destFile, err := os.OpenFile(destPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to create destination file: %w", err)
	}
	defer destFile.Close()

	written, err := io.Copy(destFile, file)
	if err != nil {
		_ = os.Remove(destPath)
		return nil, fmt.Errorf("failed to save file to disk: %w", err)
	}

	fileURL := u.baseURL + "/" + uniqueName

	return &UploadResponse{
		Status:   "success",
		FileURL:  fileURL,
		FileName: origName,
		FileType: detectedMime,
		FileSize: written,
	}, nil
}

// UploadHandler processes multipart form upload requests
func UploadHandler(uploader *Uploader, authMgr *auth.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Enforce maximum body size
		r.Body = http.MaxBytesReader(w, r.Body, uploader.maxSizeBytes+1024*1024)

		if err := r.ParseMultipartForm(uploader.maxSizeBytes); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusRequestEntityTooLarge)
			_ = json.NewEncoder(w).Encode(UploadErrorResponse{
				Status:  "error",
				Message: "Uploaded file exceeds maximum limit of 25MB",
			})
			return
		}

		file, header, err := r.FormFile("file")
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(UploadErrorResponse{
				Status:  "error",
				Message: "No file field found in form data",
			})
			return
		}
		defer file.Close()

		res, err := uploader.SaveFile(r.Context(), file, header)
		if err != nil {
			log.Printf("[upload-error] Validation failed: %v\n", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(UploadErrorResponse{
				Status:  "error",
				Message: err.Error(),
			})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(res)
	}
}

// ServeFilesHandler serves uploaded files with security headers
func ServeFilesHandler(uploader *Uploader) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Clean and sanitize request path to prevent directory traversal
		cleanPath := filepath.Clean(r.URL.Path)
		cleanPath = strings.TrimPrefix(cleanPath, "/")

		filePath := filepath.Join(uploader.uploadDir, cleanPath)

		// Verify that the resolved path stays within uploadDir
		rel, err := filepath.Rel(uploader.uploadDir, filePath)
		if err != nil || strings.HasPrefix(rel, "..") || strings.Contains(rel, "..") {
			http.Error(w, "Access denied", http.StatusForbidden)
			return
		}

		fi, err := os.Stat(filePath)
		if err != nil || fi.IsDir() {
			http.NotFound(w, r)
			return
		}

		// Security and caching headers
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Cache-Control", "public, max-age=86400")

		http.ServeFile(w, r, filePath)
	})
}
