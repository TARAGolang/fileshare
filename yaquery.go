package main

import (
	"crypto/sha1"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"net"
	"net/mail"
	"net/smtp"
	"strings"
)

type YaParams struct {
	NotificationType string `form:"notification_type"`
	OperationId      string `form:"operation_id"`
	Amount           string `form:"amount"`
	Currency         string `form:"currency"`
	Datetime         string `form:"datetime"`
	Sender           string `form:"sender"`
	Codepro          string `form:"codepro"`
	Label            string `form:"label"`

	WithDrawAmount string `form:"withdraw_amount"`
	Sha1Hash       string `form:"sha1_hash"`

	Email       string `form:"email"`
	Lastname    string `form:"lastname"`
	Firstname   string `form:"firstname"`
	Fathersname string `form:"fathersname"`
}

func (yp *YaParams) CheckSha1(nsecret string) error {
	str := yp.NotificationType + "&" +
		yp.OperationId + "&" +
		yp.Amount + "&" +
		yp.Currency + "&" +
		yp.Datetime + "&" +
		yp.Sender + "&" +
		yp.Codepro + "&" +
		nsecret + "&" +
		yp.Label

	hasher := sha1.New()
	hasher.Write([]byte(str))
	sha := hex.EncodeToString(hasher.Sum(nil))

	if !strings.EqualFold(sha, yp.Sha1Hash) {
		return fmt.Errorf("Incorrect query")
	}
	return nil
}

func SendMail(email, subj, body string) error {
	from := mail.Address{"", *mailfrom}
	to := mail.Address{"", email}

	// Setup headers
	headers := make(map[string]string)
	headers["From"] = from.String()
	headers["To"] = to.String()
	headers["Subject"] = subj

	// Setup message
	message := ""
	for k, v := range headers {
		message += fmt.Sprintf("%s: %s\r\n", k, v)
	}
	message += "\r\n" + body

	// Connect to the SMTP Server
	servername := *mailsrv

	host, _, _ := net.SplitHostPort(servername)

	auth := smtp.PlainAuth("", *mailfrom, *mailpass, host)

	// TLS config
	tlsconfig := &tls.Config{
		InsecureSkipVerify: true,
		ServerName:         host,
	}

	// Here is the key, you need to call tls.Dial instead of smtp.Dial
	// for smtp servers running on 465 that require an ssl connection
	// from the very beginning (no starttls)

	conn, err := tls.Dial("tcp", servername, tlsconfig)
	if err != nil {
		return err
	}

	c, err := smtp.NewClient(conn, host)
	if err != nil {
		return err
	}

	// Auth
	if err = c.Auth(auth); err != nil {
		return err
	}

	// To && From
	if err = c.Mail(from.Address); err != nil {
		return err
	}

	if err = c.Rcpt(to.Address); err != nil {
		return err
	}

	// Data
	w, err := c.Data()
	if err != nil {
		return err
	}

	_, err = w.Write([]byte(message))
	if err != nil {
		return err
	}

	err = w.Close()
	if err != nil {
		return err
	}

	return c.Quit()
}
