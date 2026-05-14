package httpdelivery

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"permatatex-inventory/internal/usecase"
)

func ActivityLogMiddleware(service *usecase.ActivityLogService) gin.HandlerFunc {
	return func(c *gin.Context) {
		if service == nil {
			c.Next()
			return
		}

		start := time.Now()
		c.Next()

		if !shouldLogActivity(c.Request.Method, c.Writer.Status()) {
			return
		}

		route := c.FullPath()
		if route == "" {
			route = c.Request.URL.Path
		}

		tableName, ok := resolveActivityTableName(route, c.Param("type"), c.Param("divisi"))
		if !ok {
			return
		}

		var userID *int32
		if id, ok := GetUserIDFromContext(c); ok {
			userID = &id
		}

		service.Record(usecase.ActivityLogEntry{
			UserID:      userID,
			Action:      c.Request.Method + " " + route,
			TableName:   tableName,
			Description: usecase.BuildActivityDescription(c.Request.Method, route, c.Writer.Status(), time.Since(start)),
		})
	}
}

func shouldLogActivity(method string, status int) bool {
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

func resolveActivityTableName(route, suratJalanType, division string) (string, bool) {
	switch {
	case strings.HasPrefix(route, "/api/v1/master/departemen"):
		return "departemen", true
	case strings.HasPrefix(route, "/api/v1/master/jenis-barang"):
		return "jenis_barang", true
	case strings.HasPrefix(route, "/api/v1/master/mitra"):
		return "mitra", true
	case strings.HasPrefix(route, "/api/v1/master/barang"):
		return "barang", true
	case strings.HasPrefix(route, "/api/v1/master/permissions"):
		return "hak_akses", true
	case strings.HasPrefix(route, "/api/v1/master/company"):
		return "company", true
	case strings.HasPrefix(route, "/api/v1/users"):
		return "users", true
	case strings.HasPrefix(route, "/api/v1/po-clients"):
		return "po_client", true
	case strings.HasPrefix(route, "/api/v1/pr-internals"):
		return "pr_internal", true
	case strings.HasPrefix(route, "/api/v1/po-internals"):
		return "po_internal", true
	case strings.HasPrefix(route, "/api/v1/work-orders"):
		return "work_order", true
	case strings.HasPrefix(route, "/api/v1/inventory/receive"):
		return "rekonsiliasi_material", true
	case strings.HasPrefix(route, "/api/v1/inventory/issue"):
		return "rekonsiliasi_material", true
	case strings.HasPrefix(route, "/api/v1/packing-lists"):
		return "packing_list", true
	case strings.HasPrefix(route, "/api/v1/surat-jalan/"):
		if strings.EqualFold(suratJalanType, "client") {
			return "surat_jalan_client", true
		}
		if strings.EqualFold(suratJalanType, "internal") {
			return "surat_jalan_internal", true
		}
		return "surat_jalan", true
	case strings.HasPrefix(route, "/api/v1/reports/"):
		switch strings.ToLower(division) {
		case "cutting":
			return "report_cutting", true
		case "sewing":
			return "report_sewing", true
		case "qc-finish":
			return "report_qc_finish", true
		case "packing":
			return "report_packing", true
		case "pengiriman":
			return "report_pengiriman", true
		}
		return "report", true
	default:
		return "", false
	}
}
