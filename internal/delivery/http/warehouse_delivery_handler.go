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
	v1.POST("/inventory/receive", h.ReceiveInventory)
	v1.POST("/packing-lists", h.CreatePackingList)
	v1.POST("/surat-jalan/:type", h.CreateSuratJalan)
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

func (h *WarehouseDeliveryHandler) handleError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, usecase.ErrWarehouseValidation):
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, err.Error(), model.WarehouseErrorDetail{Code: "invalid_warehouse_payload"}))
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
