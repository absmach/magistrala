// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package email

import (
	"bytes"
	"fmt"
	"html/template"
	"net/smtp"

	"github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/pkg/errors"
)

var (
	// ErrMissingEmailTemplate missing email template file
	errMissingEmailTemplate = errors.New("Missing e-mail template file")
	errParseTemplate        = errors.New("Parse e-mail template failed")
	errExecTemplate         = errors.New("Execute e-mail template failed")
	errSendMail             = errors.New("Sending e-mail failed")
)

type emailTemplate struct {
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
	Secret      string
	FromAddress string
	FromName    string
	Template    string
}

// Agent for mailing
type Agent struct {
	conf *Config
	auth smtp.Auth
	addr string
	log  logger.Logger
	tmpl *template.Template
}

// New creates new email agent
func New(c *Config) (*Agent, error) {
	a := &Agent{}
	a.conf = c
	if c.Username != "" {
		switch {
		case c.Secret != "":
			a.auth = smtp.CRAMMD5Auth(c.Username, c.Secret)
		case c.Password != "":
			a.auth = smtp.PlainAuth("", c.Username, c.Password, c.Host)
		}
	}
	a.addr = fmt.Sprintf("%s:%s", c.Host, c.Port)

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

	email := new(bytes.Buffer)
	tmpl := emailTemplate{
		To:      To,
		From:    From,
		Subject: Subject,
		Header:  Header,
		Content: Content,
		Footer:  Footer,
	}
	if From == "" {
		tmpl.From = a.conf.FromName
	}

	if err := a.tmpl.Execute(email, tmpl); err != nil {
		return errors.Wrap(errExecTemplate, err)
	}

	if err := smtp.SendMail(a.addr, a.auth, a.conf.FromAddress, To, email.Bytes()); err != nil {
		return errors.Wrap(errSendMail, err)
	}

	return nil
}
