package entity

import (
	"github.com/jackc/pgx/v5/pgtype"
)

type User struct {
	IDUser       int32              `json:"id_user"`
	Username     pgtype.Text        `json:"username"`
	Password     pgtype.Text        `json:"password"`
	Karyawan     pgtype.Bool        `json:"karyawan"`
	IsManager    pgtype.Bool        `json:"is_manager"`
	IDDepartemen pgtype.Int4        `json:"id_departemen"`
	IDMitra      pgtype.Int4        `json:"id_mitra"`
	CreatedAt    pgtype.Timestamptz `json:"created_at"`
}

type ReportPengiriman struct {
	IDReportPengiriman int32              `json:"id_report_pengiriman"`
	Date               pgtype.Date        `json:"date"`
	Quantity           pgtype.Int4        `json:"quantity"`
	IDWOShellSize      pgtype.Int4        `json:"id_wo_shell_size"`
	CreatedAt          pgtype.Timestamptz `json:"created_at"`
}
