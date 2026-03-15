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
