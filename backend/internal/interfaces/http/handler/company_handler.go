package handler

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"

	"github.com/earnlearning/backend/internal/application"
	"github.com/earnlearning/backend/internal/domain/company"
	"github.com/earnlearning/backend/internal/interfaces/http/middleware"
)

type CompanyHandler struct {
	uc *application.CompanyUsecase
}

func NewCompanyHandler(uc *application.CompanyUsecase) *CompanyHandler {
	return &CompanyHandler{uc: uc}
}

// CreateCompany godoc
//
//	@Summary		회사 설립
//	@Description	새 회사를 설립 (자본금 필요)
//	@Tags			Company
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body	body		CreateCompanyRequest	true	"회사 설립 정보"
//	@Success		201		{object}	APIResponse
//	@Failure		400		{object}	APIResponse
//	@Router			/companies [post]
func (h *CompanyHandler) CreateCompany(c echo.Context) error {
	userID := middleware.GetUserID(c)

	var input application.CreateCompanyInput
	if err := c.Bind(&input); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"success": false, "data": nil,
			"error": map[string]string{"code": "INVALID_INPUT", "message": "잘못된 입력입니다"},
		})
	}

	result, err := h.uc.CreateCompany(userID, input)
	if err != nil {
		code := http.StatusInternalServerError
		errCode := "CREATE_FAILED"
		switch err {
		case company.ErrMinCapital:
			code = http.StatusBadRequest
			errCode = "MIN_CAPITAL"
		case company.ErrInsufficientFunds:
			code = http.StatusBadRequest
			errCode = "INSUFFICIENT_FUNDS"
		}
		return c.JSON(code, map[string]interface{}{
			"success": false, "data": nil,
			"error": map[string]string{"code": errCode, "message": err.Error()},
		})
	}

	return c.JSON(http.StatusCreated, map[string]interface{}{
		"success": true, "data": result, "error": nil,
	})
}

// GetCompany godoc
//
//	@Summary		회사 상세 조회
//	@Description	특정 회사의 상세 정보 조회
//	@Tags			Company
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		int	true	"회사 ID"
//	@Success		200	{object}	APIResponse
//	@Failure		404	{object}	APIResponse
//	@Router			/companies/{id} [get]
func (h *CompanyHandler) GetCompany(c echo.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"success": false, "data": nil,
			"error": map[string]string{"code": "INVALID_ID", "message": "유효하지 않은 ID입니다"},
		})
	}

	result, err := h.uc.GetCompany(id)
	if err != nil {
		if err == company.ErrNotFound {
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
		"success": true, "data": result, "error": nil,
	})
}

// UpdateCompany godoc
//
//	@Summary		회사 정보 수정
//	@Description	회사 설명, 로고 등 수정 (소유자만)
//	@Tags			Company
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		int					true	"회사 ID"
//	@Param			body	body		UpdateCompanyRequest	true	"수정 정보"
//	@Success		200		{object}	APIResponse
//	@Failure		403		{object}	APIResponse
//	@Router			/companies/{id} [put]
func (h *CompanyHandler) UpdateCompany(c echo.Context) error {
	userID := middleware.GetUserID(c)
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"success": false, "data": nil,
			"error": map[string]string{"code": "INVALID_ID", "message": "유효하지 않은 ID입니다"},
		})
	}

	var input struct {
		Description string `json:"description"`
		LogoURL     string `json:"logo_url"`
	}
	if err := c.Bind(&input); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"success": false, "data": nil,
			"error": map[string]string{"code": "INVALID_INPUT", "message": "잘못된 입력입니다"},
		})
	}

	if err := h.uc.UpdateCompany(id, userID, input.Description, input.LogoURL); err != nil {
		if err == company.ErrNotOwner {
			return c.JSON(http.StatusForbidden, map[string]interface{}{
				"success": false, "data": nil,
				"error": map[string]string{"code": "NOT_OWNER", "message": err.Error()},
			})
		}
		if err == company.ErrNotFound {
			return c.JSON(http.StatusNotFound, map[string]interface{}{
				"success": false, "data": nil,
				"error": map[string]string{"code": "NOT_FOUND", "message": err.Error()},
			})
		}
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"success": false, "data": nil,
			"error": map[string]string{"code": "UPDATE_FAILED", "message": err.Error()},
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": true, "data": map[string]string{"message": "회사 정보가 수정되었습니다"}, "error": nil,
	})
}

// GetMyCompanies godoc
//
//	@Summary		내 회사 목록
//	@Description	내가 소유한 회사 목록 조회
//	@Tags			Company
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	APIResponse
//	@Router			/companies/mine [get]
func (h *CompanyHandler) GetMyCompanies(c echo.Context) error {
	userID := middleware.GetUserID(c)

	result, err := h.uc.GetMyCompanies(userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"success": false, "data": nil,
			"error": map[string]string{"code": "FETCH_FAILED", "message": err.Error()},
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": true, "data": result, "error": nil,
	})
}

// ListAllCompanies godoc
//
//	@Summary		전체 회사 목록
//	@Description	관리자용: 모든 회사 목록 조회
//	@Tags			Admin
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	APIResponse
//	@Router			/admin/companies [get]
func (h *CompanyHandler) ListAllCompanies(c echo.Context) error {
	result, err := h.uc.GetAllCompanies()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"success": false, "data": nil,
			"error": map[string]string{"code": "FETCH_FAILED", "message": err.Error()},
		})
	}

	if result == nil {
		result = []*company.Company{}
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": true, "data": result, "error": nil,
	})
}

// CreateBusinessCard godoc
//
//	@Summary		명함 생성/수정
//	@Description	회사 명함 생성 또는 수정
//	@Tags			Company
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id		path		int					true	"회사 ID"
//	@Param			body	body		BusinessCardRequest	true	"명함 정보"
//	@Success		200		{object}	APIResponse
//	@Failure		403		{object}	APIResponse
//	@Router			/companies/{id}/business-card [post]
func (h *CompanyHandler) CreateBusinessCard(c echo.Context) error {
	userID := middleware.GetUserID(c)
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"success": false, "data": nil,
			"error": map[string]string{"code": "INVALID_ID", "message": "유효하지 않은 ID입니다"},
		})
	}

	var card company.BusinessCard
	if err := c.Bind(&card); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"success": false, "data": nil,
			"error": map[string]string{"code": "INVALID_INPUT", "message": "잘못된 입력입니다"},
		})
	}

	if err := h.uc.CreateBusinessCard(id, userID, card); err != nil {
		if err == company.ErrNotOwner {
			return c.JSON(http.StatusForbidden, map[string]interface{}{
				"success": false, "data": nil,
				"error": map[string]string{"code": "NOT_OWNER", "message": err.Error()},
			})
		}
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"success": false, "data": nil,
			"error": map[string]string{"code": "CREATE_FAILED", "message": err.Error()},
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": true, "data": map[string]string{"message": "명함이 저장되었습니다"}, "error": nil,
	})
}

// GetBusinessCard godoc
//
//	@Summary		명함 조회
//	@Description	회사 명함 조회
//	@Tags			Company
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		int	true	"회사 ID"
//	@Success		200	{object}	APIResponse
//	@Failure		404	{object}	APIResponse
//	@Router			/companies/{id}/business-card [get]
func (h *CompanyHandler) GetBusinessCard(c echo.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"success": false, "data": nil,
			"error": map[string]string{"code": "INVALID_ID", "message": "유효하지 않은 ID입니다"},
		})
	}

	result, err := h.uc.GetBusinessCard(id)
	if err != nil {
		if err == company.ErrNotFound {
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
		"success": true, "data": result, "error": nil,
	})
}

func (h *CompanyHandler) DownloadBusinessCard(c echo.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"success": false, "data": nil,
			"error": map[string]string{"code": "INVALID_ID", "message": "유효하지 않은 ID입니다"},
		})
	}

	content, filename, err := h.uc.DownloadBusinessCard(id)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"success": false, "data": nil,
			"error": map[string]string{"code": "DOWNLOAD_FAILED", "message": err.Error()},
		})
	}

	c.Response().Header().Set("Content-Disposition", "attachment; filename=\""+filename+"\"")
	return c.Blob(http.StatusOK, "text/plain; charset=utf-8", content)
}
