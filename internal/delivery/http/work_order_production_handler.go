package httpdelivery

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"permatatex-inventory/internal/model"
	"permatatex-inventory/internal/usecase"
	"permatatex-inventory/pkg/response"
)

type WorkOrderProductionHandler struct {
	useCase *usecase.WorkOrderProductionUseCase
}

func NewWorkOrderProductionHandler(useCase *usecase.WorkOrderProductionUseCase) (*WorkOrderProductionHandler, error) {
	if useCase == nil {
		return nil, errors.New("work order production usecase is required")
	}
	return &WorkOrderProductionHandler{useCase: useCase}, nil
}

func (h *WorkOrderProductionHandler) RegisterRoutes(router gin.IRouter, authMiddleware gin.HandlerFunc) {
	v1 := router.Group("/api/v1").Use(authMiddleware)
	v1.POST("/work-orders", h.CreateWorkOrder)
	v1.POST("/reports/:divisi", h.CreateFactoryReport)
}

// CreateWorkOrder godoc
// @Summary      Create Work Order
// @Description  Create a work order with shells, shell sizes, trims, and material list in a single transaction.
// @Tags         Work Order & Production
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        payload  body      model.CreateWorkOrderRequest  true  "Work order payload"
// @Success      201      {object}  model.WorkOrderSuccessDoc
// @Failure      400      {object}  model.WorkOrderValidationErrorDoc
// @Failure      500      {object}  model.WorkOrderErrorDoc
// @Router       /api/v1/work-orders [post]
func (h *WorkOrderProductionHandler) CreateWorkOrder(c *gin.Context) {
	var req model.CreateWorkOrderRequest
	if !BindJSON(c, &req) {
		return
	}

	item, err := h.useCase.CreateWorkOrder(c.Request.Context(), req)
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusCreated, "work order created", item)
}

// CreateFactoryReport godoc
// @Summary      Create Factory Report
// @Description  Create lightweight production report for a specific division. Supported divisi: cutting, sewing, qc-finish, packing, pengiriman.
// @Tags         Work Order & Production
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        divisi   path      string                           true  "Division name"
// @Param        payload  body      model.CreateFactoryReportRequest true  "Factory report payload"
// @Success      201      {object}  model.FactoryReportSuccessDoc
// @Failure      400      {object}  model.WorkOrderValidationErrorDoc
// @Failure      500      {object}  model.WorkOrderErrorDoc
// @Router       /api/v1/reports/{divisi} [post]
func (h *WorkOrderProductionHandler) CreateFactoryReport(c *gin.Context) {
	var req model.CreateFactoryReportRequest
	if !BindJSON(c, &req) {
		return
	}

	item, err := h.useCase.CreateFactoryReport(c.Request.Context(), c.Param("divisi"), req)
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusCreated, "factory report created", item)
}

func (h *WorkOrderProductionHandler) handleError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, usecase.ErrWorkOrderValidation):
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, err.Error(), model.WorkOrderErrorDetail{Code: "invalid_work_order_payload"}))
	case errors.Is(err, usecase.ErrWorkOrderReferenceNotFound):
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, err.Error(), model.WorkOrderErrorDetail{Code: "related_data_not_found"}))
	case errors.Is(err, usecase.ErrReportDivisionUnsupported):
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, err.Error(), model.WorkOrderErrorDetail{Code: "unsupported_report_division"}))
	case errors.Is(err, usecase.ErrWorkOrderServiceUnavailable):
		AbortWithError(c, NewHTTPError(http.StatusInternalServerError, err.Error(), model.WorkOrderErrorDetail{Code: "work_order_service_unavailable"}))
	default:
		AbortWithError(c, err)
	}
}
