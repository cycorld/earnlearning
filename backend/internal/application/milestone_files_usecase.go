package application

import (
	"bytes"
	"fmt"
	"io"
	"math"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/earnlearning/backend/internal/domain/milestone"
)

// #125 — business_plan 비공개 첨부 파일 + 성적/자산 percentile.

// MaxMilestoneFileSize — 사업계획서 첨부 최대 크기 (nginx 50M 한도 내, 여유).
const MaxMilestoneFileSize = 20 * 1024 * 1024

// allowedMilestoneFileExt — 사업계획서에 흔한 포맷 (확장자 기반 검증, 소문자).
// 비공개(owner+admin) + 크기 제한이라 공개 업로드보다 폭넓게 허용.
var allowedMilestoneFileExt = map[string]bool{
	".pdf": true, ".doc": true, ".docx": true,
	".ppt": true, ".pptx": true, ".xls": true, ".xlsx": true,
	".hwp": true, ".hwpx": true, ".txt": true, ".md": true, ".csv": true,
	".zip": true, ".png": true, ".jpg": true, ".jpeg": true,
	".gif": true, ".webp": true,
	// #176 HTML 사업계획서. 다운로드 전용(attachment + octet-stream + nosniff)이며
	// 공개 /uploads 정적 경로에는 절대 노출되지 않으므로 스크립트가 실행될 수 없다.
	".html": true, ".htm": true,
}

// MaxMilestoneHTMLSize — #176 HTML 은 전체 내용을 버퍼링해 검증하므로 10MB 로 더 좁게 제한.
const MaxMilestoneHTMLSize = 10 * 1024 * 1024

// IsHTMLAttachmentExt — 확장자가 HTML 계열인지. 업로드 검증과 다운로드 헤더 결정에 함께 쓴다.
func IsHTMLAttachmentExt(ext string) bool {
	ext = strings.ToLower(ext)
	return ext == ".html" || ext == ".htm"
}

// validateMilestoneFilename — 경로 탈출·헤더 인젝션·Content-Disposition 파괴를 막는다.
// 한글 등 UTF-8 파일명은 그대로 허용한다.
func validateMilestoneFilename(name string) error {
	if name == "" || name != filepath.Base(name) ||
		strings.ContainsAny(name, "\x00\r\n\\\"") || strings.Contains(name, "/") ||
		name == "." || name == ".." {
		return fmt.Errorf("안전하지 않은 파일명입니다")
	}
	return nil
}

// SanitizeDownloadFilename — 다운로드 응답에 넣기 안전한 파일명.
// 업로드 검증 이전에 저장된 기존 레코드도 방어하기 위해 emit 시점에 한 번 더 정리한다.
func SanitizeDownloadFilename(name string, fallback string) string {
	name = filepath.Base(strings.ReplaceAll(name, "\\", "/"))
	name = strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7f || r == '"' {
			return -1
		}
		return r
	}, name)
	name = strings.TrimSpace(name)
	if name == "" || name == "." || name == ".." {
		return fallback
	}
	return name
}

// UploadFile — 학생이 business_plan 비공개 첨부를 업로드.
func (uc *MilestoneUseCase) UploadFile(studentID int, typ milestone.Type, fileHeader *multipart.FileHeader, uuidPrefix string) (*milestone.FileRef, error) {
	if typ != milestone.TypeBusinessPlan {
		return nil, fmt.Errorf("파일 첨부는 사업계획서에만 가능합니다")
	}
	if uc.privateUploadPath == "" {
		return nil, fmt.Errorf("파일 저장소가 설정되지 않았습니다")
	}
	if err := validateMilestoneFilename(fileHeader.Filename); err != nil {
		return nil, err
	}
	if fileHeader.Size > MaxMilestoneFileSize {
		return nil, fmt.Errorf("파일 크기는 최대 20MB까지 허용됩니다")
	}
	ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
	if !allowedMilestoneFileExt[ext] {
		return nil, fmt.Errorf("허용되지 않는 파일 형식입니다: %s", ext)
	}
	isHTML := IsHTMLAttachmentExt(ext)
	if isHTML && fileHeader.Size > MaxMilestoneHTMLSize {
		return nil, fmt.Errorf("HTML 파일 크기는 최대 10MB까지 허용됩니다")
	}

	storedName := uuidPrefix + ext
	if err := os.MkdirAll(uc.privateUploadPath, 0755); err != nil {
		return nil, fmt.Errorf("저장 디렉토리 생성 실패: %w", err)
	}
	storedPath := filepath.Join(uc.privateUploadPath, storedName)

	src, err := fileHeader.Open()
	if err != nil {
		return nil, fmt.Errorf("파일 열기 실패: %w", err)
	}
	defer src.Close()

	// 크기 상한을 스트림에서도 강제하고, 실패 시 조각 파일을 반드시 지운다.
	limit := int64(MaxMilestoneFileSize)
	if isHTML {
		limit = MaxMilestoneHTMLSize
	}
	limited := io.LimitReader(src, limit+1)
	var htmlBuf bytes.Buffer
	copySource := io.Reader(limited)
	if isHTML {
		// 10MB+1 로 상한이 걸린 스트림이라 버퍼 크기도 그만큼으로 제한된다.
		copySource = io.TeeReader(limited, &htmlBuf)
	}
	written, err := copyMilestoneFile(storedPath, copySource)
	if err != nil {
		return nil, fmt.Errorf("파일 복사 실패: %w", err)
	}
	if written > limit {
		_ = os.Remove(storedPath)
		if isHTML {
			return nil, fmt.Errorf("HTML 파일 크기는 최대 10MB까지 허용됩니다")
		}
		return nil, fmt.Errorf("파일 크기는 최대 20MB까지 허용됩니다")
	}
	if isHTML {
		// 전체 내용을 한 번에 검사한다(부분 버퍼는 멀티바이트 문자에서 오탐).
		// 태그·스크립트는 막지 않는다 — 다운로드 전용이라 실행될 수 없고,
		// 평범한 사업계획서 HTML 을 거부하면 기능이 무의미해진다.
		if b := htmlBuf.Bytes(); !utf8.Valid(b) || bytes.IndexByte(b, 0) >= 0 {
			_ = os.Remove(storedPath)
			return nil, fmt.Errorf("HTML 파일 내용이 올바르지 않습니다 (UTF-8 텍스트만 허용)")
		}
	}

	mimeType := fileHeader.Header.Get("Content-Type")
	if idx := strings.Index(mimeType, ";"); idx != -1 {
		mimeType = strings.TrimSpace(mimeType[:idx])
	}
	if isHTML {
		// 클라이언트가 보낸 MIME 을 그대로 신뢰하지 않는다. 다운로드는 항상 octet-stream.
		mimeType = "application/octet-stream"
	}

	f := &milestone.FileRef{
		StudentID:  studentID,
		Type:       typ,
		Filename:   fileHeader.Filename,
		StoredName: storedName,
		MimeType:   mimeType,
		Size:       written,
		Path:       storedPath,
	}
	id, err := uc.repo.AddFile(f)
	if err != nil {
		os.Remove(storedPath) // DB 실패 시 파일 정리
		return nil, fmt.Errorf("첨부 기록 저장 실패: %w", err)
	}
	f.ID = id
	return f, nil
}

// ListFiles — 학생 본인의 business_plan 첨부 목록.
func (uc *MilestoneUseCase) ListFiles(studentID int, typ milestone.Type) ([]*milestone.FileRef, error) {
	return uc.repo.ListFiles(studentID, typ)
}

// GetFileForAccess — 다운로드 권한 검증 후 FileRef 반환. owner 본인 OR 관리자만.
func (uc *MilestoneUseCase) GetFileForAccess(fileID, requesterID int, isAdmin bool) (*milestone.FileRef, error) {
	f, err := uc.repo.FindFileByID(fileID)
	if err != nil {
		return nil, err
	}
	if !isAdmin && f.StudentID != requesterID {
		return nil, milestone.ErrForbidden
	}
	return f, nil
}

// DeleteFile — 본인(또는 관리자) 첨부 삭제. DB + 디스크 정리.
func (uc *MilestoneUseCase) DeleteFile(fileID, requesterID int, isAdmin bool) error {
	f, err := uc.repo.FindFileByID(fileID)
	if err != nil {
		return err
	}
	if !isAdmin && f.StudentID != requesterID {
		return milestone.ErrForbidden
	}
	if err := uc.repo.DeleteFile(fileID); err != nil {
		return err
	}
	if f.Path != "" {
		_ = os.Remove(f.Path) // 파일 정리 실패는 무시 (DB는 이미 삭제됨)
	}
	return nil
}

// attachBusinessPlanFiles — StudentProgress 의 business_plan milestone 에 첨부 파일 목록을 채움.
// milestone row 가 없으면(미제출) 스킵 — 파일은 별도 GET /milestones/files 로도 조회 가능.
func (uc *MilestoneUseCase) attachBusinessPlanFiles(p *milestone.StudentProgress) {
	files, err := uc.repo.ListFiles(p.Student.ID, milestone.TypeBusinessPlan)
	if err != nil || len(files) == 0 {
		return
	}
	for _, m := range p.Milestones {
		if m != nil && m.Type == milestone.TypeBusinessPlan {
			m.Files = files
			return
		}
	}
}

// computeAssetPercentile — 같은 A/B/C/D 그룹 내 자산가치 순위/상위 % 산정.
func (uc *MilestoneUseCase) computeAssetPercentile(p *milestone.StudentProgress) {
	assets, err := uc.repo.ListStudentAssets()
	if err != nil {
		return
	}
	myGroup := p.Group
	var myAsset int
	found := false
	var groupAssets []int
	for _, a := range assets {
		if a.StudentID == p.Student.ID {
			myAsset = a.TotalAsset
			found = true
		}
		if milestone.ClassifyGroup(a.ApprovedCount) == myGroup {
			groupAssets = append(groupAssets, a.TotalAsset)
		}
	}
	if !found {
		return
	}
	p.AssetTotal = myAsset
	p.GroupSize = len(groupAssets)
	rank := 1
	for _, v := range groupAssets {
		if v > myAsset {
			rank++
		}
	}
	p.AssetRank = rank
	if p.GroupSize > 0 {
		pct := int(math.Ceil(float64(rank) / float64(p.GroupSize) * 100))
		if pct < 1 {
			pct = 1
		}
		if pct > 100 {
			pct = 100
		}
		p.AssetPercentile = pct
	}
}

// copyMilestoneFile — 실패 시 조각 파일을 남기지 않는다 (#176 cleanup).
func copyMilestoneFile(path string, src io.Reader) (written int64, err error) {
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
