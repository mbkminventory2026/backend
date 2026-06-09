# Audit Log V2 Implementation Plan

Date: 2026-06-09
Branch: `feat/log`
Depends on: `2026-06-09-audit-log-v2-design.md`

## Status Snapshot

Completed:

- Step 1. Database foundation
- Step 2. Query foundation
- Step 3. Backend model and usecase foundation
- Step 4. Backend HTTP endpoints
- Step 5. Users audit logging
- Step 6. Roles and permissions audit logging
- Step 7. Master data audit logging
- Step 8. Frontend API foundation
- Step 9. Frontend route and operator menu
- Step 10. Frontend detail viewer
- Step 11. Final quality control

This means the original batch-one implementation plan is complete.

## Next Active Subtasks

The remaining work has moved to a follow-up backlog:

1. `Integrated Audit Log Smoke Test`

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

## Follow-Up Breakdown

### Follow-Up 1. Integrated audit log smoke test

Goal:
- validate the finished batch one flow end-to-end with real auth/session data

Status:
- pending
- blocked on local Playwright browser availability in the current Windows environment

Work:
- login as operator
- create/update/delete records from supported modules
- confirm logs appear in backend and FE correctly
- confirm non-operator access is blocked

### Follow-Up 2. Auth and approval flow coverage

Goal:
- extend audit logs beyond CRUD resource pages into access-control workflows

Status:
- completed on `feat/log`

Work:
- `approve/reject user`
- `assign role to user`
- `password reset request create/approve/reject`
- `change password`

### Follow-Up 3. Transaction module coverage

Goal:
- expand audit logs to operational modules

Status:
- completed on `feat/log`

Recommended first targets:
- `pr-internal`
- `po-internal`
- `po-client`
- `work-order`

### Follow-Up 4. Frontend filter persistence

Goal:
- persist non-table audit log filters in URL so operator can refresh/share state

Status:
- completed on `feat/log`

Work:
- sync `action`, `module`, `entityType`, `dateFrom`, `dateTo` into route search params

### Follow-Up 5. Frontend detail polish

Goal:
- make audit log detail easier to read in daily operator use

Status:
- completed on `feat/log`

Work:
- friendlier field labels
- improved JSON formatting
- stronger module/entity badges
- optional diff grouping by section

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

Original batch one is already complete.

The next implementation cycle should start from the follow-up backlog above, not from the completed batch-one steps.
