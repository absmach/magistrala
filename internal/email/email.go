// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package email

import (
	"bytes"
	"net/mail"
	"strconv"
	"strings"
	"text/template"

	"github.com/absmach/magistrala/pkg/errors"
	"gopkg.in/gomail.v2"
)

var (
	// ErrMissingEmailTemplate missing email template file.
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
	User    string
	Content string
	Host    string
	Footer  string
}

// Config email agent configuration.
type Config struct {
	Host        string `env:"MG_EMAIL_HOST"         envDefault:"localhost"`
	Port        string `env:"MG_EMAIL_PORT"         envDefault:"25"`
	Username    string `env:"MG_EMAIL_USERNAME"     envDefault:"root"`
	Password    string `env:"MG_EMAIL_PASSWORD"     envDefault:""`
	FromAddress string `env:"MG_EMAIL_FROM_ADDRESS" envDefault:""`
	FromName    string `env:"MG_EMAIL_FROM_NAME"    envDefault:""`
	Template    string `env:"MG_EMAIL_TEMPLATE"     envDefault:"email.tmpl"`
}

// Agent for mailing.
type Agent struct {
	conf *Config
	tmpl *template.Template
	dial *gomail.Dialer
}

// New creates new email agent.
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

// Send sends e-mail.
func (a *Agent) Send(to []string, from, subject, header, user, content, footer string) error {
	if a.tmpl == nil {
		return errMissingEmailTemplate
	}

	buff := new(bytes.Buffer)
	e := email{
		To:      to,
		From:    from,
		Subject: subject,
		Header:  header,
		User:    user,
		Content: content,
		Host:    strings.Split(content, "?")[0],
		Footer:  footer,
	}
	if from == "" {
		from := mail.Address{Name: a.conf.FromName, Address: a.conf.FromAddress}
		e.From = from.String()
	}

	if err := a.tmpl.Execute(buff, e); err != nil {
		return errors.Wrap(errExecTemplate, err)
	}

	m := gomail.NewMessage()
	m.SetHeader("From", e.From)
	m.SetHeader("To", to...)
	m.SetHeader("Subject", subject)
	m.SetBody("text/plain", buff.String())

	if err := a.dial.DialAndSend(m); err != nil {
		return errors.Wrap(errSendMail, err)
	}

	return nil
}
