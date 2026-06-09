package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"permatatex-inventory/internal/entity"
	"permatatex-inventory/internal/model"
)

var (
	ErrAuditLogNotFound           = errors.New("audit log not found")
	ErrAuditLogValidation         = errors.New("invalid audit log payload")
	ErrAuditLogServiceUnavailable = errors.New("audit log service unavailable")

	auditLogSortColumns = buildSortWhitelist("created_at", "actor_username", "action", "module", "entity_type")
)

type AuditLogUseCase struct {
	repo entity.Querier
}

func NewAuditLogUseCase(repo entity.Querier) (*AuditLogUseCase, error) {
	if repo == nil {
		return nil, errors.New("audit log repository is required")
	}

	return &AuditLogUseCase{repo: repo}, nil
}

func (u *AuditLogUseCase) Record(ctx context.Context, req model.AuditLogRecordRequest) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	enrichedReq, err := u.enrichAuditLogRecordRequest(ctx, req)
	if err != nil {
		return err
	}
	if err := validateAuditLogRecordRequest(enrichedReq); err != nil {
		return err
	}

	beforeData, err := marshalOptionalAuditSnapshot(enrichedReq.BeforeData)
	if err != nil {
		return fmt.Errorf("%w: before_data", ErrAuditLogValidation)
	}

	afterData, err := marshalOptionalAuditSnapshot(enrichedReq.AfterData)
	if err != nil {
		return fmt.Errorf("%w: after_data", ErrAuditLogValidation)
	}

	changedFields, err := json.Marshal(enrichedReq.ChangedFields)
	if err != nil {
		return fmt.Errorf("%w: changed_fields", ErrAuditLogValidation)
	}

	_, err = u.repo.CreateAuditLog(ctx, entity.CreateAuditLogParams{
		ActorUserID:   toPgInt4(enrichedReq.ActorUserID),
		ActorUsername: strings.TrimSpace(enrichedReq.ActorUsername),
		ActorRole:     strings.TrimSpace(enrichedReq.ActorRole),
		Action:        strings.ToUpper(strings.TrimSpace(enrichedReq.Action)),
		Module:        strings.TrimSpace(enrichedReq.Module),
		EntityType:    strings.TrimSpace(enrichedReq.EntityType),
		EntityID:      strings.TrimSpace(enrichedReq.EntityID),
		EntityLabel:   strings.TrimSpace(enrichedReq.EntityLabel),
		Method:        strings.ToUpper(strings.TrimSpace(enrichedReq.Method)),
		Route:         strings.TrimSpace(enrichedReq.Route),
		BeforeData:    beforeData,
		AfterData:     afterData,
		ChangedFields: changedFields,
	})
	if err != nil {
		return fmt.Errorf("%w: failed to create audit log", ErrAuditLogServiceUnavailable)
	}

	return nil
}

func (u *AuditLogUseCase) enrichAuditLogRecordRequest(ctx context.Context, req model.AuditLogRecordRequest) (model.AuditLogRecordRequest, error) {
	if req.ActorUserID == nil {
		return req, nil
	}

	if strings.TrimSpace(req.ActorUsername) != "" && strings.TrimSpace(req.ActorRole) != "" {
		return req, nil
	}

	user, err := u.repo.GetUserByID(ctx, *req.ActorUserID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return req, ErrAuditLogValidation
		}
		return req, fmt.Errorf("%w: failed to resolve actor user", ErrAuditLogServiceUnavailable)
	}

	if strings.TrimSpace(req.ActorUsername) == "" {
		req.ActorUsername = user.Username
	}
	if strings.TrimSpace(req.ActorRole) == "" {
		req.ActorRole = user.NamaRole
	}

	return req, nil
}

func (u *AuditLogUseCase) List(ctx context.Context, filter model.AuditLogListFilter) (*model.AuditLogListResponse, error) {
	page, limit, offset, search, sortBy, sortDesc := normalizeListFilter(filter.ListQueryFilter, "created_at", true, auditLogSortColumns)

	listParams := entity.ListAuditLogsParams{
		SearchTerm:        search,
		ActionFilter:      strings.ToUpper(strings.TrimSpace(filter.Action)),
		ModuleFilter:      strings.TrimSpace(filter.Module),
		EntityTypeFilter:  strings.TrimSpace(filter.EntityType),
		ActorUserIDFilter: derefInt32(filter.ActorUserID),
		DateFrom:          toPgTimestamptz(filter.DateFrom),
		DateTo:            toPgTimestamptz(filter.DateTo),
		SortBy:            sortBy,
		SortDesc:          sortDesc,
		PageOffset:        offset,
		PageLimit:         limit,
	}

	rows, err := u.repo.ListAuditLogs(ctx, listParams)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to list audit logs", ErrAuditLogServiceUnavailable)
	}

	total, err := u.repo.CountAuditLogs(ctx, entity.CountAuditLogsParams{
		SearchTerm:        listParams.SearchTerm,
		ActionFilter:      listParams.ActionFilter,
		ModuleFilter:      listParams.ModuleFilter,
		EntityTypeFilter:  listParams.EntityTypeFilter,
		ActorUserIDFilter: listParams.ActorUserIDFilter,
		DateFrom:          listParams.DateFrom,
		DateTo:            listParams.DateTo,
	})
	if err != nil {
		return nil, fmt.Errorf("%w: failed to count audit logs", ErrAuditLogServiceUnavailable)
	}

	items := make([]model.AuditLogListItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, toAuditLogListItem(row))
	}

	return &model.AuditLogListResponse{
		Items:      items,
		Pagination: buildPagination(total, page, limit),
	}, nil
}

func (u *AuditLogUseCase) GetByID(ctx context.Context, id int64) (*model.AuditLogDetailResponse, error) {
	row, err := u.repo.GetAuditLogByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrAuditLogNotFound
		}
		return nil, fmt.Errorf("%w: failed to get audit log", ErrAuditLogServiceUnavailable)
	}

	detail, err := toAuditLogDetailResponse(row)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to decode audit log detail", ErrAuditLogServiceUnavailable)
	}

	return detail, nil
}

func validateAuditLogRecordRequest(req model.AuditLogRecordRequest) error {
	action := strings.ToUpper(strings.TrimSpace(req.Action))
	if action != "CREATE" && action != "UPDATE" && action != "DELETE" {
		return ErrAuditLogValidation
	}

	if strings.TrimSpace(req.Module) == "" ||
		strings.TrimSpace(req.EntityType) == "" ||
		strings.TrimSpace(req.ActorUsername) == "" {
		return ErrAuditLogValidation
	}

	return nil
}

func marshalOptionalAuditSnapshot(snapshot any) ([]byte, error) {
	if snapshot == nil {
		return nil, nil
	}

	switch value := snapshot.(type) {
	case map[string]any:
		if len(value) == 0 {
			return nil, nil
		}
	case []any:
		if len(value) == 0 {
			return nil, nil
		}
	}

	return json.Marshal(snapshot)
}

func toPgInt4(value *int32) pgtype.Int4 {
	if value == nil {
		return pgtype.Int4{}
	}

	return pgtype.Int4{
		Int32: *value,
		Valid: true,
	}
}

func toPgTimestamptz(value *time.Time) pgtype.Timestamptz {
	if value == nil {
		return pgtype.Timestamptz{}
	}

	return pgtype.Timestamptz{
		Time:  *value,
		Valid: true,
	}
}

func derefInt32(value *int32) int32 {
	if value == nil {
		return 0
	}

	return *value
}

func toAuditLogListItem(row entity.AuditLog) model.AuditLogListItem {
	item := model.AuditLogListItem{
		ID:            row.ID,
		ActorUsername: row.ActorUsername,
		ActorRole:     row.ActorRole,
		Action:        row.Action,
		Module:        row.Module,
		EntityType:    row.EntityType,
		EntityID:      row.EntityID,
		EntityLabel:   row.EntityLabel,
	}

	if row.CreatedAt.Valid {
		item.CreatedAtTime = row.CreatedAt.Time
		item.CreatedAt = row.CreatedAt.Time.Format(time.RFC3339)
	}
	if row.ActorUserID.Valid {
		actorUserID := row.ActorUserID.Int32
		item.ActorUserID = &actorUserID
	}

	return item
}

func toAuditLogDetailResponse(row entity.AuditLog) (*model.AuditLogDetailResponse, error) {
	beforeData, err := decodeAuditSnapshot(row.BeforeData)
	if err != nil {
		return nil, err
	}

	afterData, err := decodeAuditSnapshot(row.AfterData)
	if err != nil {
		return nil, err
	}

	changedFields, err := decodeAuditChangedFields(row.ChangedFields)
	if err != nil {
		return nil, err
	}

	result := &model.AuditLogDetailResponse{
		ID:            row.ID,
		ActorUsername: row.ActorUsername,
		ActorRole:     row.ActorRole,
		Action:        row.Action,
		Module:        row.Module,
		EntityType:    row.EntityType,
		EntityID:      row.EntityID,
		EntityLabel:   row.EntityLabel,
		Method:        row.Method,
		Route:         row.Route,
		BeforeData:    beforeData,
		AfterData:     afterData,
		ChangedFields: changedFields,
	}

	if row.CreatedAt.Valid {
		result.CreatedAt = row.CreatedAt.Time.Format(time.RFC3339)
	}
	if row.ActorUserID.Valid {
		actorUserID := row.ActorUserID.Int32
		result.ActorUserID = &actorUserID
	}

	return result, nil
}

func decodeAuditSnapshot(raw []byte) (map[string]any, error) {
	if len(raw) == 0 {
		return nil, nil
	}

	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return nil, err
	}

	if len(decoded) == 0 {
		return nil, nil
	}

	return decoded, nil
}

func decodeAuditChangedFields(raw []byte) ([]model.AuditLogChangedField, error) {
	if len(raw) == 0 {
		return []model.AuditLogChangedField{}, nil
	}

	var decoded []model.AuditLogChangedField
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return nil, err
	}

	if decoded == nil {
		return []model.AuditLogChangedField{}, nil
	}

	return decoded, nil
}
