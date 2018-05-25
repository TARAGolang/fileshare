package handlers

import (
	"net/http"

	"github.com/labstack/echo"
)

func IndexFake(c echo.Context) error {
	return c.String(http.StatusOK, "https://www.tsov.pro")
}
