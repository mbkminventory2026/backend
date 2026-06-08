package httpdelivery

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

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
	group.GET("/history", h.GetApprovalHistory)
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

	// Verify read permission for the specific document type
	var requiredPermission string
	switch table {
	case "PR_INTERNAL":
		requiredPermission = PermissionPRInternalRead
	case "PO_INTERNAL":
		requiredPermission = PermissionPOInternalRead
	case "WORK_ORDER":
		requiredPermission = PermissionWORead
	case "MARKER_PLAN":
		requiredPermission = PermissionMarkerPlanRead
	case "TIMELINE_PRODUKSI":
		requiredPermission = PermissionTimelineRead
	case "SPREADING_CUTTING_PLAN":
		requiredPermission = PermissionCuttingPlanRead
	case "PACKING_LIST":
		requiredPermission = PermissionPackingListRead
	default:
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, "unsupported document type", nil))
		return
	}

	if !HasPermission(c, requiredPermission) {
		AbortWithError(c, NewHTTPError(http.StatusForbidden, fmt.Sprintf("access denied: missing read permission '%s'", requiredPermission), nil))
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

// GetApprovalHistory godoc
// @Summary      Get Approval History
// @Description  Returns a list of completed (approved/rejected) document approval history.
// @Tags         Approvals
// @Produce      json
// @Security     BearerAuth
// @Param        status   query     string  false  "Filter by global status (e.g. approved, rejected)"
// @Param        table    query     string  false  "Filter by document table name"
// @Param        limit    query     int     false  "Limit"
// @Param        offset   query     int     false  "Offset"
// @Success      200      {object}  response.BaseResponse
// @Router       /api/v1/approvals/history [get]
func (h *ApprovalHandler) GetApprovalHistory(c *gin.Context) {
	status := strings.TrimSpace(c.Query("status"))
	table := strings.TrimSpace(c.Query("table"))

	filter, err := parseListQuery(c, 20)
	if err != nil {
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, "invalid list query parameters", nil))
		return
	}

	// Calculate limit and offset for query
	limit := filter.Limit
	offset := (filter.Page - 1) * limit

	result, err := h.useCase.GetApprovalHistory(c.Request.Context(), status, table, limit, offset)
	if err != nil {
		h.handleError(c, err)
		return
	}

	response.Success(c, http.StatusOK, "approval history retrieved", result)
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
