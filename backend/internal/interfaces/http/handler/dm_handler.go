package handler

import (
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"

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
	var files []*multipart.FileHeader

	// #184 multipart 첨부 전송. SendDMInput 은 json 태그만 있어 c.Bind 로는 form 값을 못 읽는다.
	if strings.HasPrefix(c.Request().Header.Get(echo.HeaderContentType), "multipart/form-data") {
		receiverID, err := strconv.Atoi(c.FormValue("receiver_id"))
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"success": false, "data": nil,
				"error": map[string]string{"code": "INVALID_INPUT", "message": "잘못된 입력입니다"},
			})
		}
		input.ReceiverID = receiverID
		input.Content = c.FormValue("content")
		form, err := c.MultipartForm()
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"success": false, "data": nil,
				"error": map[string]string{"code": "INVALID_INPUT", "message": "첨부를 읽을 수 없습니다"},
			})
		}
		files = form.File["files"]
	} else if err := c.Bind(&input); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"success": false, "data": nil,
			"error": map[string]string{"code": "INVALID_INPUT", "message": "잘못된 입력입니다"},
		})
	}

	msg, err := h.uc.SendMessageWithAttachments(userID, input, files, generateUUID)
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

// DownloadAttachment — DM 첨부 다운로드 (#184). 발신자/수신자만, 관리자 우회 없음.
func (h *DMHandler) DownloadAttachment(c echo.Context) error {
	userID := middleware.GetUserID(c)
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"success": false, "data": nil,
			"error": map[string]string{"code": "BAD_REQUEST", "message": "잘못된 첨부 ID입니다"},
		})
	}
	att, err := h.uc.GetAttachmentForAccess(id, userID)
	if err != nil {
		switch err {
		case dm.ErrForbidden:
			return c.JSON(http.StatusForbidden, map[string]interface{}{
				"success": false, "data": nil,
				"error": map[string]string{"code": "FORBIDDEN", "message": err.Error()},
			})
		case dm.ErrAttachmentNotFound, dm.ErrMessageNotFound:
			return c.JSON(http.StatusNotFound, map[string]interface{}{
				"success": false, "data": nil,
				"error": map[string]string{"code": "NOT_FOUND", "message": err.Error()},
			})
		}
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"success": false, "data": nil,
			"error": map[string]string{"code": "INTERNAL", "message": err.Error()},
		})
	}

	safeName := application.SanitizeDownloadFilename(att.Filename, att.StoredName)
	header := c.Response().Header()
	header.Set("X-Content-Type-Options", "nosniff")

	// 검증된 이미지만 inline. MIME 은 DB(클라이언트 입력) 대신 서버가 확장자로 판단한다.
	if mime := application.DMAttachmentInlineMIME(att.StoredName); mime != "" {
		header.Set(echo.HeaderContentType, mime)
		return c.Inline(att.Path, safeName)
	}
	header.Set(echo.HeaderContentType, "application/octet-stream")
	header.Set("Content-Security-Policy", "default-src 'none'; sandbox")
	return c.Attachment(att.Path, safeName)
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
