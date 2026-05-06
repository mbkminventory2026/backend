package httpdelivery

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"permatatex-inventory/internal/model"
	"permatatex-inventory/internal/usecase"
	"permatatex-inventory/pkg/response"
)

type MasterDataHandler struct {
	useCase *usecase.MasterDataUseCase
}

func NewMasterDataHandler(useCase *usecase.MasterDataUseCase) (*MasterDataHandler, error) {
	if useCase == nil {
		return nil, errors.New("master data usecase is required")
	}
	return &MasterDataHandler{useCase: useCase}, nil
}

func (h *MasterDataHandler) RegisterRoutes(router gin.IRouter, authMiddleware gin.HandlerFunc) {
	master := router.Group("/api/v1/master").Use(authMiddleware)

	// Departemen
	master.GET("/departemen", h.ListDepartemen)
	master.GET("/departemen/:id", h.GetDepartemenByID)
	master.POST("/departemen", h.CreateDepartemen)
	master.PUT("/departemen/:id", h.UpdateDepartemen)
	master.DELETE("/departemen/:id", h.DeleteDepartemen)

	// Jenis Barang
	master.GET("/jenis-barang", h.ListJenisBarang)
	master.GET("/jenis-barang/:id", h.GetJenisBarangByID)
	master.POST("/jenis-barang", h.CreateJenisBarang)
	master.PUT("/jenis-barang/:id", h.UpdateJenisBarang)
	master.DELETE("/jenis-barang/:id", h.DeleteJenisBarang)

	// Mitra
	master.GET("/mitra", h.ListMitra)
	master.GET("/mitra/:id", h.GetMitraByID)
	master.POST("/mitra", h.CreateMitra)
	master.PUT("/mitra/:id", h.UpdateMitra)
	master.DELETE("/mitra/:id", h.DeleteMitra)

	// Barang
	master.GET("/barang", h.ListBarang)
	master.GET("/barang/:id", h.GetBarangByID)
	master.POST("/barang", h.CreateBarang)
	master.PUT("/barang/:id", h.UpdateBarang)
	master.DELETE("/barang/:id", h.DeleteBarang)

	// Permissions
	master.GET("/permissions", h.ListHakAkses)
	master.GET("/permissions/:id", h.GetHakAksesByID)
	master.POST("/permissions", h.CreateHakAkses)
	master.PUT("/permissions/:id", h.UpdateHakAkses)
	master.DELETE("/permissions/:id", h.DeleteHakAkses)

	// Company
	master.GET("/company", h.GetCompany)
	master.GET("/company/:id", h.GetCompanyByID)
	master.POST("/company", h.CreateCompany)
	master.PUT("/company/:id", h.UpdateCompany)
	master.DELETE("/company/:id", h.DeleteCompany)
}

// DEPARTEMEN

// GetDepartemenByID godoc
// @Summary      Get Department Detail
// @Tags         Master Data
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      int  true  "Department ID"
// @Success      200  {object}  model.CreateDepartemenSuccessDoc
// @Router       /api/v1/master/departemen/{id} [get]
func (h *MasterDataHandler) GetDepartemenByID(c *gin.Context) {
	id, err := parsePathInt32(c, "id")
	if err != nil {
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, "invalid id", nil))
		return
	}

	item, err := h.useCase.GetDepartemenByID(c.Request.Context(), id)
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "departemen retrieved", item)
}

// ListDepartemen godoc
// @Summary      List Departments
// @Tags         Master Data
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  model.ListDepartemenSuccessDoc
// @Router       /api/v1/master/departemen [get]
func (h *MasterDataHandler) ListDepartemen(c *gin.Context) {
	items, err := h.useCase.ListDepartemen(c.Request.Context())
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "departemen retrieved", items)
}

// CreateDepartemen godoc
// @Summary      Create Department
// @Tags         Master Data
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        payload  body      model.CreateDepartemenRequest  true  "Department payload"
// @Success      201      {object}  model.CreateDepartemenSuccessDoc
// @Router       /api/v1/master/departemen [post]
func (h *MasterDataHandler) CreateDepartemen(c *gin.Context) {
	var req model.CreateDepartemenRequest
	if !BindJSON(c, &req) {
		return
	}

	item, err := h.useCase.CreateDepartemen(c.Request.Context(), req)
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusCreated, "departemen created", item)
}

// UpdateDepartemen godoc
// @Summary      Update Department
// @Tags         Master Data
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id       path      int                            true  "Department ID"
// @Param        payload  body      model.UpdateDepartemenRequest  true  "Department payload"
// @Success      200      {object}  model.CreateDepartemenSuccessDoc
// @Router       /api/v1/master/departemen/{id} [put]
func (h *MasterDataHandler) UpdateDepartemen(c *gin.Context) {
	id, err := parsePathInt32(c, "id")
	if err != nil {
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, "invalid id", nil))
		return
	}

	var req model.UpdateDepartemenRequest
	if !BindJSON(c, &req) {
		return
	}

	item, err := h.useCase.UpdateDepartemen(c.Request.Context(), id, req)
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "departemen updated", item)
}

// DeleteDepartemen godoc
// @Summary      Delete Department
// @Tags         Master Data
// @Security     BearerAuth
// @Param        id   path      int  true  "Department ID"
// @Success      200  {object}  response.BaseResponse
// @Router       /api/v1/master/departemen/{id} [delete]
func (h *MasterDataHandler) DeleteDepartemen(c *gin.Context) {
	id, err := parsePathInt32(c, "id")
	if err != nil {
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, "invalid id", nil))
		return
	}

	if err := h.useCase.DeleteDepartemen(c.Request.Context(), id); err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "departemen deleted", nil)
}

// JENIS BARANG

// GetJenisBarangByID godoc
// @Summary      Get Item Type Detail
// @Tags         Master Data
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      int  true  "Item Type ID"
// @Success      200  {object}  model.CreateJenisBarangSuccessDoc
// @Router       /api/v1/master/jenis-barang/{id} [get]
func (h *MasterDataHandler) GetJenisBarangByID(c *gin.Context) {
	id, err := parsePathInt32(c, "id")
	if err != nil {
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, "invalid id", nil))
		return
	}

	item, err := h.useCase.GetJenisBarangByID(c.Request.Context(), id)
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "jenis barang retrieved", item)
}

// ListJenisBarang godoc
// @Summary      List Item Types
// @Tags         Master Data
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  model.ListJenisBarangSuccessDoc
// @Router       /api/v1/master/jenis-barang [get]
func (h *MasterDataHandler) ListJenisBarang(c *gin.Context) {
	items, err := h.useCase.ListJenisBarang(c.Request.Context())
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "jenis barang retrieved", items)
}

// CreateJenisBarang godoc
// @Summary      Create Item Type
// @Tags         Master Data
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        payload  body      model.CreateJenisBarangRequest  true  "Item type payload"
// @Success      201      {object}  model.CreateJenisBarangSuccessDoc
// @Failure      409      {object}  model.LoginBadRequestDoc
// @Router       /api/v1/master/jenis-barang [post]
func (h *MasterDataHandler) CreateJenisBarang(c *gin.Context) {
	var req model.CreateJenisBarangRequest
	if !BindJSON(c, &req) {
		return
	}

	item, err := h.useCase.CreateJenisBarang(c.Request.Context(), req)
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusCreated, "jenis barang created", item)
}

// UpdateJenisBarang godoc
// @Summary      Update Item Type
// @Tags         Master Data
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id       path      int                             true  "Item Type ID"
// @Param        payload  body      model.UpdateJenisBarangRequest  true  "Item type payload"
// @Success      200      {object}  model.CreateJenisBarangSuccessDoc
// @Failure      409      {object}  model.LoginBadRequestDoc
// @Router       /api/v1/master/jenis-barang/{id} [put]
func (h *MasterDataHandler) UpdateJenisBarang(c *gin.Context) {
	id, err := parsePathInt32(c, "id")
	if err != nil {
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, "invalid id", nil))
		return
	}

	var req model.UpdateJenisBarangRequest
	if !BindJSON(c, &req) {
		return
	}

	item, err := h.useCase.UpdateJenisBarang(c.Request.Context(), id, req)
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "jenis barang updated", item)
}

// DeleteJenisBarang godoc
// @Summary      Delete Item Type
// @Tags         Master Data
// @Security     BearerAuth
// @Param        id   path      int  true  "Item Type ID"
// @Success      200  {object}  response.BaseResponse
// @Router       /api/v1/master/jenis-barang/{id} [delete]
func (h *MasterDataHandler) DeleteJenisBarang(c *gin.Context) {
	id, err := parsePathInt32(c, "id")
	if err != nil {
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, "invalid id", nil))
		return
	}

	if err := h.useCase.DeleteJenisBarang(c.Request.Context(), id); err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "jenis barang deleted", nil)
}

// MITRA

// GetMitraByID godoc
// @Summary      Get Partner Detail
// @Tags         Master Data
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      int  true  "Partner ID"
// @Success      200  {object}  model.CreateMitraSuccessDoc
// @Router       /api/v1/master/mitra/{id} [get]
func (h *MasterDataHandler) GetMitraByID(c *gin.Context) {
	id, err := parsePathInt32(c, "id")
	if err != nil {
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, "invalid id", nil))
		return
	}

	item, err := h.useCase.GetMitraByID(c.Request.Context(), id)
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "mitra retrieved", item)
}

// ListMitra godoc
// @Summary      List Partners
// @Tags         Master Data
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  model.ListMitraSuccessDoc
// @Router       /api/v1/master/mitra [get]
func (h *MasterDataHandler) ListMitra(c *gin.Context) {
	items, err := h.useCase.ListMitra(c.Request.Context())
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "mitra retrieved", items)
}

// CreateMitra godoc
// @Summary      Create Partner
// @Tags         Master Data
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        payload  body      model.CreateMitraRequest  true  "Partner payload"
// @Success      201      {object}  model.CreateMitraSuccessDoc
// @Router       /api/v1/master/mitra [post]
func (h *MasterDataHandler) CreateMitra(c *gin.Context) {
	var req model.CreateMitraRequest
	if !BindJSON(c, &req) {
		return
	}

	item, err := h.useCase.CreateMitra(c.Request.Context(), req)
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusCreated, "mitra created", item)
}

// UpdateMitra godoc
// @Summary      Update Partner
// @Tags         Master Data
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id       path      int                       true  "Partner ID"
// @Param        payload  body      model.UpdateMitraRequest  true  "Partner payload"
// @Success      200      {object}  model.CreateMitraSuccessDoc
// @Router       /api/v1/master/mitra/{id} [put]
func (h *MasterDataHandler) UpdateMitra(c *gin.Context) {
	id, err := parsePathInt32(c, "id")
	if err != nil {
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, "invalid id", nil))
		return
	}

	var req model.UpdateMitraRequest
	if !BindJSON(c, &req) {
		return
	}

	item, err := h.useCase.UpdateMitra(c.Request.Context(), id, req)
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "mitra updated", item)
}

// DeleteMitra godoc
// @Summary      Delete Partner
// @Tags         Master Data
// @Security     BearerAuth
// @Param        id   path      int  true  "Partner ID"
// @Success      200  {object}  response.BaseResponse
// @Router       /api/v1/master/mitra/{id} [delete]
func (h *MasterDataHandler) DeleteMitra(c *gin.Context) {
	id, err := parsePathInt32(c, "id")
	if err != nil {
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, "invalid id", nil))
		return
	}

	if err := h.useCase.DeleteMitra(c.Request.Context(), id); err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "mitra deleted", nil)
}

// BARANG

// GetBarangByID godoc
// @Summary      Get Item Detail
// @Tags         Master Data
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      int  true  "Item ID"
// @Success      200  {object}  model.CreateBarangSuccessDoc
// @Router       /api/v1/master/barang/{id} [get]
func (h *MasterDataHandler) GetBarangByID(c *gin.Context) {
	id, err := parsePathInt32(c, "id")
	if err != nil {
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, "invalid id", nil))
		return
	}

	item, err := h.useCase.GetBarangByID(c.Request.Context(), id)
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "barang retrieved", item)
}

// ListBarang godoc
// @Summary      List Items
// @Tags         Master Data
// @Produce      json
// @Security     BearerAuth
// @Param        limit   query     int  false  "Limit (default 20)"
// @Param        offset  query     int  false  "Offset (default 0)"
// @Success      200     {object}  model.ListBarangSuccessDoc
// @Router       /api/v1/master/barang [get]
func (h *MasterDataHandler) ListBarang(c *gin.Context) {
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

	items, err := h.useCase.ListBarang(c.Request.Context(), limit, offset)
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "barang retrieved", items)
}

// CreateBarang godoc
// @Summary      Create Item
// @Tags         Master Data
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        payload  body      model.CreateBarangRequest  true  "Item payload"
// @Success      201      {object}  model.CreateBarangSuccessDoc
// @Failure      409      {object}  model.LoginBadRequestDoc
// @Router       /api/v1/master/barang [post]
func (h *MasterDataHandler) CreateBarang(c *gin.Context) {
	var req model.CreateBarangRequest
	if !BindJSON(c, &req) {
		return
	}

	item, err := h.useCase.CreateBarang(c.Request.Context(), req)
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusCreated, "barang created", item)
}

// UpdateBarang godoc
// @Summary      Update Item
// @Tags         Master Data
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id       path      int                        true  "Item ID"
// @Param        payload  body      model.UpdateBarangRequest  true  "Item payload"
// @Success      200      {object}  model.CreateBarangSuccessDoc
// @Failure      409      {object}  model.LoginBadRequestDoc
// @Router       /api/v1/master/barang/{id} [put]
func (h *MasterDataHandler) UpdateBarang(c *gin.Context) {
	id, err := parsePathInt32(c, "id")
	if err != nil {
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, "invalid id", nil))
		return
	}

	var req model.UpdateBarangRequest
	if !BindJSON(c, &req) {
		return
	}

	item, err := h.useCase.UpdateBarang(c.Request.Context(), id, req)
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "barang updated", item)
}

// DeleteBarang godoc
// @Summary      Delete Item
// @Tags         Master Data
// @Security     BearerAuth
// @Param        id   path      int  true  "Item ID"
// @Success      200  {object}  response.BaseResponse
// @Router       /api/v1/master/barang/{id} [delete]
func (h *MasterDataHandler) DeleteBarang(c *gin.Context) {
	id, err := parsePathInt32(c, "id")
	if err != nil {
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, "invalid id", nil))
		return
	}

	if err := h.useCase.DeleteBarang(c.Request.Context(), id); err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "barang deleted", nil)
}

// PERMISSIONS

// ListHakAkses godoc
// @Summary      List Available Permissions
// @Tags         Master Data
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  model.ListPermissionsSuccessDoc
// @Router       /api/v1/master/permissions [get]
func (h *MasterDataHandler) ListHakAkses(c *gin.Context) {
	items, err := h.useCase.ListHakAkses(c.Request.Context())
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "permissions retrieved", items)
}

// GetHakAksesByID godoc
// @Summary      Get Permission Detail
// @Tags         Master Data
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      int  true  "Permission ID"
// @Success      200  {object}  model.HakAksesSuccessDoc
// @Router       /api/v1/master/permissions/{id} [get]
func (h *MasterDataHandler) GetHakAksesByID(c *gin.Context) {
	id, err := parsePathInt32(c, "id")
	if err != nil {
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, "invalid id", nil))
		return
	}

	item, err := h.useCase.GetHakAksesByID(c.Request.Context(), id)
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "permission retrieved", item)
}

// CreateHakAkses godoc
// @Summary      Create Permission
// @Tags         Master Data
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        payload  body      model.CreateHakAksesRequest  true  "Permission payload"
// @Success      201      {object}  model.HakAksesSuccessDoc
// @Failure      409      {object}  model.LoginBadRequestDoc
// @Router       /api/v1/master/permissions [post]
func (h *MasterDataHandler) CreateHakAkses(c *gin.Context) {
	var req model.CreateHakAksesRequest
	if !BindJSON(c, &req) {
		return
	}

	item, err := h.useCase.CreateHakAkses(c.Request.Context(), req)
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusCreated, "permission created", item)
}

// UpdateHakAkses godoc
// @Summary      Update Permission
// @Tags         Master Data
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id       path      int                         true  "Permission ID"
// @Param        payload  body      model.UpdateHakAksesRequest true  "Permission payload"
// @Success      200      {object}  model.HakAksesSuccessDoc
// @Router       /api/v1/master/permissions/{id} [put]
func (h *MasterDataHandler) UpdateHakAkses(c *gin.Context) {
	id, err := parsePathInt32(c, "id")
	if err != nil {
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, "invalid id", nil))
		return
	}

	var req model.UpdateHakAksesRequest
	if !BindJSON(c, &req) {
		return
	}

	item, err := h.useCase.UpdateHakAkses(c.Request.Context(), id, req)
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "permission updated", item)
}

// DeleteHakAkses godoc
// @Summary      Delete Permission
// @Tags         Master Data
// @Security     BearerAuth
// @Param        id   path      int  true  "Permission ID"
// @Success      200  {object}  response.BaseResponse
// @Router       /api/v1/master/permissions/{id} [delete]
func (h *MasterDataHandler) DeleteHakAkses(c *gin.Context) {
	id, err := parsePathInt32(c, "id")
	if err != nil {
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, "invalid id", nil))
		return
	}

	if err := h.useCase.DeleteHakAkses(c.Request.Context(), id); err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "permission deleted", nil)
}

// COMPANY

// GetCompany godoc
// @Summary      Get Company Data
// @Tags         Master Data
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  model.CompanySuccessDoc
// @Router       /api/v1/master/company [get]
func (h *MasterDataHandler) GetCompany(c *gin.Context) {
	item, err := h.useCase.GetCompany(c.Request.Context())
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "company data retrieved", item)
}

// GetCompanyByID godoc
// @Summary      Get Company Detail
// @Tags         Master Data
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      int  true  "Company ID"
// @Success      200  {object}  model.CompanySuccessDoc
// @Router       /api/v1/master/company/{id} [get]
func (h *MasterDataHandler) GetCompanyByID(c *gin.Context) {
	id, err := parsePathInt32(c, "id")
	if err != nil {
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, "invalid id", nil))
		return
	}

	item, err := h.useCase.GetCompanyByID(c.Request.Context(), id)
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "company data retrieved", item)
}

// CreateCompany godoc
// @Summary      Create Company Data
// @Tags         Master Data
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        payload  body      model.CreateCompanyRequest  true  "Company payload"
// @Success      201      {object}  model.CompanySuccessDoc
// @Failure      409      {object}  model.LoginBadRequestDoc
// @Router       /api/v1/master/company [post]
func (h *MasterDataHandler) CreateCompany(c *gin.Context) {
	var req model.CreateCompanyRequest
	if !BindJSON(c, &req) {
		return
	}

	item, err := h.useCase.CreateCompany(c.Request.Context(), req)
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusCreated, "company data created", item)
}

// UpdateCompany godoc
// @Summary      Update Company Data
// @Tags         Master Data
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id       path      int                         true  "Company ID"
// @Param        payload  body      model.UpdateCompanyRequest  true  "Company payload"
// @Success      200      {object}  model.CompanySuccessDoc
// @Router       /api/v1/master/company/{id} [put]
func (h *MasterDataHandler) UpdateCompany(c *gin.Context) {
	id, err := parsePathInt32(c, "id")
	if err != nil {
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, "invalid id", nil))
		return
	}

	var req model.UpdateCompanyRequest
	if !BindJSON(c, &req) {
		return
	}

	item, err := h.useCase.UpdateCompany(c.Request.Context(), id, req)
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "company data updated", item)
}

// DeleteCompany godoc
// @Summary      Delete Company Data
// @Tags         Master Data
// @Security     BearerAuth
// @Param        id   path      int  true  "Company ID"
// @Success      200  {object}  response.BaseResponse
// @Router       /api/v1/master/company/{id} [delete]
func (h *MasterDataHandler) DeleteCompany(c *gin.Context) {
	id, err := parsePathInt32(c, "id")
	if err != nil {
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, "invalid id", nil))
		return
	}

	if err := h.useCase.DeleteCompany(c.Request.Context(), id); err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "company data deleted", nil)
}

func (h *MasterDataHandler) handleError(c *gin.Context, err error) {
	if err == nil {
		return
	}
	if errors.Is(err, usecase.ErrMasterDataNotFound) {
		AbortWithError(c, NewHTTPError(http.StatusNotFound, err.Error(), nil))
		return
	}
	if errors.Is(err, usecase.ErrMasterDataDuplicateCode) || errors.Is(err, usecase.ErrCompanyAlreadyExists) {
		AbortWithError(c, NewHTTPError(http.StatusConflict, err.Error(), nil))
		return
	}
	AbortWithError(c, err)
}
