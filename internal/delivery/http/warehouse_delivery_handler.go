package httpdelivery

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"permatatex-inventory/internal/model"
	"permatatex-inventory/internal/usecase"
	"permatatex-inventory/pkg/response"
)

type WarehouseDeliveryHandler struct {
	useCase *usecase.WarehouseDeliveryUseCase
}

func NewWarehouseDeliveryHandler(useCase *usecase.WarehouseDeliveryUseCase) (*WarehouseDeliveryHandler, error) {
	if useCase == nil {
		return nil, errors.New("warehouse delivery usecase is required")
	}
	return &WarehouseDeliveryHandler{useCase: useCase}, nil
}

func (h *WarehouseDeliveryHandler) RegisterRoutes(router gin.IRouter, authMiddleware gin.HandlerFunc) {
	v1 := router.Group("/api/v1").Use(authMiddleware)
	v1.POST("/inventory/receive", RequirePermission(PermissionInventoryReceive), h.ReceiveInventory)
	v1.POST("/inventory/issue", RequirePermission(PermissionInventoryIssue), h.IssueInventory)
	v1.GET("/packing-lists", h.ListPackingLists)
	v1.GET("/packing-lists/:id", h.GetPackingListDetail)
	v1.POST("/packing-lists", RequirePermission(PermissionPackingListCreate), h.CreatePackingList)
	v1.GET("/surat-jalan-clients", h.ListSuratJalanClients)
	v1.GET("/surat-jalan-clients/:id", h.GetSuratJalanClientDetail)
	v1.GET("/surat-jalan-internals", h.ListSuratJalanInternals)
	v1.GET("/surat-jalan-internals/:id", h.GetSuratJalanInternalDetail)
	v1.POST("/surat-jalan/:type", RequirePermission(PermissionSuratJalanCreate), h.CreateSuratJalan)
}

// ReceiveInventory godoc
// @Summary      Receive Inventory
// @Description  Record received inventory, insert reconciliation receipt detail, and update reconciliation balance in one database call.
// @Tags         Warehouse & Delivery
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        payload  body      model.ReceiveInventoryRequest  true  "Receive inventory payload"
// @Success      201      {object}  model.ReceiveInventorySuccessDoc
// @Failure      400      {object}  model.WarehouseValidationErrorDoc
// @Failure      500      {object}  model.WarehouseErrorDoc
// @Router       /api/v1/inventory/receive [post]
func (h *WarehouseDeliveryHandler) ReceiveInventory(c *gin.Context) {
	var req model.ReceiveInventoryRequest
	if !BindJSON(c, &req) {
		return
	}

	item, err := h.useCase.ReceiveInventory(c.Request.Context(), req)
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusCreated, "inventory received", item)
}

// IssueInventory godoc
// @Summary      Issue Inventory
// @Description  Decrease reconciliation material balance when raw material is taken out from warehouse for production.
// @Tags         Warehouse & Delivery
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        payload  body      model.IssueInventoryRequest  true  "Issue inventory payload"
// @Success      200      {object}  model.IssueInventorySuccessDoc
// @Failure      400      {object}  model.WarehouseValidationErrorDoc
// @Failure      404      {object}  model.WarehouseErrorDoc
// @Failure      409      {object}  model.WarehouseErrorDoc
// @Failure      500      {object}  model.WarehouseErrorDoc
// @Router       /api/v1/inventory/issue [post]
func (h *WarehouseDeliveryHandler) IssueInventory(c *gin.Context) {
	var req model.IssueInventoryRequest
	if !BindJSON(c, &req) {
		return
	}

	item, err := h.useCase.IssueInventory(c.Request.Context(), req)
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "inventory issued", item)
}

// CreatePackingList godoc
// @Summary      Create Packing List
// @Description  Create packing list header with nested box items and size breakdowns in a single transaction.
// @Tags         Warehouse & Delivery
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        payload  body      model.CreatePackingListRequest  true  "Packing list payload"
// @Success      201      {object}  model.PackingListSuccessDoc
// @Failure      400      {object}  model.WarehouseValidationErrorDoc
// @Failure      500      {object}  model.WarehouseErrorDoc
// @Router       /api/v1/packing-lists [post]
func (h *WarehouseDeliveryHandler) CreatePackingList(c *gin.Context) {
	var req model.CreatePackingListRequest
	if !BindJSON(c, &req) {
		return
	}

	item, err := h.useCase.CreatePackingList(c.Request.Context(), req)
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusCreated, "packing list created", item)
}

// ListPackingLists godoc
// @Summary      List Packing Lists
// @Description  Returns a paginated list of packing list headers.
// @Tags         Warehouse & Delivery
// @Produce      json
// @Security     BearerAuth
// @Param        page    query     int     false  "Page (default 1)"
// @Param        limit   query     int     false  "Limit (default 20)"
// @Param        search  query     string  false  "Search by buyer or model"
// @Success      200     {object}  model.PackingListListSuccessDoc
// @Failure      400     {object}  model.WarehouseErrorDoc
// @Failure      500     {object}  model.WarehouseErrorDoc
// @Router       /api/v1/packing-lists [get]
func (h *WarehouseDeliveryHandler) ListPackingLists(c *gin.Context) {
	page, err := parseQueryInt32(c, "page", 1)
	if err != nil {
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, "invalid page", nil))
		return
	}
	limit, err := parseQueryInt32(c, "limit", 20)
	if err != nil {
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, "invalid limit", nil))
		return
	}

	item, err := h.useCase.ListPackingLists(c.Request.Context(), model.TransactionListFilter{
		Page:   page,
		Limit:  limit,
		Search: c.Query("search"),
	})
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "packing lists retrieved", item)
}

// GetPackingListDetail godoc
// @Summary      Get Packing List Detail
// @Description  Returns a single packing list with nested items and sizes.
// @Tags         Warehouse & Delivery
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      int  true  "Packing List ID"
// @Success      200  {object}  model.PackingListDetailSuccessDoc
// @Failure      400  {object}  model.WarehouseErrorDoc
// @Failure      404  {object}  model.WarehouseErrorDoc
// @Failure      500  {object}  model.WarehouseErrorDoc
// @Router       /api/v1/packing-lists/{id} [get]
func (h *WarehouseDeliveryHandler) GetPackingListDetail(c *gin.Context) {
	id, err := parsePathInt32(c, "id")
	if err != nil {
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, "invalid packing list id", nil))
		return
	}

	item, err := h.useCase.GetPackingListDetail(c.Request.Context(), id)
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "packing list retrieved", item)
}

// CreateSuratJalan godoc
// @Summary      Create Surat Jalan
// @Description  Create surat jalan document. Supported type: internal, client.
// @Tags         Warehouse & Delivery
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        type     path      string                              true  "Surat jalan type"
// @Param        payload  body      model.CreateSuratJalanClientRequest false "Surat jalan client payload"
// @Success      201      {object}  model.SuratJalanSuccessDoc
// @Failure      400      {object}  model.WarehouseValidationErrorDoc
// @Failure      500      {object}  model.WarehouseErrorDoc
// @Router       /api/v1/surat-jalan/{type} [post]
func (h *WarehouseDeliveryHandler) CreateSuratJalan(c *gin.Context) {
	suratJalanType := strings.TrimSpace(strings.ToLower(c.Param("type")))

	var req *model.CreateSuratJalanClientRequest
	if suratJalanType == "client" {
		payload := new(model.CreateSuratJalanClientRequest)
		if !BindJSON(c, payload) {
			return
		}
		req = payload
	}

	item, err := h.useCase.CreateSuratJalan(c.Request.Context(), suratJalanType, req)
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusCreated, "surat jalan created", item)
}

// ListSuratJalanClients godoc
// @Summary      List Surat Jalan Clients
// @Description  Returns a paginated list of client delivery notes.
// @Tags         Warehouse & Delivery
// @Produce      json
// @Security     BearerAuth
// @Param        page    query     int     false  "Page (default 1)"
// @Param        limit   query     int     false  "Limit (default 20)"
// @Param        search  query     string  false  "Search by description or material"
// @Success      200     {object}  model.SuratJalanClientListSuccessDoc
// @Failure      400     {object}  model.WarehouseErrorDoc
// @Failure      500     {object}  model.WarehouseErrorDoc
// @Router       /api/v1/surat-jalan-clients [get]
func (h *WarehouseDeliveryHandler) ListSuratJalanClients(c *gin.Context) {
	page, err := parseQueryInt32(c, "page", 1)
	if err != nil {
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, "invalid page", nil))
		return
	}
	limit, err := parseQueryInt32(c, "limit", 20)
	if err != nil {
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, "invalid limit", nil))
		return
	}

	item, err := h.useCase.ListSuratJalanClients(c.Request.Context(), model.TransactionListFilter{
		Page:   page,
		Limit:  limit,
		Search: c.Query("search"),
	})
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "surat jalan clients retrieved", item)
}

// GetSuratJalanClientDetail godoc
// @Summary      Get Surat Jalan Client Detail
// @Description  Returns a single client delivery note.
// @Tags         Warehouse & Delivery
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      int  true  "Surat Jalan Client ID"
// @Success      200  {object}  model.SuratJalanClientDetailSuccessDoc
// @Failure      400  {object}  model.WarehouseErrorDoc
// @Failure      404  {object}  model.WarehouseErrorDoc
// @Failure      500  {object}  model.WarehouseErrorDoc
// @Router       /api/v1/surat-jalan-clients/{id} [get]
func (h *WarehouseDeliveryHandler) GetSuratJalanClientDetail(c *gin.Context) {
	id, err := parsePathInt32(c, "id")
	if err != nil {
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, "invalid surat jalan client id", nil))
		return
	}

	item, err := h.useCase.GetSuratJalanClientDetail(c.Request.Context(), id)
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "surat jalan client retrieved", item)
}

// ListSuratJalanInternals godoc
// @Summary      List Surat Jalan Internals
// @Description  Returns a paginated list of internal delivery notes.
// @Tags         Warehouse & Delivery
// @Produce      json
// @Security     BearerAuth
// @Param        page    query     int  false  "Page (default 1)"
// @Param        limit   query     int  false  "Limit (default 20)"
// @Success      200     {object}  model.SuratJalanInternalListSuccessDoc
// @Failure      400     {object}  model.WarehouseErrorDoc
// @Failure      500     {object}  model.WarehouseErrorDoc
// @Router       /api/v1/surat-jalan-internals [get]
func (h *WarehouseDeliveryHandler) ListSuratJalanInternals(c *gin.Context) {
	page, err := parseQueryInt32(c, "page", 1)
	if err != nil {
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, "invalid page", nil))
		return
	}
	limit, err := parseQueryInt32(c, "limit", 20)
	if err != nil {
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, "invalid limit", nil))
		return
	}

	item, err := h.useCase.ListSuratJalanInternals(c.Request.Context(), model.TransactionListFilter{
		Page:  page,
		Limit: limit,
	})
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "surat jalan internals retrieved", item)
}

// GetSuratJalanInternalDetail godoc
// @Summary      Get Surat Jalan Internal Detail
// @Description  Returns a single internal delivery note.
// @Tags         Warehouse & Delivery
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      int  true  "Surat Jalan Internal ID"
// @Success      200  {object}  model.SuratJalanInternalDetailSuccessDoc
// @Failure      400  {object}  model.WarehouseErrorDoc
// @Failure      404  {object}  model.WarehouseErrorDoc
// @Failure      500  {object}  model.WarehouseErrorDoc
// @Router       /api/v1/surat-jalan-internals/{id} [get]
func (h *WarehouseDeliveryHandler) GetSuratJalanInternalDetail(c *gin.Context) {
	id, err := parsePathInt32(c, "id")
	if err != nil {
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, "invalid surat jalan internal id", nil))
		return
	}

	item, err := h.useCase.GetSuratJalanInternalDetail(c.Request.Context(), id)
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "surat jalan internal retrieved", item)
}

func (h *WarehouseDeliveryHandler) handleError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, usecase.ErrWarehouseValidation):
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, err.Error(), model.WarehouseErrorDetail{Code: "invalid_warehouse_payload"}))
	case errors.Is(err, usecase.ErrWarehouseNotFound):
		AbortWithError(c, NewHTTPError(http.StatusNotFound, err.Error(), model.WarehouseErrorDetail{Code: "warehouse_transaction_not_found"}))
	case errors.Is(err, usecase.ErrWarehouseInsufficientStock):
		AbortWithError(c, NewHTTPError(http.StatusConflict, err.Error(), model.WarehouseErrorDetail{Code: "insufficient_stock_balance"}))
	case errors.Is(err, usecase.ErrWarehouseReferenceNotFound):
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, err.Error(), model.WarehouseErrorDetail{Code: "related_data_not_found"}))
	case errors.Is(err, usecase.ErrSuratJalanTypeUnsupported):
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, err.Error(), model.WarehouseErrorDetail{Code: "unsupported_surat_jalan_type"}))
	case errors.Is(err, usecase.ErrWarehouseServiceUnavailable):
		AbortWithError(c, NewHTTPError(http.StatusInternalServerError, err.Error(), model.WarehouseErrorDetail{Code: "warehouse_service_unavailable"}))
	default:
		AbortWithError(c, err)
	}
}
