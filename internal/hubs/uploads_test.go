package hubs

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUploaderValidation_Extensions(t *testing.T) {
	tmpDir := t.TempDir()
	uploader, err := NewUploader(tmpDir, 1024*1024, "/uploads")
	if err != nil {
		t.Fatalf("failed to create uploader: %v", err)
	}

	safeExts := []string{".png", ".jpg", ".jpeg", ".gif", ".webp", ".pdf", ".txt", ".mp4", ".zip"}
	for _, ext := range safeExts {
		if !uploader.IsAllowedExtension(ext) {
			t.Errorf("expected extension %s to be allowed", ext)
		}
	}

	blockedExts := []string{".exe", ".bat", ".cmd", ".sh", ".ps1", ".vbs", ".dll", ".so", ".php", ".js"}
	for _, ext := range blockedExts {
		if uploader.IsAllowedExtension(ext) {
			t.Errorf("expected extension %s to be blocked", ext)
		}
	}
}

func TestUploadHandler_Success(t *testing.T) {
	tmpDir := t.TempDir()
	uploader, err := NewUploader(tmpDir, 5*1024*1024, "/uploads")
	if err != nil {
		t.Fatalf("failed to create uploader: %v", err)
	}

	// Create valid PNG byte header (8-byte signature: \x89PNG\r\n\x1a\n) + dummy data
	pngData := append([]byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}, bytes.Repeat([]byte("A"), 100)...)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", "screenshot.png")
	if err != nil {
		t.Fatalf("failed to create form file: %v", err)
	}
	_, _ = part.Write(pngData)
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/hubs/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	rr := httptest.NewRecorder()

	handler := UploadHandler(uploader, nil)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200 OK, got %d. Body: %s", rr.Code, rr.Body.String())
	}

	var res UploadResponse
	if err := json.NewDecoder(rr.Body).Decode(&res); err != nil {
		t.Fatalf("failed to parse json response: %v", err)
	}

	if res.Status != "success" {
		t.Errorf("expected status 'success', got %s", res.Status)
	}
	if !strings.HasPrefix(res.FileURL, "/uploads/att_") {
		t.Errorf("expected file_url to start with /uploads/att_, got %s", res.FileURL)
	}
	if res.FileName != "screenshot.png" {
		t.Errorf("expected file_name 'screenshot.png', got %s", res.FileName)
	}
}

func TestUploadHandler_BlockedExtension(t *testing.T) {
	tmpDir := t.TempDir()
	uploader, err := NewUploader(tmpDir, 5*1024*1024, "/uploads")
	if err != nil {
		t.Fatalf("failed to create uploader: %v", err)
	}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", "malicious_script.exe")
	if err != nil {
		t.Fatalf("failed to create form file: %v", err)
	}
	_, _ = part.Write([]byte("malicious content"))
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/hubs/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	rr := httptest.NewRecorder()

	handler := UploadHandler(uploader, nil)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400 Bad Request, got %d", rr.Code)
	}
}

func TestServeFilesHandler_SecurityAndHeaders(t *testing.T) {
	tmpDir := t.TempDir()
	uploader, err := NewUploader(tmpDir, 5*1024*1024, "/uploads")
	if err != nil {
		t.Fatalf("failed to create uploader: %v", err)
	}

	// Create test file
	testFileName := "test_image.png"
	testFilePath := filepath.Join(tmpDir, testFileName)
	if err := os.WriteFile(testFilePath, []byte("fake png content"), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	handler := ServeFilesHandler(uploader)

	// 1. Valid file request
	req := httptest.NewRequest(http.MethodGet, "/"+testFileName, nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", rr.Code)
	}
	if nosniff := rr.Header().Get("X-Content-Type-Options"); nosniff != "nosniff" {
		t.Errorf("expected X-Content-Type-Options: nosniff, got %s", nosniff)
	}
	body, _ := io.ReadAll(rr.Body)
	if string(body) != "fake png content" {
		t.Errorf("expected body 'fake png content', got %s", string(body))
	}

	// 2. Directory traversal attempt
	traversalReq := httptest.NewRequest(http.MethodGet, "/../../something_sensitive.txt", nil)
	traversalRr := httptest.NewRecorder()
	handler.ServeHTTP(traversalRr, traversalReq)

	if traversalRr.Code != http.StatusForbidden && traversalRr.Code != http.StatusNotFound {
		t.Fatalf("expected 403 or 404 for traversal attempt, got %d", traversalRr.Code)
	}

	// 3. Nonexistent file
	missingReq := httptest.NewRequest(http.MethodGet, "/nonexistent.png", nil)
	missingRr := httptest.NewRecorder()
	handler.ServeHTTP(missingRr, missingReq)

	if missingRr.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for missing file, got %d", missingRr.Code)
	}
}
