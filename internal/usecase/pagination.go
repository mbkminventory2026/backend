package usecase

import "permatatex-inventory/internal/model"

func normalizePagination(filter model.TransactionListFilter) (page int32, limit int32, offset int32) {
	page = filter.Page
	if page <= 0 {
		page = 1
	}

	limit = filter.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	offset = (page - 1) * limit
	if offset < 0 {
		offset = 0
	}

	return page, limit, offset
}

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
