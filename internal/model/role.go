package model

type CreateRoleRequest struct {
	NamaRole    string  `json:"nama_role" binding:"required"`
	HakAksesIDs []int32 `json:"hak_akses_ids"`
}

type UpdateRoleRequest struct {
	NamaRole    string  `json:"nama_role" binding:"required"`
	HakAksesIDs []int32 `json:"hak_akses_ids"`
}

type RoleListItem struct {
	IDRole    int32  `json:"id_role"`
	NamaRole  string `json:"nama_role"`
	CreatedAt string `json:"created_at"`
}

type RoleResponse struct {
	IDRole      int32    `json:"id_role"`
	NamaRole    string   `json:"nama_role"`
	CreatedAt   string   `json:"created_at"`
	Permissions []string `json:"permissions,omitempty"`
	HakAksesIDs []int32  `json:"hak_akses_ids,omitempty"`
}

type RoleListResponse struct {
	Items      []RoleListItem `json:"items"`
	Pagination PaginationMeta `json:"pagination"`
}

type RoleSuccessDoc struct {
	Status  string       `json:"status" example:"success"`
	Message string       `json:"message" example:"role created"`
	Data    RoleResponse `json:"data"`
}

type RoleListSuccessDoc struct {
	Status  string           `json:"status" example:"success"`
	Message string           `json:"message" example:"roles retrieved"`
	Data    RoleListResponse `json:"data"`
}
