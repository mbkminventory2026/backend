package httpdelivery

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

const (
	authorizationHeaderKey  = "authorization"
	authorizationTypeBearer = "bearer"
	authorizationPayloadKey = "authorization_payload"

	// PermissionAllAccess is the special permission granted to super admin/emergency access
	PermissionAllAccess = "ALL_ACCESS"

	//nolint:gosec // permission code constants are identifiers, not credentials
	PermissionAuthChangePassword = "AUTH_CHANGE_PASSWORD"
	//nolint:gosec // permission code constants are identifiers, not credentials
	PermissionPasswordResetRequestCreate = "PASSWORD_RESET_REQUEST_CREATE"
	//nolint:gosec // permission code constants are identifiers, not credentials
	PermissionUserTempPasswordCreate = "USER_TEMP_PASSWORD_CREATE"

	PermissionUserRead    = "USER_READ"
	PermissionUserCreate  = "USER_CREATE"
	PermissionUserUpdate  = "USER_UPDATE"
	PermissionUserDelete  = "USER_DELETE"
	PermissionUserApprove = "USER_APPROVE"

	PermissionRoleRead       = "ROLE_READ"
	PermissionRoleCreate     = "ROLE_CREATE"
	PermissionRoleUpdate     = "ROLE_UPDATE"
	PermissionRoleDelete     = "ROLE_DELETE"
	PermissionUserRoleAssign = "USER_ROLE_ASSIGN"

	PermissionPermissionRead   = "PERMISSION_READ"
	PermissionPermissionCreate = "PERMISSION_CREATE"
	PermissionPermissionUpdate = "PERMISSION_UPDATE"
	PermissionPermissionDelete = "PERMISSION_DELETE"

	PermissionMasterBarangRead   = "MASTER_BARANG_READ"
	PermissionMasterBarangCreate = "MASTER_BARANG_CREATE"
	PermissionMasterBarangUpdate = "MASTER_BARANG_UPDATE"
	PermissionMasterBarangDelete = "MASTER_BARANG_DELETE"

	PermissionMasterWarnaRead   = "MASTER_WARNA_READ"
	PermissionMasterWarnaCreate = "MASTER_WARNA_CREATE"
	PermissionMasterWarnaUpdate = "MASTER_WARNA_UPDATE"
	PermissionMasterWarnaDelete = "MASTER_WARNA_DELETE"

	PermissionMasterMitraRead   = "MASTER_MITRA_READ"
	PermissionMasterMitraCreate = "MASTER_MITRA_CREATE"
	PermissionMasterMitraUpdate = "MASTER_MITRA_UPDATE"
	PermissionMasterMitraDelete = "MASTER_MITRA_DELETE"

	PermissionMasterJenisBarangRead   = "MASTER_JENIS_BARANG_READ"
	PermissionMasterJenisBarangCreate = "MASTER_JENIS_BARANG_CREATE"
	PermissionMasterJenisBarangUpdate = "MASTER_JENIS_BARANG_UPDATE"
	PermissionMasterJenisBarangDelete = "MASTER_JENIS_BARANG_DELETE"

	PermissionMasterCompanyRead   = "MASTER_COMPANY_READ"
	PermissionMasterCompanyCreate = "MASTER_COMPANY_CREATE"
	PermissionMasterCompanyUpdate = "MASTER_COMPANY_UPDATE"
	PermissionMasterCompanyDelete = "MASTER_COMPANY_DELETE"

	PermissionMasterDepartemenRead   = "MASTER_DEPARTEMEN_READ"
	PermissionMasterDepartemenCreate = "MASTER_DEPARTEMEN_CREATE"
	PermissionMasterDepartemenUpdate = "MASTER_DEPARTEMEN_UPDATE"
	PermissionMasterDepartemenDelete = "MASTER_DEPARTEMEN_DELETE"

	PermissionPOClientRead   = "PO_CLIENT_READ"
	PermissionPOClientCreate = "PO_CLIENT_CREATE"
	PermissionPOClientUpdate = "PO_CLIENT_UPDATE"

	PermissionPRInternalRead    = "PR_INTERNAL_READ"
	PermissionPRInternalCreate  = "PR_INTERNAL_CREATE"
	PermissionPRInternalUpdate  = "PR_INTERNAL_UPDATE"
	PermissionPRInternalApprove = "PR_INTERNAL_APPROVE"

	PermissionPOInternalRead    = "PO_INTERNAL_READ"
	PermissionPOInternalCreate  = "PO_INTERNAL_CREATE"
	PermissionPOInternalUpdate  = "PO_INTERNAL_UPDATE"
	PermissionPOInternalApprove = "PO_INTERNAL_APPROVE"

	PermissionWORead   = "WO_READ"
	PermissionWOCreate = "WO_CREATE"
	PermissionWOUpdate = "WO_UPDATE"
	PermissionWOClose  = "WO_CLOSE"

	PermissionProductionSummaryRead  = "PRODUCTION_SUMMARY_READ"
	PermissionProductionReportRead   = "PRODUCTION_REPORT_READ"
	PermissionProductionReportCreate = "PRODUCTION_REPORT_CREATE"
	PermissionProductionReportUpdate = "PRODUCTION_REPORT_UPDATE"

	PermissionTimelineRead   = "TIMELINE_READ"
	PermissionTimelineCreate = "TIMELINE_CREATE"
	PermissionTimelineUpdate = "TIMELINE_UPDATE"

	PermissionMarkerPlanRead   = "MARKER_PLAN_READ"
	PermissionMarkerPlanCreate = "MARKER_PLAN_CREATE"
	PermissionMarkerPlanUpdate = "MARKER_PLAN_UPDATE"

	PermissionCuttingPlanRead   = "CUTTING_PLAN_READ"
	PermissionCuttingPlanCreate = "CUTTING_PLAN_CREATE"
	PermissionCuttingPlanUpdate = "CUTTING_PLAN_UPDATE"

	PermissionInventoryReceive = "INVENTORY_RECEIVE"
	PermissionInventoryIssue   = "INVENTORY_ISSUE"

	PermissionPackingListRead    = "PACKING_LIST_READ"
	PermissionPackingListCreate  = "PACKING_LIST_CREATE"
	PermissionPackingListUpdate  = "PACKING_LIST_UPDATE"
	PermissionPackingListApprove = "PACKING_LIST_APPROVE"

	PermissionSuratJalanClientRead   = "SURAT_JALAN_CLIENT_READ"
	PermissionSuratJalanInternalRead = "SURAT_JALAN_INTERNAL_READ"
	PermissionSuratJalanCreate       = "SURAT_JALAN_CREATE"
	PermissionSuratJalanUpdate       = "SURAT_JALAN_UPDATE"

	PermissionReportRead       = "REPORT_READ"
	PermissionLogRead          = "LOG_READ"
	PermissionDashboardRead    = "DASHBOARD_READ"
	PermissionAIEstimationRead = "AI_ESTIMATION_READ"

	PermissionPasswordResetRequestRead    = "PASSWORD_RESET_REQUEST_READ"
	PermissionPasswordResetRequestApprove = "PASSWORD_RESET_REQUEST_APPROVE"
	PermissionPasswordResetRequestReject  = "PASSWORD_RESET_REQUEST_REJECT"
)

// AuthMiddleware validates JWT token from Authorization header.
func AuthMiddleware(secretKey string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authorizationHeader := c.GetHeader(authorizationHeaderKey)
		if len(authorizationHeader) == 0 {
			AbortWithError(c, NewHTTPError(http.StatusUnauthorized, "authorization header is not provided", nil))
			return
		}

		fields := strings.Fields(authorizationHeader)
		if len(fields) < 2 {
			AbortWithError(c, NewHTTPError(http.StatusUnauthorized, "invalid authorization header format", nil))
			return
		}

		authorizationType := strings.ToLower(fields[0])
		if authorizationType != authorizationTypeBearer {
			AbortWithError(c, NewHTTPError(http.StatusUnauthorized, fmt.Sprintf("unsupported authorization type %s", authorizationType), nil))
			return
		}

		accessToken := fields[1]
		token, err := jwt.Parse(accessToken, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(secretKey), nil
		})

		if err != nil || !token.Valid {
			AbortWithError(c, NewHTTPError(http.StatusUnauthorized, "invalid or expired token", nil))
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			AbortWithError(c, NewHTTPError(http.StatusUnauthorized, "invalid token claims", nil))
			return
		}

		// Store user data in context for handlers to use
		c.Set(authorizationPayloadKey, claims)
		c.Next()
	}
}

// RequirePermission intercepts the request and checks if the authenticated user
// has the required permission in their JWT claims.
func RequirePermission(requiredPermission string) gin.HandlerFunc {
	return func(c *gin.Context) {
		payload, exists := c.Get(authorizationPayloadKey)
		if !exists {
			AbortWithError(c, NewHTTPError(http.StatusUnauthorized, "unauthorized", nil))
			return
		}

		claims, ok := payload.(jwt.MapClaims)
		if !ok {
			AbortWithError(c, NewHTTPError(http.StatusUnauthorized, "invalid token payload", nil))
			return
		}

		permissionsRaw, ok := claims["permissions"]
		if !ok {
			AbortWithError(c, NewHTTPError(http.StatusForbidden, "access denied: no permissions assigned", nil))
			return
		}

		hasAccess := false
		// Permissions in JWT claims are usually parsed as []interface{}
		if list, ok := permissionsRaw.([]interface{}); ok {
			for _, p := range list {
				if str, ok := p.(string); ok {
					if str == PermissionAllAccess || str == requiredPermission {
						hasAccess = true
						break
					}
				}
			}
		} else if list, ok := permissionsRaw.([]string); ok {
			// Fallback in case of manual context override or different decoder config
			for _, s := range list {
				if s == PermissionAllAccess || s == requiredPermission {
					hasAccess = true
					break
				}
			}
		}

		if !hasAccess {
			AbortWithError(c, NewHTTPError(http.StatusForbidden, fmt.Sprintf("access denied: missing required permission '%s'", requiredPermission), nil))
			return
		}

		c.Next()
	}
}

// GetUserIDFromContext retrieves the User ID from the JWT claims stored in context.
func GetUserIDFromContext(c *gin.Context) (int32, bool) {
	payload, exists := c.Get(authorizationPayloadKey)
	if !exists {
		return 0, false
	}

	claims, ok := payload.(jwt.MapClaims)
	if !ok {
		return 0, false
	}

	userIDFloat, ok := claims["user_id"].(float64) // JWT numbers are float64 by default
	if !ok {
		return 0, false
	}

	return int32(userIDFloat), true
}

// GetMitraIDFromContext retrieves the Mitra ID from the JWT claims stored in context, if present.
func GetMitraIDFromContext(c *gin.Context) (*int32, bool) {
	payload, exists := c.Get(authorizationPayloadKey)
	if !exists {
		return nil, false
	}

	claims, ok := payload.(jwt.MapClaims)
	if !ok {
		return nil, false
	}

	mitraIDVal, ok := claims["id_mitra"]
	if !ok || mitraIDVal == nil {
		return nil, true // Successfully identified that user is NOT a Mitra
	}

	mitraIDFloat, ok := mitraIDVal.(float64)
	if !ok {
		return nil, false
	}

	val := int32(mitraIDFloat)
	return &val, true
}

// GetRoleNameFromContext retrieves the role name from JWT claims stored in context.
func GetRoleNameFromContext(c *gin.Context) (string, bool) {
	payload, exists := c.Get(authorizationPayloadKey)
	if !exists {
		return "", false
	}

	claims, ok := payload.(jwt.MapClaims)
	if !ok {
		return "", false
	}

	roleName, ok := claims["role_name"].(string)
	if !ok {
		return "", false
	}

	return roleName, true
}

// IsClientContext determines whether the current authenticated principal is an external client.
func IsClientContext(c *gin.Context) (bool, bool) {
	if roleName, ok := GetRoleNameFromContext(c); ok {
		return strings.EqualFold(roleName, "CLIENT"), true
	}

	mitraID, ok := GetMitraIDFromContext(c)
	if !ok {
		return false, false
	}

	return mitraID != nil, true
}

// RequireInternalUser blocks external client principals from internal-only endpoints.
func RequireInternalUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		isClient, ok := IsClientContext(c)
		if !ok {
			AbortWithError(c, NewHTTPError(http.StatusUnauthorized, "invalid authentication context", nil))
			return
		}

		if isClient {
			AbortWithError(c, NewHTTPError(http.StatusForbidden, "access denied: internal users only", nil))
			return
		}

		c.Next()
	}
}

// HasPermission checks if the authenticated user has a specific permission.
func HasPermission(c *gin.Context, requiredPermission string) bool {
	payload, exists := c.Get(authorizationPayloadKey)
	if !exists {
		return false
	}

	claims, ok := payload.(jwt.MapClaims)
	if !ok {
		return false
	}

	permissionsRaw, ok := claims["permissions"]
	if !ok {
		return false
	}

	if list, ok := permissionsRaw.([]interface{}); ok {
		for _, p := range list {
			if str, ok := p.(string); ok {
				if str == PermissionAllAccess || str == requiredPermission {
					return true
				}
			}
		}
	} else if list, ok := permissionsRaw.([]string); ok {
		for _, s := range list {
			if s == PermissionAllAccess || s == requiredPermission {
				return true
			}
		}
	}

	return false
}
