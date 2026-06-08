package httpdelivery

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"permatatex-inventory/internal/model"
	"permatatex-inventory/internal/usecase"
	"permatatex-inventory/pkg/response"
)

type ProfilPerusahaanHandler struct {
	useCase *usecase.ProfilPerusahaanUseCase
}

func NewProfilPerusahaanHandler(useCase *usecase.ProfilPerusahaanUseCase) (*ProfilPerusahaanHandler, error) {
	if useCase == nil {
		return nil, errors.New("profil perusahaan usecase is required")
	}
	return &ProfilPerusahaanHandler{useCase: useCase}, nil
}

func (h *ProfilPerusahaanHandler) RegisterRoutes(router gin.IRouter, authMiddleware gin.HandlerFunc) {
	// Public route for landing page/login screen to get basic branding info
	router.GET("/api/v1/profil-perusahaan", h.GetProfilPerusahaan)

	// Protected routes for management
	group := router.Group("/api/v1/profil-perusahaan").Use(authMiddleware, RequireInternalUser())
	group.GET("/:id", RequirePermission(PermissionMasterProfilPerusahaanRead), h.GetProfilPerusahaanByID)
	group.POST("", RequirePermission(PermissionMasterProfilPerusahaanCreate), h.CreateProfilPerusahaan)
	group.PUT("/:id", RequirePermission(PermissionMasterProfilPerusahaanUpdate), h.UpdateProfilPerusahaan)
	group.DELETE("/:id", RequirePermission(PermissionMasterProfilPerusahaanDelete), h.DeleteProfilPerusahaan)
}

// GetProfilPerusahaan godoc
// @Summary      Get Profil Perusahaan Data
// @Tags         Profil Perusahaan
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  model.ProfilPerusahaanSuccessDoc
// @Router       /api/v1/profil-perusahaan [get]
func (h *ProfilPerusahaanHandler) GetProfilPerusahaan(c *gin.Context) {
	item, err := h.useCase.GetProfilPerusahaan(c.Request.Context())
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "profil perusahaan data retrieved", item)
}

// GetProfilPerusahaanByID godoc
// @Summary      Get Profil Perusahaan Detail
// @Tags         Profil Perusahaan
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      int  true  "Profil Perusahaan ID"
// @Success      200  {object}  model.ProfilPerusahaanSuccessDoc
// @Router       /api/v1/profil-perusahaan/{id} [get]
func (h *ProfilPerusahaanHandler) GetProfilPerusahaanByID(c *gin.Context) {
	id, err := parsePathInt32(c, "id")
	if err != nil {
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, "invalid id", nil))
		return
	}

	item, err := h.useCase.GetProfilPerusahaanByID(c.Request.Context(), id)
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "profil perusahaan data retrieved", item)
}

// CreateProfilPerusahaan godoc
// @Summary      Create Profil Perusahaan Data
// @Tags         Profil Perusahaan
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        payload  body      model.CreateProfilPerusahaanRequest  true  "Profil Perusahaan payload"
// @Success      201      {object}  model.ProfilPerusahaanSuccessDoc
// @Failure      409      {object}  model.LoginBadRequestDoc
// @Router       /api/v1/profil-perusahaan [post]
func (h *ProfilPerusahaanHandler) CreateProfilPerusahaan(c *gin.Context) {
	var req model.CreateProfilPerusahaanRequest
	if !BindJSON(c, &req) {
		return
	}

	item, err := h.useCase.CreateProfilPerusahaan(c.Request.Context(), req)
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusCreated, "profil perusahaan data created", item)
}

// UpdateProfilPerusahaan godoc
// @Summary      Update Profil Perusahaan Data
// @Tags         Profil Perusahaan
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id       path      int                                   true  "Profil Perusahaan ID"
// @Param        payload  body      model.UpdateProfilPerusahaanRequest  true  "Profil Perusahaan payload"
// @Success      200      {object}  model.ProfilPerusahaanSuccessDoc
// @Router       /api/v1/profil-perusahaan/{id} [put]
func (h *ProfilPerusahaanHandler) UpdateProfilPerusahaan(c *gin.Context) {
	id, err := parsePathInt32(c, "id")
	if err != nil {
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, "invalid id", nil))
		return
	}

	var req model.UpdateProfilPerusahaanRequest
	if !BindJSON(c, &req) {
		return
	}

	item, err := h.useCase.UpdateProfilPerusahaan(c.Request.Context(), id, req)
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "profil perusahaan data updated", item)
}

// DeleteProfilPerusahaan godoc
// @Summary      Delete Profil Perusahaan Data
// @Tags         Profil Perusahaan
// @Security     BearerAuth
// @Param        id   path      int  true  "Profil Perusahaan ID"
// @Success      200  {object}  response.BaseResponse
// @Router       /api/v1/profil-perusahaan/{id} [delete]
func (h *ProfilPerusahaanHandler) DeleteProfilPerusahaan(c *gin.Context) {
	id, err := parsePathInt32(c, "id")
	if err != nil {
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, "invalid id", nil))
		return
	}

	if err := h.useCase.DeleteProfilPerusahaan(c.Request.Context(), id); err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "profil perusahaan data deleted", nil)
}

func (h *ProfilPerusahaanHandler) handleError(c *gin.Context, err error) {
	if err == nil {
		return
	}
	if errors.Is(err, usecase.ErrProfilPerusahaanNotFound) {
		AbortWithError(c, NewHTTPError(http.StatusNotFound, err.Error(), nil))
		return
	}
	if errors.Is(err, usecase.ErrProfilPerusahaanAlreadyExists) {
		AbortWithError(c, NewHTTPError(http.StatusConflict, err.Error(), nil))
		return
	}
	AbortWithError(c, err)
}
