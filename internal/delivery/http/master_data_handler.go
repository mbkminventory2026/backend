package httpdelivery

import (
	"errors"
	"fmt"
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
	master := router.Group("/api/v1/master").Use(authMiddleware, RequireInternalUser())

	// Departemen
	master.GET("/departemen", RequirePermission(PermissionMasterDepartemenRead), h.ListDepartemen)
	master.GET("/departemen/:id", RequirePermission(PermissionMasterDepartemenRead), h.GetDepartemenByID)
	master.POST("/departemen", RequirePermission(PermissionMasterDepartemenCreate), h.CreateDepartemen)
	master.PUT("/departemen/:id", RequirePermission(PermissionMasterDepartemenUpdate), h.UpdateDepartemen)
	master.DELETE("/departemen/:id", RequirePermission(PermissionMasterDepartemenDelete), h.DeleteDepartemen)

	// Jenis Barang
	master.GET("/jenis-barang", RequirePermission(PermissionMasterJenisBarangRead), h.ListJenisBarang)
	master.GET("/jenis-barang/:id", RequirePermission(PermissionMasterJenisBarangRead), h.GetJenisBarangByID)
	master.POST("/jenis-barang", RequirePermission(PermissionMasterJenisBarangCreate), h.CreateJenisBarang)
	master.PUT("/jenis-barang/:id", RequirePermission(PermissionMasterJenisBarangUpdate), h.UpdateJenisBarang)
	master.DELETE("/jenis-barang/:id", RequirePermission(PermissionMasterJenisBarangDelete), h.DeleteJenisBarang)

	// Mitra
	master.GET("/mitra", RequirePermission(PermissionMasterMitraRead), h.ListMitra)
	master.GET("/mitra/:id", RequirePermission(PermissionMasterMitraRead), h.GetMitraByID)
	master.POST("/mitra", RequirePermission(PermissionMasterMitraCreate), h.CreateMitra)
	master.PUT("/mitra/:id", RequirePermission(PermissionMasterMitraUpdate), h.UpdateMitra)
	master.DELETE("/mitra/:id", RequirePermission(PermissionMasterMitraDelete), h.DeleteMitra)

	// Barang
	master.GET("/barang", RequirePermission(PermissionMasterBarangRead), h.ListBarang)
	master.GET("/barang/:id", RequirePermission(PermissionMasterBarangRead), h.GetBarangByID)
	master.POST("/barang", RequirePermission(PermissionMasterBarangCreate), h.CreateBarang)
	master.PUT("/barang/:id", RequirePermission(PermissionMasterBarangUpdate), h.UpdateBarang)
	master.DELETE("/barang/:id", RequirePermission(PermissionMasterBarangDelete), h.DeleteBarang)

	// Warna
	master.GET("/warna", RequirePermission(PermissionMasterWarnaRead), h.ListWarna)
	master.GET("/warna/:id", RequirePermission(PermissionMasterWarnaRead), h.GetWarnaByID)
	master.POST("/warna", RequirePermission(PermissionMasterWarnaCreate), h.CreateWarna)
	master.PUT("/warna/:id", RequirePermission(PermissionMasterWarnaUpdate), h.UpdateWarna)
	master.DELETE("/warna/:id", RequirePermission(PermissionMasterWarnaDelete), h.DeleteWarna)

	// Permissions
	master.GET("/permissions", RequirePermission(PermissionPermissionRead), h.ListHakAkses)
	master.GET("/permissions/:id", RequirePermission(PermissionPermissionRead), h.GetHakAksesByID)
	master.POST("/permissions", RequirePermission(PermissionPermissionCreate), h.CreateHakAkses)
	master.PUT("/permissions/:id", RequirePermission(PermissionPermissionUpdate), h.UpdateHakAkses)
	master.DELETE("/permissions/:id", RequirePermission(PermissionPermissionDelete), h.DeleteHakAkses)

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
	filter, err := parseListQuery(c, 20)
	if err != nil {
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, "invalid list query", nil))
		return
	}

	items, total, err := h.useCase.ListDepartemen(c.Request.Context(), filter)
	if err != nil {
		h.handleError(c, err)
		return
	}
	setTotalCountHeader(c, total)
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
	filter, err := parseListQuery(c, 20)
	if err != nil {
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, "invalid list query", nil))
		return
	}

	items, total, err := h.useCase.ListJenisBarang(c.Request.Context(), filter)
	if err != nil {
		h.handleError(c, err)
		return
	}
	setTotalCountHeader(c, total)
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
	filter, err := parseListQuery(c, 20)
	if err != nil {
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, "invalid list query", nil))
		return
	}

	items, total, err := h.useCase.ListMitra(c.Request.Context(), filter)
	if err != nil {
		h.handleError(c, err)
		return
	}
	setTotalCountHeader(c, total)
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
	filter, err := parseListQuery(c, 20)
	if err != nil {
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, "invalid list query", nil))
		return
	}

	items, total, err := h.useCase.ListBarang(c.Request.Context(), filter)
	if err != nil {
		h.handleError(c, err)
		return
	}
	setTotalCountHeader(c, total)
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
	filter, err := parseListQuery(c, 20)
	if err != nil {
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, "invalid list query", nil))
		return
	}

	items, total, err := h.useCase.ListHakAkses(c.Request.Context(), filter)
	if err != nil {
		h.handleError(c, err)
		return
	}
	setTotalCountHeader(c, total)
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

// GetWarnaByID godoc
// @Summary      Get Color Detail
// @Tags         Master Data
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      int  true  "Color ID"
// @Success      200  {object}  model.WarnaSuccessDoc
// @Router       /api/v1/master/warna/{id} [get]
func (h *MasterDataHandler) GetWarnaByID(c *gin.Context) {
	id, err := parsePathInt32(c, "id")
	if err != nil {
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, "invalid id", nil))
		return
	}

	item, err := h.useCase.GetWarnaByID(c.Request.Context(), id)
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "warna retrieved", item)
}

// ListWarna godoc
// @Summary      List Colors
// @Tags         Master Data
// @Produce      json
// @Security     BearerAuth
// @Param        limit   query     int  false  "Limit (default 20)"
// @Param        offset  query     int  false  "Offset (default 0)"
// @Param        search  query     string false "Search by name"
// @Success      200     {object}  model.ListWarnaSuccessDoc
// @Router       /api/v1/master/warna [get]
func (h *MasterDataHandler) ListWarna(c *gin.Context) {
	filter, err := parseListQuery(c, 20)
	if err != nil {
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, "invalid list query", nil))
		return
	}

	items, total, err := h.useCase.ListWarna(c.Request.Context(), filter)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.Header("X-Total-Count", fmt.Sprintf("%d", total))
	response.Success(c, http.StatusOK, "warna retrieved", items)
}

// CreateWarna godoc
// @Summary      Create Color
// @Tags         Master Data
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        payload  body      model.CreateWarnaRequest  true  "Color payload"
// @Success      201      {object}  model.WarnaSuccessDoc
// @Router       /api/v1/master/warna [post]
func (h *MasterDataHandler) CreateWarna(c *gin.Context) {
	var req model.CreateWarnaRequest
	if !BindJSON(c, &req) {
		return
	}

	item, err := h.useCase.CreateWarna(c.Request.Context(), req)
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusCreated, "warna created", item)
}

// UpdateWarna godoc
// @Summary      Update Color
// @Tags         Master Data
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id       path      int                       true  "Color ID"
// @Param        payload  body      model.UpdateWarnaRequest  true  "Color payload"
// @Success      200      {object}  model.WarnaSuccessDoc
// @Router       /api/v1/master/warna/{id} [put]
func (h *MasterDataHandler) UpdateWarna(c *gin.Context) {
	id, err := parsePathInt32(c, "id")
	if err != nil {
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, "invalid id", nil))
		return
	}

	var req model.UpdateWarnaRequest
	if !BindJSON(c, &req) {
		return
	}

	item, err := h.useCase.UpdateWarna(c.Request.Context(), id, req)
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "warna updated", item)
}

// DeleteWarna godoc
// @Summary      Delete Color
// @Tags         Master Data
// @Security     BearerAuth
// @Param        id   path      int  true  "Color ID"
// @Success      200  {object}  response.BaseResponse
// @Router       /api/v1/master/warna/{id} [delete]
func (h *MasterDataHandler) DeleteWarna(c *gin.Context) {
	id, err := parsePathInt32(c, "id")
	if err != nil {
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, "invalid id", nil))
		return
	}

	if err := h.useCase.DeleteWarna(c.Request.Context(), id); err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "warna deleted", nil)
}

func (h *MasterDataHandler) handleError(c *gin.Context, err error) {
	if err == nil {
		return
	}
	if errors.Is(err, usecase.ErrMasterDataNotFound) {
		AbortWithError(c, NewHTTPError(http.StatusNotFound, err.Error(), nil))
		return
	}
	if errors.Is(err, usecase.ErrMasterDataDuplicateCode) {
		AbortWithError(c, NewHTTPError(http.StatusConflict, err.Error(), nil))
		return
	}
	AbortWithError(c, err)
}
