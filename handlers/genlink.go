package handlers

import (
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/covrom/fileshare/store"

	"github.com/labstack/echo"
)

func GenFileLink(pth string, str *store.Store) func(c echo.Context) error {
	return func(c echo.Context) error {
		fn := c.QueryParam("file")
		days := c.QueryParam("days")
		dd, err := strconv.Atoi(days)
		if err != nil || dd < 1 {
			dd = 1
		}
		if len(fn) > 0 {
			fnb := filepath.Base(fn)
			fname := filepath.Join(pth, fnb)
			_, err := os.Stat(fname)
			if err != nil {
				return c.NoContent(http.StatusBadRequest)
			}
			return c.String(http.StatusOK, createLink(c.Request(), fnb, str.Set(fname, dd)))
		}
		return c.NoContent(http.StatusBadRequest)
	}
}
