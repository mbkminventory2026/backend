package model

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
	Satuan        string `json:"satuan" binding:"required"`
	LokasiRak     string `json:"lokasi_rak"`
	StokMinimum   int32  `json:"stok_minimum"`
}

type UpdateBarangRequest struct {
	NamaBarang    string `json:"nama_barang" binding:"required"`
	Kode          string `json:"kode" binding:"required"`
	IDJenisBarang int32  `json:"id_jenis_barang" binding:"required"`
	IDMitra       int32  `json:"id_mitra" binding:"required"`
	Satuan        string `json:"satuan" binding:"required"`
	LokasiRak     string `json:"lokasi_rak"`
	StokMinimum   int32  `json:"stok_minimum"`
}

// Profil Perusahaan
type CreateProfilPerusahaanRequest struct {
	Nama            string `json:"nama" binding:"required"`
	Alamat          string `json:"alamat"`
	Email           string `json:"email" binding:"omitempty,email"`
	NoTelp          string `json:"no_telp"`
	About           string `json:"about"`
	Logo            string `json:"logo"`
	BackgroundLogin string `json:"background_login"`
	TextFooter      string `json:"text_footer"`
	LinkWebsite     string `json:"link_website"`
	Medsos          string `json:"medsos"`
}

type UpdateProfilPerusahaanRequest struct {
	Nama            string `json:"nama" binding:"required"`
	Alamat          string `json:"alamat"`
	Email           string `json:"email" binding:"omitempty,email"`
	NoTelp          string `json:"no_telp"`
	About           string `json:"about"`
	Logo            string `json:"logo"`
	BackgroundLogin string `json:"background_login"`
	TextFooter      string `json:"text_footer"`
	LinkWebsite     string `json:"link_website"`
	Medsos          string `json:"medsos"`
}

// Hak Akses
type CreateHakAksesRequest struct {
	KodePermission   string `json:"kode_permission" binding:"required"`
	NamaHalaman      string `json:"nama_halaman" binding:"required"`
	Deskripsi        string `json:"deskripsi"`
	DomainPermission string `json:"domain_permission" binding:"required"`
	AksiPermission   string `json:"aksi_permission" binding:"required"`
}

type UpdateHakAksesRequest struct {
	KodePermission   string `json:"kode_permission" binding:"required"`
	NamaHalaman      string `json:"nama_halaman" binding:"required"`
	Deskripsi        string `json:"deskripsi"`
	DomainPermission string `json:"domain_permission" binding:"required"`
	AksiPermission   string `json:"aksi_permission" binding:"required"`
}

// Warna
type CreateWarnaRequest struct {
	NamaWarna string  `json:"nama_warna" binding:"required"`
	KodeHex   *string `json:"kode_hex" binding:"omitempty,max=7"`
}

type UpdateWarnaRequest struct {
	NamaWarna string  `json:"nama_warna" binding:"required"`
	KodeHex   *string `json:"kode_hex" binding:"omitempty,max=7"`
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
	Alamat         string `json:"alamat"`
	Kota           string `json:"kota"`
	KodePos        string `json:"kode_pos"`
	CreatedAt      string `json:"created_at"`
}

type BarangResponse struct {
	ID              int32  `json:"id_barang"`
	Nama            string `json:"nama_barang"`
	Kode            string `json:"kode"`
	NamaPerusahaan  string `json:"nama_perusahaan"`
	NamaJenisBarang string `json:"nama_jenis_barang"`
	Satuan          string `json:"satuan"`
	LokasiRak       string `json:"lokasi_rak"`
	StokMinimum     int32  `json:"stok_minimum"`
	CreatedAt       string `json:"created_at"`
}

type ProfilPerusahaanResponse struct {
	ID              int32  `json:"id_profil_perusahaan"`
	Nama            string `json:"nama"`
	Alamat          string `json:"alamat"`
	Email           string `json:"email"`
	NoTelp          string `json:"no_telp"`
	About           string `json:"about"`
	Logo            string `json:"logo"`
	BackgroundLogin string `json:"background_login"`
	TextFooter      string `json:"text_footer"`
	LinkWebsite     string `json:"link_website"`
	Medsos          string `json:"medsos"`
	CreatedAt       string `json:"created_at"`
}

type HakAksesResponse struct {
	ID               int32  `json:"id_hak_akses"`
	KodePermission   string `json:"kode_permission"`
	Nama             string `json:"nama_halaman"`
	Deskripsi        string `json:"deskripsi"`
	DomainPermission string `json:"domain_permission"`
	AksiPermission   string `json:"aksi_permission"`
	CreatedAt        string `json:"created_at"`
}

type WarnaResponse struct {
	ID        int32   `json:"id_warna"`
	NamaWarna string  `json:"nama_warna"`
	KodeHex   *string `json:"kode_hex"`
	CreatedAt string  `json:"created_at"`
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
	Status  string             `json:"status" example:"success"`
	Message string             `json:"message" example:"permissions retrieved"`
	Data    []HakAksesResponse `json:"data"`
}

type ProfilPerusahaanSuccessDoc struct {
	Status  string                   `json:"status" example:"success"`
	Message string                   `json:"message" example:"profil perusahaan data retrieved"`
	Data    ProfilPerusahaanResponse `json:"data"`
}

type HakAksesSuccessDoc struct {
	Status  string           `json:"status" example:"success"`
	Message string           `json:"message" example:"permission created"`
	Data    HakAksesResponse `json:"data"`
}

type ListWarnaSuccessDoc struct {
	Status  string          `json:"status" example:"success"`
	Message string          `json:"message" example:"warna retrieved"`
	Data    []WarnaResponse `json:"data"`
}

type WarnaSuccessDoc struct {
	Status  string        `json:"status" example:"success"`
	Message string        `json:"message" example:"warna created"`
	Data    WarnaResponse `json:"data"`
}
