package httpdelivery

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"permatatex-inventory/internal/model"
	"permatatex-inventory/internal/usecase"
	"permatatex-inventory/pkg/response"
)

type AuditLogHandler struct {
	useCase *usecase.AuditLogUseCase
}

func NewAuditLogHandler(useCase *usecase.AuditLogUseCase) (*AuditLogHandler, error) {
	if useCase == nil {
		return nil, errors.New("audit log usecase is required")
	}

	return &AuditLogHandler{useCase: useCase}, nil
}

func (h *AuditLogHandler) RegisterRoutes(router *gin.Engine, authMiddleware gin.HandlerFunc) {
	api := router.Group("/api/v1")
	api.Use(authMiddleware, RequireInternalUser(), RequireOperatorUser(), RequirePermission(PermissionLogRead))
	{
		api.GET("/activity-logs", h.List)
		api.GET("/activity-logs/:id", h.GetByID)
	}
}

// List godoc
// @Summary      List activity logs
// @Description  Retrieves paginated audit log history for operator users.
// @Tags         Audit Logs
// @Produce      json
// @Security     BearerAuth
// @Param        page         query     int     false  "Page number"
// @Param        pageSize     query     int     false  "Page size"
// @Param        limit        query     int     false  "Limit fallback"
// @Param        q            query     string  false  "Search term"
// @Param        action       query     string  false  "Action filter"
// @Param        module       query     string  false  "Module filter"
// @Param        entityType   query     string  false  "Entity type filter"
// @Param        actorUserId  query     int     false  "Actor user ID filter"
// @Param        dateFrom     query     string  false  "Date from (YYYY-MM-DD or RFC3339)"
// @Param        dateTo       query     string  false  "Date to (YYYY-MM-DD or RFC3339)"
// @Param        sortBy       query     string  false  "Sort field"
// @Param        sortDesc     query     bool    false  "Sort descending"
// @Success      200          {object}  model.AuditLogListSuccessDoc
// @Router       /api/v1/activity-logs [get]
func (h *AuditLogHandler) List(c *gin.Context) {
	baseFilter, err := parseListQuery(c, 20)
	if err != nil {
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, "invalid list query", nil))
		return
	}

	actorUserID, err := parseOptionalAuditActorUserID(c)
	if err != nil {
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, "invalid actorUserId", nil))
		return
	}

	dateFrom, err := parseOptionalAuditLogDate(c.Query("dateFrom"), false)
	if err != nil {
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, "invalid dateFrom", nil))
		return
	}

	dateTo, err := parseOptionalAuditLogDate(c.Query("dateTo"), true)
	if err != nil {
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, "invalid dateTo", nil))
		return
	}

	result, err := h.useCase.List(c.Request.Context(), model.AuditLogListFilter{
		ListQueryFilter: baseFilter,
		Action:          strings.TrimSpace(c.Query("action")),
		Module:          strings.TrimSpace(c.Query("module")),
		EntityType:      strings.TrimSpace(c.Query("entityType")),
		ActorUserID:     actorUserID,
		DateFrom:        dateFrom,
		DateTo:          dateTo,
	})
	if err != nil {
		AbortWithError(c, NewHTTPError(http.StatusInternalServerError, err.Error(), nil))
		return
	}

	response.Success(c, http.StatusOK, "activity logs retrieved", result)
}

// GetByID godoc
// @Summary      Get activity log detail
// @Description  Retrieves a single audit log detail for operator users.
// @Tags         Audit Logs
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      int  true  "Audit log ID"
// @Success      200  {object}  model.AuditLogDetailSuccessDoc
// @Router       /api/v1/activity-logs/{id} [get]
func (h *AuditLogHandler) GetByID(c *gin.Context) {
	id, err := parsePathInt64(c, "id")
	if err != nil {
		AbortWithError(c, NewHTTPError(http.StatusBadRequest, "invalid audit log id", nil))
		return
	}

	result, err := h.useCase.GetByID(c.Request.Context(), id)
	if err != nil {
		switch {
		case errors.Is(err, usecase.ErrAuditLogNotFound):
			AbortWithError(c, NewHTTPError(http.StatusNotFound, err.Error(), nil))
		default:
			AbortWithError(c, NewHTTPError(http.StatusInternalServerError, err.Error(), nil))
		}
		return
	}

	response.Success(c, http.StatusOK, "activity log detail retrieved", result)
}

func parseOptionalAuditActorUserID(c *gin.Context) (*int32, error) {
	raw := strings.TrimSpace(c.Query("actorUserId"))
	if raw == "" {
		return nil, nil
	}

	value, err := parseStringInt32(raw)
	if err != nil {
		return nil, err
	}

	return &value, nil
}

func parseOptionalAuditLogDate(raw string, endOfDay bool) (*time.Time, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, nil
	}

	layouts := []string{time.RFC3339, "2006-01-02"}
	for _, layout := range layouts {
		parsed, err := time.Parse(layout, trimmed)
		if err != nil {
			continue
		}

		if layout == "2006-01-02" && endOfDay {
			value := time.Date(parsed.Year(), parsed.Month(), parsed.Day(), 23, 59, 59, int(time.Second-time.Nanosecond), time.UTC)
			return &value, nil
		}

		return &parsed, nil
	}

	return nil, errors.New("invalid date format")
}
