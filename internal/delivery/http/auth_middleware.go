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
