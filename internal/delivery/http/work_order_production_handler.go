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
	internalOnly := RequireInternalUser()
	v1.GET("/work-orders", RequirePermission(PermissionWORead), h.ListWorkOrders)
	v1.GET("/work-orders/:id", RequirePermission(PermissionWORead), h.GetWorkOrderDetail)
	v1.GET("/work-orders/shells/:id/total-qty", internalOnly, RequirePermission(PermissionWORead), h.GetWorkOrderShellTotalQty)
	v1.GET("/production/summary", RequirePermission(PermissionProductionSummaryRead), h.ListProductionSummary)
	v1.POST("/work-orders", internalOnly, RequirePermission(PermissionWOCreate), h.CreateWorkOrder)
	v1.PATCH("/work-orders/:id/close", internalOnly, RequirePermission(PermissionWOClose), h.CloseWorkOrder)
	v1.POST("/reports/:divisi", internalOnly, RequirePermission(PermissionProductionReportCreate), h.CreateFactoryReport)
}

// ListWorkOrders godoc
// @Summary      List Work Orders
// @Description  Returns a paginated list of work orders for transaction screens.
// @Tags         Work Order & Production
// @Produce      json
// @Security     BearerAuth
// @Param        page    query     int     false  "Page (default 1)"
// @Param        limit   query     int     false  "Limit (default 20)"
// @Param        search  query     string  false  "Search by buyer, model, or PO number"
// @Success      200     {object}  model.WorkOrderListSuccessDoc
// @Failure      400     {object}  model.WorkOrderErrorDoc
// @Failure      500     {object}  model.WorkOrderErrorDoc
// @Router       /api/v1/work-orders [get]
func (h *WorkOrderProductionHandler) ListWorkOrders(c *gin.Context) {
	filter, err := parseListQuery(c, 20)
	if err != nil {
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, "invalid list query", nil))
		return
	}
	mitraID, ok := GetMitraIDFromContext(c)
	if !ok {
		AbortWithError(c, NewHTTPError(http.StatusUnauthorized, "invalid authentication context", nil))
		return
	}

	item, err := h.useCase.ListWorkOrders(c.Request.Context(), model.TransactionListFilter{
		ListQueryFilter: filter,
		IDMitra:         mitraID,
	})
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "work orders retrieved", item)
}

// GetWorkOrderDetail godoc
// @Summary      Get Work Order Detail
// @Description  Returns a single work order with shells, trims, and material lists.
// @Tags         Work Order & Production
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      int  true  "Work Order ID"
// @Success      200  {object}  model.WorkOrderDetailSuccessDoc
// @Failure      400  {object}  model.WorkOrderErrorDoc
// @Failure      404  {object}  model.WorkOrderErrorDoc
// @Failure      500  {object}  model.WorkOrderErrorDoc
// @Router       /api/v1/work-orders/{id} [get]
func (h *WorkOrderProductionHandler) GetWorkOrderDetail(c *gin.Context) {
	id, err := parsePathInt32(c, "id")
	if err != nil {
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, "invalid work order id", nil))
		return
	}
	mitraID, ok := GetMitraIDFromContext(c)
	if !ok {
		AbortWithError(c, NewHTTPError(http.StatusUnauthorized, "invalid authentication context", nil))
		return
	}

	item, err := h.useCase.GetWorkOrderDetail(c.Request.Context(), id, mitraID)
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "work order retrieved", item)
}

// ListProductionSummary godoc
// @Summary      List Production Summary
// @Description  Returns aggregated production progress per work order shell size.
// @Tags         Work Order & Production
// @Produce      json
// @Security     BearerAuth
// @Param        id_wo            query     int     false  "Work Order ID filter"
// @Param        id_wo_shell_size query     int     false  "Work Order shell size ID filter"
// @Param        search           query     string  false  "Search by model or size"
// @Param        page             query     int     false  "Page (default 1)"
// @Param        limit            query     int     false  "Limit (default 20)"
// @Success      200              {object}  model.ProductionSummaryListSuccessDoc
// @Failure      400              {object}  model.WorkOrderErrorDoc
// @Failure      500              {object}  model.WorkOrderErrorDoc
// @Router       /api/v1/production/summary [get]
func (h *WorkOrderProductionHandler) ListProductionSummary(c *gin.Context) {
	idWO, err := parseQueryInt32(c, "id_wo", 0)
	if err != nil {
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, "invalid id_wo", nil))
		return
	}
	idWOShellSize, err := parseQueryInt32(c, "id_wo_shell_size", 0)
	if err != nil {
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, "invalid id_wo_shell_size", nil))
		return
	}
	filter, err := parseListQuery(c, 20)
	if err != nil {
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, "invalid list query", nil))
		return
	}
	mitraID, ok := GetMitraIDFromContext(c)
	if !ok {
		AbortWithError(c, NewHTTPError(http.StatusUnauthorized, "invalid authentication context", nil))
		return
	}

	item, err := h.useCase.ListProductionSummary(c.Request.Context(), model.ProductionSummaryFilter{
		IDWO:            idWO,
		IDWOShellSize:   idWOShellSize,
		IDMitra:         mitraID,
		ListQueryFilter: filter,
	})
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "production summary retrieved", item)
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

// CloseWorkOrder godoc
// @Summary      Close Work Order
// @Description  Manager endpoint that only changes work order status and close audit fields.
// @Tags         Work Order & Production
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      int  true  "Work Order ID"
// @Success      200  {object}  model.WorkOrderStatusSuccessDoc
// @Failure      400  {object}  model.WorkOrderErrorDoc
// @Failure      401  {object}  model.WorkOrderErrorDoc
// @Failure      403  {object}  model.WorkOrderErrorDoc
// @Failure      404  {object}  model.WorkOrderErrorDoc
// @Failure      409  {object}  model.WorkOrderErrorDoc
// @Failure      500  {object}  model.WorkOrderErrorDoc
// @Router       /api/v1/work-orders/{id}/close [patch]
func (h *WorkOrderProductionHandler) CloseWorkOrder(c *gin.Context) {
	id, err := parsePathInt32(c, "id")
	if err != nil {
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, "invalid work order id", nil))
		return
	}
	userID, ok := GetUserIDFromContext(c)
	if !ok {
		AbortWithError(c, NewHTTPError(http.StatusUnauthorized, "unauthorized", nil))
		return
	}

	item, err := h.useCase.CloseWorkOrder(c.Request.Context(), id, userID)
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "work order closed", item)
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

// GetWorkOrderShellTotalQty godoc
// @Summary      Get Work Order Shell Total Qty
// @Description  Returns the total quantity of a work order shell from its size variations.
// @Tags         Work Order & Production
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      int  true  "Work Order Shell ID"
// @Success      200  {object}  model.WorkOrderShellTotalQtySuccessDoc
// @Failure      400  {object}  model.WorkOrderErrorDoc
// @Failure      500  {object}  model.WorkOrderErrorDoc
// @Router       /api/v1/work-orders/shells/{id}/total-qty [get]
func (h *WorkOrderProductionHandler) GetWorkOrderShellTotalQty(c *gin.Context) {
	id, err := parsePathInt32(c, "id")
	if err != nil {
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, "invalid work order shell id", nil))
		return
	}

	item, err := h.useCase.GetWorkOrderShellTotalQty(c.Request.Context(), id)
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "total qty retrieved", item)
}

func (h *WorkOrderProductionHandler) handleError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, usecase.ErrWorkOrderValidation):
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, err.Error(), model.WorkOrderErrorDetail{Code: "invalid_work_order_payload"}))
	case errors.Is(err, usecase.ErrWorkOrderNotFound):
		AbortWithError(c, NewHTTPError(http.StatusNotFound, err.Error(), model.WorkOrderErrorDetail{Code: "work_order_not_found"}))
	case errors.Is(err, usecase.ErrWorkOrderReferenceNotFound):
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, err.Error(), model.WorkOrderErrorDetail{Code: "related_data_not_found"}))
	case errors.Is(err, usecase.ErrWorkOrderAlreadyClosed):
		AbortWithError(c, NewHTTPError(http.StatusConflict, err.Error(), model.WorkOrderErrorDetail{Code: "work_order_already_closed"}))
	case errors.Is(err, usecase.ErrReportDivisionUnsupported):
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, err.Error(), model.WorkOrderErrorDetail{Code: "unsupported_report_division"}))
	case errors.Is(err, usecase.ErrWorkOrderServiceUnavailable):
		AbortWithError(c, NewHTTPError(http.StatusInternalServerError, err.Error(), model.WorkOrderErrorDetail{Code: "work_order_service_unavailable"}))
	default:
		AbortWithError(c, err)
	}
}
