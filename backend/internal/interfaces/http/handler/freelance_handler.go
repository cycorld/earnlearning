package handler

import (
	"net/http"
	"strconv"

	"github.com/earnlearning/backend/internal/application"
	"github.com/earnlearning/backend/internal/domain/freelance"
	"github.com/earnlearning/backend/internal/interfaces/http/middleware"
	"github.com/labstack/echo/v4"
)

type FreelanceHandler struct {
	uc *application.FreelanceUseCase
}

func NewFreelanceHandler(uc *application.FreelanceUseCase) *FreelanceHandler {
	return &FreelanceHandler{uc: uc}
}

// ListJobs godoc
//
//	@Summary		프리랜서 잡 목록
//	@Description	프리랜서 잡 목록 조회 (필터, 페이지네이션)
//	@Tags			Freelance
//	@Produce		json
//	@Security		BearerAuth
//	@Param			status		query		string	false	"상태 필터 (open, in_progress, completed)"
//	@Param			skills		query		string	false	"스킬 필터"
//	@Param			min_budget	query		int		false	"최소 예산"
//	@Param			page		query		int		false	"페이지"	default(1)
//	@Param			limit		query		int		false	"크기"	default(20)
//	@Success		200			{object}	APIResponse
//	@Router			/freelance/jobs [get]
func (h *FreelanceHandler) ListJobs(c echo.Context) error {
	status := c.QueryParam("status")
	skills := c.QueryParam("skills")
	minBudget, _ := strconv.Atoi(c.QueryParam("min_budget"))
	page, _ := strconv.Atoi(c.QueryParam("page"))
	limit, _ := strconv.Atoi(c.QueryParam("limit"))

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 50 {
		limit = 20
	}

	jobs, total, err := h.uc.ListJobs(status, skills, minBudget, page, limit)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, errorResp("INTERNAL", err.Error()))
	}

	totalPages := 0
	if limit > 0 {
		totalPages = (total + limit - 1) / limit
	}

	return c.JSON(http.StatusOK, successResp(map[string]interface{}{
		"data": jobs,
		"pagination": map[string]interface{}{
			"page":        page,
			"limit":       limit,
			"total":       total,
			"total_pages": totalPages,
		},
	}))
}

// CreateJob godoc
//
//	@Summary		프리랜서 잡 등록
//	@Description	새 프리랜서 잡 등록
//	@Tags			Freelance
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body	body		CreateJobRequest	true	"잡 정보"
//	@Success		201		{object}	APIResponse
//	@Router			/freelance/jobs [post]
func (h *FreelanceHandler) CreateJob(c echo.Context) error {
	userID := middleware.GetUserID(c)
	var input application.CreateJobInput
	if err := c.Bind(&input); err != nil {
		return c.JSON(http.StatusBadRequest, errorResp("BAD_REQUEST", "잘못된 요청입니다"))
	}
	job, err := h.uc.CreateJob(input, userID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errorResp("BAD_REQUEST", err.Error()))
	}
	return c.JSON(http.StatusCreated, successResp(job))
}

// GetJob godoc
//
//	@Summary		프리랜서 잡 상세
//	@Description	프리랜서 잡 상세 조회
//	@Tags			Freelance
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		int	true	"잡 ID"
//	@Success		200	{object}	APIResponse
//	@Failure		404	{object}	APIResponse
//	@Router			/freelance/jobs/{id} [get]
func (h *FreelanceHandler) GetJob(c echo.Context) error {
	userID := middleware.GetUserID(c)
	jobID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, errorResp("BAD_REQUEST", "잘못된 ID입니다"))
	}
	job, err := h.uc.GetJob(jobID, userID)
	if err != nil {
		return c.JSON(http.StatusNotFound, errorResp("NOT_FOUND", err.Error()))
	}
	return c.JSON(http.StatusOK, successResp(job))
}

// ListApplications godoc
//
//	@Summary		지원자 목록
//	@Description	프리랜서 잡의 지원자 목록 조회
//	@Tags			Freelance
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		int	true	"잡 ID"
//	@Success		200	{object}	APIResponse
//	@Router			/freelance/jobs/{id}/applications [get]
func (h *FreelanceHandler) ListApplications(c echo.Context) error {
	jobID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, errorResp("BAD_REQUEST", "잘못된 ID입니다"))
	}
	apps, err := h.uc.ListApplications(jobID)
	if err != nil {
		return c.JSON(http.StatusNotFound, errorResp("NOT_FOUND", err.Error()))
	}
	if apps == nil {
		apps = []*freelance.JobApplication{}
	}
	return c.JSON(http.StatusOK, successResp(apps))
}

// ApplyToJob godoc
//
//	@Summary		잡 지원
//	@Description	프리랜서 잡에 지원
//	@Tags			Freelance
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		int				true	"잡 ID"
//	@Param			body	body		ApplyJobRequest	true	"지원 정보"
//	@Success		200		{object}	APIResponse
//	@Router			/freelance/jobs/{id}/apply [post]
func (h *FreelanceHandler) ApplyToJob(c echo.Context) error {
	userID := middleware.GetUserID(c)
	jobID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, errorResp("BAD_REQUEST", "잘못된 ID입니다"))
	}
	var input application.ApplyJobInput
	if err := c.Bind(&input); err != nil {
		return c.JSON(http.StatusBadRequest, errorResp("BAD_REQUEST", "잘못된 요청입니다"))
	}
	app, err := h.uc.ApplyToJob(jobID, input, userID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errorResp("BAD_REQUEST", err.Error()))
	}
	return c.JSON(http.StatusOK, successResp(app))
}

// AcceptApplication godoc
//
//	@Summary		지원 수락
//	@Description	프리랜서 잡 지원을 수락
//	@Tags			Freelance
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		int							true	"잡 ID"
//	@Param			body	body		AcceptApplicationRequest	true	"수락 정보"
//	@Success		200		{object}	APIResponse
//	@Router			/freelance/jobs/{id}/accept [post]
func (h *FreelanceHandler) AcceptApplication(c echo.Context) error {
	userID := middleware.GetUserID(c)
	jobID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, errorResp("BAD_REQUEST", "잘못된 ID입니다"))
	}
	var input struct {
		ApplicationID int `json:"application_id"`
	}
	if err := c.Bind(&input); err != nil || input.ApplicationID == 0 {
		return c.JSON(http.StatusBadRequest, errorResp("BAD_REQUEST", "잘못된 지원 ID입니다"))
	}
	if err := h.uc.AcceptApplication(jobID, input.ApplicationID, userID); err != nil {
		return c.JSON(http.StatusBadRequest, errorResp("BAD_REQUEST", err.Error()))
	}
	return c.JSON(http.StatusOK, successResp(map[string]string{"message": "지원이 수락되었습니다"}))
}

// CompleteWork godoc
//
//	@Summary		작업 완료 보고
//	@Description	프리랜서 작업 완료 보고
//	@Tags			Freelance
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		int					true	"잡 ID"
//	@Param			body	body		CompleteWorkRequest	true	"완료 보고"
//	@Success		200		{object}	APIResponse
//	@Router			/freelance/jobs/{id}/complete [post]
func (h *FreelanceHandler) CompleteWork(c echo.Context) error {
	userID := middleware.GetUserID(c)
	jobID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, errorResp("BAD_REQUEST", "잘못된 ID입니다"))
	}
	var input application.CompleteWorkInput
	_ = c.Bind(&input) // optional body; empty report is fine
	if err := h.uc.CompleteWork(jobID, userID, input); err != nil {
		return c.JSON(http.StatusBadRequest, errorResp("BAD_REQUEST", err.Error()))
	}
	return c.JSON(http.StatusOK, successResp(map[string]string{"message": "작업 완료가 제출되었습니다"}))
}

// ApproveJob godoc
//
//	@Summary		작업 승인
//	@Description	프리랜서 작업 결과 승인 → 대금 지급
//	@Tags			Freelance
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		int	true	"잡 ID"
//	@Success		200	{object}	APIResponse
//	@Router			/freelance/jobs/{id}/approve [post]
func (h *FreelanceHandler) ApproveJob(c echo.Context) error {
	userID := middleware.GetUserID(c)
	jobID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, errorResp("BAD_REQUEST", "잘못된 ID입니다"))
	}
	if err := h.uc.ApproveJob(jobID, userID); err != nil {
		return c.JSON(http.StatusBadRequest, errorResp("BAD_REQUEST", err.Error()))
	}
	return c.JSON(http.StatusOK, successResp(map[string]string{"message": "작업이 승인되었습니다"}))
}

// CancelJob godoc
//
//	@Summary		잡 취소
//	@Description	프리랜서 잡 취소
//	@Tags			Freelance
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		int	true	"잡 ID"
//	@Success		200	{object}	APIResponse
//	@Router			/freelance/jobs/{id}/cancel [post]
func (h *FreelanceHandler) CancelJob(c echo.Context) error {
	userID := middleware.GetUserID(c)
	jobID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, errorResp("BAD_REQUEST", "잘못된 ID입니다"))
	}
	if err := h.uc.CancelJob(jobID, userID); err != nil {
		return c.JSON(http.StatusBadRequest, errorResp("BAD_REQUEST", err.Error()))
	}
	return c.JSON(http.StatusOK, successResp(map[string]string{"message": "작업이 취소되었습니다"}))
}


// DisputeJob godoc
//
//	@Summary		분쟁 제기
//	@Description	프리랜서 작업에 분쟁 제기
//	@Tags			Freelance
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		int	true	"잡 ID"
//	@Success		200	{object}	APIResponse
//	@Router			/freelance/jobs/{id}/dispute [post]
func (h *FreelanceHandler) DisputeJob(c echo.Context) error {
	userID := middleware.GetUserID(c)
	jobID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, errorResp("BAD_REQUEST", "잘못된 ID입니다"))
	}
	if err := h.uc.DisputeJob(jobID, userID); err != nil {
		return c.JSON(http.StatusBadRequest, errorResp("BAD_REQUEST", err.Error()))
	}
	return c.JSON(http.StatusOK, successResp(map[string]string{"message": "분쟁이 제기되었습니다"}))
}

// ReviewJob godoc
//
//	@Summary		리뷰 작성
//	@Description	완료된 프리랜서 잡에 리뷰 작성
//	@Tags			Freelance
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		int				true	"잡 ID"
//	@Param			body	body		ReviewJobRequest	true	"리뷰 정보"
//	@Success		201		{object}	APIResponse
//	@Router			/freelance/jobs/{id}/review [post]
func (h *FreelanceHandler) ReviewJob(c echo.Context) error {
	userID := middleware.GetUserID(c)
	jobID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, errorResp("BAD_REQUEST", "잘못된 ID입니다"))
	}
	var input application.ReviewJobInput
	if err := c.Bind(&input); err != nil {
		return c.JSON(http.StatusBadRequest, errorResp("BAD_REQUEST", "잘못된 요청입니다"))
	}
	review, err := h.uc.ReviewJob(jobID, input, userID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errorResp("BAD_REQUEST", err.Error()))
	}
	return c.JSON(http.StatusCreated, successResp(review))
}

// --- Helper functions ---

func successResp(data interface{}) map[string]interface{} {
	return map[string]interface{}{
		"success": true,
		"data":    data,
		"error":   nil,
	}
}

func errorResp(code, message string) map[string]interface{} {
	return map[string]interface{}{
		"success": false,
		"data":    nil,
		"error": map[string]string{
			"code":    code,
			"message": message,
		},
	}
}
