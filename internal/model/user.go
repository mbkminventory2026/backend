package model

type CreateUserRequest struct {
	Username     string  `json:"username" binding:"required"`
	Password     string  `json:"password" binding:"required,min=6"`
	IsManager    bool    `json:"is_manager"`
	IDDepartemen *int32  `json:"id_departemen"`
	IDMitra      *int32  `json:"id_mitra"`
	Status       *string `json:"status"` // Opsional, default 'active'
	HakAksesIDs  []int32 `json:"hak_akses_ids"`
}

type UpdateUserRequest struct {
	Username     string  `json:"username" binding:"required"`
	Password     *string `json:"password" binding:"omitempty,min=6"`
	IsManager    bool    `json:"is_manager"`
	IDDepartemen *int32  `json:"id_departemen"`
	IDMitra      *int32  `json:"id_mitra"`
	Status       *string `json:"status"`
	HakAksesIDs  []int32 `json:"hak_akses_ids"`
}

type UserResponse struct {
	IDUser         int32    `json:"id_user"`
	Username       string   `json:"username"`
	IsManager      bool     `json:"is_manager"`
	Status         string   `json:"status"`
	IDDepartemen   *int32   `json:"id_departemen,omitempty"`
	IDMitra        *int32   `json:"id_mitra,omitempty"`
	NamaDepartemen string   `json:"nama_departemen"`
	NamaPerusahaan string   `json:"nama_perusahaan"`
	CreatedAt      string   `json:"created_at"`
	Permissions    []string `json:"permissions,omitempty"`
	HakAksesIDs    []int32  `json:"hak_akses_ids,omitempty"`
}

type ListUsersFilter struct {
	Limit  int32 `json:"limit"`
	Offset int32 `json:"offset"`
}

// Swagger Docs
type UserSuccessDoc struct {
	Status  string       `json:"status" example:"success"`
	Message string       `json:"message" example:"user created"`
	Data    UserResponse `json:"data"`
}

type UserListSuccessDoc struct {
	Status  string         `json:"status" example:"success"`
	Message string         `json:"message" example:"users retrieved"`
	Data    []UserResponse `json:"data"`
}
