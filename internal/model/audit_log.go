package model

import "time"

type AuditLogChangedField struct {
	Field  string `json:"field"`
	Before any    `json:"before"`
	After  any    `json:"after"`
}

type AuditLogRecordRequest struct {
	ActorUserID   *int32
	ActorUsername string
	ActorRole     string
	Action        string
	Module        string
	EntityType    string
	EntityID      string
	EntityLabel   string
	Method        string
	Route         string
	BeforeData    any
	AfterData     any
	ChangedFields []AuditLogChangedField
}

type AuditLogListFilter struct {
	ListQueryFilter
	Action      string
	Module      string
	EntityType  string
	ActorUserID *int32
	DateFrom    *time.Time
	DateTo      *time.Time
}

type AuditLogListItem struct {
	ID            int64     `json:"id"`
	CreatedAt     string    `json:"created_at"`
	ActorUserID   *int32    `json:"actor_user_id,omitempty"`
	ActorUsername string    `json:"actor_username"`
	ActorRole     string    `json:"actor_role"`
	Action        string    `json:"action"`
	Module        string    `json:"module"`
	EntityType    string    `json:"entity_type"`
	EntityID      string    `json:"entity_id"`
	EntityLabel   string    `json:"entity_label"`
	CreatedAtTime time.Time `json:"-"`
}

type AuditLogDetailResponse struct {
	ID            int64                  `json:"id"`
	CreatedAt     string                 `json:"created_at"`
	ActorUserID   *int32                 `json:"actor_user_id,omitempty"`
	ActorUsername string                 `json:"actor_username"`
	ActorRole     string                 `json:"actor_role"`
	Action        string                 `json:"action"`
	Module        string                 `json:"module"`
	EntityType    string                 `json:"entity_type"`
	EntityID      string                 `json:"entity_id"`
	EntityLabel   string                 `json:"entity_label"`
	Method        string                 `json:"method"`
	Route         string                 `json:"route"`
	BeforeData    map[string]any         `json:"before_data,omitempty"`
	AfterData     map[string]any         `json:"after_data,omitempty"`
	ChangedFields []AuditLogChangedField `json:"changed_fields"`
}

type AuditLogListResponse struct {
	Items      []AuditLogListItem `json:"items"`
	Pagination PaginationMeta     `json:"pagination"`
}

type AuditLogListSuccessDoc struct {
	Status  string               `json:"status" example:"success"`
	Message string               `json:"message" example:"activity logs retrieved"`
	Data    AuditLogListResponse `json:"data"`
}

type AuditLogDetailSuccessDoc struct {
	Status  string                 `json:"status" example:"success"`
	Message string                 `json:"message" example:"activity log detail retrieved"`
	Data    AuditLogDetailResponse `json:"data"`
}
