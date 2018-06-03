package main

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"

	"github.com/covrom/fileshare/handlers"
	"github.com/covrom/fileshare/store"

	"github.com/covrom/fileshare/yamoney"
)

func TestMode(str *store.Store) {

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
			handlers.CreateLinkFromURL(u, "localhost", fnb, str.Set(fname, *testdays)), conf.MailFrom)); err != nil {
		fmt.Println(err.Error())
	}

}
