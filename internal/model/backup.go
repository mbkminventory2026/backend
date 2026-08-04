package model

type BackupStartResponse struct {
	BackupID string `json:"backup_id"`
	State    string `json:"state"`
}

type BackupSummary struct {
	BackupID          string `json:"backup_id"`
	CompletedAt       string `json:"completed_at"`
	EncryptedSize     int64  `json:"encrypted_size_bytes"`
	SHA256            string `json:"sha256"`
	DownloadAvailable bool   `json:"download_available"`
}

type BackupLastResult struct {
	BackupID      string `json:"backup_id"`
	State         string `json:"state"`
	FinishedAt    string `json:"finished_at"`
	EncryptedSize *int64 `json:"encrypted_size_bytes,omitempty"`
	SHA256        string `json:"sha256,omitempty"`
}

type BackupStatusResponse struct {
	State      string            `json:"state"`
	BackupID   string            `json:"backup_id,omitempty"`
	StartedAt  string            `json:"started_at,omitempty"`
	LastResult *BackupLastResult `json:"last_result,omitempty"`
	Latest     *BackupSummary    `json:"latest_completed_backup,omitempty"`
}

type BackupListResponse struct {
	Items      []BackupSummary `json:"items"`
	Pagination PaginationMeta  `json:"pagination"`
}

type BackupStartSuccessDoc struct {
	Status  string              `json:"status" example:"success"`
	Message string              `json:"message" example:"backup accepted"`
	Data    BackupStartResponse `json:"data"`
}

type BackupStatusSuccessDoc struct {
	Status  string               `json:"status" example:"success"`
	Message string               `json:"message" example:"backup status retrieved"`
	Data    BackupStatusResponse `json:"data"`
}

type BackupListSuccessDoc struct {
	Status  string             `json:"status" example:"success"`
	Message string             `json:"message" example:"backup history retrieved"`
	Data    BackupListResponse `json:"data"`
}
