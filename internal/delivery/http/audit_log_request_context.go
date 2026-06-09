package httpdelivery

import (
	"context"

	"github.com/gin-gonic/gin"

	"permatatex-inventory/internal/usecase"
)

func withAuditLogContext(c *gin.Context) context.Context {
	ctx := c.Request.Context()
	route := c.FullPath()
	if route == "" {
		route = c.Request.URL.Path
	}

	auditCtx := usecase.AuditLogContext{
		ActorRole: "",
		Method:    c.Request.Method,
		Route:     route,
	}

	if userID, ok := GetUserIDFromContext(c); ok {
		auditCtx.ActorUserID = &userID
	}
	if roleName, ok := GetRoleNameFromContext(c); ok {
		auditCtx.ActorRole = roleName
	}

	return usecase.WithAuditLogContext(ctx, auditCtx)
}
