package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"

	"github.com/earnlearning/backend/internal/application"
	"github.com/earnlearning/backend/internal/domain/company"
	"github.com/earnlearning/backend/internal/interfaces/http/middleware"
)

// =============================================================================
// Shareholder proposal (주주총회) handlers — live on CompanyHandler.
// =============================================================================

// CreateProposal — POST /api/companies/:id/proposals
func (h *CompanyHandler) CreateProposal(c echo.Context) error {
	userID := middleware.GetUserID(c)
	companyID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"success": false, "data": nil,
			"error": map[string]string{"code": "INVALID_ID", "message": "유효하지 않은 ID입니다"},
		})
	}

	var input application.CreateProposalInput
	if err := c.Bind(&input); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"success": false, "data": nil,
			"error": map[string]string{"code": "INVALID_INPUT", "message": "잘못된 입력입니다"},
		})
	}

	p, err := h.uc.CreateProposal(companyID, userID, input)
	if err != nil {
		if errors.Is(err, company.ErrNotShareholder) {
			return c.JSON(http.StatusForbidden, map[string]interface{}{
				"success": false, "data": nil,
				"error": map[string]string{"code": "NOT_SHAREHOLDER", "message": err.Error()},
			})
		}
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"success": false, "data": nil,
			"error": map[string]string{"code": "CREATE_FAILED", "message": err.Error()},
		})
	}

	return c.JSON(http.StatusCreated, map[string]interface{}{
		"success": true, "data": p, "error": nil,
	})
}

// GetProposals — GET /api/companies/:id/proposals
func (h *CompanyHandler) GetProposals(c echo.Context) error {
	userID := middleware.GetUserID(c)
	companyID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"success": false, "data": nil,
			"error": map[string]string{"code": "INVALID_ID", "message": "유효하지 않은 ID입니다"},
		})
	}

	proposals, err := h.uc.GetProposalsByCompanyID(companyID, userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"success": false, "data": nil,
			"error": map[string]string{"code": "FETCH_FAILED", "message": err.Error()},
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": true, "data": proposals, "error": nil,
	})
}

// GetProposal — GET /api/proposals/:pid
func (h *CompanyHandler) GetProposal(c echo.Context) error {
	userID := middleware.GetUserID(c)
	pid, err := strconv.Atoi(c.Param("pid"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"success": false, "data": nil,
			"error": map[string]string{"code": "INVALID_ID", "message": "유효하지 않은 ID입니다"},
		})
	}

	detail, err := h.uc.GetProposal(pid, userID)
	if err != nil {
		if errors.Is(err, company.ErrProposalNotFound) {
			return c.JSON(http.StatusNotFound, map[string]interface{}{
				"success": false, "data": nil,
				"error": map[string]string{"code": "NOT_FOUND", "message": err.Error()},
			})
		}
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"success": false, "data": nil,
			"error": map[string]string{"code": "FETCH_FAILED", "message": err.Error()},
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": true, "data": detail, "error": nil,
	})
}

// CastVote — POST /api/proposals/:pid/vote
func (h *CompanyHandler) CastVote(c echo.Context) error {
	userID := middleware.GetUserID(c)
	pid, err := strconv.Atoi(c.Param("pid"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"success": false, "data": nil,
			"error": map[string]string{"code": "INVALID_ID", "message": "유효하지 않은 ID입니다"},
		})
	}

	var body struct {
		Choice string `json:"choice"`
	}
	if err := c.Bind(&body); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"success": false, "data": nil,
			"error": map[string]string{"code": "INVALID_INPUT", "message": "잘못된 입력입니다"},
		})
	}

	v, err := h.uc.CastVote(pid, userID, body.Choice)
	if err != nil {
		switch {
		case errors.Is(err, company.ErrProposalNotFound):
			return c.JSON(http.StatusNotFound, map[string]interface{}{
				"success": false, "data": nil,
				"error": map[string]string{"code": "NOT_FOUND", "message": err.Error()},
			})
		case errors.Is(err, company.ErrProposalClosed):
			return c.JSON(http.StatusConflict, map[string]interface{}{
				"success": false, "data": nil,
				"error": map[string]string{"code": "PROPOSAL_CLOSED", "message": err.Error()},
			})
		case errors.Is(err, company.ErrNotShareholder):
			return c.JSON(http.StatusForbidden, map[string]interface{}{
				"success": false, "data": nil,
				"error": map[string]string{"code": "NOT_SHAREHOLDER", "message": err.Error()},
			})
		case errors.Is(err, company.ErrAlreadyVoted):
			return c.JSON(http.StatusConflict, map[string]interface{}{
				"success": false, "data": nil,
				"error": map[string]string{"code": "ALREADY_VOTED", "message": err.Error()},
			})
		default:
			return c.JSON(http.StatusBadRequest, map[string]interface{}{
				"success": false, "data": nil,
				"error": map[string]string{"code": "VOTE_FAILED", "message": err.Error()},
			})
		}
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": true, "data": v, "error": nil,
	})
}

// CancelProposal — POST /api/proposals/:pid/cancel
func (h *CompanyHandler) CancelProposal(c echo.Context) error {
	userID := middleware.GetUserID(c)
	pid, err := strconv.Atoi(c.Param("pid"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"success": false, "data": nil,
			"error": map[string]string{"code": "INVALID_ID", "message": "유효하지 않은 ID입니다"},
		})
	}

	if err := h.uc.CancelProposal(pid, userID); err != nil {
		if errors.Is(err, company.ErrProposalNotFound) {
			return c.JSON(http.StatusNotFound, map[string]interface{}{
				"success": false, "data": nil,
				"error": map[string]string{"code": "NOT_FOUND", "message": err.Error()},
			})
		}
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"success": false, "data": nil,
			"error": map[string]string{"code": "CANCEL_FAILED", "message": err.Error()},
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": true, "data": map[string]string{"message": "안건이 취소되었습니다"}, "error": nil,
	})
}
