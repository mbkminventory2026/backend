package httpdelivery

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

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
	v1.POST("/po-clients", h.CreatePOClient)
	v1.POST("/pr-internals", h.CreatePRInternal)
	v1.POST("/po-internals", h.CreatePOInternal)
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
	var req model.CreatePOClientRequest
	if !BindJSON(c, &req) {
		return
	}

	item, err := h.useCase.CreatePOClient(c.Request.Context(), req)
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusCreated, "po client created", item)
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

	item, err := h.useCase.CreatePRInternal(c.Request.Context(), req)
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusCreated, "pr internal created", item)
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
	var req model.CreatePOInternalRequest
	if !BindJSON(c, &req) {
		return
	}

	item, err := h.useCase.CreatePOInternal(c.Request.Context(), req)
	if err != nil {
		h.handleError(c, err)
		return
	}
	response.Success(c, http.StatusCreated, "po internal created", item)
}

func (h *TransactionDocumentHandler) handleError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, usecase.ErrTransactionValidation):
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, err.Error(), model.TransactionErrorDetail{Code: "invalid_transaction_payload"}))
	case errors.Is(err, usecase.ErrTransactionReferenceNotFound):
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, err.Error(), model.TransactionErrorDetail{Code: "related_data_not_found"}))
	case errors.Is(err, usecase.ErrPOClientAlreadyExists):
		AbortWithError(c, NewHTTPError(http.StatusConflict, err.Error(), model.TransactionErrorDetail{Code: "po_client_already_exists"}))
	case errors.Is(err, usecase.ErrTransactionServiceUnavailable):
		AbortWithError(c, NewHTTPError(http.StatusInternalServerError, err.Error(), model.TransactionErrorDetail{Code: "transaction_service_unavailable"}))
	default:
		AbortWithError(c, err)
	}
}
