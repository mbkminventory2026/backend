package httpdelivery

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"permatatex-inventory/internal/model"
	"permatatex-inventory/internal/usecase"
	"permatatex-inventory/pkg/response"
)

type TransactionDocumentHandler struct {
	useCase *usecase.TransactionDocumentUseCase
}

func NewTransactionDocumentHandler(useCase *usecase.TransactionDocumentUseCase) (*TransactionDocumentHandler, error) {
	if useCase == nil {
		return nil, errors.New("transaction document usecase is required")
	}
	return &TransactionDocumentHandler{useCase: useCase}, nil
}

func (h *TransactionDocumentHandler) RegisterRoutes(router gin.IRouter, authMiddleware gin.HandlerFunc) {
	v1 := router.Group("/api/v1").Use(authMiddleware)
	internalOnly := RequireInternalUser()
	v1.GET("/po-clients", RequirePermission(PermissionPOClientRead), h.ListPOClients)
	v1.GET("/po-clients/:id", RequirePermission(PermissionPOClientRead), h.GetPOClientDetail)
	v1.POST("/po-clients", internalOnly, RequirePermission(PermissionPOClientCreate), h.CreatePOClient)
	v1.PUT("/po-clients/:id", internalOnly, RequirePermission(PermissionPOClientUpdate), h.UpdatePOClient)
	v1.GET("/pr-internals", internalOnly, RequirePermission(PermissionPRInternalRead), h.ListPRInternals)
	v1.GET("/pr-internals/:id", internalOnly, RequirePermission(PermissionPRInternalRead), h.GetPRInternalDetail)
	v1.POST("/pr-internals", internalOnly, RequirePermission(PermissionPRInternalCreate), h.CreatePRInternal)
	v1.GET("/po-internals", internalOnly, RequirePermission(PermissionPOInternalRead), h.ListPOInternals)
	v1.GET("/po-internals/:id", internalOnly, RequirePermission(PermissionPOInternalRead), h.GetPOInternalDetail)
	v1.POST("/po-internals", internalOnly, RequirePermission(PermissionPOInternalCreate), h.CreatePOInternal)
}

// ListPOClients godoc
// @Summary      List PO Clients
// @Description  Returns a paginated list of PO client headers for transaction screens.
// @Tags         Transaction Documents
// @Produce      json
// @Security     BearerAuth
// @Param        page    query     int     false  "Page (default 1)"
// @Param        limit   query     int     false  "Limit (default 20)"
// @Param        search  query     string  false  "Search by PO number, season, or mitra name"
// @Success      200     {object}  model.POClientListSuccessDoc
// @Failure      400     {object}  model.TransactionErrorDoc
// @Failure      500     {object}  model.TransactionErrorDoc
// @Router       /api/v1/po-clients [get]
func (h *TransactionDocumentHandler) ListPOClients(c *gin.Context) {
	mitraID, ok := GetMitraIDFromContext(c)
	if !ok {
		AbortWithError(c, NewHTTPError(http.StatusUnauthorized, "unauthorized", nil))
		return
	}

	filter, err := parseListQuery(c, 20)
	if err != nil {
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, "invalid list query", nil))
		return
	}

	item, err := h.useCase.ListPOClients(c.Request.Context(), model.TransactionListFilter{
		ListQueryFilter: filter,
		IDMitra:         mitraID,
	})
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "po clients retrieved", item)
}

// GetPOClientDetail godoc
// @Summary      Get PO Client Detail
// @Description  Returns a single PO client with nested items and penanggung jawab.
// @Tags         Transaction Documents
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      int  true  "PO Client ID"
// @Success      200  {object}  model.POClientDetailSuccessDoc
// @Failure      400  {object}  model.TransactionErrorDoc
// @Failure      404  {object}  model.TransactionErrorDoc
// @Failure      500  {object}  model.TransactionErrorDoc
// @Router       /api/v1/po-clients/{id} [get]
func (h *TransactionDocumentHandler) GetPOClientDetail(c *gin.Context) {
	mitraID, ok := GetMitraIDFromContext(c)
	if !ok {
		AbortWithError(c, NewHTTPError(http.StatusUnauthorized, "unauthorized", nil))
		return
	}

	id, err := parsePathInt32(c, "id")
	if err != nil {
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, "invalid po client id", nil))
		return
	}

	item, err := h.useCase.GetPOClientDetail(c.Request.Context(), id, mitraID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	payload, exists := c.Get(authorizationPayloadKey)
	if exists {
		if claims, ok := payload.(jwt.MapClaims); ok {
			if !hasPricingVisibility(claims) {
				sanitizePOClientDetail(item)
			}
		}
	}

	response.Success(c, http.StatusOK, "po client retrieved", item)
}

// CreatePOClient godoc
// @Summary      Create PO Client
// @Description  Create a PO client document with nested items and penanggung jawab in a single transaction.
// @Tags         Transaction Documents
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        payload  body      model.CreatePOClientRequest  true  "PO client payload"
// @Success      201      {object}  model.POClientSuccessDoc
// @Failure      400      {object}  model.TransactionValidationErrorDoc
// @Failure      409      {object}  model.TransactionErrorDoc
// @Failure      500      {object}  model.TransactionErrorDoc
// @Router       /api/v1/po-clients [post]
func (h *TransactionDocumentHandler) CreatePOClient(c *gin.Context) {
	mitraID, ok := GetMitraIDFromContext(c)
	if !ok {
		AbortWithError(c, NewHTTPError(http.StatusUnauthorized, "unauthorized", nil))
		return
	}

	if mitraID != nil {
		AbortWithError(c, NewHTTPError(http.StatusForbidden, "access denied: client is read-only", nil))
		return
	}

	var req model.CreatePOClientRequest
	if !BindJSON(c, &req) {
		return
	}

	item, err := h.useCase.CreatePOClient(c.Request.Context(), req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	payload, exists := c.Get(authorizationPayloadKey)
	if exists {
		if claims, ok := payload.(jwt.MapClaims); ok {
			if !hasPricingVisibility(claims) {
				sanitizePOClient(item)
			}
		}
	}

	response.Success(c, http.StatusCreated, "po client created", item)
}

// UpdatePOClient godoc
// @Summary      Update PO Client
// @Description  Replace a PO client header, items, and penanggung jawab in a single transaction.
// @Tags         Transaction Documents
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id       path      int                          true  "PO Client ID"
// @Param        payload  body      model.CreatePOClientRequest  true  "PO client payload"
// @Success      200      {object}  model.POClientSuccessDoc
// @Failure      400      {object}  model.TransactionValidationErrorDoc
// @Failure      404      {object}  model.TransactionErrorDoc
// @Failure      409      {object}  model.TransactionErrorDoc
// @Failure      500      {object}  model.TransactionErrorDoc
// @Router       /api/v1/po-clients/{id} [put]
func (h *TransactionDocumentHandler) UpdatePOClient(c *gin.Context) {
	mitraID, ok := GetMitraIDFromContext(c)
	if !ok {
		AbortWithError(c, NewHTTPError(http.StatusUnauthorized, "unauthorized", nil))
		return
	}

	if mitraID != nil {
		AbortWithError(c, NewHTTPError(http.StatusForbidden, "access denied: client is read-only", nil))
		return
	}

	id, err := parsePathInt32(c, "id")
	if err != nil {
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, "invalid po client id", nil))
		return
	}

	var req model.CreatePOClientRequest
	if !BindJSON(c, &req) {
		return
	}

	item, err := h.useCase.UpdatePOClient(c.Request.Context(), id, req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	payload, exists := c.Get(authorizationPayloadKey)
	if exists {
		if claims, ok := payload.(jwt.MapClaims); ok {
			if !hasPricingVisibility(claims) {
				sanitizePOClient(item)
			}
		}
	}

	response.Success(c, http.StatusOK, "po client updated", item)
}

// ListPRInternals godoc
// @Summary      List PR Internals
// @Description  Returns a paginated list of PR internal headers.
// @Tags         Transaction Documents
// @Produce      json
// @Security     BearerAuth
// @Param        page    query     int     false  "Page (default 1)"
// @Param        limit   query     int     false  "Limit (default 20)"
// @Param        search  query     string  false  "Search by name, vendor, or projek"
// @Success      200     {object}  model.PRInternalListSuccessDoc
// @Failure      400     {object}  model.TransactionErrorDoc
// @Failure      500     {object}  model.TransactionErrorDoc
// @Router       /api/v1/pr-internals [get]
func (h *TransactionDocumentHandler) ListPRInternals(c *gin.Context) {
	filter, err := parseListQuery(c, 20)
	if err != nil {
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, "invalid list query", nil))
		return
	}

	item, err := h.useCase.ListPRInternals(c.Request.Context(), model.TransactionListFilter{
		ListQueryFilter: filter,
	})
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "pr internals retrieved", item)
}

// GetPRInternalDetail godoc
// @Summary      Get PR Internal Detail
// @Description  Returns a single PR internal with nested items.
// @Tags         Transaction Documents
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      int  true  "PR Internal ID"
// @Success      200  {object}  model.PRInternalDetailSuccessDoc
// @Failure      400  {object}  model.TransactionErrorDoc
// @Failure      404  {object}  model.TransactionErrorDoc
// @Failure      500  {object}  model.TransactionErrorDoc
// @Router       /api/v1/pr-internals/{id} [get]
func (h *TransactionDocumentHandler) GetPRInternalDetail(c *gin.Context) {
	id, err := parsePathInt32(c, "id")
	if err != nil {
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, "invalid pr internal id", nil))
		return
	}

	item, err := h.useCase.GetPRInternalDetail(c.Request.Context(), id)
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "pr internal retrieved", item)
}

// CreatePRInternal godoc
// @Summary      Create PR Internal
// @Description  Create a PR internal document with nested items in a single transaction.
// @Tags         Transaction Documents
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        payload  body      model.CreatePRInternalRequest  true  "PR internal payload"
// @Success      201      {object}  model.PRInternalSuccessDoc
// @Failure      400      {object}  model.TransactionValidationErrorDoc
// @Failure      500      {object}  model.TransactionErrorDoc
// @Router       /api/v1/pr-internals [post]
func (h *TransactionDocumentHandler) CreatePRInternal(c *gin.Context) {
	var req model.CreatePRInternalRequest
	if !BindJSON(c, &req) {
		return
	}
	userID, ok := GetUserIDFromContext(c)
	if !ok {
		AbortWithError(c, NewHTTPError(http.StatusUnauthorized, "unauthorized", nil))
		return
	}

	item, err := h.useCase.CreatePRInternal(c.Request.Context(), userID, req)
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusCreated, "pr internal created", item)
}

// ListPOInternals godoc
// @Summary      List PO Internals
// @Description  Returns a paginated list of PO internal headers.
// @Tags         Transaction Documents
// @Produce      json
// @Security     BearerAuth
// @Param        page    query     int     false  "Page (default 1)"
// @Param        limit   query     int     false  "Limit (default 20)"
// @Param        search  query     string  false  "Search by PO name, supplier, or CPO"
// @Success      200     {object}  model.POInternalListSuccessDoc
// @Failure      400     {object}  model.TransactionErrorDoc
// @Failure      500     {object}  model.TransactionErrorDoc
// @Router       /api/v1/po-internals [get]
func (h *TransactionDocumentHandler) ListPOInternals(c *gin.Context) {
	filter, err := parseListQuery(c, 20)
	if err != nil {
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, "invalid list query", nil))
		return
	}

	item, err := h.useCase.ListPOInternals(c.Request.Context(), model.TransactionListFilter{
		ListQueryFilter: filter,
	})
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusOK, "po internals retrieved", item)
}

// GetPOInternalDetail godoc
// @Summary      Get PO Internal Detail
// @Description  Returns a single PO internal with nested items.
// @Tags         Transaction Documents
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      int  true  "PO Internal ID"
// @Success      200  {object}  model.POInternalDetailSuccessDoc
// @Failure      400  {object}  model.TransactionErrorDoc
// @Failure      404  {object}  model.TransactionErrorDoc
// @Failure      500  {object}  model.TransactionErrorDoc
// @Router       /api/v1/po-internals/{id} [get]
func (h *TransactionDocumentHandler) GetPOInternalDetail(c *gin.Context) {
	id, err := parsePathInt32(c, "id")
	if err != nil {
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, "invalid po internal id", nil))
		return
	}

	item, err := h.useCase.GetPOInternalDetail(c.Request.Context(), id)
	if err != nil {
		h.handleError(c, err)
		return
	}

	payload, exists := c.Get(authorizationPayloadKey)
	if exists {
		if claims, ok := payload.(jwt.MapClaims); ok {
			if !hasPricingVisibility(claims) {
				sanitizePOInternal(item)
			}
		}
	}

	response.Success(c, http.StatusOK, "po internal retrieved", item)
}

// CreatePOInternal godoc
// @Summary      Create PO Internal
// @Description  Create a PO internal document with nested items in a single transaction.
// @Tags         Transaction Documents
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        payload  body      model.CreatePOInternalRequest  true  "PO internal payload"
// @Success      201      {object}  model.POInternalSuccessDoc
// @Failure      400      {object}  model.TransactionValidationErrorDoc
// @Failure      500      {object}  model.TransactionErrorDoc
// @Router       /api/v1/po-internals [post]
func (h *TransactionDocumentHandler) CreatePOInternal(c *gin.Context) {
	userID, ok := GetUserIDFromContext(c)
	if !ok {
		AbortWithError(c, NewHTTPError(http.StatusUnauthorized, "unauthorized", nil))
		return
	}

	var req model.CreatePOInternalRequest
	if !BindJSON(c, &req) {
		return
	}

	item, err := h.useCase.CreatePOInternal(c.Request.Context(), userID, req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	payload, exists := c.Get(authorizationPayloadKey)
	if exists {
		if claims, ok := payload.(jwt.MapClaims); ok {
			if !hasPricingVisibility(claims) {
				sanitizePOInternal(item)
			}
		}
	}

	response.Success(c, http.StatusCreated, "po internal created", item)
}

func (h *TransactionDocumentHandler) handleError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, usecase.ErrTransactionValidation):
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, err.Error(), model.TransactionErrorDetail{Code: "invalid_transaction_payload"}))
	case errors.Is(err, usecase.ErrTransactionNotFound):
		AbortWithError(c, NewHTTPError(http.StatusNotFound, err.Error(), model.TransactionErrorDetail{Code: "transaction_not_found"}))
	case errors.Is(err, usecase.ErrTransactionReferenceNotFound):
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, err.Error(), model.TransactionErrorDetail{Code: "related_data_not_found"}))
	case errors.Is(err, usecase.ErrPOClientAlreadyExists):
		AbortWithError(c, NewHTTPError(http.StatusConflict, err.Error(), model.TransactionErrorDetail{Code: "po_client_already_exists"}))
	case errors.Is(err, usecase.ErrPOClientLockedForUpdate):
		AbortWithError(c, NewHTTPError(http.StatusConflict, err.Error(), model.TransactionErrorDetail{Code: "po_client_locked_for_update"}))
	case errors.Is(err, usecase.ErrPRInternalAlreadyApproved):
		AbortWithError(c, NewHTTPError(http.StatusConflict, err.Error(), model.TransactionErrorDetail{Code: "pr_internal_already_approved"}))
	case errors.Is(err, usecase.ErrPRInternalNotApproved):
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, err.Error(), model.TransactionErrorDetail{Code: "pr_internal_not_approved"}))
	case errors.Is(err, usecase.ErrTransactionServiceUnavailable):
		AbortWithError(c, NewHTTPError(http.StatusInternalServerError, err.Error(), model.TransactionErrorDetail{Code: "transaction_service_unavailable"}))
	default:
		AbortWithError(c, err)
	}
}

func hasPricingVisibility(claims jwt.MapClaims) bool {
	roleName, ok := claims["role_name"].(string)
	if !ok {
		return false
	}
	return roleName == "SUPER_ADMIN" || roleName == "MANAGER" || roleName == "ADMIN_KEUANGAN"
}

func sanitizePOClientDetail(res *model.POClientDetailResponse) {
	if res == nil {
		return
	}
	res.PenanggungJawab = nil
	for i := range res.Items {
		res.Items[i].Price = 0.0
	}
}

func sanitizePOClient(res *model.POClientResponse) {
	if res == nil {
		return
	}
	res.PenanggungJawab = nil
	for i := range res.Items {
		res.Items[i].Price = 0.0
	}
}

func sanitizePOInternal(res *model.POInternalResponse) {
	if res == nil {
		return
	}
	res.SupplierContact = ""
	res.SupplierEmail = ""
	res.SupplierTelp = ""
	res.SupplierFax = ""
	for i := range res.Items {
		res.Items[i].UnitPrice = 0.0
	}
}
