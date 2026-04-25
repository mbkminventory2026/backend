package entity

import (
	"context"
	"time"
)

const getNextReportPengirimanID = `-- name: GetNextReportPengirimanID :one
SELECT COALESCE(MAX(ID_REPORT_PENGIRIMAN), 0) + 1 AS next_id
FROM REPORT_PENGIRIMAN
`

func (q *Queries) GetNextReportPengirimanID(ctx context.Context) (int32, error) {
	row := q.db.QueryRow(ctx, getNextReportPengirimanID)
	var nextID int32
	err := row.Scan(&nextID)
	return nextID, err
}

const createReportPengiriman = `-- name: CreateReportPengiriman :one
INSERT INTO REPORT_PENGIRIMAN (
    ID_REPORT_PENGIRIMAN,
    "DATE",
    Quantity,
    ID_WO_SHELL_SIZE
) VALUES ($1, $2, $3, $4)
RETURNING ID_REPORT_PENGIRIMAN, "DATE", Quantity, ID_WO_SHELL_SIZE, created_at
`

type CreateReportPengirimanParams struct {
	IDReportPengiriman int32
	Date               time.Time
	Quantity           int32
	IDWOShellSize      int32
}

func (q *Queries) CreateReportPengiriman(ctx context.Context, arg CreateReportPengirimanParams) (ReportPengiriman, error) {
	row := q.db.QueryRow(
		ctx,
		createReportPengiriman,
		arg.IDReportPengiriman,
		arg.Date,
		arg.Quantity,
		arg.IDWOShellSize,
	)

	var i ReportPengiriman
	err := row.Scan(
		&i.IDReportPengiriman,
		&i.Date,
		&i.Quantity,
		&i.IDWOShellSize,
		&i.CreatedAt,
	)
	return i, err
}

const getReportPengirimanByID = `-- name: GetReportPengirimanByID :one
SELECT ID_REPORT_PENGIRIMAN, "DATE", Quantity, ID_WO_SHELL_SIZE, created_at
FROM REPORT_PENGIRIMAN
WHERE ID_REPORT_PENGIRIMAN = $1
LIMIT 1
`

func (q *Queries) GetReportPengirimanByID(ctx context.Context, idReportPengiriman int32) (ReportPengiriman, error) {
	row := q.db.QueryRow(ctx, getReportPengirimanByID, idReportPengiriman)
	var i ReportPengiriman
	err := row.Scan(
		&i.IDReportPengiriman,
		&i.Date,
		&i.Quantity,
		&i.IDWOShellSize,
		&i.CreatedAt,
	)
	return i, err
}

const listReportPengiriman = `-- name: ListReportPengiriman :many
SELECT ID_REPORT_PENGIRIMAN, "DATE", Quantity, ID_WO_SHELL_SIZE, created_at
FROM REPORT_PENGIRIMAN
WHERE
    ($1::date IS NULL OR "DATE" >= $1::date)
    AND ($2::date IS NULL OR "DATE" <= $2::date)
    AND ($3::int IS NULL OR ID_WO_SHELL_SIZE = $3::int)
ORDER BY ID_REPORT_PENGIRIMAN DESC
LIMIT $4 OFFSET $5
`

type ListReportPengirimanParams struct {
	DateFrom      any
	DateTo        any
	IDWOShellSize any
	Limit         int32
	Offset        int32
}

func (q *Queries) ListReportPengiriman(ctx context.Context, arg ListReportPengirimanParams) ([]ReportPengiriman, error) {
	rows, err := q.db.Query(
		ctx,
		listReportPengiriman,
		arg.DateFrom,
		arg.DateTo,
		arg.IDWOShellSize,
		arg.Limit,
		arg.Offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []ReportPengiriman{}
	for rows.Next() {
		var i ReportPengiriman
		if err := rows.Scan(
			&i.IDReportPengiriman,
			&i.Date,
			&i.Quantity,
			&i.IDWOShellSize,
			&i.CreatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return items, nil
}

const deleteReportPengirimanByID = `-- name: DeleteReportPengirimanByID :execrows
DELETE FROM REPORT_PENGIRIMAN
WHERE ID_REPORT_PENGIRIMAN = $1
`

func (q *Queries) DeleteReportPengirimanByID(ctx context.Context, idReportPengiriman int32) (int64, error) {
	result, err := q.db.Exec(ctx, deleteReportPengirimanByID, idReportPengiriman)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected(), nil
}

const workOrderShellSizeExists = `-- name: WorkOrderShellSizeExists :one
SELECT EXISTS(
    SELECT 1
    FROM WORK_ORDER_SHELL_SIZE
    WHERE ID_WO_SHELL_SIZE = $1
) AS exists
`

func (q *Queries) WorkOrderShellSizeExists(ctx context.Context, idWOShellSize int32) (bool, error) {
	row := q.db.QueryRow(ctx, workOrderShellSizeExists, idWOShellSize)
	var exists bool
	err := row.Scan(&exists)
	return exists, err
}
