package usecase

import "github.com/jackc/pgx/v5/pgtype"

func nullableInt32Ptr(value pgtype.Int4) *int32 {
	if !value.Valid {
		return nil
	}
	v := value.Int32
	return &v
}

func nullableTimestampString(value pgtype.Timestamptz) string {
	if !value.Valid {
		return ""
	}
	return value.Time.Format("2006-01-02T15:04:05Z07:00")
}
