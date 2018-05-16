package main

import (
	"flag"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/covrom/fileshare/store"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
)

var (
	secret = flag.String("secret", "", "secret string for admin API")
	addr   = flag.String("listen", ":8000", "listen to addr:port")
	fpath  = flag.String("path", "/usr/share/covromfs/", "path to file share")
)

func main() {
	flag.Parse()
	if len(*secret) == 0 {
		log.Fatal("Secret not found, please set -secret parameter")
	}

	if err := os.MkdirAll(*fpath, 0644); err != nil {
		log.Fatal(err)
	} else {
		log.Println("Use path", *fpath)
	}

	str := store.NewStore()
	blist := NewBlackList()
	e := echo.New()
	e.HideBanner = true
	e.Use(middleware.Recover())
	e.Use(middleware.Logger())
	e.GET("/", func(c echo.Context) error {
		time.Sleep(3 * time.Second)
		return c.String(http.StatusOK, "Hello, World!")
	})
	e.GET("/favicon.ico", func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
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
		ip := ctx.RealIP()
		if len(ip) == 0 {
			ip = ctx.Request().RemoteAddr
		}
		if blist.IsBlack(ip) {
			return false, nil
		}
		if username == "admin" && password == *secret {
			return true, nil
		}
		blist.PaintBlack(ip)
		return false, nil
	}))
	g.GET("/gen", func(c echo.Context) error {
		fn := c.QueryParam("file")
		if len(fn) > 0 {
			return c.String(http.StatusOK, str.Set(fn))
		}
		return c.String(http.StatusBadRequest, "Bad request")
	})
	g.GET("/upload", func(c echo.Context) error {
		return c.HTML(http.StatusOK, indexhtml)
	})
	g.POST("/upload", func(c echo.Context) error {
		f, err := c.FormFile("file")
		if err != nil {
			return err
		}
		if len(f.Filename) > 0 {
			src, err := f.Open()
			if err != nil {
				return err
			}
			defer src.Close()

			fnb := filepath.Base(f.Filename)
			fn := filepath.Join(*fpath, fnb)
			// Destination
			dst, err := os.Create(fn)
			if err != nil {
				return err
			}
			defer dst.Close()

			// Copy
			if _, err = io.Copy(dst, src); err != nil {
				return err
			}
			u := &url.URL{}
			*u = *c.Request().URL
			u.Path = "/" + fnb
			u.Host = c.Request().Host
			u.Scheme = "http"
			key := str.Set(fn)
			uv := url.Values{"key": []string{key}}
			u.RawQuery = uv.Encode()
			return c.String(http.StatusOK, u.String())
		}
		return c.String(http.StatusBadRequest, "Bad request")
	})
	onShutdown(func() {
		str.Save()
	})
	e.Logger.Fatal(e.Start(*addr))
}
