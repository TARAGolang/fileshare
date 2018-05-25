package handlers

import (
	"net/http"
	"path/filepath"

	"github.com/covrom/fileshare/blacklist"
	"github.com/covrom/fileshare/store"
	"github.com/labstack/echo"
)

func GetFile(blist *blacklist.BlackList, str *store.Store) func(c echo.Context) error {
	return func(c echo.Context) error {
		ip := c.RealIP()
		if len(ip) == 0 {
			ip = c.Request().RemoteAddr
		}
		if blist.IsBlack(ip) {
			return c.String(http.StatusBadRequest, "You are banned. Try tommorow.")
		}
		key := c.QueryParam("key")
		if len(key) > 0 {
			fn, err := str.Get(key)
			if err != nil {
				blist.PaintBlack(ip)
				return c.String(http.StatusBadRequest, "Bad request. You can be blocked when trying to send incorrect request!")
			}
			return c.Attachment(fn, filepath.Base(fn))
		}
		blist.PaintBlack(ip)
		return c.String(http.StatusBadRequest, "Bad request. You can be blocked when trying to send incorrect request!")
	}
}
