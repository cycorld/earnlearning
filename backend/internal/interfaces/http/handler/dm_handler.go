package handler

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"

	"github.com/earnlearning/backend/internal/application"
	"github.com/earnlearning/backend/internal/domain/dm"
	"github.com/earnlearning/backend/internal/interfaces/http/middleware"
)

type DMHandler struct {
	uc *application.DMUseCase
}

func NewDMHandler(uc *application.DMUseCase) *DMHandler {
	return &DMHandler{uc: uc}
}

func (h *DMHandler) SendMessage(c echo.Context) error {
	userID := middleware.GetUserID(c)
	var input application.SendDMInput
	if err := c.Bind(&input); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"success": false, "data": nil,
			"error": map[string]string{"code": "INVALID_INPUT", "message": "잘못된 입력입니다"},
		})
	}
	msg, err := h.uc.SendMessage(userID, input)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"success": false, "data": nil,
			"error": map[string]string{"code": "SEND_FAILED", "message": err.Error()},
		})
	}
	return c.JSON(http.StatusCreated, map[string]interface{}{
		"success": true, "data": msg, "error": nil,
	})
}

func (h *DMHandler) GetConversations(c echo.Context) error {
	userID := middleware.GetUserID(c)
	convs, err := h.uc.GetConversations(userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"success": false, "data": nil,
			"error": map[string]string{"code": "FETCH_FAILED", "message": err.Error()},
		})
	}
	if convs == nil {
		convs = make([]*dm.Conversation, 0)
	}
	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": true, "data": convs, "error": nil,
	})
}

func (h *DMHandler) GetMessages(c echo.Context) error {
	userID := middleware.GetUserID(c)
	peerID, err := strconv.Atoi(c.Param("userId"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"success": false, "data": nil,
			"error": map[string]string{"code": "BAD_REQUEST", "message": "잘못된 사용자 ID입니다"},
		})
	}
	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	beforeID, _ := strconv.Atoi(c.QueryParam("before_id"))

	messages, err := h.uc.GetMessages(userID, peerID, limit, beforeID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"success": false, "data": nil,
			"error": map[string]string{"code": "FETCH_FAILED", "message": err.Error()},
		})
	}
	if messages == nil {
		messages = make([]*dm.Message, 0)
	}
	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": true, "data": messages, "error": nil,
	})
}

func (h *DMHandler) MarkAsRead(c echo.Context) error {
	userID := middleware.GetUserID(c)
	peerID, err := strconv.Atoi(c.Param("userId"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"success": false, "data": nil,
			"error": map[string]string{"code": "BAD_REQUEST", "message": "잘못된 사용자 ID입니다"},
		})
	}
	if err := h.uc.MarkAsRead(userID, peerID); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"success": false, "data": nil,
			"error": map[string]string{"code": "UPDATE_FAILED", "message": err.Error()},
		})
	}
	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": true, "data": map[string]string{"message": "읽음 처리 완료"}, "error": nil,
	})
}

func (h *DMHandler) GetUnreadCount(c echo.Context) error {
	userID := middleware.GetUserID(c)
	count, err := h.uc.GetUnreadCount(userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"success": false, "data": nil,
			"error": map[string]string{"code": "FETCH_FAILED", "message": err.Error()},
		})
	}
	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": true, "data": map[string]int{"unread_count": count}, "error": nil,
	})
}
