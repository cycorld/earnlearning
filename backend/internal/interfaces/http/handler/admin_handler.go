package handler

import (
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/earnlearning/backend/internal/application"
	"github.com/earnlearning/backend/internal/interfaces/http/middleware"
	"github.com/earnlearning/backend/internal/interfaces/ws"
	"github.com/labstack/echo/v4"
)

// forceReloadRateLimit — 관리자 실수로 force-reload 가 연타되어 전 사용자가
// 반복적으로 새로고침 당하지 않도록 최소 1분 간격으로 제한한다 (#027).
const forceReloadRateLimit = 60 * time.Second

type AdminHandler struct {
	authUC *application.AuthUseCase
	hub    *ws.Hub

	// force-reload rate limit: 마지막 실행 시각
	frMu   sync.Mutex
	frLast time.Time
}

// NewAdminHandler — hub 은 nil 이어도 된다 (unit test 등). nil 이면 force-reload 는 503.
func NewAdminHandler(uc *application.AuthUseCase, hub *ws.Hub) *AdminHandler {
	return &AdminHandler{authUC: uc, hub: hub}
}

// GetPendingUsers godoc
//
//	@Summary		승인 대기 사용자 목록
//	@Description	관리자용: 승인 대기 중인 사용자 목록 조회
//	@Tags			Admin
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	APIResponse
//	@Router			/admin/users/pending [get]
func (h *AdminHandler) GetPendingUsers(c echo.Context) error {
	users, err := h.authUC.AdminGetPending()
	if err != nil {
		return errorResponse(c, http.StatusInternalServerError, "INTERNAL_ERROR", "서버 오류가 발생했습니다")
	}

	viewerRole := middleware.GetUserRole(c)
	var result []userResponse
	for _, u := range users {
		result = append(result, userToResponse(u, viewerRole))
	}

	if result == nil {
		result = []userResponse{}
	}
	return successResponse(c, http.StatusOK, result)
}

// ApproveUser godoc
//
//	@Summary		사용자 승인
//	@Description	관리자용: 사용자 가입 승인
//	@Tags			Admin
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		int	true	"사용자 ID"
//	@Success		200	{object}	APIResponse
//	@Failure		404	{object}	APIResponse
//	@Router			/admin/users/{id}/approve [put]
func (h *AdminHandler) ApproveUser(c echo.Context) error {
	id, err := intParam(c, "id")
	if err != nil {
		return errorResponse(c, http.StatusBadRequest, "INVALID_ID", "유효하지 않은 ID입니다")
	}

	if err := h.authUC.AdminApprove(id); err != nil {
		return errorResponse(c, http.StatusNotFound, "NOT_FOUND", err.Error())
	}

	return successResponse(c, http.StatusOK, map[string]string{"message": "승인되었습니다"})
}

// RejectUser godoc
//
//	@Summary		사용자 거절
//	@Description	관리자용: 사용자 가입 거절
//	@Tags			Admin
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		int	true	"사용자 ID"
//	@Success		200	{object}	APIResponse
//	@Failure		404	{object}	APIResponse
//	@Router			/admin/users/{id}/reject [put]
func (h *AdminHandler) RejectUser(c echo.Context) error {
	id, err := intParam(c, "id")
	if err != nil {
		return errorResponse(c, http.StatusBadRequest, "INVALID_ID", "유효하지 않은 ID입니다")
	}

	if err := h.authUC.AdminReject(id); err != nil {
		return errorResponse(c, http.StatusNotFound, "NOT_FOUND", err.Error())
	}

	return successResponse(c, http.StatusOK, map[string]string{"message": "거절되었습니다"})
}

// ListUsers godoc
//
//	@Summary		전체 사용자 목록
//	@Description	관리자용: 전체 사용자 목록 (페이지네이션)
//	@Tags			Admin
//	@Produce		json
//	@Security		BearerAuth
//	@Param			page	query		int	false	"페이지 번호"	default(1)
//	@Param			limit	query		int	false	"페이지 크기"	default(20)
//	@Success		200		{object}	APIResponse
//	@Router			/admin/users [get]
func (h *AdminHandler) ListUsers(c echo.Context) error {
	page := intQuery(c, "page", 1)
	limit := intQuery(c, "limit", 20)

	result, err := h.authUC.AdminListUsers(page, limit)
	if err != nil {
		return errorResponse(c, http.StatusInternalServerError, "INTERNAL_ERROR", "서버 오류가 발생했습니다")
	}

	viewerRole := middleware.GetUserRole(c)
	var userList []userResponse
	for _, u := range result.Users {
		userList = append(userList, userToResponse(u, viewerRole))
	}

	if userList == nil {
		userList = []userResponse{}
	}

	return successResponse(c, http.StatusOK, map[string]interface{}{
		"users":       userList,
		"total":       result.Total,
		"total_pages": result.TotalPages,
	})
}

// ImpersonateUser godoc
//
//	@Summary		사용자 대리 로그인
//	@Description	관리자용: 특정 사용자로 대리 로그인 (디버깅용)
//	@Tags			Admin
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		int	true	"사용자 ID"
//	@Success		200	{object}	APIResponse
//	@Failure		404	{object}	APIResponse
//	@Router			/admin/users/{id}/impersonate [post]
func (h *AdminHandler) ImpersonateUser(c echo.Context) error {
	id, err := intParam(c, "id")
	if err != nil {
		return errorResponse(c, http.StatusBadRequest, "INVALID_ID", "유효하지 않은 ID입니다")
	}

	resp, err := h.authUC.ImpersonateUser(id)
	if err != nil {
		return errorResponse(c, http.StatusNotFound, "NOT_FOUND", "사용자를 찾을 수 없습니다")
	}

	return successResponse(c, http.StatusOK, resp)
}

// ForceReload godoc
//
//	@Summary		전체 접속 클라이언트 강제 새로고침 (#027)
//	@Description	관리자용: WebSocket 으로 모든 클라이언트에 force_reload 이벤트를 브로드캐스트해
//	@Description	구버전 PWA 를 새 버전으로 강제 전환시킨다. Rate limit: 1회/분.
//	@Tags			Admin
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	APIResponse
//	@Failure		429	{object}	APIResponse	"1분 간격 제한"
//	@Failure		503	{object}	APIResponse	"WS hub 미설정"
//	@Router			/admin/force-reload [post]
func (h *AdminHandler) ForceReload(c echo.Context) error {
	if h.hub == nil {
		return errorResponse(c, http.StatusServiceUnavailable, "WS_UNAVAILABLE", "WebSocket 허브가 설정되지 않았습니다")
	}

	// Rate limit — 관리자 실수 연타 방지.
	h.frMu.Lock()
	if !h.frLast.IsZero() && time.Since(h.frLast) < forceReloadRateLimit {
		remain := forceReloadRateLimit - time.Since(h.frLast)
		h.frMu.Unlock()
		return errorResponse(c, http.StatusTooManyRequests, "RATE_LIMITED",
			"force-reload는 1분에 1회만 실행할 수 있습니다. "+remain.Truncate(time.Second).String()+" 남음")
	}
	h.frLast = time.Now()
	h.frMu.Unlock()

	var body struct {
		Reason string `json:"reason"`
	}
	_ = c.Bind(&body) // body 는 선택

	actorID := middleware.GetUserID(c)
	log.Printf("admin: force-reload triggered by user %d, reason=%q", actorID, body.Reason)

	// 현재는 target=all 만 지원. 추후 user:<id> 도 고려 가능.
	h.hub.Broadcast(map[string]interface{}{
		"event": "force_reload",
		"data": map[string]interface{}{
			"reason":    body.Reason,
			"at":        time.Now().Unix(),
			"actor_id":  actorID,
		},
	})

	return successResponse(c, http.StatusOK, map[string]interface{}{
		"message": "force-reload broadcast 완료",
	})
}
