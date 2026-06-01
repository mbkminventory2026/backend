package usecase

import "permatatex-inventory/internal/model"

func buildPagination(total int64, page int32, limit int32) model.PaginationMeta {
	totalPages := int64(0)
	if total > 0 && limit > 0 {
		totalPages = (total + int64(limit) - 1) / int64(limit)
	}

	return model.PaginationMeta{
		Page:       page,
		Limit:      limit,
		TotalItems: total,
		TotalPages: totalPages,
	}
}
