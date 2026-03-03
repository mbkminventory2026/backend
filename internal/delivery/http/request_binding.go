package httpdelivery

import "github.com/gin-gonic/gin"

// BindJSON binds and validates JSON request, then aborts with formatted bad request on failure.
func BindJSON(c *gin.Context, target any) bool {
	if err := c.ShouldBindJSON(target); err != nil {
		AbortWithBadRequest(c, err)
		return false
	}

	return true
}
