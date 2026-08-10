package httpdelivery

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"

	"permatatex-inventory/internal/model"
	"permatatex-inventory/internal/usecase"
	"permatatex-inventory/pkg/response"
)

type MaterialListHandler struct {
	useCase                    *usecase.MaterialListUseCase
	excelExportUseCase         materialListExcelExporter
	excelImportUseCase         materialListExcelImporter
	excelImportTemplateUseCase materialListExcelImportTemplater
}

type materialListExcelExporter interface {
	ExportByID(context.Context, int32) (*model.ExportedFile, error)
}
type materialListExcelImporter interface {
	Import(context.Context, int32, io.Reader) (*usecase.MaterialListExcelImportResult, error)
}
type materialListExcelImportTemplater interface {
	Generate(context.Context, int32) (*model.ExportedFile, error)
}

func NewMaterialListHandler(useCase *usecase.MaterialListUseCase, excelExportUseCase materialListExcelExporter, importUseCases ...any) (*MaterialListHandler, error) {
	if useCase == nil {
		return nil, errors.New("material list usecase is required")
	}
	if excelExportUseCase == nil {
		return nil, errors.New("material list excel export usecase is required")
	}
	if len(importUseCases) != 0 && len(importUseCases) != 2 {
		return nil, errors.New("material list excel import usecases must be supplied together")
	}
	var excelImportUseCase materialListExcelImporter
	var excelImportTemplateUseCase materialListExcelImportTemplater
	if len(importUseCases) == 2 {
		var ok bool
		excelImportUseCase, ok = importUseCases[0].(materialListExcelImporter)
		if !ok || excelImportUseCase == nil {
			return nil, errors.New("material list excel import usecase is required")
		}
		excelImportTemplateUseCase, ok = importUseCases[1].(materialListExcelImportTemplater)
		if !ok || excelImportTemplateUseCase == nil {
			return nil, errors.New("material list excel import template usecase is required")
		}
	}
	if len(importUseCases) == 2 && (excelImportUseCase == nil || excelImportTemplateUseCase == nil) {
		return nil, errors.New("material list excel import usecases are required")
	}
	return &MaterialListHandler{useCase: useCase, excelExportUseCase: excelExportUseCase, excelImportUseCase: excelImportUseCase, excelImportTemplateUseCase: excelImportTemplateUseCase}, nil
}

func (h *MaterialListHandler) RegisterRoutes(router gin.IRouter, authMiddleware gin.HandlerFunc) {
	v1 := router.Group("/api/v1").Use(authMiddleware)
	internalOnly := RequireInternalUser()

	v1.GET("/material-lists", RequirePermission(PermissionMaterialListRead), h.ListPaginated)
	v1.GET("/work-orders/:id/material-lists", RequirePermission(PermissionMaterialListRead), h.ListByWO)
	v1.POST("/work-orders/:id/material-lists", internalOnly, RequirePermission(PermissionMaterialListUpdate), h.Create)

	v1.GET("/material-lists/:id", RequirePermission(PermissionMaterialListRead), h.Get)
	v1.GET("/material-lists/:id/export/excel", internalOnly, RequirePermission(PermissionMaterialListRead), h.ExportExcel)
	v1.GET("/material-lists/:id/import/excel/template", internalOnly, RequirePermission(PermissionMaterialListUpdate), h.ImportExcelTemplate)
	v1.POST("/material-lists/:id/import/excel", internalOnly, RequirePermission(PermissionMaterialListUpdate), h.ImportExcel)
	v1.PATCH("/material-lists/:id", internalOnly, RequirePermission(PermissionMaterialListUpdate), h.Update)
	v1.DELETE("/material-lists/:id", internalOnly, RequirePermission(PermissionMaterialListUpdate), h.Delete)

	v1.POST("/material-lists/:id/items", internalOnly, RequirePermission(PermissionMaterialListUpdate), h.CreateItem)
	v1.GET("/material-list-items/:id", RequirePermission(PermissionMaterialListRead), h.GetItem)
	v1.PATCH("/material-list-items/:id", internalOnly, RequirePermission(PermissionMaterialListUpdate), h.UpdateItem)
	v1.DELETE("/material-list-items/:id", internalOnly, RequirePermission(PermissionMaterialListUpdate), h.DeleteItem)
}

// ImportExcelTemplate godoc
// @Summary      Download Material List import template
// @Description  Generates an XLSX import template for one unlocked material list.
// @Tags         Material List
// @Produce      application/vnd.openxmlformats-officedocument.spreadsheetml.sheet
// @Security     BearerAuth
// @Param        id path int true "Material List ID"
// @Success      200 {file} binary
// @Failure      403 {object} model.WorkOrderErrorDoc
// @Failure      404 {object} model.WorkOrderErrorDoc
// @Failure      409 {object} model.WorkOrderErrorDoc
// @Failure      500 {object} model.WorkOrderErrorDoc
// @Router       /api/v1/material-lists/{id}/import/excel/template [get]
func (h *MaterialListHandler) ImportExcelTemplate(c *gin.Context) {
	id, err := parsePathInt32(c, "id")
	if err != nil {
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, "invalid material list id", nil))
		return
	}
	file, err := h.excelImportTemplateUseCase.Generate(c.Request.Context(), id)
	if err != nil {
		h.handleImportError(c, err)
		return
	}
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, sanitizeMaterialListAttachmentFileName(file.FileName)))
	c.Data(http.StatusOK, file.ContentType, file.Content)
}

const materialListImportHTTPMaxBytes = (2 << 20) + (64 << 10)

// ImportExcel godoc
// @Summary      Import Material List XLSX
// @Description  Validates and imports an XLSX workbook atomically.
// @Tags         Material List
// @Accept       multipart/form-data
// @Produce      json
// @Security     BearerAuth
// @Param        id path int true "Material List ID"
// @Param        file formData file true "XLSX import workbook"
// @Success      201 {object} response.BaseResponse
// @Failure      400 {object} model.WorkOrderErrorDoc
// @Failure      403 {object} model.WorkOrderErrorDoc
// @Failure      404 {object} model.WorkOrderErrorDoc
// @Failure      409 {object} model.WorkOrderErrorDoc
// @Failure      413 {object} model.WorkOrderErrorDoc
// @Failure      500 {object} model.WorkOrderErrorDoc
// @Router       /api/v1/material-lists/{id}/import/excel [post]
func (h *MaterialListHandler) ImportExcel(c *gin.Context) {
	id, err := parsePathInt32(c, "id")
	if err != nil {
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, "invalid material list id", nil))
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, materialListImportHTTPMaxBytes)
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) || errors.Is(err, http.ErrBodyReadAfterClose) || strings.Contains(err.Error(), "request body too large") {
			AbortWithError(c, NewHTTPError(http.StatusRequestEntityTooLarge, "file too large", model.WorkOrderErrorDetail{Code: "file_too_large"}))
			return
		}
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, "import file is required", model.WorkOrderErrorDetail{Code: "invalid_workbook"}))
		return
	}
	if c.Request.MultipartForm != nil {
		defer c.Request.MultipartForm.RemoveAll()
		if len(c.Request.MultipartForm.File["file"]) != 1 {
			_ = file.Close()
			AbortWithError(c, NewHTTPError(http.StatusBadRequest, "exactly one import file is required", model.WorkOrderErrorDetail{Code: "invalid_workbook"}))
			return
		}
	}
	defer file.Close()
	if strings.ToLower(filepath.Ext(header.Filename)) != ".xlsx" {
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, "import file must be an .xlsx workbook", model.WorkOrderErrorDetail{Code: "invalid_workbook"}))
		return
	}
	if header.Size > 2<<20 {
		AbortWithError(c, NewHTTPError(http.StatusRequestEntityTooLarge, "file too large", model.WorkOrderErrorDetail{Code: "file_too_large"}))
		return
	}
	result, err := h.excelImportUseCase.Import(c.Request.Context(), id, file)
	if err != nil {
		h.handleImportError(c, err)
		return
	}
	response.Success(c, http.StatusCreated, "material list imported", gin.H{"id_material_list": id, "created_count": result.CreatedCount})
}

func (h *MaterialListHandler) handleImportError(c *gin.Context, err error) {
	var validation *usecase.MaterialListExcelImportValidationError
	if errors.As(err, &validation) {
		code, status := importErrorStatus(validation)
		detail := gin.H{"code": code, "row_errors": validation.Errors, "error_count": validation.ErrorCount, "truncated": validation.ErrorCount > len(validation.Errors)}
		AbortWithError(c, NewHTTPError(status, "material list import failed", detail))
		return
	}
	AbortWithError(c, NewHTTPError(http.StatusInternalServerError, "failed to import material list", model.WorkOrderErrorDetail{Code: "material_list_import_failed"}))
}

func importErrorStatus(v *usecase.MaterialListExcelImportValidationError) (string, int) {
	allConflict := len(v.Errors) > 0
	for _, e := range v.Errors {
		switch e.Code {
		case "material_list_not_found":
			return e.Code, http.StatusNotFound
		case "material_list_locked":
			return e.Code, http.StatusConflict
		case "file_too_large":
			return e.Code, http.StatusRequestEntityTooLarge
		case "duplicate_in_file", "duplicate_existing_item":
		default:
			allConflict = false
		}
	}
	if allConflict {
		return "duplicate_in_file", http.StatusConflict
	}
	for _, e := range v.Errors {
		if e.Code == "unsupported_template_version" || e.Code == "target_material_list_mismatch" {
			return e.Code, http.StatusBadRequest
		}
	}
	return "invalid_workbook", http.StatusBadRequest
}

// ExportExcel godoc
// @Summary      Export Material List Excel
// @Description  Generates a downloadable Excel workbook for one material list.
// @Tags         Material List
// @Produce      application/vnd.openxmlformats-officedocument.spreadsheetml.sheet
// @Security     BearerAuth
// @Param        id   path      int  true  "Material List ID"
// @Success      200  {file}    binary
// @Failure      400  {object}  model.WorkOrderErrorDoc
// @Failure      403  {object}  model.WorkOrderErrorDoc
// @Failure      404  {object}  model.WorkOrderErrorDoc
// @Failure      409  {object}  model.WorkOrderErrorDoc
// @Failure      500  {object}  model.WorkOrderErrorDoc
// @Router       /api/v1/material-lists/{id}/export/excel [get]
func (h *MaterialListHandler) ExportExcel(c *gin.Context) {
	id, err := parsePathInt32(c, "id")
	if err != nil {
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, "invalid material list id", nil))
		return
	}

	exportedFile, err := h.excelExportUseCase.ExportByID(c.Request.Context(), id)
	if err != nil {
		h.handleExportError(c, err)
		return
	}

	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, sanitizeMaterialListAttachmentFileName(exportedFile.FileName)))
	c.Data(http.StatusOK, exportedFile.ContentType, exportedFile.Content)
}

func (h *MaterialListHandler) Create(c *gin.Context) {
	idWo, err := parsePathInt32(c, "id")
	if err != nil {
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, "invalid work order id", nil))
		return
	}
	var req model.CreateMaterialListRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, "invalid payload", err))
		return
	}
	item, err := h.useCase.CreateMaterialList(c.Request.Context(), idWo, req)
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusCreated, "material list created", item)
}

func (h *MaterialListHandler) ListByWO(c *gin.Context) {
	idWo, err := parsePathInt32(c, "id")
	if err != nil {
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, "invalid work order id", nil))
		return
	}
	unlockedOnly := c.Query("unlocked") == "true"
	res, err := h.useCase.ListByWO(c.Request.Context(), idWo, unlockedOnly)
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "material lists retrieved", res)
}

func (h *MaterialListHandler) Get(c *gin.Context) {
	id, err := parsePathInt32(c, "id")
	if err != nil {
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, "invalid material list id", nil))
		return
	}
	res, err := h.useCase.Get(c.Request.Context(), id)
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "material list retrieved", res)
}

func (h *MaterialListHandler) Update(c *gin.Context) {
	id, err := parsePathInt32(c, "id")
	if err != nil {
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, "invalid material list id", nil))
		return
	}
	var req model.UpdateMaterialListRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, "invalid payload", err))
		return
	}
	res, err := h.useCase.Update(c.Request.Context(), id, req)
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "material list updated", res)
}

func (h *MaterialListHandler) Delete(c *gin.Context) {
	id, err := parsePathInt32(c, "id")
	if err != nil {
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, "invalid material list id", nil))
		return
	}
	if err := h.useCase.Delete(c.Request.Context(), id); err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "material list deleted", nil)
}

func (h *MaterialListHandler) CreateItem(c *gin.Context) {
	idML, err := parsePathInt32(c, "id")
	if err != nil {
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, "invalid material list id", nil))
		return
	}
	var req model.CreateMaterialListItemBody
	if err := c.ShouldBindJSON(&req); err != nil {
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, "invalid payload", err))
		return
	}
	item, err := h.useCase.CreateItem(c.Request.Context(), idML, req)
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusCreated, "material list item created", item)
}

func (h *MaterialListHandler) UpdateItem(c *gin.Context) {
	id, err := parsePathInt32(c, "id")
	if err != nil {
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, "invalid material list item id", nil))
		return
	}
	var req model.UpdateMaterialListItemBody
	if err := c.ShouldBindJSON(&req); err != nil {
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, "invalid payload", err))
		return
	}
	res, err := h.useCase.UpdateItem(c.Request.Context(), id, req)
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "material list item updated", res)
}

func (h *MaterialListHandler) DeleteItem(c *gin.Context) {
	id, err := parsePathInt32(c, "id")
	if err != nil {
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, "invalid material list item id", nil))
		return
	}
	if err := h.useCase.DeleteItem(c.Request.Context(), id); err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "material list item deleted", nil)
}

func (h *MaterialListHandler) GetItem(c *gin.Context) {
	id, err := parsePathInt32(c, "id")
	if err != nil {
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, "invalid material list item id", nil))
		return
	}
	res, err := h.useCase.GetItemDetail(c.Request.Context(), id)
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "material list item retrieved", res)
}

func (h *MaterialListHandler) ListPaginated(c *gin.Context) {
	filter, err := parseListQuery(c, 20)
	if err != nil {
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, "invalid query params", nil))
		return
	}
	lockedOnly := c.Query("locked_only") != "false"
	res, err := h.useCase.ListMaterialListsPaginated(c.Request.Context(), filter.Search, lockedOnly, filter.Limit, filter.Offset)
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "material lists retrieved", res)
}

func (h *MaterialListHandler) handleError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, usecase.ErrMaterialListNotFound), errors.Is(err, usecase.ErrMaterialListItemNotFound):
		AbortWithError(c, NewHTTPError(http.StatusNotFound, err.Error(), nil))
	case errors.Is(err, usecase.ErrMaterialListLocked):
		AbortWithError(c, NewHTTPError(http.StatusConflict, err.Error(), nil))
	case errors.Is(err, usecase.ErrMaterialListValidation):
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, err.Error(), nil))
	default:
		AbortWithError(c, NewHTTPError(http.StatusInternalServerError, err.Error(), nil))
	}
}

func (h *MaterialListHandler) handleExportError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, usecase.ErrMaterialListNotFound):
		AbortWithError(c, NewHTTPError(http.StatusNotFound, err.Error(), model.WorkOrderErrorDetail{Code: "material_list_not_found"}))
	case errors.Is(err, usecase.ErrMaterialListExportMarkerPlanConflict):
		AbortWithError(c, NewHTTPError(http.StatusConflict, "material list export conflict", model.WorkOrderErrorDetail{Code: "marker_plan_conflict"}))
	case errors.Is(err, usecase.ErrMaterialListExportDataConflict):
		AbortWithError(c, NewHTTPError(http.StatusConflict, "material list export conflict", model.WorkOrderErrorDetail{Code: "material_list_export_data_conflict"}))
	default:
		AbortWithError(c, NewHTTPError(http.StatusInternalServerError, "failed to export material list", model.WorkOrderErrorDetail{Code: "material_list_export_failed"}))
	}
}

func sanitizeMaterialListAttachmentFileName(fileName string) string {
	name := strings.TrimSpace(fileName)
	if name == "" {
		return "MATERIAL_LIST.xlsx"
	}
	return strings.NewReplacer(`"`, "", "\r", "", "\n", "").Replace(name)
}
