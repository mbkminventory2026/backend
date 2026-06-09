package httpdelivery

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"permatatex-inventory/internal/model"
	"permatatex-inventory/internal/usecase"
	"permatatex-inventory/pkg/response"
)

type UserHandler struct {
	useCase *usecase.UserUseCase
}

func NewUserHandler(useCase *usecase.UserUseCase) (*UserHandler, error) {
	if useCase == nil {
		return nil, errors.New("user usecase is required")
	}

	return &UserHandler{
		useCase: useCase,
	}, nil
}

func (h *UserHandler) RegisterRoutes(router gin.IRouter, authMiddleware gin.HandlerFunc) {
	group := router.Group("/api/v1/users").Use(authMiddleware, RequireInternalUser())

	group.POST("", RequirePermission(PermissionUserCreate), h.Create)
	group.GET("", RequirePermission(PermissionUserRead), h.List)
	group.GET("/:id", RequirePermission(PermissionUserRead), h.GetByID)
	group.PUT("/:id", RequirePermission(PermissionUserUpdate), h.Update)
	group.PUT("/:id/role", RequirePermission(PermissionUserRoleAssign), h.AssignRole)
	group.PUT("/:id/permissions", RequirePermission(PermissionUserUpdate), h.AssignPermissions)
	group.PUT("/:id/approve", RequirePermission(PermissionUserApprove), h.Approve)
	group.PUT("/:id/reject", RequirePermission(PermissionUserApprove), h.Reject)
	group.DELETE("/:id", RequirePermission(PermissionUserDelete), h.Delete)
}

// Create godoc
// @Summary      Create New User
// @Description  Creates a new user and assigns permissions in a single transaction.
// @Tags         Users
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        payload  body      model.CreateUserRequest  true  "Create user payload"
// @Success      201      {object}  model.UserSuccessDoc
// @Failure      400      {object}  model.LoginBadRequestDoc
// @Failure      409      {object}  model.LoginBadRequestDoc
// @Failure      401      {object}  model.GetMeUnauthorizedDoc
// @Failure      503      {object}  model.LoginServiceUnavailableDoc
// @Router       /api/v1/users [post]
func (h *UserHandler) Create(c *gin.Context) {
	var req model.CreateUserRequest
	if !BindJSON(c, &req) {
		return
	}

	var actorUserID *int32
	if userID, ok := GetUserIDFromContext(c); ok {
		actorUserID = &userID
	}

	result, err := h.useCase.Create(withAuditLogContext(c), actorUserID, req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	response.Success(c, http.StatusCreated, "user created", result)
}

// List godoc
// @Summary      List Users
// @Description  Returns a paginated list of users with their department and mitra names.
// @Tags         Users
// @Produce      json
// @Security     BearerAuth
// @Param        limit   query     int  false  "Limit (default 20)"
// @Param        offset  query     int  false  "Offset (default 0)"
// @Success      200     {object}  model.UserListSuccessDoc
// @Failure      401     {object}  model.GetMeUnauthorizedDoc
// @Router       /api/v1/users [get]
func (h *UserHandler) List(c *gin.Context) {
	filter, err := parseListQuery(c, 20)
	if err != nil {
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, "invalid list query", nil))
		return
	}

	result, total, err := h.useCase.List(c.Request.Context(), model.ListUsersFilter{ListQueryFilter: filter})
	if err != nil {
		h.handleError(c, err)
		return
	}

	setTotalCountHeader(c, total)
	response.Success(c, http.StatusOK, "users retrieved", result)
}

// GetByID godoc
// @Summary      Get User Detail
// @Description  Returns a single user detail with permissions.
// @Tags         Users
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      int  true  "User ID"
// @Success      200  {object}  model.UserSuccessDoc
// @Failure      401  {object}  model.GetMeUnauthorizedDoc
// @Failure      404  {object}  model.LoginBadRequestDoc
// @Router       /api/v1/users/{id} [get]
func (h *UserHandler) GetByID(c *gin.Context) {
	id, err := parsePathInt32(c, "id")
	if err != nil {
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, "invalid user id", nil))
		return
	}

	result, err := h.useCase.GetByID(c.Request.Context(), id)
	if err != nil {
		h.handleError(c, err)
		return
	}

	response.Success(c, http.StatusOK, "user retrieved", result)
}

// Update godoc
// @Summary      Update User
// @Description  Updates user profile and permissions in a single transaction. Password is optional.
// @Tags         Users
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id       path      int                      true  "User ID"
// @Param        payload  body      model.UpdateUserRequest  true  "Update user payload"
// @Success      200      {object}  model.UserSuccessDoc
// @Failure      400      {object}  model.LoginBadRequestDoc
// @Failure      409      {object}  model.LoginBadRequestDoc
// @Failure      401      {object}  model.GetMeUnauthorizedDoc
// @Failure      404      {object}  model.LoginBadRequestDoc
// @Router       /api/v1/users/{id} [put]
func (h *UserHandler) Update(c *gin.Context) {
	id, err := parsePathInt32(c, "id")
	if err != nil {
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, "invalid user id", nil))
		return
	}

	var req model.UpdateUserRequest
	if !BindJSON(c, &req) {
		return
	}

	var actorUserID *int32
	if userID, ok := GetUserIDFromContext(c); ok {
		actorUserID = &userID
	}

	result, err := h.useCase.Update(withAuditLogContext(c), id, actorUserID, req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	response.Success(c, http.StatusOK, "user updated", result)
}

// AssignRole godoc
// @Summary      Assign Role to User
// @Description  Updates only the user's primary role assignment.
// @Tags         Users
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id       path      int                       true  "User ID"
// @Param        payload  body      model.AssignUserRoleRequest true  "Assign role payload"
// @Success      200      {object}  model.UserSuccessDoc
// @Failure      400      {object}  model.LoginBadRequestDoc
// @Failure      401      {object}  model.GetMeUnauthorizedDoc
// @Failure      404      {object}  model.LoginBadRequestDoc
// @Failure      503      {object}  model.LoginServiceUnavailableDoc
// @Router       /api/v1/users/{id}/role [put]
func (h *UserHandler) AssignRole(c *gin.Context) {
	id, err := parsePathInt32(c, "id")
	if err != nil {
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, "invalid user id", nil))
		return
	}

	var req model.AssignUserRoleRequest
	if !BindJSON(c, &req) {
		return
	}

	result, err := h.useCase.AssignRole(withAuditLogContext(c), id, req.IDRole)
	if err != nil {
		h.handleError(c, err)
		return
	}

	response.Success(c, http.StatusOK, "user role assigned", result)
}

// AssignPermissions godoc
// @Summary      Replace User Permission Overrides
// @Description  Replaces additive USER_AKSES overrides for a user without changing the primary role.
// @Tags         Users
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id       path      int                              true  "User ID"
// @Param        payload  body      model.AssignUserPermissionsRequest true  "Assign permissions payload"
// @Success      200      {object}  model.UserSuccessDoc
// @Failure      400      {object}  model.LoginBadRequestDoc
// @Failure      401      {object}  model.GetMeUnauthorizedDoc
// @Failure      404      {object}  model.LoginBadRequestDoc
// @Failure      503      {object}  model.LoginServiceUnavailableDoc
// @Router       /api/v1/users/{id}/permissions [put]
func (h *UserHandler) AssignPermissions(c *gin.Context) {
	id, err := parsePathInt32(c, "id")
	if err != nil {
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, "invalid user id", nil))
		return
	}

	var req model.AssignUserPermissionsRequest
	if !BindJSON(c, &req) {
		return
	}

	result, err := h.useCase.ReplacePermissions(c.Request.Context(), id, req.HakAksesIDs)
	if err != nil {
		h.handleError(c, err)
		return
	}

	response.Success(c, http.StatusOK, "user permissions updated", result)
}

// Delete godoc
// @Summary      Delete User
// @Description  Deletes a user by ID. Protects Super Admin (ID 1) from deletion.
// @Tags         Users
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      int  true  "User ID"
// @Success      200  {object}  response.BaseResponse
// @Failure      400  {object}  model.LoginBadRequestDoc
// @Failure      401  {object}  model.GetMeUnauthorizedDoc
// @Failure      404  {object}  model.LoginBadRequestDoc
// @Router       /api/v1/users/{id} [delete]
func (h *UserHandler) Delete(c *gin.Context) {
	id, err := parsePathInt32(c, "id")
	if err != nil {
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, "invalid user id", nil))
		return
	}

	err = h.useCase.Delete(withAuditLogContext(c), id)
	if err != nil {
		h.handleError(c, err)
		return
	}

	response.Success(c, http.StatusOK, "user deleted", nil)
}

type ApproveUserRequest struct {
	Username string `json:"username" binding:"required,min=3"`
}

// Approve godoc
// @Summary      Approve User Pendaftaran
// @Description  Approves a pending user registration, updating status to active and setting username/password.
// @Tags         Users
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      int  true  "User ID"
// @Param        payload  body      httpdelivery.ApproveUserRequest  true  "Approval payload"
// @Success      200  {object}  model.UserSuccessDoc
// @Router       /api/v1/users/{id}/approve [put]
func (h *UserHandler) Approve(c *gin.Context) {
	id, err := parsePathInt32(c, "id")
	if err != nil {
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, "invalid user id", nil))
		return
	}

	var req ApproveUserRequest
	if !BindJSON(c, &req) {
		return
	}

	result, err := h.useCase.Approve(withAuditLogContext(c), id, req.Username)
	if err != nil {
		h.handleError(c, err)
		return
	}

	response.Success(c, http.StatusOK, "user approved", result)
}

// Reject godoc
// @Summary      Reject User Pendaftaran
// @Description  Rejects a pending user registration, updating status to rejected.
// @Tags         Users
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      int  true  "User ID"
// @Success      200  {object}  response.BaseResponse
// @Router       /api/v1/users/{id}/reject [put]
func (h *UserHandler) Reject(c *gin.Context) {
	id, err := parsePathInt32(c, "id")
	if err != nil {
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, "invalid user id", nil))
		return
	}

	err = h.useCase.Reject(withAuditLogContext(c), id)
	if err != nil {
		h.handleError(c, err)
		return
	}

	response.Success(c, http.StatusOK, "user rejected", nil)
}

func (h *UserHandler) handleError(c *gin.Context, err error) {
	if errors.Is(err, usecase.ErrCannotDeleteSuperAdmin) {
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, err.Error(), nil))
		return
	}
	if errors.Is(err, usecase.ErrUsernameAlreadyExists) {
		AbortWithError(c, NewHTTPError(http.StatusConflict, err.Error(), nil))
		return
	}
	if errors.Is(err, usecase.ErrUserNotFound) {
		AbortWithError(c, NewHTTPError(http.StatusNotFound, err.Error(), nil))
		return
	}
	if errors.Is(err, usecase.ErrUserValidation) {
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, err.Error(), nil))
		return
	}

	AbortWithError(c, err)
}
