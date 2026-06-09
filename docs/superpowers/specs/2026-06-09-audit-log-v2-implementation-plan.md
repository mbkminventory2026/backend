# Audit Log V2 Implementation Plan

Date: 2026-06-09
Branch: `feat/log`
Depends on: `2026-06-09-audit-log-v2-design.md`

## Objective

Implement Audit Log V2 batch one for operator-only history log with field-level before/after changes for:

- `users`
- `roles`
- `permissions`
- `departemen`
- `barang`
- `jenis_barang`
- `mitra`
- `warna`

## Delivery Strategy

The work will be delivered in small, reviewable commits. Each step must keep the project buildable before moving to the next one.

## Step Breakdown

### Step 1. Database foundation

Goal:
- create new `audit_logs` table

Work:
- add migration `up/down`
- include indexes for common list filters:
  - `created_at`
  - `actor_user_id`
  - `module`
  - `entity_type`
  - `action`

Expected commit:
- `feat(db): add audit logs table`

### Step 2. Query foundation

Goal:
- provide sqlc queries for creating and reading audit logs

Work:
- add `db/query/audit_logs.sql`
- generate sqlc code
- support:
  - insert audit log
  - list audit logs with filters
  - count audit logs
  - get audit log by id

Expected commit:
- `feat(db): add audit log sqlc queries`

### Step 3. Backend model and usecase foundation

Goal:
- introduce normalized DTOs and core usecase methods

Work:
- add request/response models
- add diff item model
- add `AuditLogUseCase`
- add helper functions for:
  - snapshot normalization
  - changed field calculation
  - operator-only list/detail handling

Expected commit:
- `feat(backend): add audit log models and usecase foundation`

### Step 4. Backend HTTP endpoints

Goal:
- expose dedicated operator-only API

Work:
- add `AuditLogHandler`
- register:
  - `GET /api/v1/activity-logs`
  - `GET /api/v1/activity-logs/:id`
- enforce:
  - auth
  - internal user
  - operator role
  - `LOG_READ`
- update Swagger

Expected commit:
- `feat(backend): add activity log API endpoints`

### Step 5. Users audit logging

Goal:
- record audit logs for user create, update, delete

Work:
- capture `before_data`, `after_data`, and `changed_fields`
- set:
  - `module = user-management`
  - `entity_type = users`
  - `entity_label = username`

Expected commit:
- `feat(backend): record audit logs for users`

### Step 6. Roles and permissions audit logging

Goal:
- record audit logs for role and permission create, update, delete

Work:
- `roles`
- `hak_akses`

Expected commit:
- `feat(backend): record audit logs for roles and permissions`

### Step 7. Master data audit logging

Goal:
- record audit logs for selected master data

Work:
- `departemen`
- `barang`
- `jenis_barang`
- `mitra`
- `warna`

Expected commit:
- `feat(backend): record audit logs for master data`

### Step 8. Frontend API foundation

Goal:
- add FE client for the new operator log module

Work:
- add API methods for:
  - list logs
  - get log detail
- define FE-side types

Expected commit:
- `feat(frontend): add activity log API client`

### Step 9. Frontend route and operator menu

Goal:
- expose the module in operator navigation only

Work:
- add sidenav item `History Log`
- add route `/history-log`
- add route-level permission check

Expected commit:
- `feat(frontend): add operator history log route and page`

### Step 10. Frontend detail viewer

Goal:
- allow operator to inspect changes

Work:
- build table with filters and pagination
- add detail modal or drawer
- prioritize `changed_fields`
- show `before_data` and `after_data`

Expected commit:
- `feat(frontend): add activity log detail viewer`

### Step 11. Final quality control

Goal:
- make the full batch releasable

Work:
- backend:
  - `make migrate-up-docker`
  - `make db-gen`
  - `go build ./...`
  - `make swag`
  - `make lint`
- frontend:
  - `npm run build`

Expected commit:
- `chore: regenerate docs and run quality checks`

## Testing Strategy

### Backend

- confirm new records are inserted only on `POST`, `PUT`, and `DELETE` for batch-one modules
- confirm `GET` requests do not generate Audit Log V2 records
- confirm list filtering works
- confirm detail endpoint returns `before_data`, `after_data`, and `changed_fields`
- confirm non-operator users cannot access log endpoints

### Frontend

- confirm operator sees the `History Log` menu
- confirm other roles do not see or access the page
- confirm list loads with filters and pagination
- confirm detail modal shows changed fields first

## Risks and Controls

### Risk: wide code touch in usecases

Control:
- implement by module in separate commits

### Risk: noisy diffs from nullable values

Control:
- normalize snapshots before diffing

### Risk: permission drift

Control:
- enforce both operator role and `LOG_READ`

## Done Definition

This batch is complete when:

- operator can open a dedicated `History Log` page
- only operator can access it
- list and detail endpoints work
- users, roles, permissions, and selected master data produce audit logs on CUD
- update logs include field-level before/after changes
- quality checks pass
