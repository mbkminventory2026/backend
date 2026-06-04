package model

type CreateUserRequest struct {
	Username     string  `json:"username" binding:"required"`
	Password     string  `json:"password" binding:"required,min=6"`
	IDRole       int32   `json:"id_role" binding:"required"`
	IDDepartemen *int32  `json:"id_departemen"`
	IDMitra      *int32  `json:"id_mitra"`
	Status       *string `json:"status"` // Opsional, default 'active'
	HakAksesIDs  []int32 `json:"hak_akses_ids"`
}

type UpdateUserRequest struct {
	Username     string  `json:"username" binding:"required"`
	Password     *string `json:"password" binding:"omitempty,min=6"`
	IDDepartemen *int32  `json:"id_departemen"`
	IDMitra      *int32  `json:"id_mitra"`
	Status       *string `json:"status"`
}

type AssignUserRoleRequest struct {
	IDRole int32 `json:"id_role" binding:"required"`
}

type AssignUserPermissionsRequest struct {
	HakAksesIDs []int32 `json:"hak_akses_ids"`
}

type UserResponse struct {
	IDUser         int32    `json:"id_user"`
	Username       string   `json:"username"`
	Status         string   `json:"status"`
	IDRole         int32    `json:"id_role"`
	NamaRole       string   `json:"nama_role"`
	IDDepartemen   *int32   `json:"id_departemen,omitempty"`
	IDMitra        *int32   `json:"id_mitra,omitempty"`
	NamaDepartemen string   `json:"nama_departemen"`
	NamaPerusahaan string   `json:"nama_perusahaan"`
	CreatedAt      string   `json:"created_at"`
	Permissions    []string `json:"permissions,omitempty"`
	HakAksesIDs    []int32  `json:"hak_akses_ids,omitempty"`
}

type ListUsersFilter struct {
	ListQueryFilter
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
