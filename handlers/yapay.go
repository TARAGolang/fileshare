package handlers

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/covrom/fileshare/store"

	"github.com/covrom/fileshare/config"
	"github.com/covrom/fileshare/yamoney"
	"github.com/labstack/echo"
)

func YandexMoneyPush(elog echo.Logger, conf *config.Conf, str *store.Store) func(c echo.Context) error {
	return func(c echo.Context) error {

		yap := &yamoney.YaParams{}

		if err := c.Bind(yap); err != nil {
			return c.NoContent(http.StatusOK)
		}

		elog.Warn("Receive payment:\n", yap)

		if err := yap.CheckSha1(conf.YaKey); err != nil {
			elog.Error("Wrong SHA1")
			return c.NoContent(http.StatusOK)
		}

		fnb := filepath.Base(yap.Label)

		if am, ok := conf.FixPrices[fnb]; ok && am != yap.WithDrawAmount {
			elog.Error("Wrong price:", yap.WithDrawAmount)
			return c.NoContent(http.StatusOK)
		}

		fname := filepath.Join(conf.Path, fnb)
		_, err := os.Stat(fname)
		if err != nil {
			elog.Error("File not exists:", fname)
			return c.NoContent(http.StatusOK)
		}

		if err := yamoney.SendMail(conf.MailSrv, conf.MailFrom, conf.MailPass, yap.Email,
			"Ссылка на загрузку "+fnb, fmt.Sprintf(`Ваша ссылка для скачивания %s
Ссылка действительна в течение одного дня!
Если Вам не удается сачать файл, напишите пожалуйста письмо на %s`,
				createLink(c.Request(), fnb, str.Set(fname, 1)), conf.MailFrom)); err != nil {
			elog.Error(err.Error())
			return c.NoContent(http.StatusOK)
		}

		return c.NoContent(http.StatusOK)
	}
}
