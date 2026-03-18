package handler

import (
	"net/http"

	"github.com/earnlearning/backend/internal/application"
	"github.com/earnlearning/backend/internal/domain/classroom"
	"github.com/earnlearning/backend/internal/interfaces/http/middleware"
	"github.com/labstack/echo/v4"
)

type ClassroomHandler struct {
	classroomUC *application.ClassroomUseCase
}

func NewClassroomHandler(uc *application.ClassroomUseCase) *ClassroomHandler {
	return &ClassroomHandler{classroomUC: uc}
}

func (h *ClassroomHandler) CreateClassroom(c echo.Context) error {
	var input application.CreateClassroomInput
	if err := c.Bind(&input); err != nil {
		return errorResponse(c, http.StatusBadRequest, "INVALID_INPUT", "잘못된 입력입니다")
	}

	userID := middleware.GetUserID(c)
	classroom, err := h.classroomUC.CreateClassroom(input, userID)
	if err != nil {
		return errorResponse(c, http.StatusInternalServerError, "INTERNAL_ERROR", "클래스룸 생성에 실패했습니다")
	}

	return successResponse(c, http.StatusCreated, classroom)
}

func (h *ClassroomHandler) JoinClassroom(c echo.Context) error {
	var input application.JoinClassroomInput
	if err := c.Bind(&input); err != nil {
		return errorResponse(c, http.StatusBadRequest, "INVALID_INPUT", "잘못된 입력입니다")
	}

	userID := middleware.GetUserID(c)
	classroom, err := h.classroomUC.JoinClassroom(input.Code, userID)
	if err != nil {
		return errorResponse(c, http.StatusBadRequest, "JOIN_FAILED", err.Error())
	}

	return successResponse(c, http.StatusOK, classroom)
}

func (h *ClassroomHandler) GetClassroom(c echo.Context) error {
	id, err := intParam(c, "id")
	if err != nil {
		return errorResponse(c, http.StatusBadRequest, "INVALID_ID", "유효하지 않은 ID입니다")
	}

	classroom, err := h.classroomUC.GetClassroom(id)
	if err != nil {
		return errorResponse(c, http.StatusNotFound, "NOT_FOUND", err.Error())
	}

	channels, _ := h.classroomUC.GetClassroomChannels(id)
	members, _ := h.classroomUC.GetClassroomMembers(id)

	return successResponse(c, http.StatusOK, map[string]interface{}{
		"classroom": classroom,
		"channels":  channels,
		"members":   members,
	})
}

func (h *ClassroomHandler) GetClassroomDashboard(c echo.Context) error {
	id, err := intParam(c, "id")
	if err != nil {
		return errorResponse(c, http.StatusBadRequest, "INVALID_ID", "유효하지 않은 ID입니다")
	}

	cr, err := h.classroomUC.GetClassroom(id)
	if err != nil {
		return errorResponse(c, http.StatusNotFound, "NOT_FOUND", err.Error())
	}

	members, err := h.classroomUC.GetMemberDashboard(id)
	if err != nil {
		return errorResponse(c, http.StatusInternalServerError, "INTERNAL_ERROR", "멤버 데이터 조회에 실패했습니다")
	}

	if members == nil {
		members = []*classroom.MemberDashboard{}
	}

	return successResponse(c, http.StatusOK, map[string]interface{}{
		"classroom": cr,
		"members":   members,
	})
}

func (h *ClassroomHandler) ListMyClassrooms(c echo.Context) error {
	userID := middleware.GetUserID(c)
	classrooms, err := h.classroomUC.ListMyClassrooms(userID)
	if err != nil {
		return errorResponse(c, http.StatusInternalServerError, "INTERNAL_ERROR", "서버 오류가 발생했습니다")
	}

	if classrooms == nil {
		classrooms = []*classroom.Classroom{}
	}

	return successResponse(c, http.StatusOK, classrooms)
}
