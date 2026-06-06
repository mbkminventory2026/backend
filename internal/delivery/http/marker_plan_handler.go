package httpdelivery

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"permatatex-inventory/internal/model"
	"permatatex-inventory/internal/usecase"
	"permatatex-inventory/pkg/response"
)

type MarkerPlanHandler struct {
	useCase *usecase.MarkerPlanUseCase
}

func NewMarkerPlanHandler(useCase *usecase.MarkerPlanUseCase) (*MarkerPlanHandler, error) {
	if useCase == nil {
		return nil, errors.New("marker plan usecase is required")
	}
	return &MarkerPlanHandler{useCase: useCase}, nil
}

func (h *MarkerPlanHandler) RegisterRoutes(router gin.IRouter, authMiddleware gin.HandlerFunc) {
	v1 := router.Group("/api/v1").Use(authMiddleware)

	v1.POST("/marker-plans", RequirePermission(PermissionMarkerPlanCreate), h.CreateMarkerPlan)
	v1.GET("/marker-plans/:id", RequirePermission(PermissionMarkerPlanRead), h.GetMarkerPlan)
}

// CreateMarkerPlan godoc
// @Summary      Create Marker Plan
// @Description  Create a marker plan with components, ratios, and ratio size markers in a single transaction.
// @Tags         Marker Plan
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        payload  body      model.CreateMarkerPlanRequest  true  "Marker plan payload"
// @Success      201      {object}  model.MarkerPlanSuccessDoc
// @Failure      400      {object}  model.MarkerPlanValidationErrorDoc
// @Failure      500      {object}  model.MarkerPlanErrorDoc
// @Router       /api/v1/marker-plans [post]
func (h *MarkerPlanHandler) CreateMarkerPlan(c *gin.Context) {
	userID, ok := GetUserIDFromContext(c)
	if !ok {
		AbortWithError(c, NewHTTPError(http.StatusUnauthorized, "unauthorized", nil))
		return
	}

	var req model.CreateMarkerPlanRequest
	if !BindJSON(c, &req) {
		return
	}

	item, err := h.useCase.CreateMarkerPlan(c.Request.Context(), userID, req)
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusCreated, "marker plan created", item)
}

// GetMarkerPlan godoc
// @Summary      Get Marker Plan Detail
// @Description  Returns a single marker plan with components, ratios, and size breakdown.
// @Tags         Marker Plan
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      int  true  "Marker Plan ID"
// @Success      200  {object}  model.MarkerPlanSuccessDoc
// @Failure      400  {object}  model.MarkerPlanErrorDoc
// @Failure      404  {object}  model.MarkerPlanErrorDoc
// @Failure      500  {object}  model.MarkerPlanErrorDoc
// @Router       /api/v1/marker-plans/{id} [get]
func (h *MarkerPlanHandler) GetMarkerPlan(c *gin.Context) {
	id, err := parsePathInt32(c, "id")
	if err != nil {
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, "invalid marker plan id", nil))
		return
	}

	item, err := h.useCase.GetMarkerPlan(c.Request.Context(), id)
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "marker plan retrieved", item)
}

func (h *MarkerPlanHandler) handleError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, usecase.ErrMarkerPlanValidation):
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, err.Error(), model.MarkerPlanErrorDetail{Code: "invalid_marker_plan_payload"}))
	case errors.Is(err, usecase.ErrMarkerPlanNotFound):
		AbortWithError(c, NewHTTPError(http.StatusNotFound, err.Error(), model.MarkerPlanErrorDetail{Code: "marker_plan_not_found"}))
	case errors.Is(err, usecase.ErrMarkerPlanReferenceNotFound):
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, err.Error(), model.MarkerPlanErrorDetail{Code: "related_data_not_found"}))
	default:
		AbortWithError(c, err)
	}
}
