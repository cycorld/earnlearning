package handler

import (
	"net/http"

	"github.com/earnlearning/backend/internal/infrastructure/persistence"
	"github.com/labstack/echo/v4"
)

type TaskHandler struct {
	repo *persistence.TaskRepo
}

func NewTaskHandler(repo *persistence.TaskRepo) *TaskHandler {
	return &TaskHandler{repo: repo}
}

// ListTasks godoc
//
//	@Summary		태스크 목록
//	@Description	관리자용: 칸반 태스크 목록 조회
//	@Tags			Admin
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	APIResponse
//	@Router			/admin/tasks [get]
func (h *TaskHandler) ListTasks(c echo.Context) error {
	tasks, err := h.repo.List()
	if err != nil {
		return errorResponse(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
	}
	return successResponse(c, http.StatusOK, tasks)
}
