package httpdelivery

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"permatatex-inventory/internal/model"
	"permatatex-inventory/internal/usecase"
	"permatatex-inventory/pkg/response"
)

const (
	messageReportPengirimanCreated       = "report pengiriman created"
	messageReportPengirimanListRetrieved = "report pengiriman list retrieved"
	messageReportPengirimanRetrieved     = "report pengiriman retrieved"
	messageReportPengirimanDeleted       = "report pengiriman deleted"
)

type ReportPengirimanHandler struct {
	useCase *usecase.ReportPengirimanUseCase
}

func NewReportPengirimanHandler(useCase *usecase.ReportPengirimanUseCase) (*ReportPengirimanHandler, error) {
	if useCase == nil {
		return nil, errors.New("report pengiriman usecase is required")
	}

	return &ReportPengirimanHandler{
		useCase: useCase,
	}, nil
}

func (h *ReportPengirimanHandler) RegisterRoutes(router gin.IRouter, authMiddleware gin.HandlerFunc) {
	group := router.Group("/api/v1/report-pengiriman").Use(authMiddleware)
	group.POST("", h.Create)
	group.GET("", h.List)
	group.GET("/:id", h.GetByID)
	group.DELETE("/:id", h.DeleteByID)
}

// Create godoc
// @Summary      Create Report Pengiriman
// @Description  Creates a new report pengiriman entry.
// @Tags         Report Pengiriman
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        payload  body      model.CreateReportPengirimanRequest  true  "Create payload"
// @Success      201      {object}  model.ReportPengirimanSuccessDoc
// @Failure      400      {object}  model.ReportPengirimanBadRequestDoc
// @Failure      401      {object}  model.ReportPengirimanUnauthorizedDoc
// @Failure      503      {object}  model.ReportPengirimanServiceUnavailableDoc
// @Router       /api/v1/report-pengiriman [post]
func (h *ReportPengirimanHandler) Create(c *gin.Context) {
	var req model.CreateReportPengirimanRequest
	if !BindJSON(c, &req) {
		return
	}

	result, err := h.useCase.Create(c.Request.Context(), req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	response.Success(c, http.StatusCreated, messageReportPengirimanCreated, result)
}

// List godoc
// @Summary      List Report Pengiriman
// @Description  Returns report pengiriman list with optional filters.
// @Tags         Report Pengiriman
// @Produce      json
// @Security     BearerAuth
// @Param        date_from         query     string  false  "Start date filter (YYYY-MM-DD)"
// @Param        date_to           query     string  false  "End date filter (YYYY-MM-DD)"
// @Param        id_wo_shell_size  query     int     false  "Work order shell size ID filter"
// @Param        limit             query     int     false  "Result limit (default: 20, max: 100)"
// @Param        offset            query     int     false  "Result offset (default: 0)"
// @Success      200               {object}  model.ReportPengirimanListSuccessDoc
// @Failure      400               {object}  model.ReportPengirimanBadRequestDoc
// @Failure      401               {object}  model.ReportPengirimanUnauthorizedDoc
// @Failure      503               {object}  model.ReportPengirimanServiceUnavailableDoc
// @Router       /api/v1/report-pengiriman [get]
func (h *ReportPengirimanHandler) List(c *gin.Context) {
	filter, err := parseListReportPengirimanFilter(c)
	if err != nil {
		AbortWithError(c, err)
		return
	}

	result, err := h.useCase.List(c.Request.Context(), filter)
	if err != nil {
		h.handleError(c, err)
		return
	}

	response.Success(c, http.StatusOK, messageReportPengirimanListRetrieved, result)
}

// GetByID godoc
// @Summary      Get Report Pengiriman Detail
// @Description  Returns report pengiriman detail by ID.
// @Tags         Report Pengiriman
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      int  true  "Report Pengiriman ID"
// @Success      200  {object}  model.ReportPengirimanSuccessDoc
// @Failure      400  {object}  model.ReportPengirimanBadRequestDoc
// @Failure      401  {object}  model.ReportPengirimanUnauthorizedDoc
// @Failure      404  {object}  model.ReportPengirimanNotFoundDoc
// @Failure      503  {object}  model.ReportPengirimanServiceUnavailableDoc
// @Router       /api/v1/report-pengiriman/{id} [get]
func (h *ReportPengirimanHandler) GetByID(c *gin.Context) {
	id, err := parseIDParam(c.Param("id"))
	if err != nil {
		AbortWithError(c, err)
		return
	}

	result, err := h.useCase.GetByID(c.Request.Context(), id)
	if err != nil {
		h.handleError(c, err)
		return
	}

	response.Success(c, http.StatusOK, messageReportPengirimanRetrieved, result)
}

// DeleteByID godoc
// @Summary      Delete Report Pengiriman
// @Description  Deletes report pengiriman by ID.
// @Tags         Report Pengiriman
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      int  true  "Report Pengiriman ID"
// @Success      200  {object}  model.ReportPengirimanDeleteSuccessDoc
// @Failure      400  {object}  model.ReportPengirimanBadRequestDoc
// @Failure      401  {object}  model.ReportPengirimanUnauthorizedDoc
// @Failure      404  {object}  model.ReportPengirimanNotFoundDoc
// @Failure      503  {object}  model.ReportPengirimanServiceUnavailableDoc
// @Router       /api/v1/report-pengiriman/{id} [delete]
func (h *ReportPengirimanHandler) DeleteByID(c *gin.Context) {
	id, err := parseIDParam(c.Param("id"))
	if err != nil {
		AbortWithError(c, err)
		return
	}

	if err := h.useCase.Delete(c.Request.Context(), id); err != nil {
		h.handleError(c, err)
		return
	}

	response.Success(c, http.StatusOK, messageReportPengirimanDeleted, model.DeleteReportPengirimanResponse{
		IDReportPengiriman: id,
	})
}

func parseListReportPengirimanFilter(c *gin.Context) (model.ListReportPengirimanFilter, error) {
	filter := model.ListReportPengirimanFilter{
		DateFrom: strings.TrimSpace(c.Query("date_from")),
		DateTo:   strings.TrimSpace(c.Query("date_to")),
		Limit:    20,
		Offset:   0,
	}

	if value := strings.TrimSpace(c.Query("id_wo_shell_size")); value != "" {
		parsed, err := strconv.ParseInt(value, 10, 32)
		if err != nil || parsed <= 0 {
			return filter, NewHTTPError(http.StatusBadRequest, "id_wo_shell_size must be a positive integer", nil)
		}
		filter.IDWOShellSize = int32(parsed)
	}

	if value := strings.TrimSpace(c.Query("limit")); value != "" {
		parsed, err := strconv.ParseInt(value, 10, 32)
		if err != nil {
			return filter, NewHTTPError(http.StatusBadRequest, "limit must be an integer", nil)
		}
		filter.Limit = int32(parsed)
	}

	if value := strings.TrimSpace(c.Query("offset")); value != "" {
		parsed, err := strconv.ParseInt(value, 10, 32)
		if err != nil {
			return filter, NewHTTPError(http.StatusBadRequest, "offset must be an integer", nil)
		}
		filter.Offset = int32(parsed)
	}

	return filter, nil
}

func parseIDParam(raw string) (int32, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return 0, NewHTTPError(http.StatusBadRequest, "id is required", nil)
	}

	parsed, err := strconv.ParseInt(value, 10, 32)
	if err != nil || parsed <= 0 {
		return 0, NewHTTPError(http.StatusBadRequest, "id must be a positive integer", nil)
	}

	return int32(parsed), nil
}

func (h *ReportPengirimanHandler) handleError(c *gin.Context, err error) {
	if errors.Is(err, usecase.ErrReportPengirimanValidation) {
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, err.Error(), nil))
		return
	}

	if errors.Is(err, usecase.ErrReportPengirimanNotFound) {
		AbortWithError(c, NewHTTPError(http.StatusNotFound, usecase.ErrReportPengirimanNotFound.Error(), nil))
		return
	}

	if errors.Is(err, usecase.ErrReportPengirimanServiceUnavailable) {
		AbortWithError(c, NewHTTPError(http.StatusServiceUnavailable, usecase.ErrReportPengirimanServiceUnavailable.Error(), nil))
		return
	}

	AbortWithError(c, err)
}
