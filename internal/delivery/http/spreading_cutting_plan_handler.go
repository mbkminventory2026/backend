package httpdelivery

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"permatatex-inventory/internal/model"
	"permatatex-inventory/internal/usecase"
	"permatatex-inventory/pkg/response"
)

type SpreadingCuttingPlanHandler struct {
	useCase *usecase.SpreadingCuttingPlanUseCase
}

func NewSpreadingCuttingPlanHandler(useCase *usecase.SpreadingCuttingPlanUseCase) (*SpreadingCuttingPlanHandler, error) {
	if useCase == nil {
		return nil, errors.New("spreading cutting plan usecase is required")
	}
	return &SpreadingCuttingPlanHandler{useCase: useCase}, nil
}

func (h *SpreadingCuttingPlanHandler) RegisterRoutes(router gin.IRouter, authMiddleware gin.HandlerFunc) {
	v1 := router.Group("/api/v1").Use(authMiddleware)

	v1.POST("/spreading-cutting-plans", RequirePermission(PermissionCuttingPlanCreate), h.CreateSpreadingCuttingPlan)
	v1.GET("/spreading-cutting-plans", RequirePermission(PermissionCuttingPlanRead), h.ListSpreadingCuttingPlans)
	v1.GET("/spreading-cutting-plans/:id", RequirePermission(PermissionCuttingPlanRead), h.GetSpreadingCuttingPlan)
}

// CreateSpreadingCuttingPlan godoc
// @Summary      Create Spreading Cutting Plan
// @Description  Create a spreading cutting plan with components, ratios, and ratio size markers in a single transaction.
// @Tags         Spreading Cutting Plan
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        payload  body      model.CreateSpreadingCuttingPlanRequest  true  "Spreading cutting plan payload"
// @Success      201      {object}  model.SpreadingCuttingPlanSuccessDoc
// @Failure      400      {object}  model.SpreadingCuttingPlanValidationErrorDoc
// @Failure      500      {object}  model.SpreadingCuttingPlanErrorDoc
// @Router       /api/v1/spreading-cutting-plans [post]
func (h *SpreadingCuttingPlanHandler) CreateSpreadingCuttingPlan(c *gin.Context) {
	userID, ok := GetUserIDFromContext(c)
	if !ok {
		AbortWithError(c, NewHTTPError(http.StatusUnauthorized, "unauthorized", nil))
		return
	}

	var req model.CreateSpreadingCuttingPlanRequest
	if !BindJSON(c, &req) {
		return
	}

	item, err := h.useCase.CreateSpreadingCuttingPlan(c.Request.Context(), userID, req)
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusCreated, "spreading cutting plan created", item)
}

// GetSpreadingCuttingPlan godoc
// @Summary      Get Spreading Cutting Plan Detail
// @Description  Returns a single spreading cutting plan with components, ratios, and size breakdown.
// @Tags         Spreading Cutting Plan
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      int  true  "Spreading Cutting Plan ID"
// @Success      200  {object}  model.SpreadingCuttingPlanSuccessDoc
// @Failure      400  {object}  model.SpreadingCuttingPlanErrorDoc
// @Failure      404  {object}  model.SpreadingCuttingPlanErrorDoc
// @Failure      500  {object}  model.SpreadingCuttingPlanErrorDoc
// @Router       /api/v1/spreading-cutting-plans/{id} [get]
func (h *SpreadingCuttingPlanHandler) GetSpreadingCuttingPlan(c *gin.Context) {
	id, err := parsePathInt32(c, "id")
	if err != nil {
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, "invalid spreading cutting plan id", nil))
		return
	}

	item, err := h.useCase.GetSpreadingCuttingPlan(c.Request.Context(), id)
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "spreading cutting plan retrieved", item)
}

// ListSpreadingCuttingPlans godoc
// @Summary      List Spreading Cutting Plans
// @Description  Returns a paginated list of spreading cutting plans.
// @Tags         Spreading Cutting Plan
// @Produce      json
// @Security     BearerAuth
// @Param        page    query     int     false  "Page (default 1)"
// @Param        limit   query     int     false  "Limit (default 20)"
// @Param        search  query     string  false  "Search by document number, model, buyer"
// @Success      200     {object}  model.SpreadingCuttingPlanListSuccessDoc
// @Failure      400     {object}  model.SpreadingCuttingPlanErrorDoc
// @Failure      500     {object}  model.SpreadingCuttingPlanErrorDoc
// @Router       /api/v1/spreading-cutting-plans [get]
func (h *SpreadingCuttingPlanHandler) ListSpreadingCuttingPlans(c *gin.Context) {
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

	item, err := h.useCase.ListSpreadingCuttingPlans(c.Request.Context(), model.TransactionListFilter{
		ListQueryFilter: filter,
		IDMitra:         mitraID,
	})
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "spreading cutting plans retrieved", item)
}

func (h *SpreadingCuttingPlanHandler) handleError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, usecase.ErrSpreadingCuttingPlanValidation):
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, err.Error(), model.SpreadingCuttingPlanErrorDetail{Code: "invalid_spreading_cutting_plan_payload"}))
	case errors.Is(err, usecase.ErrSpreadingCuttingPlanNotFound):
		AbortWithError(c, NewHTTPError(http.StatusNotFound, err.Error(), model.SpreadingCuttingPlanErrorDetail{Code: "spreading_cutting_plan_not_found"}))
	case errors.Is(err, usecase.ErrSpreadingCuttingPlanReferenceNotFound):
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, err.Error(), model.SpreadingCuttingPlanErrorDetail{Code: "related_data_not_found"}))
	default:
		AbortWithError(c, err)
	}
}
