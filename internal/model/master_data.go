package model

import "permatatex-inventory/internal/entity"

// --- REQUESTS ---

// Departemen
type CreateDepartemenRequest struct {
	NamaDepartemen string `json:"nama_departemen" binding:"required"`
}

type UpdateDepartemenRequest struct {
	NamaDepartemen string `json:"nama_departemen" binding:"required"`
}

// Jenis Barang
type CreateJenisBarangRequest struct {
	NamaJenisBarang string `json:"nama_jenis_barang" binding:"required"`
	Kode            string `json:"kode" binding:"required"`
}

type UpdateJenisBarangRequest struct {
	NamaJenisBarang string `json:"nama_jenis_barang" binding:"required"`
	Kode            string `json:"kode" binding:"required"`
}

// Mitra
type CreateMitraRequest struct {
	NamaPerusahaan string `json:"nama_perusahaan" binding:"required"`
	TipePerusahaan string `json:"tipe_perusahaan" binding:"required"`
	Email          string `json:"email" binding:"omitempty,email"`
	NoTelp         string `json:"no_telp" binding:"required"`
	Alamat         string `json:"alamat"`
	Kota           string `json:"kota"`
	KodePos        string `json:"kode_pos"`
}

type UpdateMitraRequest struct {
	NamaPerusahaan string `json:"nama_perusahaan" binding:"required"`
	TipePerusahaan string `json:"tipe_perusahaan" binding:"required"`
	Email          string `json:"email" binding:"omitempty,email"`
	NoTelp         string `json:"no_telp" binding:"required"`
	Alamat         string `json:"alamat"`
	Kota           string `json:"kota"`
	KodePos        string `json:"kode_pos"`
}

// Barang
type CreateBarangRequest struct {
	NamaBarang    string `json:"nama_barang" binding:"required"`
	Kode          string `json:"kode" binding:"required"`
	IDJenisBarang int32  `json:"id_jenis_barang" binding:"required"`
	IDMitra       int32  `json:"id_mitra" binding:"required"`
}

type UpdateBarangRequest struct {
	NamaBarang    string `json:"nama_barang" binding:"required"`
	Kode          string `json:"kode" binding:"required"`
	IDJenisBarang int32  `json:"id_jenis_barang" binding:"required"`
	IDMitra       int32  `json:"id_mitra" binding:"required"`
}

// Company
type UpdateCompanyRequest struct {
	Nama   string `json:"nama" binding:"required"`
	Alamat string `json:"alamat"`
	Email  string `json:"email" binding:"omitempty,email"`
	NoTelp string `json:"no_telp"`
	About  string `json:"about"`
	Logo   string `json:"logo"`
}

// --- RESPONSES ---

type DepartemenResponse struct {
	ID        int32  `json:"id_departemen"`
	Nama      string `json:"nama_departemen"`
	CreatedAt string `json:"created_at"`
}

type JenisBarangResponse struct {
	ID        int32  `json:"id_jenis_barang"`
	Nama      string `json:"nama_jenis_barang"`
	Kode      string `json:"kode"`
	CreatedAt string `json:"created_at"`
}

type MitraResponse struct {
	ID             int32  `json:"id_mitra"`
	NamaPerusahaan string `json:"nama_perusahaan"`
	TipePerusahaan string `json:"tipe_perusahaan"`
	Email          string `json:"email"`
	NoTelp         string `json:"no_telp"`
	CreatedAt      string `json:"created_at"`
}

type BarangResponse struct {
	ID              int32  `json:"id_barang"`
	Nama            string `json:"nama_barang"`
	Kode            string `json:"kode"`
	NamaPerusahaan  string `json:"nama_perusahaan"`
	NamaJenisBarang string `json:"nama_jenis_barang"`
	CreatedAt       string `json:"created_at"`
}

type CompanyResponse struct {
	ID        int32  `json:"id_company"`
	Nama      string `json:"nama"`
	Alamat    string `json:"alamat"`
	Email     string `json:"email"`
	NoTelp    string `json:"no_telp"`
	About     string `json:"about"`
	Logo      string `json:"logo"`
	CreatedAt string `json:"created_at"`
}

// --- SWAGGER SUCCESS DOCS ---

type ListDepartemenSuccessDoc struct {
	Status  string               `json:"status" example:"success"`
	Message string               `json:"message" example:"departemen retrieved"`
	Data    []DepartemenResponse `json:"data"`
}

type CreateDepartemenSuccessDoc struct {
	Status  string             `json:"status" example:"success"`
	Message string             `json:"message" example:"departemen created"`
	Data    DepartemenResponse `json:"data"`
}

type ListJenisBarangSuccessDoc struct {
	Status  string                `json:"status" example:"success"`
	Message string                `json:"message" example:"jenis barang retrieved"`
	Data    []JenisBarangResponse `json:"data"`
}

type CreateJenisBarangSuccessDoc struct {
	Status  string              `json:"status" example:"success"`
	Message string              `json:"message" example:"jenis barang created"`
	Data    JenisBarangResponse `json:"data"`
}

type ListMitraSuccessDoc struct {
	Status  string          `json:"status" example:"success"`
	Message string          `json:"message" example:"mitra retrieved"`
	Data    []MitraResponse `json:"data"`
}

type CreateMitraSuccessDoc struct {
	Status  string        `json:"status" example:"success"`
	Message string        `json:"message" example:"mitra created"`
	Data    MitraResponse `json:"data"`
}

type ListBarangSuccessDoc struct {
	Status  string           `json:"status" example:"success"`
	Message string           `json:"message" example:"barang retrieved"`
	Data    []BarangResponse `json:"data"`
}

type CreateBarangSuccessDoc struct {
	Status  string         `json:"status" example:"success"`
	Message string         `json:"message" example:"barang created"`
	Data    BarangResponse `json:"data"`
}

type ListPermissionsSuccessDoc struct {
	Status  string           `json:"status" example:"success"`
	Message string           `json:"message" example:"permissions retrieved"`
	Data    []entity.HakAkse `json:"data"`
}

type CompanySuccessDoc struct {
	Status  string          `json:"status" example:"success"`
	Message string          `json:"message" example:"company data retrieved"`
	Data    CompanyResponse `json:"data"`
}
