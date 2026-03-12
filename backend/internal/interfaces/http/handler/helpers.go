package handler

import (
	"strconv"

	"github.com/labstack/echo/v4"
)

func successResponse(c echo.Context, status int, data interface{}) error {
	return c.JSON(status, map[string]interface{}{
		"success": true,
		"data":    data,
		"error":   nil,
	})
}

func errorResponse(c echo.Context, status int, code, message string) error {
	return c.JSON(status, map[string]interface{}{
		"success": false,
		"data":    nil,
		"error": map[string]string{
			"code":    code,
			"message": message,
		},
	})
}

func intParam(c echo.Context, name string) (int, error) {
	return strconv.Atoi(c.Param(name))
}

func intQuery(c echo.Context, name string, defaultVal int) int {
	v := c.QueryParam(name)
	if v == "" {
		return defaultVal
	}
	i, err := strconv.Atoi(v)
	if err != nil {
		return defaultVal
	}
	return i
}
