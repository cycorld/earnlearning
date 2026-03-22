package handler

import (
	"net/http"
	"os"
	"path/filepath"
	"runtime"

	"github.com/labstack/echo/v4"
)

// DocsHandler serves API documentation.
type DocsHandler struct {
	docsDir string
}

// NewDocsHandler creates a new DocsHandler.
// docsDir is the path to the directory containing swagger.json.
func NewDocsHandler(docsDir string) *DocsHandler {
	return &DocsHandler{docsDir: docsDir}
}

// ServeUI serves the Scalar API Reference UI.
//
//	@Summary		API 문서 UI
//	@Description	Scalar 기반 API 문서 페이지
//	@Tags			Docs
//	@Produce		html
//	@Success		200	{string}	string	"HTML page"
//	@Router			/docs [get]
func (h *DocsHandler) ServeUI(c echo.Context) error {
	html := `<!DOCTYPE html>
<html>
<head>
  <title>EarnLearning API Docs</title>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
</head>
<body>
  <script id="api-reference" data-url="/docs/openapi.json"></script>
  <script src="https://cdn.jsdelivr.net/npm/@scalar/api-reference"></script>
</body>
</html>`
	return c.HTML(http.StatusOK, html)
}

// ServeSpec serves the OpenAPI JSON spec.
//
//	@Summary		OpenAPI 스펙 (JSON)
//	@Description	swaggo로 생성된 OpenAPI 3.0 JSON 스펙
//	@Tags			Docs
//	@Produce		json
//	@Success		200	{object}	object
//	@Router			/docs/openapi.json [get]
func (h *DocsHandler) ServeSpec(c echo.Context) error {
	specPath := filepath.Join(h.docsDir, "swagger.json")

	data, err := os.ReadFile(specPath)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"success": false,
			"data":    nil,
			"error":   map[string]string{"code": "SPEC_NOT_FOUND", "message": "API 스펙 파일을 찾을 수 없습니다"},
		})
	}

	return c.Blob(http.StatusOK, "application/json", data)
}

// getDocsDir returns the docs directory relative to the binary or source.
func getDocsDir() string {
	// Try relative to executable first
	ex, err := os.Executable()
	if err == nil {
		dir := filepath.Join(filepath.Dir(ex), "docs")
		if _, err := os.Stat(filepath.Join(dir, "swagger.json")); err == nil {
			return dir
		}
	}

	// Fallback: relative to source file
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "..", "..", "..", "..", "docs")
}
