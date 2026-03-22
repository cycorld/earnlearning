package handler

import (
	"net/http"

	"github.com/earnlearning/backend/internal/application"
	"github.com/earnlearning/backend/internal/interfaces/http/middleware"
	"github.com/labstack/echo/v4"
)

type NotificationHandler struct {
	uc *application.NotificationUseCase
}

func NewNotificationHandler(uc *application.NotificationUseCase) *NotificationHandler {
	return &NotificationHandler{uc: uc}
}

// GetNotifications godoc
//
//	@Summary		알림 목록 조회
//	@Description	내 알림 목록 조회 (읽음/안읽음 필터)
//	@Tags			Notification
//	@Produce		json
//	@Security		BearerAuth
//	@Param			page	query		int		false	"페이지"	default(1)
//	@Param			limit	query		int		false	"크기"	default(20)
//	@Param			is_read	query		string	false	"읽음 여부 (true/false)"
//	@Success		200		{object}	APIResponse
//	@Router			/notifications [get]
func (h *NotificationHandler) GetNotifications(c echo.Context) error {
	userID := middleware.GetUserID(c)
	page := intQuery(c, "page", 1)
	limit := intQuery(c, "limit", 20)

	var isRead *bool
	if v := c.QueryParam("is_read"); v != "" {
		b := v == "true" || v == "1"
		isRead = &b
	}

	result, err := h.uc.GetNotifications(userID, isRead, page, limit)
	if err != nil {
		return errorResponse(c, http.StatusInternalServerError, "INTERNAL_ERROR", "서버 오류가 발생했습니다")
	}
	return successResponse(c, http.StatusOK, result)
}

// MarkRead godoc
//
//	@Summary		알림 읽음 처리
//	@Description	특정 알림 읽음 처리
//	@Tags			Notification
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		int	true	"알림 ID"
//	@Success		200	{object}	APIResponse
//	@Router			/notifications/{id}/read [put]
func (h *NotificationHandler) MarkRead(c echo.Context) error {
	userID := middleware.GetUserID(c)
	notifID, err := intParam(c, "id")
	if err != nil {
		return errorResponse(c, http.StatusBadRequest, "INVALID_ID", "유효하지 않은 알림 ID입니다")
	}

	if err := h.uc.MarkRead(notifID, userID); err != nil {
		return errorResponse(c, http.StatusBadRequest, "NOTIFICATION_ERROR", err.Error())
	}
	return successResponse(c, http.StatusOK, map[string]string{"message": "읽음 처리되었습니다"})
}

// MarkAllRead godoc
//
//	@Summary		전체 알림 읽음 처리
//	@Description	모든 알림 일괄 읽음 처리
//	@Tags			Notification
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	APIResponse
//	@Router			/notifications/read-all [put]
func (h *NotificationHandler) MarkAllRead(c echo.Context) error {
	userID := middleware.GetUserID(c)

	if err := h.uc.MarkAllRead(userID); err != nil {
		return errorResponse(c, http.StatusInternalServerError, "INTERNAL_ERROR", "서버 오류가 발생했습니다")
	}
	return successResponse(c, http.StatusOK, map[string]string{"message": "모든 알림이 읽음 처리되었습니다"})
}

// SubscribePush godoc
//
//	@Summary		푸시 구독
//	@Description	웹 푸시 알림 구독 등록
//	@Tags			Notification
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body	body		SubscribePushRequest	true	"구독 정보"
//	@Success		200		{object}	APIResponse
//	@Router			/notifications/push/subscribe [post]
func (h *NotificationHandler) SubscribePush(c echo.Context) error {
	userID := middleware.GetUserID(c)

	var input application.SubscribePushInput
	if err := c.Bind(&input); err != nil {
		return errorResponse(c, http.StatusBadRequest, "INVALID_INPUT", "입력값이 올바르지 않습니다")
	}

	if err := h.uc.SubscribePush(userID, input); err != nil {
		return errorResponse(c, http.StatusInternalServerError, "PUSH_ERROR", err.Error())
	}
	return successResponse(c, http.StatusOK, map[string]string{"message": "푸시 구독이 등록되었습니다"})
}

// UnsubscribePush godoc
//
//	@Summary		푸시 구독 해제
//	@Description	웹 푸시 알림 구독 해제
//	@Tags			Notification
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body	body		UnsubscribePushRequest	true	"해제 정보"
//	@Success		200		{object}	APIResponse
//	@Router			/notifications/push/subscribe [delete]
func (h *NotificationHandler) UnsubscribePush(c echo.Context) error {
	userID := middleware.GetUserID(c)

	var input application.UnsubscribePushInput
	if err := c.Bind(&input); err != nil {
		return errorResponse(c, http.StatusBadRequest, "INVALID_INPUT", "입력값이 올바르지 않습니다")
	}

	if err := h.uc.UnsubscribePush(userID, input); err != nil {
		return errorResponse(c, http.StatusInternalServerError, "PUSH_ERROR", err.Error())
	}
	return successResponse(c, http.StatusOK, map[string]string{"message": "푸시 구독이 해제되었습니다"})
}

// GetVAPIDPublicKey godoc
//
//	@Summary		VAPID 공개키 조회
//	@Description	웹 푸시용 VAPID 공개키 조회
//	@Tags			Notification
//	@Produce		json
//	@Success		200	{object}	APIResponse
//	@Router			/push/vapid-public-key [get]
func (h *NotificationHandler) GetVAPIDPublicKey(c echo.Context) error {
	key := h.uc.GetVAPIDPublicKey()
	return successResponse(c, http.StatusOK, map[string]string{"vapid_public_key": key})
}

// GetEmailPreference godoc
//
//	@Summary		이메일 알림 설정 조회
//	@Description	이메일 알림 수신 설정 조회
//	@Tags			Notification
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	APIResponse
//	@Router			/notifications/email/preference [get]
func (h *NotificationHandler) GetEmailPreference(c echo.Context) error {
	userID := middleware.GetUserID(c)
	pref, err := h.uc.GetEmailPreference(userID)
	if err != nil {
		return errorResponse(c, http.StatusInternalServerError, "INTERNAL_ERROR", "서버 오류가 발생했습니다")
	}
	return successResponse(c, http.StatusOK, pref)
}

// UpdateEmailPreference godoc
//
//	@Summary		이메일 알림 설정 변경
//	@Description	이메일 알림 수신 여부 변경
//	@Tags			Notification
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body	body		UpdateEmailPrefRequest	true	"설정"
//	@Success		200		{object}	APIResponse
//	@Router			/notifications/email/preference [put]
func (h *NotificationHandler) UpdateEmailPreference(c echo.Context) error {
	userID := middleware.GetUserID(c)
	var input struct {
		EmailEnabled bool `json:"email_enabled"`
	}
	if err := c.Bind(&input); err != nil {
		return errorResponse(c, http.StatusBadRequest, "INVALID_INPUT", "입력값이 올바르지 않습니다")
	}

	if err := h.uc.UpdateEmailPreference(userID, input.EmailEnabled); err != nil {
		return errorResponse(c, http.StatusInternalServerError, "INTERNAL_ERROR", "서버 오류가 발생했습니다")
	}
	return successResponse(c, http.StatusOK, map[string]string{"message": "이메일 알림 설정이 변경되었습니다"})
}

// AdminSendAnnouncement godoc
//
//	@Summary		공지 알림 전송
//	@Description	관리자용: 공지 알림 전송 (전체 또는 특정 사용자)
//	@Tags			Admin
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body	body		AnnouncementRequest	true	"공지 정보"
//	@Success		200		{object}	APIResponse
//	@Router			/admin/notifications/announce [post]
func (h *NotificationHandler) AdminSendAnnouncement(c echo.Context) error {
	var input struct {
		Title      string `json:"title"`
		Body       string `json:"body"`
		UserIDs    []int  `json:"user_ids"`     // 비어있으면 전체 유저에게 전송
		SendNotify *bool  `json:"send_notify"`  // true면 푸시+이메일도 전송 (기본: true)
	}
	if err := c.Bind(&input); err != nil {
		return errorResponse(c, http.StatusBadRequest, "INVALID_INPUT", "입력값이 올바르지 않습니다")
	}
	if input.Title == "" {
		return errorResponse(c, http.StatusBadRequest, "INVALID_INPUT", "제목은 필수입니다")
	}
	if input.Body == "" {
		return errorResponse(c, http.StatusBadRequest, "INVALID_INPUT", "내용은 필수입니다")
	}

	sendNotify := true
	if input.SendNotify != nil {
		sendNotify = *input.SendNotify
	}

	sent, err := h.uc.SendAnnouncement(input.Title, input.Body, input.UserIDs, sendNotify)
	if err != nil {
		return errorResponse(c, http.StatusInternalServerError, "NOTIFICATION_ERROR", err.Error())
	}
	return successResponse(c, http.StatusOK, map[string]interface{}{
		"message": "공지 알림이 전송되었습니다",
		"sent":    sent,
	})
}
