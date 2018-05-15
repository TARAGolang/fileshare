package main

import (
	"flag"
	"log"
	"net/http"
	"path/filepath"
	"time"

	"github.com/covrom/fileshare/store"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
)

var (
	secret = flag.String("secret", "", "secret string for admin API")
	addr   = flag.String("listen", ":8000", "listen to addr:port")
)

func main() {
	flag.Parse()
	if len(*secret) == 0 {
		log.Fatal("Secret not found, please set -secret parameter")
	}
	str := store.NewStore()
	e := echo.New()
	e.HideBanner = true
	e.Use(middleware.Recover())
	e.Use(middleware.Logger())
	e.GET("/", func(c echo.Context) error {
		time.Sleep(3 * time.Second)
		return c.String(http.StatusOK, "Hello, World!")
	})
	e.GET("/:fname", func(c echo.Context) error {
		key := c.QueryParam("key")
		if len(key) > 0 {
			fn, err := str.Get(key)
			if err != nil {
				return c.String(http.StatusBadRequest, "Bad request")
			}
			return c.Attachment(fn, filepath.Base(fn))
		}
		return c.String(http.StatusBadRequest, "Bad request")
	})
	g := e.Group("/newlink")
	g.Use(middleware.BasicAuth(func(username, password string, ctx echo.Context) (bool, error) {
		if username == "admin" && password == *secret {
			return true, nil
		}
		return false, nil
	}))
	g.GET("/new", func(c echo.Context) error {
		fn := c.QueryParam("file")
		if len(fn) > 0 {
			return c.String(http.StatusOK, str.Set(fn))
		}
		return c.String(http.StatusBadRequest, "Bad request")
	})
	e.Logger.Fatal(e.Start(*addr))
}
