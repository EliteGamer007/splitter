package handlers

import (
	"github.com/labstack/echo/v4"
	"strconv"
)

func requestPage(c echo.Context, fallback int) (int, int) {
	limit, offset := fallback, 0
	if n, err := strconv.Atoi(c.QueryParam("limit")); err == nil && n > 0 {
		limit = n
	}
	if limit > 50 {
		limit = 50
	}
	if n, err := strconv.Atoi(c.QueryParam("offset")); err == nil && n >= 0 {
		offset = n
	}
	return limit, offset
}
