package main

import (
	"flag"
	"log"
	"net/http"
	"os"

	"github.com/covrom/fileshare/config"
	"golang.org/x/crypto/acme/autocert"

	"github.com/BurntSushi/toml"
	"github.com/covrom/fileshare/blacklist"
	"github.com/covrom/fileshare/handlers"
	"github.com/covrom/fileshare/pages"
	"github.com/covrom/fileshare/store"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
)

var (
	conffile = flag.String("c", "config.toml", "toml config file")

	testmode  = flag.Bool("t", false, "send testing mail with link to file")
	testdays  = flag.Int("d", 1, "testing mail link days to expiration")
	testfile  = flag.String("f", "kanban.zip", "file for send testing mail")
	testemail = flag.String("e", "rs@tsov.pro", "email for send testing mail")

	sslmode = flag.String("ssl", "file", "ssl mode: 'auto' - automatically letsencrypt, 'file' - from cert.pem and key.pem, empty for http")

	conf = &config.Conf{
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
		TestMode(str)
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

	e.GET("/", handlers.IndexFake)

	e.GET("/favicon.ico", func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})

	e.GET("/:fname", handlers.GetFile(blist, str))

	e.GET("/buy/files", handlers.BuyPage(conf.FixPrices, conf.Descriptions))

	g := e.Group("/newlink")

	g.Use(middleware.BasicAuth(func(username, password string, ctx echo.Context) (bool, error) {
		ip := ctx.RealIP()
		if len(ip) == 0 {
			ip = ctx.Request().RemoteAddr
		}
		if username == "admin" && password == conf.AdminPassword {
			blist.White(ip)
			return true, nil
		}
		if blist.IsBlack(ip) {
			return false, nil
		}
		blist.PaintBlack(ip)
		return false, nil
	}))

	g.GET("/gen", handlers.GenFileLink(conf.Path, str))

	g.GET("/upload", handlers.UploadHtml(conf.Path))

	g.POST("/upload", handlers.UploadFile(conf.Path, str))

	if len(conf.YaKey) > 0 {
		e.POST("/yapayment", handlers.YandexMoneyPush(e.Logger, conf, str))
	}

	onShutdown(func() {
		str.Save()
		if logf != nil {
			logf.Close()
		}
	})

	switch *sslmode {
	case "file":
		// openssl req -x509 -nodes -days 365 -newkey rsa:2048 -keyout key.pem -out cert.pem
		e.Logger.Error(e.StartTLS(conf.Listen, "cert.pem", "key.pem"))
	case "auto":
		// LetsEncrypt automatically certificating
		e.AutoTLSManager.Cache = autocert.DirCache(".cache")
		e.Logger.Error(e.StartAutoTLS(conf.Listen))
	default:
		e.Logger.Error(e.Start(conf.Listen))
	}

	if logf != nil {
		logf.Close()
	}

	os.Exit(1)
}
