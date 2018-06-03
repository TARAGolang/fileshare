package config

type Conf struct {
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
	// descriptions for files
	Descriptions map[string]string
}
