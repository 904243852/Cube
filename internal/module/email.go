package module

import (
	"crypto/tls"
	"fmt"
	"net/smtp"
	"strings"
)

func init() {
	register("email", func(worker Worker, db Db) interface{} {
		return func(host string, port int, username string, password string) *EmailClient {
			return &EmailClient{
				host:     host,
				port:     port,
				username: username,
				password: password,
			}
		}
	})
}

type EmailClient struct {
	host     string
	port     int
	username string
	password string
}

func (e *EmailClient) Send(receivers []string, subject string, content string, attachments []struct {
	Name        string
	ContentType string
	Base64      string
},
) error {
	address := fmt.Sprintf("%s:%d", e.host, e.port)
	auth := smtp.PlainAuth("", e.username, e.password, e.host)
	msg := []byte(strings.Join([]string{
		"To: " + strings.Join(receivers, ";"),
		"From: " + e.username + "<" + e.username + ">",
		"Subject: " + subject,
		"Content-Type: multipart/mixed; boundary=WebKitBoundary",
		"",
		"--WebKitBoundary",
		"Content-Type: text/plain; charset=utf-8",
		"",
		content,
	}, "\r\n"))
	for _, a := range attachments {
		msg = append(
			msg,
			[]byte(strings.Join([]string{
				"",
				"--WebKitBoundary",
				"Content-Transfer-Encoding: base64",
				"Content-Disposition: attachment",
				"Content-Type: " + a.ContentType + "; name=" + a.Name,
				"",
				a.Base64,
			}, "\r\n"))...,
		)
	}

	if e.port == 25 { // 25 端口直接发送
		return smtp.SendMail(address, auth, e.username, receivers, msg)
	}

	config := &tls.Config{ // 其他端口如 465 需要 TLS 加密
		InsecureSkipVerify: true, // 不校验服务端证书
		ServerName:         e.host,
	}
	conn, err := tls.Dial("tcp", address, config)
	if err != nil {
		return err
	}
	client, err := smtp.NewClient(conn, e.host)
	if err != nil {
		return err
	}
	defer client.Close()
	if ok, _ := client.Extension("AUTH"); ok {
		if err = client.Auth(auth); err != nil {
			return err
		}
	}
	if err = client.Mail(e.username); err != nil {
		return err
	}

	for _, receiver := range receivers {
		if err = client.Rcpt(receiver); err != nil {
			return err
		}
	}
	w, err := client.Data()
	if err != nil {
		return err
	}
	_, err = w.Write(msg)
	if err != nil {
		return err
	}
	err = w.Close()
	if err == nil {
		client.Quit()
	}
	return nil
}
