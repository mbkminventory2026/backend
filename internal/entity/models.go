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
