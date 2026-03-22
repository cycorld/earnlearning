package handler

import (
	"crypto/rand"
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/earnlearning/backend/internal/application"
	"github.com/earnlearning/backend/internal/interfaces/http/middleware"
)

type UploadHandler struct {
	uc *application.UploadUsecase
}

func NewUploadHandler(uc *application.UploadUsecase) *UploadHandler {
	return &UploadHandler{uc: uc}
}

// Upload godoc
//
//	@Summary		파일 업로드
//	@Description	이미지/문서 파일 업로드
//	@Tags			Upload
//	@Accept			multipart/form-data
//	@Produce		json
//	@Security		BearerAuth
//	@Param			file	formData	file	true	"업로드할 파일"
//	@Success		201		{object}	APIResponse
//	@Failure		400		{object}	APIResponse
//	@Router			/upload [post]
func (h *UploadHandler) Upload(c echo.Context) error {
	userID := middleware.GetUserID(c)

	file, err := c.FormFile("file")
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"success": false, "data": nil,
			"error": map[string]string{"code": "NO_FILE", "message": "파일이 첨부되지 않았습니다"},
		})
	}

	// Generate UUID prefix
	uuid := generateUUID()

	result, err := h.uc.Upload(userID, file, uuid)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"success": false, "data": nil,
			"error": map[string]string{"code": "UPLOAD_FAILED", "message": err.Error()},
		})
	}

	return c.JSON(http.StatusCreated, map[string]interface{}{
		"success": true, "data": result, "error": nil,
	})
}

// generateUUID generates a simple UUID v4 string without external dependency.
func generateUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // variant
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
