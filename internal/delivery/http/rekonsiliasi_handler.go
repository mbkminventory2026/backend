package httpdelivery

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"permatatex-inventory/internal/model"
	"permatatex-inventory/internal/usecase"
	"permatatex-inventory/pkg/response"
)

type RekonsiliasiHandler struct {
	useCase *usecase.RekonsiliasiUseCase
}

func NewRekonsiliasiHandler(useCase *usecase.RekonsiliasiUseCase) (*RekonsiliasiHandler, error) {
	if useCase == nil {
		return nil, errors.New("rekonsiliasi usecase is required")
	}

	return &RekonsiliasiHandler{useCase: useCase}, nil
}

func (h *RekonsiliasiHandler) RegisterRoutes(router gin.IRouter, authMiddleware gin.HandlerFunc) {
	group := router.Group("/api/v1/rekonsiliasi").Use(authMiddleware, RequireInternalUser())
	group.GET("", RequirePermission(PermissionRekonsiliasiRead), h.List)
	group.POST("", RequirePermission(PermissionRekonsiliasiCreate), h.Create)
	group.GET("/:id", RequirePermission(PermissionRekonsiliasiRead), h.Get)
	group.PUT("/:id", RequirePermission(PermissionRekonsiliasiUpdate), h.Update)
	group.POST("/:id/refresh", RequirePermission(PermissionRekonsiliasiUpdate), h.Refresh)
}

// List godoc
// @Summary      List rekonsiliasi
// @Description  Retrieves paginated material reconciliation documents.
// @Tags         Rekonsiliasi
// @Produce      json
// @Security     BearerAuth
// @Param        page      query     int     false  "Page number"
// @Param        pageSize  query     int     false  "Page size"
// @Param        limit     query     int     false  "Limit fallback"
// @Param        q         query     string  false  "Search term"
// @Param        idWo      query     int     false  "Filter by work order ID"
// @Param        sortBy    query     string  false  "Sort field"
// @Param        sortDesc  query     bool    false  "Sort descending"
// @Success      200       {object}  model.RekonsiliasiListSuccessDoc
// @Failure      400       {object}  model.RekonsiliasiErrorDoc
// @Failure      500       {object}  model.RekonsiliasiErrorDoc
// @Router       /api/v1/rekonsiliasi [get]
func (h *RekonsiliasiHandler) List(c *gin.Context) {
	filter, err := parseListQuery(c, 20)
	if err != nil {
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, "invalid query params", nil))
		return
	}

	idWo, err := parseOptionalRekonsiliasiWOID(c.Query("idWo"))
	if err != nil {
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, "invalid work order id", nil))
		return
	}

	result, err := h.useCase.ListRekonsiliasis(c.Request.Context(), model.RekonsiliasiListFilter{
		ListQueryFilter: filter,
		IDWo:            idWo,
	})
	if err != nil {
		h.handleError(c, err)
		return
	}

	response.Success(c, http.StatusOK, "rekonsiliasi retrieved", result)
}

// Create godoc
// @Summary      Create rekonsiliasi
// @Description  Creates a material reconciliation document from work order source data.
// @Tags         Rekonsiliasi
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        payload  body      model.CreateRekonsiliasiRequest  true  "Create rekonsiliasi payload"
// @Success      201      {object}  model.RekonsiliasiSuccessDoc
// @Failure      400      {object}  model.RekonsiliasiValidationErrorDoc
// @Failure      404      {object}  model.RekonsiliasiErrorDoc
// @Failure      409      {object}  model.RekonsiliasiErrorDoc
// @Failure      500      {object}  model.RekonsiliasiErrorDoc
// @Router       /api/v1/rekonsiliasi [post]
func (h *RekonsiliasiHandler) Create(c *gin.Context) {
	userID, ok := GetUserIDFromContext(c)
	if !ok {
		AbortWithError(c, NewHTTPError(http.StatusUnauthorized, "unauthorized", nil))
		return
	}

	var req model.CreateRekonsiliasiRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, "invalid payload", err))
		return
	}

	result, err := h.useCase.CreateRekonsiliasi(c.Request.Context(), userID, req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	response.Success(c, http.StatusCreated, "rekonsiliasi created", result)
}

// Get godoc
// @Summary      Get rekonsiliasi detail
// @Description  Retrieves a full material reconciliation document.
// @Tags         Rekonsiliasi
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      int  true  "Rekonsiliasi ID"
// @Success      200  {object}  model.RekonsiliasiSuccessDoc
// @Failure      400  {object}  model.RekonsiliasiErrorDoc
// @Failure      404  {object}  model.RekonsiliasiErrorDoc
// @Failure      500  {object}  model.RekonsiliasiErrorDoc
// @Router       /api/v1/rekonsiliasi/{id} [get]
func (h *RekonsiliasiHandler) Get(c *gin.Context) {
	id, err := parsePathInt32(c, "id")
	if err != nil {
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, "invalid rekonsiliasi id", nil))
		return
	}

	result, err := h.useCase.GetRekonsiliasi(c.Request.Context(), id)
	if err != nil {
		h.handleError(c, err)
		return
	}

	response.Success(c, http.StatusOK, "rekonsiliasi retrieved", result)
}

// Update godoc
// @Summary      Update rekonsiliasi
// @Description  Updates editable manual fields in a material reconciliation document.
// @Tags         Rekonsiliasi
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id       path      int                               true  "Rekonsiliasi ID"
// @Param        payload  body      model.UpdateRekonsiliasiRequest  true  "Update rekonsiliasi payload"
// @Success      200      {object}  model.RekonsiliasiSuccessDoc
// @Failure      400      {object}  model.RekonsiliasiValidationErrorDoc
// @Failure      404      {object}  model.RekonsiliasiErrorDoc
// @Failure      500      {object}  model.RekonsiliasiErrorDoc
// @Router       /api/v1/rekonsiliasi/{id} [put]
func (h *RekonsiliasiHandler) Update(c *gin.Context) {
	userID, ok := GetUserIDFromContext(c)
	if !ok {
		AbortWithError(c, NewHTTPError(http.StatusUnauthorized, "unauthorized", nil))
		return
	}

	id, err := parsePathInt32(c, "id")
	if err != nil {
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, "invalid rekonsiliasi id", nil))
		return
	}

	var req model.UpdateRekonsiliasiRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, "invalid payload", err))
		return
	}

	result, err := h.useCase.UpdateRekonsiliasi(c.Request.Context(), id, userID, req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	response.Success(c, http.StatusOK, "rekonsiliasi updated", result)
}

// Refresh godoc
// @Summary      Refresh rekonsiliasi
// @Description  Refreshes source-driven snapshot data while preserving manual input.
// @Tags         Rekonsiliasi
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      int  true  "Rekonsiliasi ID"
// @Success      200  {object}  model.RekonsiliasiSuccessDoc
// @Failure      400  {object}  model.RekonsiliasiErrorDoc
// @Failure      404  {object}  model.RekonsiliasiErrorDoc
// @Failure      500  {object}  model.RekonsiliasiErrorDoc
// @Router       /api/v1/rekonsiliasi/{id}/refresh [post]
func (h *RekonsiliasiHandler) Refresh(c *gin.Context) {
	userID, ok := GetUserIDFromContext(c)
	if !ok {
		AbortWithError(c, NewHTTPError(http.StatusUnauthorized, "unauthorized", nil))
		return
	}

	id, err := parsePathInt32(c, "id")
	if err != nil {
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, "invalid rekonsiliasi id", nil))
		return
	}

	result, err := h.useCase.RefreshRekonsiliasi(c.Request.Context(), id, userID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	response.Success(c, http.StatusOK, "rekonsiliasi refreshed", result)
}

func (h *RekonsiliasiHandler) handleError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, usecase.ErrRekonsiliasiValidation):
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, err.Error(), nil))
	case errors.Is(err, usecase.ErrRekonsiliasiNotFound), errors.Is(err, usecase.ErrRekonsiliasiSourceNotFound):
		AbortWithError(c, NewHTTPError(http.StatusNotFound, err.Error(), nil))
	case errors.Is(err, usecase.ErrRekonsiliasiAlreadyExists):
		AbortWithError(c, NewHTTPError(http.StatusConflict, err.Error(), nil))
	default:
		AbortWithError(c, NewHTTPError(http.StatusInternalServerError, err.Error(), nil))
	}
}

func parseOptionalRekonsiliasiWOID(raw string) (*int32, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, nil
	}

	value, err := parseStringInt32(trimmed)
	if err != nil {
		return nil, err
	}

	return &value, nil
}
