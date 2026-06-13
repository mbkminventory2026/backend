package httpdelivery

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"permatatex-inventory/internal/model"
	"permatatex-inventory/internal/usecase"
	"permatatex-inventory/pkg/response"
)

type DataApproveCuttingPlanHandler struct {
	useCase *usecase.DataApproveCuttingPlanUseCase
}

func NewDataApproveCuttingPlanHandler(uc *usecase.DataApproveCuttingPlanUseCase) (*DataApproveCuttingPlanHandler, error) {
	if uc == nil {
		return nil, errors.New("data approve cutting plan usecase is required")
	}
	return &DataApproveCuttingPlanHandler{useCase: uc}, nil
}

func (h *DataApproveCuttingPlanHandler) RegisterRoutes(router gin.IRouter, authMiddleware gin.HandlerFunc) {
	v1 := router.Group("/api/v1").Use(authMiddleware)

	v1.POST("/data-approve-cutting-plans", RequirePermission(PermissionDataApproveCuttingPlanCreate), h.Create)
	v1.GET("/data-approve-cutting-plans", RequirePermission(PermissionDataApproveCuttingPlanRead), h.List)
	v1.GET("/data-approve-cutting-plans/:id", RequirePermission(PermissionDataApproveCuttingPlanRead), h.Get)
}

// Create godoc
// @Summary      Create Data Approve Cutting Plan
// @Description  Creates a new Data Approve Cutting Plan document linked to a Work Order and initializes the approval workflow.
// @Tags         Data Approve Cutting Plan
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        payload  body      model.CreateDataApproveCuttingPlanRequest  true  "Payload"
// @Success      201      {object}  model.DataApproveCuttingPlanSuccessDoc
// @Failure      400      {object}  model.DataApproveCuttingPlanValidationErrorDoc
// @Failure      500      {object}  model.DataApproveCuttingPlanErrorDoc
// @Router       /api/v1/data-approve-cutting-plans [post]
func (h *DataApproveCuttingPlanHandler) Create(c *gin.Context) {
	userID, ok := GetUserIDFromContext(c)
	if !ok {
		AbortWithError(c, NewHTTPError(http.StatusUnauthorized, "unauthorized", nil))
		return
	}

	var req model.CreateDataApproveCuttingPlanRequest
	if !BindJSON(c, &req) {
		return
	}

	item, err := h.useCase.CreateDataApproveCuttingPlan(c.Request.Context(), userID, req)
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusCreated, "data approve cutting plan created", item)
}

// Get godoc
// @Summary      Get Data Approve Cutting Plan Detail
// @Description  Returns a single Data Approve Cutting Plan with the aggregated size breakdown table.
// @Tags         Data Approve Cutting Plan
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      int  true  "Document ID"
// @Success      200  {object}  model.DataApproveCuttingPlanSuccessDoc
// @Failure      404  {object}  model.DataApproveCuttingPlanErrorDoc
// @Failure      500  {object}  model.DataApproveCuttingPlanErrorDoc
// @Router       /api/v1/data-approve-cutting-plans/{id} [get]
func (h *DataApproveCuttingPlanHandler) Get(c *gin.Context) {
	id, err := parsePathInt32(c, "id")
	if err != nil {
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, "invalid id", nil))
		return
	}

	item, err := h.useCase.GetDataApproveCuttingPlan(c.Request.Context(), id)
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "data approve cutting plan retrieved", item)
}

// List godoc
// @Summary      List Data Approve Cutting Plans
// @Description  Returns a paginated list of Data Approve Cutting Plan documents.
// @Tags         Data Approve Cutting Plan
// @Produce      json
// @Security     BearerAuth
// @Param        page    query     int     false  "Page (default 1)"
// @Param        limit   query     int     false  "Limit (default 20)"
// @Param        search  query     string  false  "Search by document number, buyer, model"
// @Success      200     {object}  model.DataApproveCuttingPlanListSuccessDoc
// @Failure      500     {object}  model.DataApproveCuttingPlanErrorDoc
// @Router       /api/v1/data-approve-cutting-plans [get]
func (h *DataApproveCuttingPlanHandler) List(c *gin.Context) {
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

	item, err := h.useCase.ListDataApproveCuttingPlans(c.Request.Context(), model.TransactionListFilter{
		ListQueryFilter: filter,
		IDMitra:         mitraID,
	})
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "data approve cutting plans retrieved", item)
}

func (h *DataApproveCuttingPlanHandler) handleError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, usecase.ErrDACPValidation):
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, err.Error(), model.DataApproveCuttingPlanErrorDetail{Code: "invalid_payload"}))
	case errors.Is(err, usecase.ErrDACPNotFound):
		AbortWithError(c, NewHTTPError(http.StatusNotFound, err.Error(), model.DataApproveCuttingPlanErrorDetail{Code: "not_found"}))
	case errors.Is(err, usecase.ErrDACPReferenceNotFound):
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, err.Error(), model.DataApproveCuttingPlanErrorDetail{Code: "reference_not_found"}))
	default:
		AbortWithError(c, err)
	}
}
