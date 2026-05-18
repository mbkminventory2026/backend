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
	group := router.Group("/api/v1/users").Use(authMiddleware)

	group.POST("", RequirePermission("USER_CREATE"), h.Create)
	group.GET("", RequirePermission("USER_READ"), h.List)
	group.GET("/:id", RequirePermission("USER_READ"), h.GetByID)
	group.PUT("/:id", RequirePermission("USER_UPDATE"), h.Update)
	group.PUT("/:id/approve", RequirePermission("USER_UPDATE"), h.Approve)
	group.PUT("/:id/reject", RequirePermission("USER_UPDATE"), h.Reject)
	group.DELETE("/:id", RequirePermission("USER_DELETE"), h.Delete)
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

	result, err := h.useCase.Create(c.Request.Context(), req)
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
	limit, err := parseQueryInt32(c, "limit", 20)
	if err != nil {
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, "invalid limit", nil))
		return
	}
	offset, err := parseQueryInt32(c, "offset", 0)
	if err != nil {
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, "invalid offset", nil))
		return
	}

	result, err := h.useCase.List(c.Request.Context(), model.ListUsersFilter{
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		h.handleError(c, err)
		return
	}

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

	result, err := h.useCase.Update(c.Request.Context(), id, req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	response.Success(c, http.StatusOK, "user updated", result)
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

	err = h.useCase.Delete(c.Request.Context(), id)
	if err != nil {
		h.handleError(c, err)
		return
	}

	response.Success(c, http.StatusOK, "user deleted", nil)
}

// Approve godoc
// @Summary      Approve User Pendaftaran
// @Description  Approves a pending user registration, updating status to active.
// @Tags         Users
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      int  true  "User ID"
// @Success      200  {object}  model.UserSuccessDoc
// @Router       /api/v1/users/{id}/approve [put]
func (h *UserHandler) Approve(c *gin.Context) {
	id, err := parsePathInt32(c, "id")
	if err != nil {
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, "invalid user id", nil))
		return
	}

	result, err := h.useCase.Approve(c.Request.Context(), id)
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

	err = h.useCase.Reject(c.Request.Context(), id)
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
