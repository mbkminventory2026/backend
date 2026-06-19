package httpdelivery

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"permatatex-inventory/internal/model"
	"permatatex-inventory/internal/usecase"
	"permatatex-inventory/pkg/response"
)

type MasterPlanHandler struct {
	useCase *usecase.MasterPlanUseCase
}

func NewMasterPlanHandler(useCase *usecase.MasterPlanUseCase) (*MasterPlanHandler, error) {
	if useCase == nil {
		return nil, errors.New("master plan usecase is required")
	}
	return &MasterPlanHandler{useCase: useCase}, nil
}

func (h *MasterPlanHandler) RegisterRoutes(router gin.IRouter, authMiddleware gin.HandlerFunc) {
	v1 := router.Group("/api/v1").Use(authMiddleware)

	v1.POST("/master-plans", RequirePermission(PermissionMasterPlanCreate), h.CreateMasterPlan)
	v1.GET("/master-plans", RequirePermission(PermissionMasterPlanRead), h.ListMasterPlans)
	v1.GET("/master-plans/:id", RequirePermission(PermissionMasterPlanRead), h.GetMasterPlan)
	v1.PUT("/master-plans/:id", RequirePermission(PermissionMasterPlanUpdate), h.UpdateMasterPlan)
	v1.DELETE("/master-plans/:id", RequirePermission(PermissionMasterPlanDelete), h.DeleteMasterPlan)

	v1.POST("/master-plans/:id/items", RequirePermission(PermissionMasterPlanUpdate), h.AddItem)
	v1.DELETE("/master-plans/:id/items/:itemId", RequirePermission(PermissionMasterPlanUpdate), h.RemoveItem)

	v1.PUT("/master-plans/:id/items/:itemId/target-harian", RequirePermission(PermissionMasterPlanUpdate), h.UpsertTargetHarian)
	v1.PUT("/master-plans/:id/items/:itemId/output-harian", RequirePermission(PermissionMasterPlanUpdate), h.UpsertOutputHarian)
	v1.PUT("/master-plans/:id/items/:itemId/target-proses", RequirePermission(PermissionMasterPlanUpdate), h.UpsertTargetProses)
	v1.DELETE("/master-plans/:id/items/:itemId/target-proses/:tanggal", RequirePermission(PermissionMasterPlanUpdate), h.DeleteTargetProses)
}

// CreateMasterPlan godoc
// @Summary      Create Master Plan
// @Tags         Master Plan
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        payload  body      model.CreateMasterPlanRequest  true  "Master plan payload"
// @Success      201      {object}  model.MasterPlanSuccessDoc
// @Failure      400      {object}  model.MasterPlanValidationErrorDoc
// @Failure      500      {object}  model.MasterPlanErrorDoc
// @Router       /api/v1/master-plans [post]
func (h *MasterPlanHandler) CreateMasterPlan(c *gin.Context) {
	userID, ok := GetUserIDFromContext(c)
	if !ok {
		AbortWithError(c, NewHTTPError(http.StatusUnauthorized, "unauthorized", nil))
		return
	}

	var req model.CreateMasterPlanRequest
	if !BindJSON(c, &req) {
		return
	}

	item, err := h.useCase.CreateMasterPlan(c.Request.Context(), userID, req)
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusCreated, "master plan created", item)
}

// GetMasterPlan godoc
// @Summary      Get Master Plan
// @Tags         Master Plan
// @Produce      json
// @Security     BearerAuth
// @Param        id  path  int  true  "Master Plan ID"
// @Success      200  {object}  model.MasterPlanSuccessDoc
// @Failure      404  {object}  model.MasterPlanErrorDoc
// @Router       /api/v1/master-plans/{id} [get]
func (h *MasterPlanHandler) GetMasterPlan(c *gin.Context) {
	id, err := parsePathInt32(c, "id")
	if err != nil {
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, "invalid master plan id", nil))
		return
	}

	item, err := h.useCase.GetMasterPlan(c.Request.Context(), id)
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "master plan retrieved", item)
}

// ListMasterPlans godoc
// @Summary      List Master Plans
// @Tags         Master Plan
// @Produce      json
// @Security     BearerAuth
// @Param        page    query  int     false  "Page"
// @Param        limit   query  int     false  "Limit"
// @Param        search  query  string  false  "Search by nama, departemen, or line"
// @Success      200  {object}  model.MasterPlanListSuccessDoc
// @Router       /api/v1/master-plans [get]
func (h *MasterPlanHandler) ListMasterPlans(c *gin.Context) {
	filter, err := parseListQuery(c, 20)
	if err != nil {
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, "invalid list query", nil))
		return
	}

	result, err := h.useCase.ListMasterPlans(c.Request.Context(), filter)
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "master plans retrieved", result)
}

// UpdateMasterPlan godoc
// @Summary      Update Master Plan
// @Tags         Master Plan
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id       path  int                             true  "Master Plan ID"
// @Param        payload  body  model.UpdateMasterPlanRequest   true  "Update payload"
// @Success      200  {object}  model.MasterPlanSuccessDoc
// @Router       /api/v1/master-plans/{id} [put]
func (h *MasterPlanHandler) UpdateMasterPlan(c *gin.Context) {
	id, err := parsePathInt32(c, "id")
	if err != nil {
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, "invalid master plan id", nil))
		return
	}

	var req model.UpdateMasterPlanRequest
	if !BindJSON(c, &req) {
		return
	}

	item, err := h.useCase.UpdateMasterPlan(c.Request.Context(), id, req)
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "master plan updated", item)
}

// DeleteMasterPlan godoc
// @Summary      Delete Master Plan
// @Tags         Master Plan
// @Produce      json
// @Security     BearerAuth
// @Param        id  path  int  true  "Master Plan ID"
// @Success      200  {object}  model.MasterPlanSuccessDoc
// @Router       /api/v1/master-plans/{id} [delete]
func (h *MasterPlanHandler) DeleteMasterPlan(c *gin.Context) {
	id, err := parsePathInt32(c, "id")
	if err != nil {
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, "invalid master plan id", nil))
		return
	}

	if err := h.useCase.DeleteMasterPlan(c.Request.Context(), id); err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "master plan deleted", nil)
}

// AddItem godoc
// @Summary      Add WO item to Master Plan
// @Tags         Master Plan
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id       path  int                              true  "Master Plan ID"
// @Param        payload  body  model.AddMasterPlanItemRequest   true  "Item payload"
// @Success      201  {object}  model.MasterPlanSuccessDoc
// @Router       /api/v1/master-plans/{id}/items [post]
func (h *MasterPlanHandler) AddItem(c *gin.Context) {
	planID, err := parsePathInt32(c, "id")
	if err != nil {
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, "invalid master plan id", nil))
		return
	}

	var req model.AddMasterPlanItemRequest
	if !BindJSON(c, &req) {
		return
	}

	item, err := h.useCase.AddItem(c.Request.Context(), planID, req)
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusCreated, "item added", item)
}

// RemoveItem godoc
// @Summary      Remove WO item from Master Plan
// @Tags         Master Plan
// @Produce      json
// @Security     BearerAuth
// @Param        id      path  int  true  "Master Plan ID"
// @Param        itemId  path  int  true  "Item ID"
// @Success      200  {object}  model.MasterPlanSuccessDoc
// @Router       /api/v1/master-plans/{id}/items/{itemId} [delete]
func (h *MasterPlanHandler) RemoveItem(c *gin.Context) {
	planID, err := parsePathInt32(c, "id")
	if err != nil {
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, "invalid master plan id", nil))
		return
	}
	itemID, err := parsePathInt32(c, "itemId")
	if err != nil {
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, "invalid item id", nil))
		return
	}

	if err := h.useCase.RemoveItem(c.Request.Context(), planID, itemID); err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "item removed", nil)
}

// UpsertTargetHarian godoc
// @Summary      Set daily targets for a Master Plan item
// @Tags         Master Plan
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id      path  int                               true  "Master Plan ID"
// @Param        itemId  path  int                               true  "Item ID"
// @Param        payload body  model.UpsertTargetHarianRequest   true  "Target harian payload"
// @Success      200  {object}  model.MasterPlanSuccessDoc
// @Router       /api/v1/master-plans/{id}/items/{itemId}/target-harian [put]
func (h *MasterPlanHandler) UpsertTargetHarian(c *gin.Context) {
	planID, itemID, ok := h.parsePlanAndItemID(c)
	if !ok {
		return
	}

	var req model.UpsertTargetHarianRequest
	if !BindJSON(c, &req) {
		return
	}

	if err := h.useCase.UpsertTargetHarian(c.Request.Context(), planID, itemID, req); err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "target harian updated", nil)
}

// UpsertOutputHarian godoc
// @Summary      Set daily output for a Master Plan item
// @Tags         Master Plan
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id      path  int                                true  "Master Plan ID"
// @Param        itemId  path  int                                true  "Item ID"
// @Param        payload body  model.UpsertOutputHarianRequest    true  "Output harian payload"
// @Success      200  {object}  model.MasterPlanSuccessDoc
// @Router       /api/v1/master-plans/{id}/items/{itemId}/output-harian [put]
func (h *MasterPlanHandler) UpsertOutputHarian(c *gin.Context) {
	planID, itemID, ok := h.parsePlanAndItemID(c)
	if !ok {
		return
	}

	var req model.UpsertOutputHarianRequest
	if !BindJSON(c, &req) {
		return
	}

	if err := h.useCase.UpsertOutputHarian(c.Request.Context(), planID, itemID, req); err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "output harian updated", nil)
}

// UpsertTargetProses godoc
// @Summary      Set a process milestone for a Master Plan item on a specific date
// @Tags         Master Plan
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id      path  int                               true  "Master Plan ID"
// @Param        itemId  path  int                               true  "Item ID"
// @Param        payload body  model.UpsertTargetProsesRequest   true  "Target proses payload"
// @Success      200  {object}  model.MasterPlanSuccessDoc
// @Router       /api/v1/master-plans/{id}/items/{itemId}/target-proses [put]
func (h *MasterPlanHandler) UpsertTargetProses(c *gin.Context) {
	planID, itemID, ok := h.parsePlanAndItemID(c)
	if !ok {
		return
	}

	var req model.UpsertTargetProsesRequest
	if !BindJSON(c, &req) {
		return
	}

	if err := h.useCase.UpsertTargetProses(c.Request.Context(), planID, itemID, req); err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "target proses updated", nil)
}

// DeleteTargetProses godoc
// @Summary      Remove a process milestone from a Master Plan item
// @Tags         Master Plan
// @Produce      json
// @Security     BearerAuth
// @Param        id       path  int     true  "Master Plan ID"
// @Param        itemId   path  int     true  "Item ID"
// @Param        tanggal  path  string  true  "Date (YYYY-MM-DD)"
// @Success      200  {object}  model.MasterPlanSuccessDoc
// @Router       /api/v1/master-plans/{id}/items/{itemId}/target-proses/{tanggal} [delete]
func (h *MasterPlanHandler) DeleteTargetProses(c *gin.Context) {
	planID, itemID, ok := h.parsePlanAndItemID(c)
	if !ok {
		return
	}
	tanggal := c.Param("tanggal")

	if err := h.useCase.DeleteTargetProses(c.Request.Context(), planID, itemID, tanggal); err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "target proses deleted", nil)
}

func (h *MasterPlanHandler) parsePlanAndItemID(c *gin.Context) (planID, itemID int32, ok bool) {
	var err error
	planID, err = parsePathInt32(c, "id")
	if err != nil {
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, "invalid master plan id", nil))
		return 0, 0, false
	}
	itemID, err = parsePathInt32(c, "itemId")
	if err != nil {
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, "invalid item id", nil))
		return 0, 0, false
	}
	return planID, itemID, true
}

func (h *MasterPlanHandler) handleError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, usecase.ErrMasterPlanValidation):
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, err.Error(), model.MasterPlanErrorDetail{Code: "invalid_master_plan_payload"}))
	case errors.Is(err, usecase.ErrMasterPlanNotFound):
		AbortWithError(c, NewHTTPError(http.StatusNotFound, err.Error(), model.MasterPlanErrorDetail{Code: "master_plan_not_found"}))
	case errors.Is(err, usecase.ErrMasterPlanItemNotFound):
		AbortWithError(c, NewHTTPError(http.StatusNotFound, err.Error(), model.MasterPlanErrorDetail{Code: "master_plan_item_not_found"}))
	case errors.Is(err, usecase.ErrMasterPlanReferenceNotFound):
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, err.Error(), model.MasterPlanErrorDetail{Code: "related_data_not_found"}))
	case errors.Is(err, usecase.ErrMasterPlanDuplicate):
		AbortWithError(c, NewHTTPError(http.StatusConflict, err.Error(), model.MasterPlanErrorDetail{Code: "duplicate_entry"}))
	default:
		AbortWithError(c, err)
	}
}
