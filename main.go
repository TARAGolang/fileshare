package main

import (
	"flag"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"net/http"
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
	e.Renderer = &Template{
		templates: template.Must(template.New("upload.html").Parse(indexhtml)),
	}
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
		// TODO: block brute force
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
			fnb := filepath.Base(fn)
			fname := filepath.Join(*fpath, fnb)
			_, err := os.Stat(fname)
			if err != nil {
				return c.NoContent(http.StatusBadRequest)
			}
			return c.String(http.StatusOK, createLink(c.Request(), fnb, str.Set(fname)))
		}
		return c.NoContent(http.StatusBadRequest)
	})

	g.GET("/upload", func(c echo.Context) error {
		files, err := ioutil.ReadDir(*fpath)
		if err != nil {
			return c.String(http.StatusInternalServerError, err.Error())
		}
		allf := make([]string, 0, len(files))
		for _, info := range files {
			if info.Mode().IsRegular() {
				allf = append(allf, info.Name())
			}
		}
		return c.Render(http.StatusOK, "upload.html", allf)
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
			return c.String(http.StatusOK, createLink(c.Request(), fnb, str.Set(fn)))
		}
		return c.String(http.StatusBadRequest, "Bad request")
	})

	onShutdown(func() {
		str.Save()
	})

	e.Logger.Fatal(e.Start(*addr))
}
