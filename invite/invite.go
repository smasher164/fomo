package invite

import (
	"crypto/tls"
	"html/template"
	"net"
	"net/smtp"
)

var tMsg = template.Must(template.New("msg").Parse(
	`{{.FromName}} is inviting you to {{.EventName}}!
To accept the invitation, visit {{.URL}} to sign up.`))

type Mdata struct {
	FromName  string
	EventName string
	URL       string
}

func Mail(auth smtp.Auth, to []string, from string, msg Mdata) error {
	const address = "smtp.gmail.com:587"
	conn, err := net.Dial("tcp4", address)
	if err != nil {
		return err
	}
	host, _, _ := net.SplitHostPort(address)
	c, err := smtp.NewClient(conn, host)
	if err != nil {
		return err
	}
	defer c.Close()
	if err = c.Hello(host); err != nil {
		return err
	}
	if ok, _ := c.Extension("STARTTLS"); ok {
		config := &tls.Config{ServerName: host}
		if err = c.StartTLS(config); err != nil {
			return err
		}
	}
	if auth != nil {
		if ok, _ := c.Extension("AUTH"); ok {
			if err = c.Auth(auth); err != nil {
				return err
			}
		}
	}
	if err = c.Mail(from); err != nil {
		return err
	}
	for _, addr := range to {
		if err = c.Rcpt(addr); err != nil {
			return err
		}
	}
	// Send body
	w, err := c.Data()
	if err != nil {
		return err
	}
	if err = tMsg.Execute(w, msg); err != nil {
		return err
	}
	if err = w.Close(); err != nil {
		return err
	}
	return c.Quit()
}
