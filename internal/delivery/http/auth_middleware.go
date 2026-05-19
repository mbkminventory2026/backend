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

	// PermissionAllAccess is the special permission granted to managers/superadmin
	PermissionAllAccess   = "ALL_ACCESS"
	PermissionUserRead    = "USER_READ"
	PermissionUserCreate  = "USER_CREATE"
	PermissionUserUpdate  = "USER_UPDATE"
	PermissionUserDelete  = "USER_DELETE"
	PermissionUserApprove = "USER_APPROVE"

	PermissionMasterRead   = "MASTER_READ"
	PermissionMasterCreate = "MASTER_CREATE"
	PermissionMasterUpdate = "MASTER_UPDATE"
	PermissionMasterDelete = "MASTER_DELETE"

	PermissionPORead    = "PO_READ"
	PermissionPOCreate  = "PO_CREATE"
	PermissionPOUpdate  = "PO_UPDATE"
	PermissionPRApprove = "PR_APPROVE"

	PermissionWORead   = "WO_READ"
	PermissionWOCreate = "WO_CREATE"
	PermissionWOClose  = "WO_CLOSE"

	PermissionReportRead   = "REPORT_READ"
	PermissionReportCreate = "REPORT_CREATE"

	PermissionInventoryReceive  = "INVENTORY_RECEIVE"
	PermissionInventoryIssue    = "INVENTORY_ISSUE"
	PermissionPackingListCreate = "PACKING_LIST_CREATE"
	PermissionSuratJalanCreate  = "SURAT_JALAN_CREATE"

	PermissionLogRead       = "LOG_READ"
	PermissionDashboardRead = "DASHBOARD_READ"
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

	// Managers or ALL_ACCESS always have full access
	if isManager, ok := claims["is_manager"].(bool); ok && isManager {
		return true
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
