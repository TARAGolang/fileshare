package main

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/covrom/fileshare/blacklist"
	"github.com/covrom/fileshare/pages"
	"github.com/covrom/fileshare/store"
	"github.com/covrom/fileshare/yamoney"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
)

var (
	conffile = flag.String("c", "config.toml", "toml config file")

	testmode  = flag.Bool("t", false, "send testing mail with link to file")
	testfile  = flag.String("f", "kanban.zip", "file for send testing mail")
	testemail = flag.String("e", "rs@tsov.pro", "email for send testing mail")

	conf = &struct {
		LogFile string
		// basic auth password for "admin" API
		AdminPassword string
		// listen to addr:port
		Listen string
		// path to file share
		Path string
		// Yandex Money secret string
		// https://tech.yandex.ru/money/doc/dg/reference/notification-p2p-incoming-docpage/
		// https://money.yandex.ru/myservices/online.xml
		YaKey string
		// SMTP TLS/SSL server auth for sending emails
		MailFrom string
		MailPass string
		MailSrv  string
		// fix prices for files, if not defined then any payment allow
		FixPrices map[string]string
	}{
		Listen: ":443",
	}
)

func main() {

	flag.Parse()

	if _, err := toml.DecodeFile(*conffile, conf); err != nil {
		log.Fatal(err)
	}

	// log.Printf("Config: %#v\n", *conf)
	log.Println("Fix prices:", conf.FixPrices)

	str := store.NewStore()
	blist := blacklist.NewBlackList()

	if *testmode {

		fnb := filepath.Base(*testfile)
		fname := filepath.Join(conf.Path, fnb)
		_, err := os.Stat(fname)
		if err != nil {
			fmt.Println("File not exists:", fname)
			return
		}

		u := &url.URL{}

		if err := yamoney.SendMail(conf.MailSrv, conf.MailFrom, conf.MailPass, *testemail,
			"Ссылка на загрузку "+fnb, fmt.Sprintf(`Ваша ссылка для скачивания %s
Ссылка действительна в течение одного дня!
Если Вам не удается сачать файл, напишите пожалуйста письмо на %s`,
				createLinkFromURL(u, "localhost", fnb, str.Set(fname)), conf.MailFrom)); err != nil {
			fmt.Println(err.Error())
		}

		return
	}

	if len(conf.AdminPassword) == 0 {
		log.Fatal("AdminPassword settings not found")
	}

	if len(conf.Path) > 0 {
		if err := os.MkdirAll(conf.Path, 0644); err != nil {
			log.Fatal(err)
		} else {
			log.Println("Files sharing path is", conf.Path)
		}
	}

	e := echo.New()
	e.HideBanner = true
	e.Renderer = pages.MainRenderer()
	e.Use(middleware.Recover())

	var logf *os.File

	if len(conf.LogFile) > 0 {
		var err error

		logf, err = os.OpenFile(conf.LogFile, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0640)
		if err != nil {
			log.Fatal(err)
		}
		defer logf.Close()

		e.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
			Output: logf,
		}))
	} else {
		e.Use(middleware.Logger())
	}

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

	e.GET("/buy/files", func(c echo.Context) error {
		return c.Render(http.StatusOK, "payment.html", conf.FixPrices)
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
		if username == "admin" && password == conf.AdminPassword {
			return true, nil
		}
		blist.PaintBlack(ip)
		return false, nil
	}))

	g.GET("/gen", func(c echo.Context) error {
		fn := c.QueryParam("file")
		if len(fn) > 0 {
			fnb := filepath.Base(fn)
			fname := filepath.Join(conf.Path, fnb)
			_, err := os.Stat(fname)
			if err != nil {
				return c.NoContent(http.StatusBadRequest)
			}
			return c.String(http.StatusOK, createLink(c.Request(), fnb, str.Set(fname)))
		}
		return c.NoContent(http.StatusBadRequest)
	})

	g.GET("/upload", func(c echo.Context) error {
		files, err := ioutil.ReadDir(conf.Path)
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
			fn := filepath.Join(conf.Path, fnb)
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

	if len(conf.YaKey) > 0 {
		e.POST("/yapayment", func(c echo.Context) error {
			yap := &yamoney.YaParams{}

			if err := c.Bind(yap); err != nil {
				return c.NoContent(http.StatusOK)
			}

			e.Logger.Warn("Receive payment:\n", yap)

			if err := yap.CheckSha1(conf.YaKey); err != nil {
				e.Logger.Error("Wrong SHA1")
				return c.NoContent(http.StatusOK)
			}

			fnb := filepath.Base(yap.Label)

			if am, ok := conf.FixPrices[fnb]; ok && am != yap.WithDrawAmount {
				e.Logger.Error("Wrong price:", yap.WithDrawAmount)
				return c.NoContent(http.StatusOK)
			}

			fname := filepath.Join(conf.Path, fnb)
			_, err := os.Stat(fname)
			if err != nil {
				e.Logger.Error("File not exists:", fname)
				return c.NoContent(http.StatusOK)
			}

			if err := yamoney.SendMail(conf.MailSrv, conf.MailFrom, conf.MailPass, yap.Email,
				"Ссылка на загрузку "+fnb, fmt.Sprintf(`Ваша ссылка для скачивания %s
Ссылка действительна в течение одного дня!
Если Вам не удается сачать файл, напишите пожалуйста письмо на %s`,
					createLink(c.Request(), fnb, str.Set(fname)), conf.MailFrom)); err != nil {
				e.Logger.Error(err.Error())
				return c.NoContent(http.StatusOK)
			}

			return c.NoContent(http.StatusOK)
		})
	}

	onShutdown(func() {
		str.Save()
		if logf != nil {
			logf.Close()
		}
	})

	// openssl req -x509 -nodes -days 365 -newkey rsa:2048 -keyout key.pem -out cert.pem
	e.Logger.Error(e.StartTLS(conf.Listen, "cert.pem", "key.pem"))

	// e.AutoTLSManager.Cache = autocert.DirCache("./.cache")
	// e.Logger.Error(e.StartAutoTLS(conf.Listen))

	// e.Logger.Error(e.Start(conf.Listen))

	if logf != nil {
		logf.Close()
	}

	os.Exit(1)
}
