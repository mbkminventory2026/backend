# Audit Log V2 Design

Date: 2026-06-09
Branch: `feat/log`
Scope: Backend + Frontend foundation for operator-only history log with field-level before/after tracking

## Current Status

Implemented in `feat/log`:

- new `audit_logs` table and sqlc queries
- dedicated backend API:
  - `GET /api/v1/activity-logs`
  - `GET /api/v1/activity-logs/:id`
- operator-only frontend `History Log` page
- field-level audit logging for:
  - `users`
  - `roles`
  - `permissions`
  - `departemen`
  - `barang`
  - `jenis_barang`
  - `mitra`
  - `warna`

Remaining backlog after batch one:

- integrated end-to-end smoke testing with real operator login
- audit logging for transaction modules

Already completed after batch one:

- audit logging for auth and approval flows:
  - `approve/reject user`
  - `assign role to user`
  - `password reset request create/approve/reject`
  - `change password`
- FE filter persistence to URL for non-table filters
- FE detail formatting polish

## Background

Permatatex already has a legacy activity log foundation:

- tables `LOG_AKTIVITAS` and `LOG_AKTIVITAS_DETAIL`
- async `ActivityLogService`
- global `ActivityLogMiddleware`
- `GET /api/v1/logs`

That foundation is useful, but still too generic for the production requirement:

- operator must be able to view system history log from the UI
- only `Create`, `Update`, and `Delete` actions should be logged
- `Read` actions must not be logged
- updates should show field-level changes with `before` and `after`
- the feature must be added incrementally to avoid destabilizing existing modules

Because the codebase is already broad and active, this feature will use a hybrid approach instead of a full redesign.

## Goals

- Introduce a new audit log structure for richer change history.
- Restrict audit log access to operator users only.
- Capture `CREATE`, `UPDATE`, and `DELETE` only.
- Record field-level changes for prioritized modules in the first batch.
- Expose a dedicated backend API and frontend operator page for viewing logs.
- Keep the legacy log mechanism intact during rollout.

## Non-Goals

- Replacing the legacy `LOG_AKTIVITAS` system immediately.
- Logging every module in the system in the first batch.
- Logging `GET` or read-only operations.
- Real-time audit log streaming.
- Exporting audit log to Excel/PDF in this batch.
- Full payload diff visualization for every nested structure in batch one.

## Recommended Approach

Use a hybrid design:

1. Keep the legacy activity log tables and middleware untouched.
2. Add a new `audit_logs` table for rich audit entries.
3. Record `audit_logs` from usecase-level business operations, not from middleware.
4. Add a dedicated operator-only API for list and detail views.
5. Add a frontend operator page `History Log`.

This gives better audit quality than patching the legacy structure, while avoiding risky big-bang changes.

## Batch 1 Scope

Batch one covers:

- `users`
- `roles`
- `permissions`
- `departemen`
- `barang`
- `jenis_barang`
- `mitra`
- `warna`

Batch one is now implemented on branch `feat/log`.

## Architecture

### Why usecase-level logging

Field-level `before` and `after` snapshots cannot be reliably produced from a global middleware, because middleware does not know:

- the selected entity
- the old persisted state
- the normalized business payload
- the final saved state after transactional updates

Therefore, audit logging will be triggered from the specific usecases that already handle `create`, `update`, and `delete`.

### Compatibility model

- Legacy activity log continues to run through existing middleware.
- Audit Log V2 is additive and independent.
- Operator UI uses Audit Log V2 endpoints only.
- Legacy `GET /api/v1/logs` remains available unless explicitly deprecated later.

## Data Model

Create new table `audit_logs`.

### Proposed columns

- `id BIGSERIAL PRIMARY KEY`
- `actor_user_id INT NULL`
- `actor_username VARCHAR(100) NOT NULL DEFAULT ''`
- `actor_role VARCHAR(100) NOT NULL DEFAULT ''`
- `action VARCHAR(20) NOT NULL`
- `module VARCHAR(100) NOT NULL`
- `entity_type VARCHAR(100) NOT NULL`
- `entity_id VARCHAR(100) NOT NULL DEFAULT ''`
- `entity_label VARCHAR(255) NOT NULL DEFAULT ''`
- `method VARCHAR(10) NOT NULL DEFAULT ''`
- `route VARCHAR(255) NOT NULL DEFAULT ''`
- `before_data JSONB NULL`
- `after_data JSONB NULL`
- `changed_fields JSONB NOT NULL DEFAULT '[]'::jsonb`
- `created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()`

### Notes

- `entity_id` is stored as string for flexible support across current and future tables.
- `before_data` and `after_data` store compact entity snapshots, not arbitrary whole request bodies.
- `changed_fields` stores a normalized diff list that is easier for FE rendering than raw JSON comparison.

### Changed field item shape

`changed_fields` will be a JSON array with objects like:

```json
[
  {
    "field": "nama_barang",
    "before": "Cotton 30s",
    "after": "Cotton 40s"
  }
]
```

## Data Capture Strategy

### Create

- No persisted `before_data`.
- Build `after_data` from the final saved record.
- `changed_fields` may contain every inserted field or remain empty in batch one.

Recommendation for batch one:
- keep `before_data = null`
- keep `after_data = saved snapshot`
- keep `changed_fields = []` for create

### Update

- Load persisted state before mutation.
- Execute update successfully.
- Reload or reconstruct final saved state.
- Compute field-level diff.
- Save `before_data`, `after_data`, and `changed_fields`.

### Delete

- Load persisted state before delete.
- Delete record successfully.
- Save `before_data = deleted snapshot`
- Save `after_data = null`
- Save `changed_fields = []`

## Module Mapping

### Module values

- `user-management`
- `role-management`
- `master-data`

### Entity type values

- `users`
- `roles`
- `hak_akses`
- `departemen`
- `barang`
- `jenis_barang`
- `mitra`
- `warna`

### Entity label guidance

Human-readable identifiers should be stored for table view convenience.

Examples:

- user: username
- role: role name
- permission: permission code
- barang: `nama_barang`
- departemen: `nama_departemen`
- jenis barang: `nama_jenis_barang`
- mitra: `nama_perusahaan`
- warna: `nama`

## Backend Components

### Database

- migration for `audit_logs`
- sqlc query file for audit logs

### Models

New DTOs for:

- list query filter
- list item response
- detail response
- internal diff item structure

### Usecases

New dedicated usecase:

- `AuditLogUseCase`

Responsibilities:

- write audit log entry
- list paginated logs
- get log detail by id
- compute or accept changed field list

### Delivery layer

New dedicated handler:

- `AuditLogHandler`

New endpoints:

- `GET /api/v1/activity-logs`
- `GET /api/v1/activity-logs/:id`

### Authorization

The new log endpoints must be:

- authenticated
- internal-user only
- operator-only by role
- additionally protected by `LOG_READ`

Role gating is required even if another role accidentally gets `LOG_READ` later.

## API Contract

### List endpoint

`GET /api/v1/activity-logs`

Supported query params:

- `page`
- `pageSize`
- `q`
- `action`
- `module`
- `entityType`
- `actorUserId`
- `dateFrom`
- `dateTo`
- `sortBy`
- `sortDesc`

Recommended sortable fields in batch one:

- `created_at`
- `actor_username`
- `action`
- `module`
- `entity_type`

### List response fields

- `id`
- `created_at`
- `actor_user_id`
- `actor_username`
- `actor_role`
- `action`
- `module`
- `entity_type`
- `entity_id`
- `entity_label`

### Detail endpoint

`GET /api/v1/activity-logs/:id`

Detail response fields:

- all list fields
- `method`
- `route`
- `before_data`
- `after_data`
- `changed_fields`

## Frontend Components

### Navigation

Add operator-only sidenav item:

- `History Log`

### Page

New page:

- route: `/history-log`

### FE page contents

- filter bar
- paginated table
- row click opens detail drawer or modal

### Table columns

- waktu
- pengguna
- role
- aksi
- modul
- jenis entitas
- label entitas

### Detail view sections

- metadata
- before
- after
- changed fields

Priority in detail UI:

- show `changed_fields` first
- `before_data` and `after_data` can be collapsible or secondary blocks

## Incremental Commit Plan

Recommended commit sequence:

1. `docs: add audit log v2 design spec`
2. `feat(db): add audit logs table`
3. `feat(db): add audit log sqlc queries`
4. `feat(backend): add audit log models and usecase foundation`
5. `feat(backend): add activity log API endpoints`
6. `feat(backend): record audit logs for users`
7. `feat(backend): record audit logs for roles and permissions`
8. `feat(backend): record audit logs for master data`
9. `feat(frontend): add activity log API client`
10. `feat(frontend): add operator history log route and page`
11. `feat(frontend): add activity log detail viewer`
12. `chore: regenerate docs and run quality checks`

## Risks

### Snapshot drift

If `before_data` and `after_data` are built from inconsistent source structs, diffs may be noisy.

Mitigation:

- use normalized snapshot builders per entity
- keep batch one to small modules only

### Over-logging or duplicate logs

If some modules later also log through middleware-like hooks, one action may create multiple log records.

Mitigation:

- Audit Log V2 writes only from usecases in batch one
- legacy activity log remains logically separate

### Performance overhead

Extra reads for old-state snapshots can add cost on update/delete.

Mitigation:

- only apply to small batch-one modules
- log compact snapshots only

### Role leakage

If only permission check is used, non-operator roles may eventually see logs.

Mitigation:

- require both operator role and `LOG_READ`

## Acceptance Criteria

### Backend

- `audit_logs` table exists
- operator-only list/detail endpoints exist
- `users`, `roles`, `permissions`, and selected master data create audit log entries on create/update/delete
- update logs include field-level `changed_fields`
- read endpoints do not create Audit Log V2 entries

### Frontend

- operator sidenav contains `History Log`
- operator can open log list page
- operator can filter and paginate logs
- operator can inspect detail with `before`, `after`, and `changed_fields`
- non-operator users cannot access the page

## Out of Scope for Next Batch

- approvals logs
- PR/PO/WO audit coverage
- production and warehouse module audit coverage
- export audit logs to Excel
- realtime operator notifications
- migration of old `LOG_AKTIVITAS` data into `audit_logs`
