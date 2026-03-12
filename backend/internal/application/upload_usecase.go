package application

import (
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"

	"github.com/earnlearning/backend/internal/domain/post"
)

// Maximum upload size: 10MB
const MaxUploadSize = 10 * 1024 * 1024

// Allowed MIME types
var allowedMIMETypes = map[string]bool{
	"image/jpeg":                                                                true,
	"image/png":                                                                 true,
	"image/gif":                                                                 true,
	"image/webp":                                                                true,
	"application/pdf":                                                           true,
	"application/msword":                                                        true,
	"application/vnd.openxmlformats-officedocument.wordprocessingml.document":    true,
	"application/vnd.ms-excel":                                                  true,
	"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet":         true,
	"application/vnd.ms-powerpoint":                                             true,
	"application/vnd.openxmlformats-officedocument.presentationml.presentation": true,
	"text/plain":       true,
	"application/zip":  true,
	"video/mp4":        true,
	"audio/mpeg":       true,
	"application/json": true,
}

type UploadUsecase struct {
	postRepo   post.PostRepository
	uploadPath string
}

func NewUploadUsecase(pr post.PostRepository, uploadPath string) *UploadUsecase {
	return &UploadUsecase{
		postRepo:   pr,
		uploadPath: uploadPath,
	}
}

type UploadResult struct {
	ID       int    `json:"id"`
	Filename string `json:"filename"`
	MimeType string `json:"mime_type"`
	Size     int64  `json:"size"`
	URL      string `json:"url"`
}

func (uc *UploadUsecase) Upload(userID int, fileHeader *multipart.FileHeader, uuidPrefix string) (*UploadResult, error) {
	// Check file size
	if fileHeader.Size > MaxUploadSize {
		return nil, fmt.Errorf("파일 크기는 최대 10MB까지 허용됩니다")
	}

	// Check MIME type
	mimeType := fileHeader.Header.Get("Content-Type")
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}
	// Normalize MIME type (remove charset etc.)
	if idx := strings.Index(mimeType, ";"); idx != -1 {
		mimeType = strings.TrimSpace(mimeType[:idx])
	}
	if !allowedMIMETypes[mimeType] {
		return nil, fmt.Errorf("허용되지 않는 파일 형식입니다: %s", mimeType)
	}

	// Generate stored filename with UUID prefix
	ext := filepath.Ext(fileHeader.Filename)
	storedName := uuidPrefix + ext

	// Ensure upload directory exists
	if err := os.MkdirAll(uc.uploadPath, 0755); err != nil {
		return nil, fmt.Errorf("업로드 디렉토리 생성 실패: %w", err)
	}

	// Save file
	storedPath := filepath.Join(uc.uploadPath, storedName)
	src, err := fileHeader.Open()
	if err != nil {
		return nil, fmt.Errorf("파일 열기 실패: %w", err)
	}
	defer src.Close()

	dst, err := os.Create(storedPath)
	if err != nil {
		return nil, fmt.Errorf("파일 저장 실패: %w", err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return nil, fmt.Errorf("파일 복사 실패: %w", err)
	}

	// Save upload record
	upload := &post.Upload{
		UserID:     userID,
		Filename:   fileHeader.Filename,
		StoredName: storedName,
		MimeType:   mimeType,
		Size:       fileHeader.Size,
		Path:       storedPath,
	}

	uploadID, err := uc.postRepo.CreateUpload(upload)
	if err != nil {
		// Clean up file on DB failure
		os.Remove(storedPath)
		return nil, fmt.Errorf("업로드 기록 저장 실패: %w", err)
	}

	return &UploadResult{
		ID:       uploadID,
		Filename: fileHeader.Filename,
		MimeType: mimeType,
		Size:     fileHeader.Size,
		URL:      "/uploads/" + storedName,
	}, nil
}
