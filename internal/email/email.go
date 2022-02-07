// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package email

import (
	"bytes"
	"net/mail"
	"strconv"
	"text/template"

	"github.com/mainflux/mainflux/pkg/errors"
	"gopkg.in/gomail.v2"
)

var (
	// ErrMissingEmailTemplate missing email template file
	errMissingEmailTemplate = errors.New("Missing e-mail template file")
	errParseTemplate        = errors.New("Parse e-mail template failed")
	errExecTemplate         = errors.New("Execute e-mail template failed")
	errSendMail             = errors.New("Sending e-mail failed")
)

type email struct {
	To      []string
	From    string
	Subject string
	Header  string
	Content string
	Footer  string
}

// Config email agent configuration.
type Config struct {
	Host        string
	Port        string
	Username    string
	Password    string
	FromAddress string
	FromName    string
	Template    string
}

// Agent for mailing
type Agent struct {
	conf *Config
	tmpl *template.Template
	dial *gomail.Dialer
}

// New creates new email agent
func New(c *Config) (*Agent, error) {
	a := &Agent{}
	a.conf = c
	port, err := strconv.Atoi(c.Port)
	if err != nil {
		return a, err
	}
	d := gomail.NewDialer(c.Host, port, c.Username, c.Password)
	a.dial = d

	tmpl, err := template.ParseFiles(c.Template)
	if err != nil {
		return a, errors.Wrap(errParseTemplate, err)
	}
	a.tmpl = tmpl
	return a, nil
}

// Send sends e-mail
func (a *Agent) Send(To []string, From, Subject, Header, Content, Footer string) error {
	if a.tmpl == nil {
		return errMissingEmailTemplate
	}

	buff := new(bytes.Buffer)
	e := email{
		To:      To,
		From:    From,
		Subject: Subject,
		Header:  Header,
		Content: Content,
		Footer:  Footer,
	}
	if From == "" {
		from := mail.Address{Name: a.conf.FromName, Address: a.conf.FromAddress}
		e.From = from.String()
	}

	if err := a.tmpl.Execute(buff, e); err != nil {
		return errors.Wrap(errExecTemplate, err)
	}

	m := gomail.NewMessage()
	m.SetHeader("From", e.From)
	m.SetHeader("To", To...)
	m.SetHeader("Subject", Subject)
	m.SetBody("text/plain", buff.String())

	if err := a.dial.DialAndSend(m); err != nil {
		return errors.Wrap(errSendMail, err)
	}

	return nil
}
