package entity

import (
	"context"
)

type Querier interface {
	GetUserByUsername(ctx context.Context, username string) (User, error)
	GetNextReportPengirimanID(ctx context.Context) (int32, error)
	CreateReportPengiriman(ctx context.Context, arg CreateReportPengirimanParams) (ReportPengiriman, error)
	GetReportPengirimanByID(ctx context.Context, idReportPengiriman int32) (ReportPengiriman, error)
	ListReportPengiriman(ctx context.Context, arg ListReportPengirimanParams) ([]ReportPengiriman, error)
	DeleteReportPengirimanByID(ctx context.Context, idReportPengiriman int32) (int64, error)
	WorkOrderShellSizeExists(ctx context.Context, idWOShellSize int32) (bool, error)
}
