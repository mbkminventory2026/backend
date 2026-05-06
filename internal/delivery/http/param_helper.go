package httpdelivery

import (
	"fmt"
	"math"
	"strconv"

	"github.com/gin-gonic/gin"
)

func parsePathInt32(c *gin.Context, key string) (int32, error) {
	return parseStringInt32(c.Param(key))
}

func parseQueryInt32(c *gin.Context, key string, defaultValue int32) (int32, error) {
	raw := c.DefaultQuery(key, strconv.FormatInt(int64(defaultValue), 10))
	return parseStringInt32(raw)
}

func parseStringInt32(raw string) (int32, error) {
	value, err := strconv.ParseInt(raw, 10, 32)
	if err != nil {
		return 0, err
	}

	if value < math.MinInt32 || value > math.MaxInt32 {
		return 0, fmt.Errorf("value out of int32 range")
	}

	return int32(value), nil
}
