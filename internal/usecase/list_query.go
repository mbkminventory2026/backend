package usecase

import (
	"strings"

	"permatatex-inventory/internal/model"
)

func normalizeListFilter(filter model.ListQueryFilter, defaultSort string, defaultDesc bool, allowed map[string]struct{}) (page int32, limit int32, offset int32, search string, sortBy string, sortDesc bool) {
	page = filter.Page
	limit = filter.Limit
	offset = filter.Offset
	search = strings.TrimSpace(filter.Search)

	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	if page <= 0 {
		if offset > 0 {
			page = (offset / limit) + 1
		} else {
			page = 1
		}
	}

	offset = (page - 1) * limit
	if offset < 0 {
		offset = 0
	}

	sortBy = strings.TrimSpace(filter.SortBy)
	if _, ok := allowed[sortBy]; !ok {
		sortBy = defaultSort
	}

	sortDesc = filter.SortDesc
	if sortBy == defaultSort && filter.SortBy == "" {
		sortDesc = defaultDesc
	}

	return page, limit, offset, search, sortBy, sortDesc
}

func buildSortWhitelist(columns ...string) map[string]struct{} {
	whitelist := make(map[string]struct{}, len(columns))
	for _, column := range columns {
		whitelist[column] = struct{}{}
	}
	return whitelist
}
