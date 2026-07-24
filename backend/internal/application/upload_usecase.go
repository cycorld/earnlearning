package application

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/earnlearning/backend/internal/domain/post"
)

// Maximum upload size: 10MB
const MaxUploadSize = 10 * 1024 * 1024

var allowedMIMETypesByExtension = map[string]map[string]bool{
	".jpg": {"image/jpeg": true}, ".jpeg": {"image/jpeg": true}, ".png": {"image/png": true},
	".gif": {"image/gif": true}, ".webp": {"image/webp": true}, ".pdf": {"application/pdf": true},
	".doc": {"application/msword": true}, ".docx": {"application/vnd.openxmlformats-officedocument.wordprocessingml.document": true},
	".xls": {"application/vnd.ms-excel": true}, ".xlsx": {"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet": true},
	".ppt": {"application/vnd.ms-powerpoint": true}, ".pptx": {"application/vnd.openxmlformats-officedocument.presentationml.presentation": true},
	".txt": {"text/plain": true}, ".md": {"text/markdown": true, "text/plain": true},
	".zip": {"application/zip": true}, ".mp4": {"video/mp4": true}, ".mp3": {"audio/mpeg": true},
	".json": {"application/json": true, "text/plain": true},
}

var activeHTMLPattern = regexp.MustCompile(`(?is)<\s*(?:!doctype\s+html|html\b|head\b|body\b|script\b|iframe\b|object\b|embed\b|svg\b|form\b|meta\b|link\b)`)

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
	filename := fileHeader.Filename
	if filename == "" || filename != filepath.Base(filename) || strings.ContainsAny(filename, "\x00\r\n") {
		return nil, fmt.Errorf("안전하지 않은 파일명입니다")
	}
	ext := strings.ToLower(filepath.Ext(filename))
	allowedMIMEs, extensionAllowed := allowedMIMETypesByExtension[ext]
	if ext == "" || !extensionAllowed {
		return nil, fmt.Errorf("허용되지 않는 파일 확장자입니다: %s", ext)
	}

	// Check file size
	if fileHeader.Size < 0 || fileHeader.Size > MaxUploadSize {
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
	if mimeType != "application/octet-stream" && !allowedMIMEs[mimeType] {
		return nil, fmt.Errorf("파일 확장자와 MIME 형식이 일치하지 않습니다")
	}

	// Generate stored filename with UUID prefix
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
	reader := bufio.NewReader(src)
	textUpload := ext == ".md" || ext == ".txt" || ext == ".json"
	if !textUpload {
		sample, err := reader.Peek(512)
		if err != nil && err != io.EOF && err != bufio.ErrBufferFull {
			return nil, fmt.Errorf("파일 검사 실패: %w", err)
		}
		if err := validateUploadContent(ext, sample); err != nil {
			return nil, err
		}
	}

	limited := io.LimitReader(reader, MaxUploadSize+1)
	var textContent bytes.Buffer
	copySource := io.Reader(limited)
	if textUpload {
		// This buffer is bounded by the same 10MB+1 stream saved to disk.
		copySource = io.TeeReader(limited, &textContent)
	}
	written, err := copyUploadFile(storedPath, copySource)
	if err != nil {
		return nil, fmt.Errorf("파일 복사 실패: %w", err)
	}
	if written > MaxUploadSize {
		_ = os.Remove(storedPath)
		return nil, fmt.Errorf("파일 크기는 최대 10MB까지 허용됩니다")
	}
	if textUpload {
		if err := validateUploadContent(ext, textContent.Bytes()); err != nil {
			_ = os.Remove(storedPath)
			return nil, err
		}
	}

	// Save upload record
	upload := &post.Upload{
		UserID:     userID,
		Filename:   filename,
		StoredName: storedName,
		MimeType:   mimeType,
		Size:       written,
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
		Filename: filename,
		MimeType: mimeType,
		Size:     written,
		URL:      "/uploads/" + storedName,
	}, nil
}

func validateUploadContent(ext string, sample []byte) error {
	detected := strings.Split(http.DetectContentType(sample), ";")[0]
	if ext == ".md" || ext == ".txt" || ext == ".json" {
		// Reject active/document-level HTML anywhere; ordinary Markdown remains valid.
		if !utf8.Valid(sample) || bytes.IndexByte(sample, 0) >= 0 || detected == "text/html" || detected == "text/xml" || activeHTMLPattern.Match(sample) {
			return fmt.Errorf("텍스트 파일 내용이 올바르지 않습니다")
		}
		return nil
	}
	expected := map[string]string{".jpg": "image/jpeg", ".jpeg": "image/jpeg", ".png": "image/png", ".gif": "image/gif", ".webp": "image/webp", ".pdf": "application/pdf", ".mp4": "video/mp4"}
	if want := expected[ext]; want != "" && detected != want {
		return fmt.Errorf("파일 내용과 확장자가 일치하지 않습니다")
	}
	if ext == ".zip" || ext == ".docx" || ext == ".xlsx" || ext == ".pptx" {
		if len(sample) < 4 || !bytes.Equal(sample[:2], []byte("PK")) {
			return fmt.Errorf("ZIP 기반 파일 내용이 올바르지 않습니다")
		}
	}
	if ext == ".doc" || ext == ".xls" || ext == ".ppt" {
		ole := []byte{0xd0, 0xcf, 0x11, 0xe0, 0xa1, 0xb1, 0x1a, 0xe1}
		if len(sample) < len(ole) || !bytes.Equal(sample[:len(ole)], ole) {
			return fmt.Errorf("Office 파일 내용이 올바르지 않습니다")
		}
	}
	if ext == ".mp3" && detected != "audio/mpeg" && !(len(sample) >= 3 && string(sample[:3]) == "ID3") {
		return fmt.Errorf("파일 내용과 확장자가 일치하지 않습니다")
	}
	return nil
}

func copyUploadFile(path string, src io.Reader) (written int64, err error) {
	dst, err := os.Create(path)
	if err != nil {
		return 0, err
	}
	ok := false
	defer func() {
		if closeErr := dst.Close(); err == nil && closeErr != nil {
			err = closeErr
		}
		if !ok || err != nil {
			_ = os.Remove(path)
		}
	}()
	written, err = io.Copy(dst, src)
	ok = err == nil
	return written, err
}
