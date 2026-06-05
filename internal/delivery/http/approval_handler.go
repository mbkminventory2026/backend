package httpdelivery

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"permatatex-inventory/internal/model"
	"permatatex-inventory/internal/usecase"
	"permatatex-inventory/pkg/response"
)

type ApprovalHandler struct {
	useCase *usecase.ApprovalUseCase
}

func NewApprovalHandler(useCase *usecase.ApprovalUseCase) (*ApprovalHandler, error) {
	if useCase == nil {
		return nil, errors.New("approval usecase is required")
	}
	return &ApprovalHandler{useCase: useCase}, nil
}

func (h *ApprovalHandler) RegisterRoutes(router gin.IRouter, authMiddleware gin.HandlerFunc) {
	group := router.Group("/api/v1/approvals").Use(authMiddleware, RequireInternalUser())

	group.GET("/pending", h.GetPendingApprovals)
	group.POST("/action", h.ProcessApprovalAction)
	group.GET("/document/:table/:id", h.GetDocumentAuditTrail)
}

// GetPendingApprovals godoc
// @Summary      Get Pending Approvals
// @Description  Returns a list of documents currently waiting for the logged-in user's approval.
// @Tags         Approvals
// @Produce      json
// @Security     BearerAuth
// @Success      200      {object}  response.BaseResponse
// @Router       /api/v1/approvals/pending [get]
func (h *ApprovalHandler) GetPendingApprovals(c *gin.Context) {
	userID, ok := GetUserIDFromContext(c)
	if !ok {
		AbortWithError(c, NewHTTPError(http.StatusUnauthorized, "unauthorized", nil))
		return
	}

	result, err := h.useCase.GetPendingApprovals(c.Request.Context(), userID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	response.Success(c, http.StatusOK, "pending approvals retrieved", result)
}

// ProcessApprovalAction godoc
// @Summary      Process Approval Action
// @Description  Allows a user to approve or reject their pending approval step.
// @Tags         Approvals
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        payload  body      model.ApprovalActionRequest  true  "Action payload"
// @Success      200      {object}  response.BaseResponse
// @Router       /api/v1/approvals/action [post]
func (h *ApprovalHandler) ProcessApprovalAction(c *gin.Context) {
	userID, ok := GetUserIDFromContext(c)
	if !ok {
		AbortWithError(c, NewHTTPError(http.StatusUnauthorized, "unauthorized", nil))
		return
	}

	var req model.ApprovalActionRequest
	if !BindJSON(c, &req) {
		return
	}

	err := h.useCase.ProcessApprovalAction(c.Request.Context(), userID, req)
	if err != nil {
		h.handleError(c, err)
		return
	}

	response.Success(c, http.StatusOK, "approval action processed successfully", nil)
}

// GetDocumentAuditTrail godoc
// @Summary      Get Document Audit Trail
// @Description  Returns the complete approval steps history for a specific document.
// @Tags         Approvals
// @Produce      json
// @Security     BearerAuth
// @Param        table   path      string  true  "Database table name (e.g. PR_INTERNAL, WORK_ORDER)"
// @Param        id      path      int     true  "Document ID"
// @Success      200      {object}  response.BaseResponse
// @Router       /api/v1/approvals/document/{table}/{id} [get]
func (h *ApprovalHandler) GetDocumentAuditTrail(c *gin.Context) {
	table := c.Param("table")
	if table == "" {
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, "table parameter is required", nil))
		return
	}

	id, err := parsePathInt32(c, "id")
	if err != nil {
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, "invalid document id", nil))
		return
	}

	result, err := h.useCase.GetDocumentAuditTrail(c.Request.Context(), table, id)
	if err != nil {
		h.handleError(c, err)
		return
	}

	response.Success(c, http.StatusOK, "audit trail retrieved", result)
}

func (h *ApprovalHandler) handleError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, usecase.ErrApprovalNotFound):
		AbortWithError(c, NewHTTPError(http.StatusNotFound, err.Error(), nil))
	case errors.Is(err, usecase.ErrPreviousStepPending):
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, err.Error(), nil))
	case errors.Is(err, usecase.ErrDocumentNotPending):
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, err.Error(), nil))
	case errors.Is(err, usecase.ErrUnauthorizedApproval):
		AbortWithError(c, NewHTTPError(http.StatusForbidden, err.Error(), nil))
	case errors.Is(err, usecase.ErrApprovalServiceUnavailable):
		AbortWithError(c, NewHTTPError(http.StatusServiceUnavailable, err.Error(), nil))
	default:
		AbortWithError(c, err)
	}
}
