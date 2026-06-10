package usecase

import "context"

type auditLogContextKey struct{}

type AuditLogContext struct {
	ActorUserID *int32
	ActorRole   string
	Method      string
	Route       string
}

func WithAuditLogContext(ctx context.Context, value AuditLogContext) context.Context {
	return context.WithValue(ctx, auditLogContextKey{}, value)
}

func GetAuditLogContext(ctx context.Context) (AuditLogContext, bool) {
	value, ok := ctx.Value(auditLogContextKey{}).(AuditLogContext)
	return value, ok
}
