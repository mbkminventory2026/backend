package httpdelivery

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"permatatex-inventory/internal/model"
	"permatatex-inventory/internal/usecase"
	excelexporter "permatatex-inventory/pkg/exporter/excel"
	"permatatex-inventory/pkg/response"
)

type ExcelExportHandler struct {
	useCase *usecase.ExcelExportUseCase
}

func NewExcelExportHandler(useCase *usecase.ExcelExportUseCase) (*ExcelExportHandler, error) {
	if useCase == nil {
		return nil, errors.New("excel export usecase is required")
	}

	return &ExcelExportHandler{useCase: useCase}, nil
}

func (h *ExcelExportHandler) RegisterRoutes(router gin.IRouter, authMiddleware gin.HandlerFunc) {
	group := router.Group("/api/v1/exports/excel").Use(authMiddleware, RequireInternalUser(), RequirePermission(PermissionReportRead))
	{
		group.GET("/templates", h.ListTemplates)
		group.POST("/render", h.RenderTemplate)
	}
}

// ListTemplates godoc
// @Summary      List Excel Export Templates
// @Description  Returns available Excel templates from the configured export template directory.
// @Tags         Exports
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  model.ExcelTemplateListSuccessDoc
// @Failure      401  {object}  response.BaseResponse
// @Failure      403  {object}  response.BaseResponse
// @Failure      500  {object}  response.BaseResponse
// @Router       /api/v1/exports/excel/templates [get]
func (h *ExcelExportHandler) ListTemplates(c *gin.Context) {
	result, err := h.useCase.ListTemplates(c.Request.Context())
	if err != nil {
		AbortWithError(c, err)
		return
	}

	response.Success(c, http.StatusOK, "excel templates retrieved", result)
}

// RenderTemplate godoc
// @Summary      Render Excel Template
// @Description  Renders an Excel workbook from a registered template and returns it as a downloadable .xlsx file.
// @Tags         Exports
// @Accept       json
// @Produce      application/vnd.openxmlformats-officedocument.spreadsheetml.sheet
// @Security     BearerAuth
// @Param        payload  body      model.ExcelRenderRequest  true  "Excel render payload"
// @Success      200      {file}    binary
// @Failure      400      {object}  response.BaseResponse
// @Failure      401      {object}  response.BaseResponse
// @Failure      403      {object}  response.BaseResponse
// @Failure      500      {object}  response.BaseResponse
// @Router       /api/v1/exports/excel/render [post]
func (h *ExcelExportHandler) RenderTemplate(c *gin.Context) {
	var req model.ExcelRenderRequest
	if !BindJSON(c, &req) {
		return
	}

	exportedFile, err := h.useCase.RenderTemplate(c.Request.Context(), req)
	if err != nil {
		if errors.Is(err, excelexporter.ErrTemplateNameRequired) {
			AbortWithError(c, NewHTTPError(http.StatusBadRequest, err.Error(), nil))
			return
		}

		if errors.Is(err, excelexporter.ErrTemplateNotFound) {
			AbortWithError(c, NewHTTPError(http.StatusNotFound, err.Error(), nil))
			return
		}

		AbortWithError(c, err)
		return
	}

	disposition := fmt.Sprintf(`attachment; filename="%s"`, sanitizeAttachmentFileName(exportedFile.FileName))
	c.Header("Content-Disposition", disposition)
	c.Data(http.StatusOK, exportedFile.ContentType, exportedFile.Content)
}

func sanitizeAttachmentFileName(fileName string) string {
	name := strings.TrimSpace(fileName)
	if name == "" {
		return "export.xlsx"
	}

	replacer := strings.NewReplacer(`"`, "", "\r", "", "\n", "")
	return replacer.Replace(name)
}
