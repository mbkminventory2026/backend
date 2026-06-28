package httpdelivery

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"permatatex-inventory/internal/model"
	"permatatex-inventory/internal/usecase"
)

type auditFallbackMetadata struct {
	module      string
	entityType  string
	entityID    string
	entityLabel string
}

func AuditLogFallbackMiddleware(auditLog *usecase.AuditLogUseCase) gin.HandlerFunc {
	return func(c *gin.Context) {
		if auditLog == nil {
			c.Next()
			return
		}

		c.Next()

		if !shouldLogFallbackAudit(c.Request.Method, c.Writer.Status()) {
			return
		}

		route := c.FullPath()
		if route == "" {
			route = c.Request.URL.Path
		}

		metadata, ok := resolveFallbackAuditMetadata(c, route)
		if !ok {
			return
		}

		auditCtx := usecase.AuditLogContext{
			Method:    c.Request.Method,
			Route:     route,
			ActorRole: "",
		}
		if userID, ok := GetUserIDFromContext(c); ok {
			auditCtx.ActorUserID = &userID
		}
		if roleName, ok := GetRoleNameFromContext(c); ok {
			auditCtx.ActorRole = roleName
		}

		ctx := usecase.WithAuditLogContext(c.Request.Context(), auditCtx)
		beforeSnapshot, afterSnapshot := buildFallbackAuditSnapshots(c, route)

		if err := auditLog.Record(ctx, model.AuditLogRecordRequest{
			ActorUserID:   auditCtx.ActorUserID,
			ActorRole:     auditCtx.ActorRole,
			Action:        resolveAuditAction(c.Request.Method),
			Module:        metadata.module,
			EntityType:    metadata.entityType,
			EntityID:      metadata.entityID,
			EntityLabel:   metadata.entityLabel,
			Method:        auditCtx.Method,
			Route:         auditCtx.Route,
			BeforeData:    beforeSnapshot,
			AfterData:     afterSnapshot,
			ChangedFields: []model.AuditLogChangedField{},
		}); err != nil {
			slog.Error(
				"failed to record fallback audit log",
				slog.String("route", route),
				slog.String("method", c.Request.Method),
				slog.String("entity_type", metadata.entityType),
				slog.String("error", err.Error()),
			)
		}
	}
}

func shouldLogFallbackAudit(method string, status int) bool {
	if status < http.StatusOK || status >= http.StatusMultipleChoices {
		return false
	}

	switch method {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	default:
		return false
	}
}

func resolveFallbackAuditMetadata(c *gin.Context, route string) (auditFallbackMetadata, bool) {
	switch {
	case strings.HasPrefix(route, "/api/v1/master-plans"):
		return resolveMasterPlanFallbackMetadata(c, route), true
	case route == "/api/v1/marker-plans":
		return auditFallbackMetadata{
			module:      "work-order-production",
			entityType:  "marker_plans",
			entityLabel: "Marker Plan",
		}, true
	case route == "/api/v1/spreading-cutting-plans":
		return auditFallbackMetadata{
			module:      "work-order-production",
			entityType:  "spreading_cutting_plans",
			entityLabel: "Spreading & Cutting Plan",
		}, true
	case route == "/api/v1/data-approve-cutting-plans":
		return auditFallbackMetadata{
			module:      "work-order-production",
			entityType:  "data_approve_cutting_plans",
			entityLabel: "Data Approve Cutting Plan",
		}, true
	case strings.HasPrefix(route, "/api/v1/timeline-plans"):
		return auditFallbackMetadata{
			module:      "work-order-production",
			entityType:  "timeline_plans",
			entityID:    firstNonEmptyAudit(c.Param("id"), c.Param("wo_id")),
			entityLabel: buildEntityLabel("Timeline Plan", firstNonEmptyAudit(c.Param("id"), c.Param("wo_id"))),
		}, true
	case route == "/api/v1/work-orders/:id/material-lists":
		woID := c.Param("id")
		return auditFallbackMetadata{
			module:      "work-order-production",
			entityType:  "material_lists",
			entityID:    woID,
			entityLabel: buildEntityLabel("Material List WO", woID),
		}, true
	case strings.HasPrefix(route, "/api/v1/material-lists/"):
		id := c.Param("id")
		return auditFallbackMetadata{
			module:      "work-order-production",
			entityType:  "material_lists",
			entityID:    id,
			entityLabel: buildEntityLabel("Material List", id),
		}, true
	case strings.HasPrefix(route, "/api/v1/material-list-items/"):
		id := c.Param("id")
		return auditFallbackMetadata{
			module:      "work-order-production",
			entityType:  "material_list_items",
			entityID:    id,
			entityLabel: buildEntityLabel("Material List Item", id),
		}, true
	case route == "/api/v1/rekonsiliasi":
		return auditFallbackMetadata{
			module:      "work-order-production",
			entityType:  "rekonsiliasis",
			entityLabel: "Rekonsiliasi Material",
		}, true
	case route == "/api/v1/rekonsiliasi/:id":
		id := c.Param("id")
		return auditFallbackMetadata{
			module:      "work-order-production",
			entityType:  "rekonsiliasis",
			entityID:    id,
			entityLabel: buildEntityLabel("Rekonsiliasi Material", id),
		}, true
	case route == "/api/v1/rekonsiliasi/:id/refresh":
		id := c.Param("id")
		return auditFallbackMetadata{
			module:      "work-order-production",
			entityType:  "rekonsiliasis",
			entityID:    id,
			entityLabel: buildEntityLabel("Refresh Rekonsiliasi", id),
		}, true
	case route == "/api/v1/reports/:divisi":
		return resolveFactoryReportFallbackMetadata(c), true
	case route == "/api/v1/work-orders/:id/retur":
		id := c.Param("id")
		return auditFallbackMetadata{
			module:      "work-order-production",
			entityType:  "retur_clients",
			entityID:    id,
			entityLabel: buildEntityLabel("Retur Client WO", id),
		}, true
	default:
		return auditFallbackMetadata{}, false
	}
}

func resolveMasterPlanFallbackMetadata(c *gin.Context, route string) auditFallbackMetadata {
	id := firstNonEmptyAudit(c.Param("itemId"), c.Param("id"))

	if strings.Contains(route, "/items") {
		return auditFallbackMetadata{
			module:      "work-order-production",
			entityType:  "master_plan_items",
			entityID:    id,
			entityLabel: buildEntityLabel("Master Plan Item", id),
		}
	}

	return auditFallbackMetadata{
		module:      "work-order-production",
		entityType:  "master_plans",
		entityID:    id,
		entityLabel: buildEntityLabel("Master Plan", id),
	}
}

func resolveFactoryReportFallbackMetadata(c *gin.Context) auditFallbackMetadata {
	division := strings.ToLower(strings.TrimSpace(c.Param("divisi")))
	entityType := "factory_reports"
	entityLabel := "Factory Report"

	switch division {
	case "cutting":
		entityType = "report_cutting"
		entityLabel = "Report Cutting"
	case "sewing":
		entityType = "report_sewing"
		entityLabel = "Report Sewing"
	case "qc-finish":
		entityType = "report_qc_finish"
		entityLabel = "Report QC Finish"
	case "packing":
		entityType = "report_packing"
		entityLabel = "Report Packing"
	case "pengiriman":
		entityType = "report_pengiriman"
		entityLabel = "Report Pengiriman"
	}

	return auditFallbackMetadata{
		module:      "work-order-production",
		entityType:  entityType,
		entityLabel: entityLabel,
	}
}

func buildFallbackAuditSnapshots(c *gin.Context, route string) (map[string]any, map[string]any) {
	snapshot := map[string]any{
		"route":       route,
		"method":      c.Request.Method,
		"status":      c.Writer.Status(),
		"path_params": buildFallbackAuditPathParams(c),
	}

	if c.Request.URL.RawQuery != "" {
		snapshot["query"] = c.Request.URL.RawQuery
	}

	if c.Request.Method == http.MethodDelete {
		return snapshot, nil
	}

	return nil, snapshot
}

func buildFallbackAuditPathParams(c *gin.Context) map[string]string {
	if len(c.Params) == 0 {
		return nil
	}

	params := make(map[string]string, len(c.Params))
	for _, param := range c.Params {
		if strings.TrimSpace(param.Key) == "" || strings.TrimSpace(param.Value) == "" {
			continue
		}
		params[param.Key] = param.Value
	}

	if len(params) == 0 {
		return nil
	}

	return params
}

func resolveAuditAction(method string) string {
	switch method {
	case http.MethodPost:
		return "CREATE"
	case http.MethodPut, http.MethodPatch:
		return "UPDATE"
	case http.MethodDelete:
		return "DELETE"
	default:
		return strings.ToUpper(strings.TrimSpace(method))
	}
}

func buildEntityLabel(prefix, id string) string {
	if strings.TrimSpace(id) == "" {
		return prefix
	}

	return prefix + " #" + id
}

func firstNonEmptyAudit(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}

	return ""
}
