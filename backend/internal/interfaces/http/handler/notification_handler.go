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

func (h *NotificationHandler) MarkAllRead(c echo.Context) error {
	userID := middleware.GetUserID(c)

	if err := h.uc.MarkAllRead(userID); err != nil {
		return errorResponse(c, http.StatusInternalServerError, "INTERNAL_ERROR", "서버 오류가 발생했습니다")
	}
	return successResponse(c, http.StatusOK, map[string]string{"message": "모든 알림이 읽음 처리되었습니다"})
}

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

func (h *NotificationHandler) GetVAPIDPublicKey(c echo.Context) error {
	key := h.uc.GetVAPIDPublicKey()
	return successResponse(c, http.StatusOK, map[string]string{"vapid_public_key": key})
}

func (h *NotificationHandler) GetEmailPreference(c echo.Context) error {
	userID := middleware.GetUserID(c)
	pref, err := h.uc.GetEmailPreference(userID)
	if err != nil {
		return errorResponse(c, http.StatusInternalServerError, "INTERNAL_ERROR", "서버 오류가 발생했습니다")
	}
	return successResponse(c, http.StatusOK, pref)
}

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

func (h *NotificationHandler) AdminSendAnnouncement(c echo.Context) error {
	var input struct {
		Title   string `json:"title"`
		Body    string `json:"body"`
		UserIDs []int  `json:"user_ids"` // 비어있으면 전체 유저에게 전송
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

	sent, err := h.uc.SendAnnouncement(input.Title, input.Body, input.UserIDs)
	if err != nil {
		return errorResponse(c, http.StatusInternalServerError, "NOTIFICATION_ERROR", err.Error())
	}
	return successResponse(c, http.StatusOK, map[string]interface{}{
		"message": "공지 알림이 전송되었습니다",
		"sent":    sent,
	})
}
