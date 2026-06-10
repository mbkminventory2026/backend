package httpdelivery

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"permatatex-inventory/internal/model"
	"permatatex-inventory/internal/usecase"
	"permatatex-inventory/pkg/response"
)

type MaterialListHandler struct {
	useCase *usecase.MaterialListUseCase
}

func NewMaterialListHandler(useCase *usecase.MaterialListUseCase) (*MaterialListHandler, error) {
	if useCase == nil {
		return nil, errors.New("material list usecase is required")
	}
	return &MaterialListHandler{useCase: useCase}, nil
}

func (h *MaterialListHandler) RegisterRoutes(router gin.IRouter, authMiddleware gin.HandlerFunc) {
	v1 := router.Group("/api/v1").Use(authMiddleware)
	internalOnly := RequireInternalUser()

	v1.GET("/material-lists", RequirePermission(PermissionWORead), h.ListPaginated)
	v1.GET("/work-orders/:id/material-lists", RequirePermission(PermissionWORead), h.ListByWO)
	v1.POST("/work-orders/:id/material-lists", internalOnly, RequirePermission(PermissionWOUpdate), h.Create)

	v1.GET("/material-lists/:id", RequirePermission(PermissionWORead), h.Get)
	v1.PATCH("/material-lists/:id", internalOnly, RequirePermission(PermissionWOUpdate), h.Update)
	v1.DELETE("/material-lists/:id", internalOnly, RequirePermission(PermissionWOUpdate), h.Delete)

	v1.POST("/material-lists/:id/items", internalOnly, RequirePermission(PermissionWOUpdate), h.CreateItem)
	v1.GET("/material-list-items/:id", RequirePermission(PermissionWORead), h.GetItem)
	v1.PATCH("/material-list-items/:id", internalOnly, RequirePermission(PermissionWOUpdate), h.UpdateItem)
	v1.DELETE("/material-list-items/:id", internalOnly, RequirePermission(PermissionWOUpdate), h.DeleteItem)
}

func (h *MaterialListHandler) Create(c *gin.Context) {
	idWo, err := parsePathInt32(c, "id")
	if err != nil {
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, "invalid work order id", nil))
		return
	}
	var req model.CreateMaterialListRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, "invalid payload", err))
		return
	}
	item, err := h.useCase.CreateMaterialList(c.Request.Context(), idWo, req)
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusCreated, "material list created", item)
}

func (h *MaterialListHandler) ListByWO(c *gin.Context) {
	idWo, err := parsePathInt32(c, "id")
	if err != nil {
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, "invalid work order id", nil))
		return
	}
	unlockedOnly := c.Query("unlocked") == "true"
	res, err := h.useCase.ListByWO(c.Request.Context(), idWo, unlockedOnly)
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "material lists retrieved", res)
}

func (h *MaterialListHandler) Get(c *gin.Context) {
	id, err := parsePathInt32(c, "id")
	if err != nil {
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, "invalid material list id", nil))
		return
	}
	res, err := h.useCase.Get(c.Request.Context(), id)
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "material list retrieved", res)
}

func (h *MaterialListHandler) Update(c *gin.Context) {
	id, err := parsePathInt32(c, "id")
	if err != nil {
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, "invalid material list id", nil))
		return
	}
	var req model.UpdateMaterialListRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, "invalid payload", err))
		return
	}
	res, err := h.useCase.Update(c.Request.Context(), id, req)
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "material list updated", res)
}

func (h *MaterialListHandler) Delete(c *gin.Context) {
	id, err := parsePathInt32(c, "id")
	if err != nil {
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, "invalid material list id", nil))
		return
	}
	if err := h.useCase.Delete(c.Request.Context(), id); err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "material list deleted", nil)
}

func (h *MaterialListHandler) CreateItem(c *gin.Context) {
	idML, err := parsePathInt32(c, "id")
	if err != nil {
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, "invalid material list id", nil))
		return
	}
	var req model.CreateMaterialListItemBody
	if err := c.ShouldBindJSON(&req); err != nil {
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, "invalid payload", err))
		return
	}
	item, err := h.useCase.CreateItem(c.Request.Context(), idML, req)
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusCreated, "material list item created", item)
}

func (h *MaterialListHandler) UpdateItem(c *gin.Context) {
	id, err := parsePathInt32(c, "id")
	if err != nil {
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, "invalid material list item id", nil))
		return
	}
	var req model.UpdateMaterialListItemBody
	if err := c.ShouldBindJSON(&req); err != nil {
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, "invalid payload", err))
		return
	}
	res, err := h.useCase.UpdateItem(c.Request.Context(), id, req)
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "material list item updated", res)
}

func (h *MaterialListHandler) DeleteItem(c *gin.Context) {
	id, err := parsePathInt32(c, "id")
	if err != nil {
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, "invalid material list item id", nil))
		return
	}
	if err := h.useCase.DeleteItem(c.Request.Context(), id); err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "material list item deleted", nil)
}

func (h *MaterialListHandler) GetItem(c *gin.Context) {
	id, err := parsePathInt32(c, "id")
	if err != nil {
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, "invalid material list item id", nil))
		return
	}
	res, err := h.useCase.GetItemDetail(c.Request.Context(), id)
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "material list item retrieved", res)
}

func (h *MaterialListHandler) ListPaginated(c *gin.Context) {
	filter, err := parseListQuery(c, 20)
	if err != nil {
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, "invalid query params", nil))
		return
	}
	lockedOnly := c.Query("locked_only") != "false"
	res, err := h.useCase.ListMaterialListsPaginated(c.Request.Context(), filter.Search, lockedOnly, filter.Limit, filter.Offset)
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "material lists retrieved", res)
}

func (h *MaterialListHandler) handleError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, usecase.ErrMaterialListNotFound), errors.Is(err, usecase.ErrMaterialListItemNotFound):
		AbortWithError(c, NewHTTPError(http.StatusNotFound, err.Error(), nil))
	case errors.Is(err, usecase.ErrMaterialListLocked):
		AbortWithError(c, NewHTTPError(http.StatusConflict, err.Error(), nil))
	case errors.Is(err, usecase.ErrMaterialListValidation):
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, err.Error(), nil))
	default:
		AbortWithError(c, NewHTTPError(http.StatusInternalServerError, err.Error(), nil))
	}
}
