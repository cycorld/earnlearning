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
	case mail.ErrAlreadyClaimed, mail.ErrAddressTaken, mail.ErrAlreadyApproved:
		return errorResponse(c, http.StatusConflict, "CONFLICT", err.Error())
	case mail.ErrNoAddress:
		return errorResponse(c, http.StatusBadRequest, "NO_ADDRESS", err.Error())
	case mail.ErrNotApproved:
		return errorResponse(c, http.StatusForbidden, "NOT_APPROVED", err.Error())
	case mail.ErrForbidden:
		return errorResponse(c, http.StatusForbidden, "FORBIDDEN", err.Error())
	case mail.ErrNotFound:
		return errorResponse(c, http.StatusNotFound, "NOT_FOUND", err.Error())
	case mail.ErrSendFailed:
		return errorResponse(c, http.StatusBadGateway, "SEND_FAILED", err.Error())
	case mail.ErrSendDisabled:
		return errorResponse(c, http.StatusServiceUnavailable, "SEND_DISABLED", err.Error())
	case mail.ErrAttachmentsTooLarge:
		return errorResponse(c, http.StatusRequestEntityTooLarge, "TOO_LARGE", err.Error())
	default:
		return errorResponse(c, http.StatusInternalServerError, "INTERNAL", "서버 오류가 발생했습니다")
	}
}

func addressPayload(a *mail.Address) map[string]interface{} {
	if a == nil {
		return map[string]interface{}{"local_part": nil, "email": nil, "status": nil}
	}
	return map[string]interface{}{
		"local_part": a.LocalPart,
		"email":      mail.EmailFor(a.LocalPart),
		"status":     a.Status,
	}
}

// GetAddress — GET /mail/address (개인 주소 하위호환).
func (h *MailHandler) GetAddress(c echo.Context) error {
	userID := middleware.GetUserID(c)
	addr, err := h.uc.GetAddress(userID)
	if err != nil {
		return mailErr(c, err)
	}
	return successResponse(c, http.StatusOK, addressPayload(addr))
}

// GetMailboxes — GET /mail/mailboxes (개인 + 소유 회사 + 권한 있는 공용).
func (h *MailHandler) GetMailboxes(c echo.Context) error {
	userID := middleware.GetUserID(c)
	items, err := h.uc.ListMailboxes(userID)
	if err != nil {
		return mailErr(c, err)
	}
	return successResponse(c, http.StatusOK, map[string]interface{}{"mailboxes": items})
}

// ClaimAddress — POST /mail/address (개인).
func (h *MailHandler) ClaimAddress(c echo.Context) error {
	var body struct {
		LocalPart string `json:"local_part"`
	}
	if err := c.Bind(&body); err != nil {
		return errorResponse(c, http.StatusBadRequest, "INVALID_INPUT", "잘못된 입력입니다")
	}
	userID := middleware.GetUserID(c)
	addr, err := h.uc.ClaimPersonalAddress(userID, body.LocalPart)
	if err != nil {
		return mailErr(c, err)
	}
	return successResponse(c, http.StatusCreated, addressPayload(addr))
}

// ClaimCompanyAddress — POST /companies/:id/mail-address (회사 소유주만).
func (h *MailHandler) ClaimCompanyAddress(c echo.Context) error {
	companyID, err := intParam(c, "id")
	if err != nil {
		return errorResponse(c, http.StatusBadRequest, "INVALID_ID", "유효하지 않은 ID입니다")
	}
	var body struct {
		LocalPart string `json:"local_part"`
	}
	if err := c.Bind(&body); err != nil {
		return errorResponse(c, http.StatusBadRequest, "INVALID_INPUT", "잘못된 입력입니다")
	}
	userID := middleware.GetUserID(c)
	addr, err := h.uc.ClaimCompanyAddress(companyID, userID, body.LocalPart)
	if err != nil {
		return mailErr(c, err)
	}
	return successResponse(c, http.StatusCreated, addressPayload(addr))
}

// ListBox — GET /mail?box=inbox|sent&address_id=N&limit&offset. address_id 필수.
func (h *MailHandler) ListBox(c echo.Context) error {
	userID := middleware.GetUserID(c)
	isAdmin := middleware.GetUserRole(c) == "admin"
	box := c.QueryParam("box")
	addressID := intQuery(c, "address_id", 0)
	if addressID <= 0 {
		return errorResponse(c, http.StatusBadRequest, "MISSING_ADDRESS_ID", "address_id 가 필요합니다")
	}
	limit, offset := paginate(c)
	items, total, err := h.uc.ListBox(userID, addressID, box, isAdmin, limit, offset)
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

// Send — POST /mail/send {address_id,to,subject,body_text,in_reply_to_id}
func (h *MailHandler) Send(c echo.Context) error {
	var body struct {
		AddressID   int    `json:"address_id"`
		To          string `json:"to"`
		Subject     string `json:"subject"`
		BodyText    string `json:"body_text"`
		InReplyToID *int   `json:"in_reply_to_id"`
	}
	if err := c.Bind(&body); err != nil {
		return errorResponse(c, http.StatusBadRequest, "INVALID_INPUT", "잘못된 입력입니다")
	}
	if body.AddressID <= 0 {
		return errorResponse(c, http.StatusBadRequest, "MISSING_ADDRESS_ID", "address_id 가 필요합니다")
	}
	userID := middleware.GetUserID(c)
	id, err := h.uc.Send(userID, body.AddressID, application.SendInput{
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

// AdminListAddresses — GET /admin/mail/addresses?status=pending (기본 pending, status=all 이면 전체).
func (h *MailHandler) AdminListAddresses(c echo.Context) error {
	status := c.QueryParam("status")
	if status == "" {
		status = mail.StatusPending
	}
	if status == "all" {
		status = ""
	}
	items, err := h.uc.ListAddressesAdmin(status)
	if err != nil {
		return mailErr(c, err)
	}
	return successResponse(c, http.StatusOK, items)
}

// AdminApproveAddress — POST /admin/mail/addresses/:id/approve
func (h *MailHandler) AdminApproveAddress(c echo.Context) error {
	id, err := intParam(c, "id")
	if err != nil {
		return errorResponse(c, http.StatusBadRequest, "INVALID_ID", "유효하지 않은 ID입니다")
	}
	addr, err := h.uc.ApproveAddress(id)
	if err != nil {
		return mailErr(c, err)
	}
	return successResponse(c, http.StatusOK, adminAddressResult(addr))
}

// AdminRejectAddress — POST /admin/mail/addresses/:id/reject
func (h *MailHandler) AdminRejectAddress(c echo.Context) error {
	id, err := intParam(c, "id")
	if err != nil {
		return errorResponse(c, http.StatusBadRequest, "INVALID_ID", "유효하지 않은 ID입니다")
	}
	addr, err := h.uc.RejectAddress(id)
	if err != nil {
		return mailErr(c, err)
	}
	return successResponse(c, http.StatusOK, adminAddressResult(addr))
}

func adminAddressResult(a *mail.Address) map[string]interface{} {
	return map[string]interface{}{
		"id":         a.ID,
		"local_part": a.LocalPart,
		"email":      mail.EmailFor(a.LocalPart),
		"status":     a.Status,
	}
}

// AdminCreateShared — POST /admin/mail/shared {local_part,display_name}
func (h *MailHandler) AdminCreateShared(c echo.Context) error {
	var body struct {
		LocalPart   string `json:"local_part"`
		DisplayName string `json:"display_name"`
	}
	if err := c.Bind(&body); err != nil {
		return errorResponse(c, http.StatusBadRequest, "INVALID_INPUT", "잘못된 입력입니다")
	}
	adminID := middleware.GetUserID(c)
	addr, err := h.uc.CreateSharedAddress(adminID, body.LocalPart, body.DisplayName)
	if err != nil {
		return mailErr(c, err)
	}
	return successResponse(c, http.StatusCreated, map[string]interface{}{
		"address_id":   addr.ID,
		"local_part":   addr.LocalPart,
		"display_name": addr.DisplayName,
		"email":        mail.EmailFor(addr.LocalPart),
		"status":       addr.Status,
	})
}

// AdminListShared — GET /admin/mail/shared → [{address_id,local_part,display_name,email,grants:[...]}]
func (h *MailHandler) AdminListShared(c echo.Context) error {
	items, err := h.uc.ListSharedAddresses()
	if err != nil {
		return mailErr(c, err)
	}
	return successResponse(c, http.StatusOK, items)
}

// AdminGrantShared — POST /admin/mail/shared/:addressId/grants {user_id}
func (h *MailHandler) AdminGrantShared(c echo.Context) error {
	addressID, err := intParam(c, "addressId")
	if err != nil {
		return errorResponse(c, http.StatusBadRequest, "INVALID_ID", "유효하지 않은 ID입니다")
	}
	var body struct {
		UserID int `json:"user_id"`
	}
	if err := c.Bind(&body); err != nil || body.UserID <= 0 {
		return errorResponse(c, http.StatusBadRequest, "INVALID_INPUT", "user_id 가 필요합니다")
	}
	if err := h.uc.GrantSharedAccess(addressID, body.UserID); err != nil {
		return mailErr(c, err)
	}
	return successResponse(c, http.StatusOK, map[string]interface{}{"granted": true})
}

// AdminRevokeShared — POST /admin/mail/shared/:addressId/grants/:userId/revoke
func (h *MailHandler) AdminRevokeShared(c echo.Context) error {
	addressID, err := intParam(c, "addressId")
	if err != nil {
		return errorResponse(c, http.StatusBadRequest, "INVALID_ID", "유효하지 않은 ID입니다")
	}
	userID, err := intParam(c, "userId")
	if err != nil {
		return errorResponse(c, http.StatusBadRequest, "INVALID_ID", "유효하지 않은 ID입니다")
	}
	if err := h.uc.RevokeSharedAccess(addressID, userID); err != nil {
		return mailErr(c, err)
	}
	return successResponse(c, http.StatusOK, map[string]interface{}{"revoked": true})
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
