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

// CreateClassroom godoc
//
//	@Summary		클래스룸 생성
//	@Description	새 클래스룸 생성 (관리자)
//	@Tags			Classroom
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body	body		CreateClassroomRequest	true	"클래스룸 정보"
//	@Success		201		{object}	APIResponse
//	@Router			/classrooms [post]
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

// JoinClassroom godoc
//
//	@Summary		클래스룸 참여
//	@Description	초대 코드로 클래스룸 참여
//	@Tags			Classroom
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body	body		JoinClassroomRequest	true	"참여 코드"
//	@Success		200		{object}	APIResponse
//	@Failure		400		{object}	APIResponse
//	@Router			/classrooms/join [post]
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

// GetClassroom godoc
//
//	@Summary		클래스룸 상세 조회
//	@Description	클래스룸 정보, 채널, 멤버 조회
//	@Tags			Classroom
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		int	true	"클래스룸 ID"
//	@Success		200	{object}	APIResponse
//	@Failure		404	{object}	APIResponse
//	@Router			/classrooms/{id} [get]
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

// GetClassroomDashboard godoc
//
//	@Summary		클래스룸 대시보드
//	@Description	관리자용: 클래스룸 멤버 활동 대시보드
//	@Tags			Admin
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		int	true	"클래스룸 ID"
//	@Success		200	{object}	APIResponse
//	@Router			/admin/classrooms/{id}/dashboard [get]
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

// ListMyClassrooms godoc
//
//	@Summary		내 클래스룸 목록
//	@Description	내가 속한 클래스룸 목록 조회
//	@Tags			Classroom
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	APIResponse
//	@Router			/classrooms [get]
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
