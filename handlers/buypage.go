package handlers

import (
	"net/http"

	"github.com/labstack/echo"
)

func BuyPage(fprices map[string]string) func(c echo.Context) error {
	return func(c echo.Context) error {
		return c.Render(http.StatusOK, "payment.html", fprices)
	}
}
