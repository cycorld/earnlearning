package handler

import (
	"crypto/subtle"
	"net/http"
	"time"

	"github.com/earnlearning/backend/internal/application"
	"github.com/earnlearning/backend/internal/domain/mail"
	"github.com/earnlearning/backend/internal/interfaces/http/middleware"
	"github.com/labstack/echo/v4"
)

// MailHandler — #166 학생 메일함.
type MailHandler struct {
	uc            *application.MailUseCase
	webhookSecret string // 비면 inbound webhook 비활성(503)
}

func NewMailHandler(uc *application.MailUseCase, webhookSecret string) *MailHandler {
	return &MailHandler{uc: uc, webhookSecret: webhookSecret}
}

// mailErr — 도메인 에러 → HTTP 상태 매핑.
func mailErr(c echo.Context, err error) error {
	switch err {
	case mail.ErrInvalidLocalPart, mail.ErrReserved:
		return errorResponse(c, http.StatusBadRequest, "INVALID_LOCAL_PART", err.Error())
	case mail.ErrAlreadyClaimed, mail.ErrAddressTaken:
		return errorResponse(c, http.StatusConflict, "CONFLICT", err.Error())
	case mail.ErrNoAddress:
		return errorResponse(c, http.StatusBadRequest, "NO_ADDRESS", err.Error())
	case mail.ErrForbidden:
		return errorResponse(c, http.StatusForbidden, "FORBIDDEN", err.Error())
	case mail.ErrNotFound:
		return errorResponse(c, http.StatusNotFound, "NOT_FOUND", err.Error())
	case mail.ErrSendFailed:
		return errorResponse(c, http.StatusBadGateway, "SEND_FAILED", err.Error())
	case mail.ErrAttachmentsTooLarge:
		return errorResponse(c, http.StatusRequestEntityTooLarge, "TOO_LARGE", err.Error())
	default:
		return errorResponse(c, http.StatusInternalServerError, "INTERNAL", "서버 오류가 발생했습니다")
	}
}

func addressPayload(a *mail.Address) map[string]interface{} {
	if a == nil {
		return map[string]interface{}{"local_part": nil, "email": nil}
	}
	return map[string]interface{}{
		"local_part": a.LocalPart,
		"email":      mail.EmailFor(a.LocalPart),
	}
}

// GetAddress — GET /mail/address
func (h *MailHandler) GetAddress(c echo.Context) error {
	userID := middleware.GetUserID(c)
	addr, err := h.uc.GetAddress(userID)
	if err != nil {
		return mailErr(c, err)
	}
	return successResponse(c, http.StatusOK, addressPayload(addr))
}

// ClaimAddress — POST /mail/address
func (h *MailHandler) ClaimAddress(c echo.Context) error {
	var body struct {
		LocalPart string `json:"local_part"`
	}
	if err := c.Bind(&body); err != nil {
		return errorResponse(c, http.StatusBadRequest, "INVALID_INPUT", "잘못된 입력입니다")
	}
	userID := middleware.GetUserID(c)
	addr, err := h.uc.ClaimAddress(userID, body.LocalPart)
	if err != nil {
		return mailErr(c, err)
	}
	return successResponse(c, http.StatusCreated, addressPayload(addr))
}

// ListBox — GET /mail?box=inbox|sent&limit&offset
func (h *MailHandler) ListBox(c echo.Context) error {
	userID := middleware.GetUserID(c)
	box := c.QueryParam("box")
	limit, offset := paginate(c)
	items, total, err := h.uc.ListBox(userID, box, limit, offset)
	if err != nil {
		return mailErr(c, err)
	}
	if items == nil {
		items = []*mail.EmailListItem{}
	}
	return successResponse(c, http.StatusOK, map[string]interface{}{"emails": items, "total": total})
}

// mailAttachmentResp — 상세 응답용 첨부 (경로 미노출).
type mailAttachmentResp struct {
	ID       int    `json:"id"`
	Filename string `json:"filename"`
	Mime     string `json:"mime"`
	Size     int64  `json:"size"`
}

// mailDetailResp — GET /mail/:id 응답.
type mailDetailResp struct {
	ID          int                  `json:"id"`
	Direction   string               `json:"direction"`
	FromAddr    string               `json:"from_addr"`
	ToAddr      string               `json:"to_addr"`
	Subject     string               `json:"subject"`
	BodyText    string               `json:"body_text"`
	BodyHTML    string               `json:"body_html"`
	MessageID   string               `json:"message_id"`
	InReplyTo   string               `json:"in_reply_to"`
	Refs        string               `json:"refs"`
	Read        bool                 `json:"read"`
	CreatedAt   time.Time            `json:"created_at"`
	Attachments []mailAttachmentResp `json:"attachments"`
}

// GetEmail — GET /mail/:id
func (h *MailHandler) GetEmail(c echo.Context) error {
	id, err := intParam(c, "id")
	if err != nil {
		return errorResponse(c, http.StatusBadRequest, "INVALID_ID", "유효하지 않은 ID입니다")
	}
	userID := middleware.GetUserID(c)
	isAdmin := middleware.GetUserRole(c) == "admin"
	e, atts, err := h.uc.GetEmail(id, userID, isAdmin)
	if err != nil {
		return mailErr(c, err)
	}
	resp := mailDetailResp{
		ID: e.ID, Direction: e.Direction, FromAddr: e.FromAddr, ToAddr: e.ToAddr,
		Subject: e.Subject, BodyText: e.BodyText, BodyHTML: e.BodyHTML,
		MessageID: e.MessageID, InReplyTo: e.InReplyTo, Refs: e.Refs,
		Read: e.Read, CreatedAt: e.CreatedAt,
		Attachments: []mailAttachmentResp{},
	}
	for _, a := range atts {
		resp.Attachments = append(resp.Attachments, mailAttachmentResp{
			ID: a.ID, Filename: a.Filename, Mime: a.Mime, Size: a.Size,
		})
	}
	return successResponse(c, http.StatusOK, resp)
}

// DownloadAttachment — GET /mail/attachments/:id
func (h *MailHandler) DownloadAttachment(c echo.Context) error {
	id, err := intParam(c, "id")
	if err != nil {
		return errorResponse(c, http.StatusBadRequest, "INVALID_ID", "유효하지 않은 ID입니다")
	}
	userID := middleware.GetUserID(c)
	isAdmin := middleware.GetUserRole(c) == "admin"
	a, err := h.uc.GetAttachmentForAccess(id, userID, isAdmin)
	if err != nil {
		return mailErr(c, err)
	}
	return c.Attachment(a.StoredPath, a.Filename)
}

// Send — POST /mail/send
func (h *MailHandler) Send(c echo.Context) error {
	var body struct {
		To          string `json:"to"`
		Subject     string `json:"subject"`
		BodyText    string `json:"body_text"`
		InReplyToID *int   `json:"in_reply_to_id"`
	}
	if err := c.Bind(&body); err != nil {
		return errorResponse(c, http.StatusBadRequest, "INVALID_INPUT", "잘못된 입력입니다")
	}
	userID := middleware.GetUserID(c)
	id, err := h.uc.Send(userID, application.SendInput{
		To:          body.To,
		Subject:     body.Subject,
		BodyText:    body.BodyText,
		InReplyToID: body.InReplyToID,
	})
	if err != nil {
		return mailErr(c, err)
	}
	return successResponse(c, http.StatusCreated, map[string]interface{}{"id": id})
}

// Inbound — POST /mail/inbound (JWT 없음, X-Mail-Webhook-Secret 헤더 인증).
// 봉투(envelope) 없이 worker 계약에 맞는 raw JSON 을 반환한다.
func (h *MailHandler) Inbound(c echo.Context) error {
	if h.webhookSecret == "" {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{"error": "mail inbound disabled"})
	}
	got := c.Request().Header.Get("X-Mail-Webhook-Secret")
	if subtle.ConstantTimeCompare([]byte(got), []byte(h.webhookSecret)) != 1 {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	var in application.InboundInput
	if err := c.Bind(&in); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid payload"})
	}
	id, err := h.uc.ReceiveInbound(in)
	if err != nil {
		switch err {
		case mail.ErrNotFound:
			return c.JSON(http.StatusNotFound, map[string]string{"error": "unknown recipient"})
		case mail.ErrAttachmentsTooLarge:
			return c.JSON(http.StatusRequestEntityTooLarge, map[string]string{"error": "attachments too large"})
		default:
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal error"})
		}
	}
	return c.JSON(http.StatusCreated, map[string]int{"id": id})
}

// AdminListMail — GET /admin/mail?limit&offset
func (h *MailHandler) AdminListMail(c echo.Context) error {
	limit, offset := paginate(c)
	items, total, err := h.uc.ListAll(limit, offset)
	if err != nil {
		return mailErr(c, err)
	}
	if items == nil {
		items = []*mail.EmailListItem{}
	}
	return successResponse(c, http.StatusOK, map[string]interface{}{"emails": items, "total": total})
}

// paginate — limit(1..100, 기본 20) / offset(>=0) 파싱.
func paginate(c echo.Context) (int, int) {
	limit := intQuery(c, "limit", 20)
	if limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	offset := intQuery(c, "offset", 0)
	if offset < 0 {
		offset = 0
	}
	return limit, offset
}
