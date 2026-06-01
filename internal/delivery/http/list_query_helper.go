package httpdelivery

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"permatatex-inventory/internal/model"
)

func parseListQuery(c *gin.Context, defaultLimit int32) (model.ListQueryFilter, error) {
	limit, err := parsePageSizeOrLimit(c, defaultLimit)
	if err != nil {
		return model.ListQueryFilter{}, err
	}

	offset, err := parseOptionalQueryInt32(c, "offset")
	if err != nil {
		return model.ListQueryFilter{}, err
	}

	page, err := parsePageOrOffset(c, limit)
	if err != nil {
		return model.ListQueryFilter{}, err
	}

	sortDesc, err := parseQueryBool(c, "sortDesc", false)
	if err != nil {
		return model.ListQueryFilter{}, err
	}

	return model.ListQueryFilter{
		Page:     page,
		Limit:    limit,
		Offset:   offset,
		Search:   firstNonEmpty(c.Query("search"), c.Query("q")),
		SortBy:   strings.TrimSpace(c.Query("sortBy")),
		SortDesc: sortDesc,
	}, nil
}

func parsePageSizeOrLimit(c *gin.Context, defaultValue int32) (int32, error) {
	if raw := strings.TrimSpace(c.Query("pageSize")); raw != "" {
		return parseStringInt32(raw)
	}
	return parseQueryInt32(c, "limit", defaultValue)
}

func parsePageOrOffset(c *gin.Context, limit int32) (int32, error) {
	if raw := strings.TrimSpace(c.Query("page")); raw != "" {
		return parseStringInt32(raw)
	}

	offset, err := parseOptionalQueryInt32(c, "offset")
	if err != nil {
		return 0, err
	}
	if offset > 0 && limit > 0 {
		return (offset / limit) + 1, nil
	}

	return 1, nil
}

func parseOptionalQueryInt32(c *gin.Context, key string) (int32, error) {
	raw := strings.TrimSpace(c.Query(key))
	if raw == "" {
		return 0, nil
	}

	return parseStringInt32(raw)
}

func parseQueryBool(c *gin.Context, key string, defaultValue bool) (bool, error) {
	raw := strings.TrimSpace(c.Query(key))
	if raw == "" {
		return defaultValue, nil
	}

	return strconv.ParseBool(raw)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}

	return ""
}

func setTotalCountHeader(c *gin.Context, total int64) {
	c.Header("X-Total-Count", strconv.FormatInt(total, 10))
}
