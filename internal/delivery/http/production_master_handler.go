package httpdelivery

import (
	"net/http"

	"permatatex-inventory/internal/model"
	"permatatex-inventory/internal/usecase"
	"permatatex-inventory/pkg/response"

	"github.com/gin-gonic/gin"
)

type ProductionMasterHandler struct {
	usecase *usecase.ProductionMasterUseCase
}

func NewProductionMasterHandler(u *usecase.ProductionMasterUseCase) (*ProductionMasterHandler, error) {
	return &ProductionMasterHandler{
		usecase: u,
	}, nil
}

// PRODUCTION LINE

func (h *ProductionMasterHandler) GetProductionLineByID(c *gin.Context) {
	id, err := parsePathInt32(c, "id")
	if err != nil {
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, "invalid id", nil))
		return
	}

	item, err := h.usecase.GetProductionLineByID(c.Request.Context(), id)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, response.MessageInternalServerError, err.Error())
		return
	}
	response.Success(c, http.StatusOK, "production line retrieved", item)
}

func (h *ProductionMasterHandler) ListProductionLines(c *gin.Context) {
	filter, err := parseListQuery(c, 20)
	if err != nil {
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, "invalid list query", nil))
		return
	}

	items, total, err := h.usecase.ListProductionLines(c.Request.Context(), filter)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, response.MessageInternalServerError, err.Error())
		return
	}
	setTotalCountHeader(c, total)
	response.Success(c, http.StatusOK, "production lines retrieved", items)
}

func (h *ProductionMasterHandler) CreateProductionLine(c *gin.Context) {
	var req model.CreateProductionLineRequest
	if !BindJSON(c, &req) {
		return
	}

	item, err := h.usecase.CreateProductionLine(c.Request.Context(), req)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, response.MessageInternalServerError, err.Error())
		return
	}
	response.Success(c, http.StatusCreated, "production line created", item)
}

func (h *ProductionMasterHandler) UpdateProductionLine(c *gin.Context) {
	id, err := parsePathInt32(c, "id")
	if err != nil {
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, "invalid id", nil))
		return
	}

	var req model.UpdateProductionLineRequest
	if !BindJSON(c, &req) {
		return
	}

	item, err := h.usecase.UpdateProductionLine(c.Request.Context(), id, req)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, response.MessageInternalServerError, err.Error())
		return
	}
	response.Success(c, http.StatusOK, "production line updated", item)
}

func (h *ProductionMasterHandler) DeleteProductionLine(c *gin.Context) {
	id, err := parsePathInt32(c, "id")
	if err != nil {
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, "invalid id", nil))
		return
	}

	if err := h.usecase.DeleteProductionLine(c.Request.Context(), id); err != nil {
		response.Fail(c, http.StatusInternalServerError, response.MessageInternalServerError, err.Error())
		return
	}
	response.Success(c, http.StatusOK, "production line deleted", nil)
}

// PRODUCTION STATUS PLAN

func (h *ProductionMasterHandler) GetProductionStatusPlanByID(c *gin.Context) {
	id, err := parsePathInt32(c, "id")
	if err != nil {
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, "invalid id", nil))
		return
	}

	item, err := h.usecase.GetProductionStatusPlanByID(c.Request.Context(), id)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, response.MessageInternalServerError, err.Error())
		return
	}
	response.Success(c, http.StatusOK, "production status plan retrieved", item)
}

func (h *ProductionMasterHandler) ListProductionStatusPlans(c *gin.Context) {
	filter, err := parseListQuery(c, 20)
	if err != nil {
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, "invalid list query", nil))
		return
	}

	items, total, err := h.usecase.ListProductionStatusPlans(c.Request.Context(), filter)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, response.MessageInternalServerError, err.Error())
		return
	}
	setTotalCountHeader(c, total)
	response.Success(c, http.StatusOK, "production status plans retrieved", items)
}

func (h *ProductionMasterHandler) CreateProductionStatusPlan(c *gin.Context) {
	var req model.CreateProductionStatusPlanRequest
	if !BindJSON(c, &req) {
		return
	}

	item, err := h.usecase.CreateProductionStatusPlan(c.Request.Context(), req)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, response.MessageInternalServerError, err.Error())
		return
	}
	response.Success(c, http.StatusCreated, "production status plan created", item)
}

func (h *ProductionMasterHandler) UpdateProductionStatusPlan(c *gin.Context) {
	id, err := parsePathInt32(c, "id")
	if err != nil {
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, "invalid id", nil))
		return
	}

	var req model.UpdateProductionStatusPlanRequest
	if !BindJSON(c, &req) {
		return
	}

	item, err := h.usecase.UpdateProductionStatusPlan(c.Request.Context(), id, req)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, response.MessageInternalServerError, err.Error())
		return
	}
	response.Success(c, http.StatusOK, "production status plan updated", item)
}

func (h *ProductionMasterHandler) DeleteProductionStatusPlan(c *gin.Context) {
	id, err := parsePathInt32(c, "id")
	if err != nil {
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, "invalid id", nil))
		return
	}

	if err := h.usecase.DeleteProductionStatusPlan(c.Request.Context(), id); err != nil {
		response.Fail(c, http.StatusInternalServerError, response.MessageInternalServerError, err.Error())
		return
	}
	response.Success(c, http.StatusOK, "production status plan deleted", nil)
}
