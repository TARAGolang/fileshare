package main

import (
	"flag"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"

	"github.com/covrom/fileshare/store"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
)

var (
	testmode  = flag.Bool("t", false, "send testing mail with link to file")
	testfile  = flag.String("f", "kanban.zip", "file for send testing mail")
	testemail = flag.String("e", "rs@tsov.pro", "email for send testing mail")

	secret = flag.String("secret", "", "secret string for admin API")
	addr   = flag.String("listen", ":443", "listen to addr:port")
	fpath  = flag.String("path", "/usr/share/covromfs/", "path to file share")
	// https://tech.yandex.ru/money/doc/dg/reference/notification-p2p-incoming-docpage/
	// https://money.yandex.ru/myservices/online.xml
	yakey    = flag.String("yakey", "", "secret string for Yandex Money incoming payments")
	mailfrom = flag.String("mailfrom", "", "email from")
	mailpass = flag.String("mailpass", "", "password for email from")
	mailsrv  = flag.String("mailsrv", "smtp.yandex.ru:465", "mail server (smtp ssl)")
)

func main() {
	flag.Parse()

	str := store.NewStore()
	blist := NewBlackList()

	if *testmode {
		fnb := filepath.Base(*testfile)
		fname := filepath.Join(*fpath, fnb)
		_, err := os.Stat(fname)
		if err != nil {
			fmt.Println("File not exists:", fname)
			return
		}

		u := &url.URL{}

		if err := SendMail(*testemail, "Ссылка на загрузку "+fnb, fmt.Sprintf(`Ваша ссылка для скачивания %s
Ссылка действительна в течение одного дня!
Если Вам не удается сачать файл, напишите пожалуйста письмо на rs@tsov.pro`,
			createLinkFromURL(u, "localhost", fnb, str.Set(fname)))); err != nil {
			fmt.Println(err.Error())
		}

		return
	}

	if len(*secret) == 0 {
		log.Fatal("Secret not found, please set -secret parameter")
	}

	if err := os.MkdirAll(*fpath, 0644); err != nil {
		log.Fatal(err)
	} else {
		log.Println("Use path", *fpath)
	}

	e := echo.New()
	e.HideBanner = true
	e.Renderer = &Template{
		templates: template.Must(template.New("upload.html").Parse(indexhtml)),
	}
	e.Use(middleware.Recover())
	e.Use(middleware.Logger())

	e.GET("/", func(c echo.Context) error {
		return c.String(http.StatusOK, "https://www.tsov.pro")
	})

	e.GET("/favicon.ico", func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})

	e.GET("/:fname", func(c echo.Context) error {
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

	if len(*yakey) > 0 {
		e.POST("/yapayment", func(c echo.Context) error {
			yap := &YaParams{}
			if err := c.Bind(yap); err != nil {
				return c.NoContent(http.StatusOK)
			}

			e.Logger.Info("Receive payment:\n", yap)

			if err := yap.CheckSha1(*yakey); err != nil {
				e.Logger.Error("Wrong SHA1")
				return c.NoContent(http.StatusOK)
			}

			fixprices := map[string]string{
				"kanban.zip": "10000.00",
			}

			fnb := filepath.Base(yap.Label)

			if am, ok := fixprices[fnb]; ok && am != yap.WithDrawAmount {
				e.Logger.Error("Wrong price:", yap.WithDrawAmount)
				return c.NoContent(http.StatusOK)
			}

			fname := filepath.Join(*fpath, fnb)
			_, err := os.Stat(fname)
			if err != nil {
				e.Logger.Error("File not exists:", fname)
				return c.NoContent(http.StatusOK)
			}

			if err := SendMail(yap.Email, "Ссылка на загрузку "+fnb, fmt.Sprintf(`Ваша ссылка для скачивания %s
Ссылка действительна в течение одного дня!
Если Вам не удается сачать файл, напишите пожалуйста письмо на rs@tsov.pro`,
				createLink(c.Request(), fnb, str.Set(fname)))); err != nil {
				e.Logger.Error(err.Error())
				return c.NoContent(http.StatusOK)
			}

			return c.NoContent(http.StatusOK)
		})
	}

	onShutdown(func() {
		str.Save()
	})

	e.Logger.Fatal(e.StartAutoTLS(*addr))
	// e.Logger.Fatal(e.Start(*addr))
}
