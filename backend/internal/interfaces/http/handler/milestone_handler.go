package handler

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/earnlearning/backend/internal/application"
	"github.com/earnlearning/backend/internal/domain/milestone"
	"github.com/earnlearning/backend/internal/interfaces/http/middleware"
	"github.com/labstack/echo/v4"
)

type MilestoneHandler struct {
	uc *application.MilestoneUseCase
}

func NewMilestoneHandler(uc *application.MilestoneUseCase) *MilestoneHandler {
	return &MilestoneHandler{uc: uc}
}

// GetMyMilestones godoc
//
//	@Summary  내 평가지표 진행 현황
//	@Tags     Milestone
//	@Produce  json
//	@Security BearerAuth
//	@Success  200 {object} APIResponse
//	@Router   /milestones/mine [get]
func (h *MilestoneHandler) GetMyMilestones(c echo.Context) error {
	userID := middleware.GetUserID(c)
	// 호출 시점에 자동 집계 동기화 (회사·grant URL 변경 즉시 반영).
	if _, err := h.uc.SyncAuto(userID); err != nil {
		return c.JSON(http.StatusInternalServerError, errorResp("INTERNAL", err.Error()))
	}
	prog, err := h.uc.ListForStudent(userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, errorResp("INTERNAL", err.Error()))
	}
	return c.JSON(http.StatusOK, successResp(prog))
}

// SubmitMilestone godoc
//
//	@Summary  평가지표 수동 제출 (사업계획서/회고 또는 MVP URL 직접 입력)
//	@Tags     Milestone
//	@Accept   json
//	@Produce  json
//	@Security BearerAuth
//	@Success  200 {object} APIResponse
//	@Router   /milestones [post]
func (h *MilestoneHandler) SubmitMilestone(c echo.Context) error {
	userID := middleware.GetUserID(c)
	var in application.SubmitManualInput
	if err := c.Bind(&in); err != nil {
		return c.JSON(http.StatusBadRequest, errorResp("BAD_REQUEST", "잘못된 요청입니다"))
	}
	m, err := h.uc.SubmitManual(userID, in)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errorResp("BAD_REQUEST", err.Error()))
	}
	return c.JSON(http.StatusOK, successResp(m))
}

// ScoreEssay godoc
//
//	@Summary  회고 에세이 AI 작성 확률 셀프체크 (저장 안 함)
//	@Tags     Milestone
//	@Accept   json
//	@Produce  json
//	@Security BearerAuth
//	@Success  200 {object} APIResponse
//	@Router   /milestones/essay/score [post]
func (h *MilestoneHandler) ScoreEssay(c echo.Context) error {
	var body struct {
		Text string `json:"text"`
	}
	if err := c.Bind(&body); err != nil {
		return c.JSON(http.StatusBadRequest, errorResp("BAD_REQUEST", "잘못된 요청입니다"))
	}
	if len([]rune(body.Text)) < 200 {
		return c.JSON(http.StatusBadRequest, errorResp("TOO_SHORT", "200자 이상이어야 평가 가능합니다"))
	}
	// 셀프체크는 평가 자체만 — DB 저장 안 함. 학생당 rate-limit 없음 (학생 수 적음).
	ctx, cancel := context.WithTimeout(c.Request().Context(), 35*time.Second)
	defer cancel()
	result := h.uc.EvaluateEssay(ctx, body.Text)
	return c.JSON(http.StatusOK, successResp(result))
}

// AdminListMilestones godoc
//
//	@Summary  관리자: 전체 학생 매트릭스
//	@Tags     Admin
//	@Produce  json
//	@Security BearerAuth
//	@Success  200 {object} APIResponse
//	@Router   /admin/milestones [get]
func (h *MilestoneHandler) AdminListMilestones(c echo.Context) error {
	// admin이 매트릭스 보기 전에 모든 학생 sync (옵션 — 시간이 좀 걸릴 수 있음).
	// 학생 수가 적은 ewha 사이즈에서는 빠름.
	if c.QueryParam("sync") == "1" {
		// list 먼저 가져와서 student id 모음.
		all, err := h.uc.ListAll()
		if err != nil {
			return c.JSON(http.StatusInternalServerError, errorResp("INTERNAL", err.Error()))
		}
		for _, p := range all {
			_, _ = h.uc.SyncAuto(p.Student.ID)
		}
	}

	all, err := h.uc.ListAll()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, errorResp("INTERNAL", err.Error()))
	}
	return c.JSON(http.StatusOK, successResp(all))
}

// UploadFile godoc
//
//	@Summary  사업계획서 비공개 첨부 업로드 (다중 파일은 반복 호출)
//	@Tags     Milestone
//	@Accept   multipart/form-data
//	@Produce  json
//	@Security BearerAuth
//	@Param    file formData file true "첨부 파일"
//	@Success  201 {object} APIResponse
//	@Router   /milestones/files [post]
func (h *MilestoneHandler) UploadFile(c echo.Context) error {
	userID := middleware.GetUserID(c)
	file, err := c.FormFile("file")
	if err != nil {
		return c.JSON(http.StatusBadRequest, errorResp("NO_FILE", "파일이 첨부되지 않았습니다"))
	}
	typ := milestone.Type(c.FormValue("type"))
	if typ == "" {
		typ = milestone.TypeBusinessPlan
	}
	f, err := h.uc.UploadFile(userID, typ, file, generateUUID())
	if err != nil {
		return c.JSON(http.StatusBadRequest, errorResp("UPLOAD_FAILED", err.Error()))
	}
	return c.JSON(http.StatusCreated, successResp(f))
}

// ListFiles godoc
//
//	@Summary  내 사업계획서 첨부 목록
//	@Tags     Milestone
//	@Produce  json
//	@Security BearerAuth
//	@Success  200 {object} APIResponse
//	@Router   /milestones/files [get]
func (h *MilestoneHandler) ListFiles(c echo.Context) error {
	userID := middleware.GetUserID(c)
	typ := milestone.Type(c.QueryParam("type"))
	if typ == "" {
		typ = milestone.TypeBusinessPlan
	}
	files, err := h.uc.ListFiles(userID, typ)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, errorResp("INTERNAL", err.Error()))
	}
	if files == nil {
		files = []*milestone.FileRef{}
	}
	return c.JSON(http.StatusOK, successResp(files))
}

// DownloadFile godoc
//
//	@Summary  첨부 다운로드/열람 (업로더 본인 또는 관리자만)
//	@Tags     Milestone
//	@Security BearerAuth
//	@Param    id path int true "file id"
//	@Success  200 {file} file
//	@Router   /milestones/files/{id} [get]
func (h *MilestoneHandler) DownloadFile(c echo.Context) error {
	userID := middleware.GetUserID(c)
	isAdmin := middleware.GetUserRole(c) == "admin"
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, errorResp("INVALID_ID", "유효하지 않은 ID"))
	}
	f, err := h.uc.GetFileForAccess(id, userID, isAdmin)
	if err != nil {
		switch err {
		case milestone.ErrForbidden:
			return c.JSON(http.StatusForbidden, errorResp("FORBIDDEN", err.Error()))
		case milestone.ErrFileNotFound:
			return c.JSON(http.StatusNotFound, errorResp("NOT_FOUND", err.Error()))
		}
		return c.JSON(http.StatusInternalServerError, errorResp("INTERNAL", err.Error()))
	}
	// Inline: PDF/이미지는 브라우저에서 바로 열람, 그 외는 다운로드.
	return c.Inline(f.Path, f.Filename)
}

// DeleteFile godoc
//
//	@Summary  첨부 삭제 (업로더 본인 또는 관리자만)
//	@Tags     Milestone
//	@Security BearerAuth
//	@Param    id path int true "file id"
//	@Success  200 {object} APIResponse
//	@Router   /milestones/files/{id} [delete]
func (h *MilestoneHandler) DeleteFile(c echo.Context) error {
	userID := middleware.GetUserID(c)
	isAdmin := middleware.GetUserRole(c) == "admin"
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, errorResp("INVALID_ID", "유효하지 않은 ID"))
	}
	if err := h.uc.DeleteFile(id, userID, isAdmin); err != nil {
		switch err {
		case milestone.ErrForbidden:
			return c.JSON(http.StatusForbidden, errorResp("FORBIDDEN", err.Error()))
		case milestone.ErrFileNotFound:
			return c.JSON(http.StatusNotFound, errorResp("NOT_FOUND", err.Error()))
		}
		return c.JSON(http.StatusInternalServerError, errorResp("INTERNAL", err.Error()))
	}
	return c.JSON(http.StatusOK, successResp(map[string]string{"message": "삭제되었습니다"}))
}

// AdminApproveMilestone godoc
//
//	@Summary  관리자: 평가지표 승인
//	@Tags     Admin
//	@Accept   json
//	@Produce  json
//	@Security BearerAuth
//	@Param    id path int true "milestone id"
//	@Success  200 {object} APIResponse
//	@Router   /admin/milestones/{id}/approve [post]
func (h *MilestoneHandler) AdminApproveMilestone(c echo.Context) error {
	adminID := middleware.GetUserID(c)
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, errorResp("INVALID_ID", "유효하지 않은 ID"))
	}
	var body struct {
		AdminNote string `json:"admin_note"`
	}
	_ = c.Bind(&body)

	if err := h.uc.Approve(id, adminID, body.AdminNote); err != nil {
		if err == milestone.ErrNotFound {
			return c.JSON(http.StatusNotFound, errorResp("NOT_FOUND", err.Error()))
		}
		return c.JSON(http.StatusBadRequest, errorResp("BAD_REQUEST", err.Error()))
	}
	return c.JSON(http.StatusOK, successResp(map[string]string{"message": "승인되었습니다"}))
}

// AdminRejectMilestone godoc
//
//	@Summary  관리자: 평가지표 반려
//	@Tags     Admin
//	@Accept   json
//	@Produce  json
//	@Security BearerAuth
//	@Param    id path int true "milestone id"
//	@Success  200 {object} APIResponse
//	@Router   /admin/milestones/{id}/reject [post]
func (h *MilestoneHandler) AdminRejectMilestone(c echo.Context) error {
	adminID := middleware.GetUserID(c)
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, errorResp("INVALID_ID", "유효하지 않은 ID"))
	}
	var body struct {
		AdminNote string `json:"admin_note"`
	}
	_ = c.Bind(&body)

	if err := h.uc.Reject(id, adminID, body.AdminNote); err != nil {
		if err == milestone.ErrNotFound {
			return c.JSON(http.StatusNotFound, errorResp("NOT_FOUND", err.Error()))
		}
		return c.JSON(http.StatusBadRequest, errorResp("BAD_REQUEST", err.Error()))
	}
	return c.JSON(http.StatusOK, successResp(map[string]string{"message": "반려되었습니다"}))
}
