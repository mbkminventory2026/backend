package usecase

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"permatatex-inventory/internal/entity"
	"permatatex-inventory/internal/model"
)

var (
	ErrRoleValidation         = errors.New("invalid role payload")
	ErrRoleManagementNotFound = errors.New("role not found")
	ErrRoleServiceUnavailable = errors.New("role service unavailable")
	ErrRoleNameAlreadyExists  = errors.New("role name already exists")
	ErrRoleInUse              = errors.New("role is still assigned to users")
	ErrReservedRoleProtected  = errors.New("reserved role cannot be modified or deleted")

	roleSortColumns = buildSortWhitelist("created_at", "id_role", "nama_role")
)

type RoleUseCase struct {
	repo     entity.Querier
	dbPool   *pgxpool.Pool
	auditLog *AuditLogUseCase
}

func NewRoleUseCase(repo entity.Querier, dbPool *pgxpool.Pool, auditLog *AuditLogUseCase) (*RoleUseCase, error) {
	if repo == nil {
		return nil, errors.New("role repository is required")
	}
	if dbPool == nil {
		return nil, errors.New("database pool is required")
	}

	return &RoleUseCase{
		repo:     repo,
		dbPool:   dbPool,
		auditLog: auditLog,
	}, nil
}

func (u *RoleUseCase) List(ctx context.Context, filter model.ListQueryFilter) (*model.RoleListResponse, error) {
	page, limit, offset, search, sortBy, sortDesc := normalizeListFilter(filter, "id_role", false, roleSortColumns)
	rows, err := u.repo.ListRoles(ctx, entity.ListRolesParams{
		SearchTerm: search,
		SortBy:     sortBy,
		SortDesc:   sortDesc,
		PageLimit:  limit,
		PageOffset: offset,
	})
	if err != nil {
		return nil, fmt.Errorf("%w: failed to list roles", ErrRoleServiceUnavailable)
	}

	items := make([]model.RoleListItem, 0, len(rows))
	total := int64(0)
	for _, row := range rows {
		total = row.TotalCount
		items = append(items, model.RoleListItem{
			IDRole:    row.IDRole,
			NamaRole:  row.NamaRole,
			CreatedAt: row.CreatedAt.Time.Format(time.RFC3339),
		})
	}

	return &model.RoleListResponse{
		Items:      items,
		Pagination: buildPagination(total, page, limit),
	}, nil
}

func (u *RoleUseCase) GetByID(ctx context.Context, id int32) (*model.RoleResponse, error) {
	role, err := u.repo.GetRoleByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrRoleManagementNotFound
		}
		return nil, fmt.Errorf("%w: failed to get role", ErrRoleServiceUnavailable)
	}

	return u.buildRoleResponse(ctx, role)
}

func (u *RoleUseCase) Create(ctx context.Context, req model.CreateRoleRequest) (*model.RoleResponse, error) {
	tx, err := u.dbPool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to start transaction", ErrRoleServiceUnavailable)
	}
	defer func() {
		if rollbackErr := tx.Rollback(ctx); rollbackErr != nil && !errors.Is(rollbackErr, pgx.ErrTxClosed) {
			_ = rollbackErr
		}
	}()

	qtx := entity.New(tx)
	role, err := qtx.CreateRole(ctx, req.NamaRole)
	if err != nil {
		if isRoleUniqueViolation(err) {
			return nil, ErrRoleNameAlreadyExists
		}
		return nil, fmt.Errorf("%w: failed to create role", ErrRoleServiceUnavailable)
	}

	if err := syncRolePermissions(ctx, qtx, role.IDRole, req.HakAksesIDs); err != nil {
		if isRoleForeignKeyViolation(err) {
			return nil, ErrRoleValidation
		}
		return nil, fmt.Errorf("%w: failed to assign role permissions", ErrRoleServiceUnavailable)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("%w: failed to commit transaction", ErrRoleServiceUnavailable)
	}

	result, err := u.buildRoleResponse(ctx, role)
	if err != nil {
		return nil, err
	}

	u.recordCreateRoleAudit(ctx, result)

	return result, nil
}

func (u *RoleUseCase) Update(ctx context.Context, id int32, req model.UpdateRoleRequest) (*model.RoleResponse, error) {
	existing, err := u.repo.GetRoleByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrRoleManagementNotFound
		}
		return nil, fmt.Errorf("%w: failed to get role", ErrRoleServiceUnavailable)
	}

	if isReservedRole(existing.NamaRole) && existing.NamaRole != req.NamaRole {
		return nil, ErrReservedRoleProtected
	}

	beforeRole, err := u.buildRoleResponse(ctx, existing)
	if err != nil {
		return nil, err
	}

	tx, err := u.dbPool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to start transaction", ErrRoleServiceUnavailable)
	}
	defer func() {
		if rollbackErr := tx.Rollback(ctx); rollbackErr != nil && !errors.Is(rollbackErr, pgx.ErrTxClosed) {
			_ = rollbackErr
		}
	}()

	qtx := entity.New(tx)
	role, err := qtx.UpdateRole(ctx, entity.UpdateRoleParams{
		IDRole:   id,
		NamaRole: req.NamaRole,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrRoleManagementNotFound
		}
		if isRoleUniqueViolation(err) {
			return nil, ErrRoleNameAlreadyExists
		}
		return nil, fmt.Errorf("%w: failed to update role", ErrRoleServiceUnavailable)
	}

	if err := syncRolePermissions(ctx, qtx, id, req.HakAksesIDs); err != nil {
		if isRoleForeignKeyViolation(err) {
			return nil, ErrRoleValidation
		}
		return nil, fmt.Errorf("%w: failed to update role permissions", ErrRoleServiceUnavailable)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("%w: failed to commit transaction", ErrRoleServiceUnavailable)
	}

	result, err := u.buildRoleResponse(ctx, role)
	if err != nil {
		return nil, err
	}

	beforeSnapshot := buildRoleAuditSnapshot(beforeRole)
	afterSnapshot := buildRoleAuditSnapshot(result)
	u.recordUpdateRoleAudit(ctx, result, beforeSnapshot, afterSnapshot)

	return result, nil
}

func (u *RoleUseCase) Delete(ctx context.Context, id int32) error {
	existing, err := u.repo.GetRoleByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrRoleManagementNotFound
		}
		return fmt.Errorf("%w: failed to get role", ErrRoleServiceUnavailable)
	}

	if isReservedRole(existing.NamaRole) {
		return ErrReservedRoleProtected
	}

	existingDetail, err := u.buildRoleResponse(ctx, existing)
	if err != nil {
		return err
	}

	affected, err := u.repo.DeleteRole(ctx, id)
	if err != nil {
		if isRoleForeignKeyViolation(err) {
			return ErrRoleInUse
		}
		return fmt.Errorf("%w: failed to delete role", ErrRoleServiceUnavailable)
	}
	if affected == 0 {
		return ErrRoleManagementNotFound
	}

	u.recordDeleteRoleAudit(ctx, existingDetail)

	return nil
}

func (u *RoleUseCase) buildRoleResponse(ctx context.Context, role entity.Role) (*model.RoleResponse, error) {
	permissions, err := u.repo.ListRolePermissions(ctx, role.IDRole)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to get role permissions", ErrRoleServiceUnavailable)
	}
	permissionIDs, err := u.repo.ListRolePermissionIDs(ctx, role.IDRole)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to get role permission ids", ErrRoleServiceUnavailable)
	}

	return &model.RoleResponse{
		IDRole:      role.IDRole,
		NamaRole:    role.NamaRole,
		CreatedAt:   role.CreatedAt.Time.Format(time.RFC3339),
		Permissions: permissions,
		HakAksesIDs: permissionIDs,
	}, nil
}

func syncRolePermissions(ctx context.Context, qtx *entity.Queries, roleID int32, hakAksesIDs []int32) error {
	if _, err := qtx.DeleteRoleHakAksesByRoleID(ctx, roleID); err != nil {
		return err
	}

	for _, hakAksesID := range hakAksesIDs {
		if err := qtx.CreateRoleHakAkses(ctx, entity.CreateRoleHakAksesParams{
			IDRole:     roleID,
			IDHakAkses: hakAksesID,
		}); err != nil {
			return err
		}
	}

	return nil
}

func isRoleUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func isRoleForeignKeyViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23503"
}

func isReservedRole(name string) bool {
	upperName := strings.ToUpper(strings.TrimSpace(name))
	switch upperName {
	case "SUPER_ADMIN", "ADMIN_SISTEM", "ADMIN_KEUANGAN", "ADMIN_PRODUKSI", "ADMIN_GUDANG", "MANAGER", "CLIENT":
		return true
	}
	return false
}

func (u *RoleUseCase) recordCreateRoleAudit(ctx context.Context, role *model.RoleResponse) {
	if u.auditLog == nil || role == nil {
		return
	}

	auditCtx, ok := GetAuditLogContext(ctx)
	if !ok {
		return
	}

	if err := u.auditLog.Record(ctx, model.AuditLogRecordRequest{
		ActorUserID: auditCtx.ActorUserID,
		ActorRole:   auditCtx.ActorRole,
		Action:      "CREATE",
		Module:      "role-management",
		EntityType:  "roles",
		EntityID:    fmt.Sprintf("%d", role.IDRole),
		EntityLabel: role.NamaRole,
		Method:      auditCtx.Method,
		Route:       auditCtx.Route,
		AfterData:   buildRoleAuditSnapshot(role),
	}); err != nil {
		slog.Error("failed to record role create audit log", slog.String("error", err.Error()))
	}
}

func (u *RoleUseCase) recordUpdateRoleAudit(ctx context.Context, role *model.RoleResponse, beforeSnapshot, afterSnapshot map[string]any) {
	if u.auditLog == nil || role == nil {
		return
	}

	auditCtx, ok := GetAuditLogContext(ctx)
	if !ok {
		return
	}

	if err := u.auditLog.Record(ctx, model.AuditLogRecordRequest{
		ActorUserID:   auditCtx.ActorUserID,
		ActorRole:     auditCtx.ActorRole,
		Action:        "UPDATE",
		Module:        "role-management",
		EntityType:    "roles",
		EntityID:      fmt.Sprintf("%d", role.IDRole),
		EntityLabel:   role.NamaRole,
		Method:        auditCtx.Method,
		Route:         auditCtx.Route,
		BeforeData:    beforeSnapshot,
		AfterData:     afterSnapshot,
		ChangedFields: buildChangedFieldsFromSnapshots(beforeSnapshot, afterSnapshot),
	}); err != nil {
		slog.Error("failed to record role update audit log", slog.String("error", err.Error()))
	}
}

func (u *RoleUseCase) recordDeleteRoleAudit(ctx context.Context, role *model.RoleResponse) {
	if u.auditLog == nil || role == nil {
		return
	}

	auditCtx, ok := GetAuditLogContext(ctx)
	if !ok {
		return
	}

	if err := u.auditLog.Record(ctx, model.AuditLogRecordRequest{
		ActorUserID: auditCtx.ActorUserID,
		ActorRole:   auditCtx.ActorRole,
		Action:      "DELETE",
		Module:      "role-management",
		EntityType:  "roles",
		EntityID:    fmt.Sprintf("%d", role.IDRole),
		EntityLabel: role.NamaRole,
		Method:      auditCtx.Method,
		Route:       auditCtx.Route,
		BeforeData:  buildRoleAuditSnapshot(role),
	}); err != nil {
		slog.Error("failed to record role delete audit log", slog.String("error", err.Error()))
	}
}

func buildRoleAuditSnapshot(role *model.RoleResponse) map[string]any {
	if role == nil {
		return nil
	}

	return map[string]any{
		"id_role":       role.IDRole,
		"nama_role":     role.NamaRole,
		"permissions":   role.Permissions,
		"hak_akses_ids": role.HakAksesIDs,
	}
}
