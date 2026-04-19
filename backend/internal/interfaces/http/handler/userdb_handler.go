package handler

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/earnlearning/backend/internal/application"
	"github.com/earnlearning/backend/internal/domain/userdb"
	"github.com/earnlearning/backend/internal/interfaces/http/middleware"
)

type UserDBHandler struct {
	uc *application.UserDBUseCase
}

func NewUserDBHandler(uc *application.UserDBUseCase) *UserDBHandler {
	return &UserDBHandler{uc: uc}
}

// ListMyDatabases godoc
//
//	@Summary		내 데이터베이스 목록
//	@Description	내가 프로비저닝 한 학생용 PostgreSQL DB 목록
//	@Tags			UserDB
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	APIResponse
//	@Router			/users/me/databases [get]
func (h *UserDBHandler) ListMyDatabases(c echo.Context) error {
	userID := middleware.GetUserID(c)
	list, err := h.uc.List(userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, errorResp("INTERNAL", err.Error()))
	}
	if list == nil {
		list = []*userdb.UserDatabase{}
	}
	return c.JSON(http.StatusOK, successResp(list))
}

// CreateMyDatabase godoc
//
//	@Summary		새 데이터베이스 생성
//	@Description	학생 개인용 PostgreSQL DB 프로비저닝. 비밀번호는 응답에 1회만 포함.
//	@Tags			UserDB
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body	body		application.CreateUserDBInput	true	"프로젝트명"
//	@Success		201		{object}	APIResponse
//	@Failure		400		{object}	APIResponse
//	@Failure		409		{object}	APIResponse
//	@Router			/users/me/databases [post]
func (h *UserDBHandler) CreateMyDatabase(c echo.Context) error {
	userID := middleware.GetUserID(c)
	var input application.CreateUserDBInput
	if err := c.Bind(&input); err != nil {
		return c.JSON(http.StatusBadRequest, errorResp("BAD_REQUEST", "잘못된 요청입니다"))
	}

	out, err := h.uc.Create(userID, input)
	if err != nil {
		return userdbErrorResponse(c, err)
	}
	return c.JSON(http.StatusCreated, successResp(out))
}

// RotateMyDatabasePassword godoc
//
//	@Summary		비밀번호 재발급
//	@Description	데이터베이스 비밀번호 재발급 (기존 비번은 즉시 무효)
//	@Tags			UserDB
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		int	true	"DB ID"
//	@Success		200	{object}	APIResponse
//	@Router			/users/me/databases/{id}/rotate [post]
func (h *UserDBHandler) RotateMyDatabasePassword(c echo.Context) error {
	userID := middleware.GetUserID(c)
	dbID, err := intParam(c, "id")
	if err != nil {
		return c.JSON(http.StatusBadRequest, errorResp("BAD_REQUEST", "잘못된 ID입니다"))
	}
	out, err := h.uc.Rotate(userID, dbID)
	if err != nil {
		return userdbErrorResponse(c, err)
	}
	return c.JSON(http.StatusOK, successResp(out))
}

// DeleteMyDatabase godoc
//
//	@Summary		데이터베이스 삭제
//	@Description	학생 DB 와 PG 계정을 완전히 삭제합니다 (복구 불가)
//	@Tags			UserDB
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		int	true	"DB ID"
//	@Success		200	{object}	APIResponse
//	@Router			/users/me/databases/{id} [delete]
func (h *UserDBHandler) DeleteMyDatabase(c echo.Context) error {
	userID := middleware.GetUserID(c)
	dbID, err := intParam(c, "id")
	if err != nil {
		return c.JSON(http.StatusBadRequest, errorResp("BAD_REQUEST", "잘못된 ID입니다"))
	}
	if err := h.uc.Delete(userID, dbID); err != nil {
		return userdbErrorResponse(c, err)
	}
	return c.JSON(http.StatusOK, successResp(map[string]string{"message": "데이터베이스가 삭제되었습니다"}))
}

// AdminReconcileUserDBs — POST /api/admin/user-databases/reconcile (#016).
// SQLite 의 user_databases 행을 PG 와 대조하여 고아 행 (PG 없음) 자동 정리.
// `sudo earnlearning-db delete` 로 PG 만 지운 케이스 정리용.
func (h *UserDBHandler) AdminReconcileUserDBs(c echo.Context) error {
	res, err := h.uc.AdminReconcile()
	if err != nil {
		return userdbErrorResponse(c, err)
	}
	return c.JSON(http.StatusOK, successResp(res))
}

// AdminDeleteUserDBByName — DELETE /api/admin/user-databases/by-dbname/:db_name (#016).
// db_name 으로 PG + SQLite 양쪽 정리. PG 에 이미 없으면 SQLite 만.
func (h *UserDBHandler) AdminDeleteUserDBByName(c echo.Context) error {
	dbName := c.Param("db_name")
	if dbName == "" {
		return c.JSON(http.StatusBadRequest, errorResp("BAD_REQUEST", "db_name required"))
	}
	if err := h.uc.AdminDeleteByDBName(dbName); err != nil {
		return userdbErrorResponse(c, err)
	}
	return c.JSON(http.StatusOK, successResp(map[string]string{"db_name": dbName, "status": "deleted"}))
}

// userdbErrorResponse 는 userdb 도메인 에러를 적절한 HTTP 응답으로 매핑한다.
func userdbErrorResponse(c echo.Context, err error) error {
	switch {
	case errors.Is(err, userdb.ErrNotFound):
		return c.JSON(http.StatusNotFound, errorResp("NOT_FOUND", err.Error()))
	case errors.Is(err, userdb.ErrForbidden):
		return c.JSON(http.StatusForbidden, errorResp("FORBIDDEN", err.Error()))
	case errors.Is(err, userdb.ErrInvalidName), errors.Is(err, userdb.ErrNameTooLong):
		return c.JSON(http.StatusBadRequest, errorResp("INVALID_NAME", err.Error()))
	case errors.Is(err, userdb.ErrDuplicate):
		return c.JSON(http.StatusConflict, errorResp("DUPLICATE", err.Error()))
	case errors.Is(err, userdb.ErrSlugConflict):
		return c.JSON(http.StatusConflict, errorResp("SLUG_CONFLICT", err.Error()))
	case errors.Is(err, userdb.ErrQuotaExceeded):
		return c.JSON(http.StatusForbidden, errorResp("QUOTA_EXCEEDED", err.Error()))
	case errors.Is(err, userdb.ErrProvisionerDown):
		return c.JSON(http.StatusServiceUnavailable, errorResp("PROVISIONER_DOWN", "DB 서비스가 설정되지 않았습니다"))
	case errors.Is(err, userdb.ErrProvisionFailed):
		return c.JSON(http.StatusBadGateway, errorResp("PROVISION_FAILED", err.Error()))
	default:
		return c.JSON(http.StatusInternalServerError, errorResp("INTERNAL", err.Error()))
	}
}
