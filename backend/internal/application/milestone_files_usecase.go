package application

import (
	"fmt"
	"io"
	"math"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"

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
}

// UploadFile — 학생이 business_plan 비공개 첨부를 업로드.
func (uc *MilestoneUseCase) UploadFile(studentID int, typ milestone.Type, fileHeader *multipart.FileHeader, uuidPrefix string) (*milestone.FileRef, error) {
	if typ != milestone.TypeBusinessPlan {
		return nil, fmt.Errorf("파일 첨부는 사업계획서에만 가능합니다")
	}
	if uc.privateUploadPath == "" {
		return nil, fmt.Errorf("파일 저장소가 설정되지 않았습니다")
	}
	if fileHeader.Size > MaxMilestoneFileSize {
		return nil, fmt.Errorf("파일 크기는 최대 20MB까지 허용됩니다")
	}
	ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
	if !allowedMilestoneFileExt[ext] {
		return nil, fmt.Errorf("허용되지 않는 파일 형식입니다: %s", ext)
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
	dst, err := os.Create(storedPath)
	if err != nil {
		return nil, fmt.Errorf("파일 저장 실패: %w", err)
	}
	defer dst.Close()
	if _, err := io.Copy(dst, src); err != nil {
		return nil, fmt.Errorf("파일 복사 실패: %w", err)
	}

	mimeType := fileHeader.Header.Get("Content-Type")
	if idx := strings.Index(mimeType, ";"); idx != -1 {
		mimeType = strings.TrimSpace(mimeType[:idx])
	}

	f := &milestone.FileRef{
		StudentID:  studentID,
		Type:       typ,
		Filename:   fileHeader.Filename,
		StoredName: storedName,
		MimeType:   mimeType,
		Size:       fileHeader.Size,
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
