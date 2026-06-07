package httpdelivery

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"permatatex-inventory/internal/model"
	"permatatex-inventory/internal/usecase"
	"permatatex-inventory/pkg/response"
)

type TimelineProduksiHandler struct {
	useCase *usecase.TimelineProduksiUseCase
}

func NewTimelineProduksiHandler(useCase *usecase.TimelineProduksiUseCase) (*TimelineProduksiHandler, error) {
	if useCase == nil {
		return nil, errors.New("timeline produksi usecase is required")
	}
	return &TimelineProduksiHandler{useCase: useCase}, nil
}

func (h *TimelineProduksiHandler) RegisterRoutes(router gin.IRouter, authMiddleware gin.HandlerFunc) {
	v1 := router.Group("/api/v1").Use(authMiddleware, RequireInternalUser())

	v1.POST("/timeline-plans", RequirePermission(PermissionTimelineCreate), h.CreateTimelinePlan)
	v1.GET("/timeline-plans", RequirePermission(PermissionTimelineRead), h.GetTimelinePlans)
	v1.GET("/timeline-plans/:id", RequirePermission(PermissionTimelineRead), h.GetTimelinePlan)
	v1.PATCH("/timeline-plans/wo-shell-plans/:id/status", RequirePermission(PermissionTimelineUpdate), h.UpdateWOShellPlanStatus)
}

// GetTimelinePlans godoc
// @Summary      Get List of Timeline Plans
// @Description  Get a paginated list of timeline plans
// @Tags         Timeline & Production Plan
// @Produce      json
// @Security     BearerAuth
// @Param        page    query     int     false  "Page number" default(1)
// @Param        limit   query     int     false  "Number of items per page" default(10)
// @Param        search  query     string  false  "Search term"
// @Param        sort    query     string  false  "Sort by column" default(created_at)
// @Param        desc    query     bool    false  "Sort descending" default(true)
// @Success      200     {object}  model.TimelinePlanListSuccessDoc
// @Failure      400     {object}  model.TimelinePlanErrorDoc
// @Failure      500     {object}  model.TimelinePlanErrorDoc
// @Router       /api/v1/timeline-plans [get]
func (h *TimelineProduksiHandler) GetTimelinePlans(c *gin.Context) {
	filter, err := parseListQuery(c, 10)
	if err != nil {
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, "invalid list query parameter", nil))
		return
	}

	result, err := h.useCase.GetTimelinePlans(c.Request.Context(), filter)
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "timeline plans retrieved", result)
}

// CreateTimelinePlan godoc
// @Summary      Create Timeline Plan
// @Description  Create a production timeline plan with multiple wo shell plans.
// @Tags         Timeline & Production Plan
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        payload  body      model.CreateTimelinePlanRequest  true  "Timeline plan payload"
// @Success      201      {object}  model.TimelinePlanSuccessDoc
// @Failure      400      {object}  model.TimelinePlanValidationErrorDoc
// @Failure      500      {object}  model.TimelinePlanErrorDoc
// @Router       /api/v1/timeline-plans [post]
func (h *TimelineProduksiHandler) CreateTimelinePlan(c *gin.Context) {
	userID, ok := GetUserIDFromContext(c)
	if !ok {
		AbortWithError(c, NewHTTPError(http.StatusUnauthorized, "unauthorized", nil))
		return
	}

	var req model.CreateTimelinePlanRequest
	if !BindJSON(c, &req) {
		return
	}

	item, err := h.useCase.CreateTimelinePlan(c.Request.Context(), userID, req)
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusCreated, "timeline plan created", item)
}

// GetTimelinePlan godoc
// @Summary      Get Timeline Plan Detail
// @Description  Returns a single timeline plan with its wo shell plans.
// @Tags         Timeline & Production Plan
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      int  true  "Timeline ID"
// @Success      200  {object}  model.TimelinePlanSuccessDoc
// @Failure      400  {object}  model.TimelinePlanErrorDoc
// @Failure      404  {object}  model.TimelinePlanErrorDoc
// @Failure      500  {object}  model.TimelinePlanErrorDoc
// @Router       /api/v1/timeline-plans/{id} [get]
func (h *TimelineProduksiHandler) GetTimelinePlan(c *gin.Context) {
	id, err := parsePathInt32(c, "id")
	if err != nil {
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, "invalid timeline id", nil))
		return
	}

	item, err := h.useCase.GetTimelinePlan(c.Request.Context(), id)
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "timeline plan retrieved", item)
}

// UpdateWOShellPlanStatus godoc
// @Summary      Update WO Shell Plan Status
// @Description  Updates progress status for a specific work order shell plan (cutting, embroidery, sewing, finishing/packing).
// @Tags         Timeline & Production Plan
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id       path      int                                   true  "WO Shell Plan ID"
// @Param        payload  body      model.UpdateWOShellPlanStatusRequest  true  "Statuses update payload"
// @Success      200      {object}  model.UpdateWOShellPlanStatusSuccessDoc
// @Failure      400      {object}  model.TimelinePlanErrorDoc
// @Failure      404      {object}  model.TimelinePlanErrorDoc
// @Failure      500      {object}  model.TimelinePlanErrorDoc
// @Router       /api/v1/timeline-plans/wo-shell-plans/{id}/status [patch]
func (h *TimelineProduksiHandler) UpdateWOShellPlanStatus(c *gin.Context) {
	id, err := parsePathInt32(c, "id")
	if err != nil {
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, "invalid wo shell plan id", nil))
		return
	}

	var req model.UpdateWOShellPlanStatusRequest
	if !BindJSON(c, &req) {
		return
	}

	err = h.useCase.UpdateWOShellPlanStatus(c.Request.Context(), id, req)
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "wo shell plan status updated", nil)
}

func (h *TimelineProduksiHandler) handleError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, usecase.ErrTimelinePlanValidation):
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, err.Error(), model.TimelinePlanErrorDetail{Code: "invalid_timeline_payload"}))
	case errors.Is(err, usecase.ErrTimelinePlanNotFound):
		AbortWithError(c, NewHTTPError(http.StatusNotFound, err.Error(), model.TimelinePlanErrorDetail{Code: "timeline_plan_not_found"}))
	case errors.Is(err, usecase.ErrWOShellPlanNotFound):
		AbortWithError(c, NewHTTPError(http.StatusNotFound, err.Error(), model.TimelinePlanErrorDetail{Code: "wo_shell_plan_not_found"}))
	case errors.Is(err, usecase.ErrTimelinePlanReferenceNotFound):
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, err.Error(), model.TimelinePlanErrorDetail{Code: "related_data_not_found"}))
	default:
		AbortWithError(c, err)
	}
}
