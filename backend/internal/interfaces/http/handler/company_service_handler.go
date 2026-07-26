package handler

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/earnlearning/backend/internal/application"
	"github.com/earnlearning/backend/internal/domain/company"
	"github.com/earnlearning/backend/internal/interfaces/http/middleware"
)

// CompanyServiceHandler — #181 회사 등록 서비스 (URL 검증 + Rybbit 연동).
type CompanyServiceHandler struct {
	uc *application.CompanyServiceUseCase
}

func NewCompanyServiceHandler(uc *application.CompanyServiceUseCase) *CompanyServiceHandler {
	return &CompanyServiceHandler{uc: uc}
}

// companyServiceErr — 도메인 에러 → HTTP 상태/코드 매핑 (#181).
// 외부 시스템 실패 메시지는 그대로 노출하지 않는다 (내부 정보 유출 방지).
func companyServiceErr(c echo.Context, err error) error {
	switch {
	case errors.Is(err, company.ErrNotFound), errors.Is(err, company.ErrServiceNotFound):
		return errorResponse(c, http.StatusNotFound, "NOT_FOUND", err.Error())
	case errors.Is(err, company.ErrNotOwner):
		return errorResponse(c, http.StatusForbidden, "NOT_OWNER", err.Error())
	case errors.Is(err, company.ErrNotStudent):
		return errorResponse(c, http.StatusForbidden, "NOT_STUDENT", err.Error())
	case errors.Is(err, company.ErrNotApproved):
		return errorResponse(c, http.StatusForbidden, "NOT_APPROVED", err.Error())
	case errors.Is(err, company.ErrNoClassroom):
		return errorResponse(c, http.StatusForbidden, "NO_CLASSROOM", err.Error())
	case errors.Is(err, company.ErrWrongClassroom):
		return errorResponse(c, http.StatusForbidden, "WRONG_CLASSROOM", err.Error())
	case errors.Is(err, company.ErrInvalidServiceURL):
		return errorResponse(c, http.StatusBadRequest, "INVALID_SERVICE_URL", err.Error())
	case errors.Is(err, company.ErrServiceNameTooLong):
		return errorResponse(c, http.StatusBadRequest, "INVALID_INPUT", err.Error())
	case errors.Is(err, company.ErrDuplicateServiceURL):
		return errorResponse(c, http.StatusConflict, "DUPLICATE_SERVICE_URL", err.Error())
	case errors.Is(err, company.ErrValidationRequired):
		return errorResponse(c, http.StatusConflict, "VALIDATION_REQUIRED", err.Error())
	case errors.Is(err, company.ErrValidationStale):
		return errorResponse(c, http.StatusConflict, "VALIDATION_STALE", err.Error())
	case errors.Is(err, company.ErrRybbitNotConfigured):
		return errorResponse(c, http.StatusServiceUnavailable, "RYBBIT_NOT_CONFIGURED",
			company.ErrRybbitNotConfigured.Error())
	case errors.Is(err, company.ErrRybbitFailed):
		return errorResponse(c, http.StatusBadGateway, "RYBBIT_CONNECT_FAILED",
			company.ErrRybbitFailed.Error())
	default:
		return errorResponse(c, http.StatusInternalServerError, "INTERNAL", "서버 오류가 발생했습니다")
	}
}

// serviceIDs — 경로의 회사 id / 서비스 id 를 함께 파싱한다.
func serviceIDs(c echo.Context) (companyID, serviceID int, err error) {
	companyID, err = intParam(c, "id")
	if err != nil {
		return 0, 0, err
	}
	serviceID, err = intParam(c, "serviceId")
	if err != nil {
		return 0, 0, err
	}
	return companyID, serviceID, nil
}

// ListServices godoc
//
//	@Summary		회사 등록 서비스 목록
//	@Description	회사에 등록된 서비스 목록 (승인 사용자 누구나 조회)
//	@Tags			Company
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		int	true	"회사 ID"
//	@Success		200	{object}	APIResponse
//	@Failure		404	{object}	APIResponse
//	@Router			/companies/{id}/services [get]
func (h *CompanyServiceHandler) ListServices(c echo.Context) error {
	companyID, err := intParam(c, "id")
	if err != nil {
		return errorResponse(c, http.StatusBadRequest, "INVALID_ID", "유효하지 않은 ID입니다")
	}
	list, err := h.uc.ListServices(companyID)
	if err != nil {
		return companyServiceErr(c, err)
	}
	return successResponse(c, http.StatusOK, list)
}

// CreateService godoc
//
//	@Summary		서비스 등록
//	@Description	회사에 서비스 URL 등록 (소유자만, https 전용)
//	@Tags			Company
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		int	true	"회사 ID"
//	@Success		201	{object}	APIResponse
//	@Failure		400	{object}	APIResponse
//	@Failure		403	{object}	APIResponse
//	@Failure		409	{object}	APIResponse
//	@Router			/companies/{id}/services [post]
func (h *CompanyServiceHandler) CreateService(c echo.Context) error {
	userID := middleware.GetUserID(c)
	companyID, err := intParam(c, "id")
	if err != nil {
		return errorResponse(c, http.StatusBadRequest, "INVALID_ID", "유효하지 않은 ID입니다")
	}
	var input application.CompanyServiceInput
	if err := c.Bind(&input); err != nil {
		return errorResponse(c, http.StatusBadRequest, "INVALID_INPUT", "잘못된 입력입니다")
	}
	created, err := h.uc.CreateService(companyID, userID, input)
	if err != nil {
		return companyServiceErr(c, err)
	}
	return successResponse(c, http.StatusCreated, created)
}

// UpdateService godoc
//
//	@Summary		서비스 수정
//	@Description	서비스 이름/URL 수정 (소유자만). URL 이 바뀌면 검증 초기화 + 재연동 필요
//	@Tags			Company
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id			path		int	true	"회사 ID"
//	@Param			serviceId	path		int	true	"서비스 ID"
//	@Success		200			{object}	APIResponse
//	@Failure		403			{object}	APIResponse
//	@Failure		404			{object}	APIResponse
//	@Router			/companies/{id}/services/{serviceId} [put]
func (h *CompanyServiceHandler) UpdateService(c echo.Context) error {
	userID := middleware.GetUserID(c)
	companyID, serviceID, err := serviceIDs(c)
	if err != nil {
		return errorResponse(c, http.StatusBadRequest, "INVALID_ID", "유효하지 않은 ID입니다")
	}
	var input application.CompanyServiceInput
	if err := c.Bind(&input); err != nil {
		return errorResponse(c, http.StatusBadRequest, "INVALID_INPUT", "잘못된 입력입니다")
	}
	updated, err := h.uc.UpdateService(companyID, serviceID, userID, input)
	if err != nil {
		return companyServiceErr(c, err)
	}
	return successResponse(c, http.StatusOK, updated)
}

// DeleteService godoc
//
//	@Summary		서비스 삭제
//	@Description	등록한 서비스 삭제 (소유자만). Rybbit 사이트는 그대로 둔다
//	@Tags			Company
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id			path		int	true	"회사 ID"
//	@Param			serviceId	path		int	true	"서비스 ID"
//	@Success		200			{object}	APIResponse
//	@Failure		403			{object}	APIResponse
//	@Failure		404			{object}	APIResponse
//	@Router			/companies/{id}/services/{serviceId} [delete]
func (h *CompanyServiceHandler) DeleteService(c echo.Context) error {
	userID := middleware.GetUserID(c)
	companyID, serviceID, err := serviceIDs(c)
	if err != nil {
		return errorResponse(c, http.StatusBadRequest, "INVALID_ID", "유효하지 않은 ID입니다")
	}
	if err := h.uc.DeleteService(companyID, serviceID, userID); err != nil {
		return companyServiceErr(c, err)
	}
	return successResponse(c, http.StatusOK, map[string]string{"message": "서비스를 삭제했어요"})
}

// ValidateService godoc
//
//	@Summary		서비스 URL 검증
//	@Description	서버가 직접 접속해 도달 가능한지 확인하고 결과를 기록 (소유자만)
//	@Tags			Company
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id			path		int	true	"회사 ID"
//	@Param			serviceId	path		int	true	"서비스 ID"
//	@Success		200			{object}	APIResponse
//	@Failure		403			{object}	APIResponse
//	@Failure		404			{object}	APIResponse
//	@Router			/companies/{id}/services/{serviceId}/validate [post]
func (h *CompanyServiceHandler) ValidateService(c echo.Context) error {
	userID := middleware.GetUserID(c)
	companyID, serviceID, err := serviceIDs(c)
	if err != nil {
		return errorResponse(c, http.StatusBadRequest, "INVALID_ID", "유효하지 않은 ID입니다")
	}
	// 검사가 끝났으면 결과가 invalid 여도 200 (실패 사유는 응답 본문의 validation_detail).
	updated, err := h.uc.ValidateService(c.Request().Context(), companyID, serviceID, userID)
	if err != nil {
		return companyServiceErr(c, err)
	}
	return successResponse(c, http.StatusOK, updated)
}

// ConnectRybbit godoc
//
//	@Summary		Rybbit 연동
//	@Description	검증된 서비스를 Rybbit(웹 애널리틱스) 사이트로 등록하고 소유자의 회사 전체 사이트 접근을 재조정 (소유자=승인된 학생 + 강의실 일치)
//	@Tags			Company
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id			path		int	true	"회사 ID"
//	@Param			serviceId	path		int	true	"서비스 ID"
//	@Success		200			{object}	APIResponse
//	@Failure		409			{object}	APIResponse
//	@Failure		502			{object}	APIResponse
//	@Failure		503			{object}	APIResponse
//	@Router			/companies/{id}/services/{serviceId}/rybbit/connect [post]
func (h *CompanyServiceHandler) ConnectRybbit(c echo.Context) error {
	userID := middleware.GetUserID(c)
	companyID, serviceID, err := serviceIDs(c)
	if err != nil {
		return errorResponse(c, http.StatusBadRequest, "INVALID_ID", "유효하지 않은 ID입니다")
	}
	updated, err := h.uc.ConnectRybbit(c.Request().Context(), companyID, serviceID, userID)
	if err != nil {
		return companyServiceErr(c, err)
	}
	return successResponse(c, http.StatusOK, updated)
}
