package httpdelivery

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"permatatex-inventory/internal/model"
	"permatatex-inventory/internal/usecase"
	"permatatex-inventory/pkg/response"
)

type RoleHandler struct {
	useCase *usecase.RoleUseCase
}

func NewRoleHandler(useCase *usecase.RoleUseCase) (*RoleHandler, error) {
	if useCase == nil {
		return nil, errors.New("role usecase is required")
	}

	return &RoleHandler{useCase: useCase}, nil
}

func (h *RoleHandler) RegisterRoutes(router gin.IRouter, authMiddleware gin.HandlerFunc) {
	group := router.Group("/api/v1/roles").Use(authMiddleware)

	group.GET("", RequirePermission(PermissionRoleRead), h.List)
	group.GET("/:id", RequirePermission(PermissionRoleRead), h.GetByID)
	group.POST("", RequirePermission(PermissionRoleCreate), h.Create)
	group.PUT("/:id", RequirePermission(PermissionRoleUpdate), h.Update)
	group.DELETE("/:id", RequirePermission(PermissionRoleDelete), h.Delete)
}

// List godoc
// @Summary      List Roles
// @Description  Returns a paginated list of roles.
// @Tags         Roles
// @Produce      json
// @Security     BearerAuth
// @Param        page    query     int     false  "Page (default 1)"
// @Param        limit   query     int     false  "Limit (default 20)"
// @Param        search  query     string  false  "Search by role name"
// @Success      200     {object}  model.RoleListSuccessDoc
// @Failure      400     {object}  model.LoginBadRequestDoc
// @Failure      401     {object}  model.GetMeUnauthorizedDoc
// @Failure      503     {object}  model.LoginServiceUnavailableDoc
// @Router       /api/v1/roles [get]
func (h *RoleHandler) List(c *gin.Context) {
	filter, err := parseListQuery(c, 20)
	if err != nil {
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, "invalid list query", nil))
		return
	}

	result, err := h.useCase.List(c.Request.Context(), filter)
	if err != nil {
		h.handleError(c, err)
		return
	}

	response.Success(c, http.StatusOK, "roles retrieved", result)
}

// GetByID godoc
// @Summary      Get Role Detail
// @Description  Returns a single role with assigned permissions.
// @Tags         Roles
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      int  true  "Role ID"
// @Success      200  {object}  model.RoleSuccessDoc
// @Failure      400  {object}  model.LoginBadRequestDoc
// @Failure      401  {object}  model.GetMeUnauthorizedDoc
// @Failure      404  {object}  model.LoginBadRequestDoc
// @Failure      503  {object}  model.LoginServiceUnavailableDoc
// @Router       /api/v1/roles/{id} [get]
func (h *RoleHandler) GetByID(c *gin.Context) {
	id, err := parsePathInt32(c, "id")
	if err != nil {
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, "invalid role id", nil))
		return
	}

	result, err := h.useCase.GetByID(c.Request.Context(), id)
	if err != nil {
		h.handleError(c, err)
		return
	}

	response.Success(c, http.StatusOK, "role retrieved", result)
}

// Create godoc
// @Summary      Create Role
// @Description  Creates a new role and assigns permission overrides for the role.
// @Tags         Roles
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        payload  body      model.CreateRoleRequest  true  "Create role payload"
// @Success      201      {object}  model.RoleSuccessDoc
// @Failure      400      {object}  model.LoginBadRequestDoc
// @Failure      401      {object}  model.GetMeUnauthorizedDoc
// @Failure      409      {object}  model.LoginBadRequestDoc
// @Failure      503      {object}  model.LoginServiceUnavailableDoc
// @Router       /api/v1/roles [post]
func (h *RoleHandler) Create(c *gin.Context) {
	var req model.CreateRoleRequest
	if !BindJSON(c, &req) {
		return
	}

	result, err := h.useCase.Create(c.Request.Context(), req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	response.Success(c, http.StatusCreated, "role created", result)
}

// Update godoc
// @Summary      Update Role
// @Description  Updates role metadata and replaces assigned permissions.
// @Tags         Roles
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id       path      int                    true  "Role ID"
// @Param        payload  body      model.UpdateRoleRequest true  "Update role payload"
// @Success      200      {object}  model.RoleSuccessDoc
// @Failure      400      {object}  model.LoginBadRequestDoc
// @Failure      401      {object}  model.GetMeUnauthorizedDoc
// @Failure      404      {object}  model.LoginBadRequestDoc
// @Failure      409      {object}  model.LoginBadRequestDoc
// @Failure      503      {object}  model.LoginServiceUnavailableDoc
// @Router       /api/v1/roles/{id} [put]
func (h *RoleHandler) Update(c *gin.Context) {
	id, err := parsePathInt32(c, "id")
	if err != nil {
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, "invalid role id", nil))
		return
	}

	var req model.UpdateRoleRequest
	if !BindJSON(c, &req) {
		return
	}

	result, err := h.useCase.Update(c.Request.Context(), id, req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	response.Success(c, http.StatusOK, "role updated", result)
}

// Delete godoc
// @Summary      Delete Role
// @Description  Deletes a role if it is not currently assigned to users.
// @Tags         Roles
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      int  true  "Role ID"
// @Success      200  {object}  response.BaseResponse
// @Failure      400  {object}  model.LoginBadRequestDoc
// @Failure      401  {object}  model.GetMeUnauthorizedDoc
// @Failure      404  {object}  model.LoginBadRequestDoc
// @Failure      409  {object}  model.LoginBadRequestDoc
// @Failure      503  {object}  model.LoginServiceUnavailableDoc
// @Router       /api/v1/roles/{id} [delete]
func (h *RoleHandler) Delete(c *gin.Context) {
	id, err := parsePathInt32(c, "id")
	if err != nil {
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, "invalid role id", nil))
		return
	}

	if err := h.useCase.Delete(c.Request.Context(), id); err != nil {
		h.handleError(c, err)
		return
	}

	response.Success(c, http.StatusOK, "role deleted", nil)
}

func (h *RoleHandler) handleError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, usecase.ErrRoleValidation):
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, err.Error(), nil))
	case errors.Is(err, usecase.ErrRoleManagementNotFound):
		AbortWithError(c, NewHTTPError(http.StatusNotFound, err.Error(), nil))
	case errors.Is(err, usecase.ErrReservedRoleProtected):
		AbortWithError(c, NewHTTPError(http.StatusForbidden, err.Error(), nil))
	case errors.Is(err, usecase.ErrRoleNameAlreadyExists), errors.Is(err, usecase.ErrRoleInUse):
		AbortWithError(c, NewHTTPError(http.StatusConflict, err.Error(), nil))
	case errors.Is(err, usecase.ErrRoleServiceUnavailable):
		AbortWithError(c, NewHTTPError(http.StatusServiceUnavailable, err.Error(), nil))
	default:
		AbortWithError(c, err)
	}
}
