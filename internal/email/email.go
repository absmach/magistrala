// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package email

import (
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"net/smtp"

	"github.com/mainflux/mainflux/logger"
)

var (
	// ErrMissingEmailTemplate missing email template file
	ErrMissingEmailTemplate = errors.New("Missing e-mail template file")
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
	Driver      string
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
	auth smtp.Auth
	addr string
	log  logger.Logger
	tmpl *template.Template
}

// New creates new email agent
func New(c *Config) (*Agent, error) {
	a := &Agent{}
	a.conf = c
	a.auth = smtp.PlainAuth("", c.Username, c.Password, c.Host)
	a.addr = fmt.Sprintf("%s:%s", c.Host, c.Port)

	tmpl, err := template.ParseFiles(c.Template)
	if err != nil {
		return nil, err
	}
	a.tmpl = tmpl
	return a, nil
}

// Send sends e-mail
func (a *Agent) Send(To []string, From, Subject, Header, Content, Footer string) error {
	if a.tmpl == nil {
		return ErrMissingEmailTemplate
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
		return err
	}

	return smtp.SendMail(a.addr, a.auth, a.conf.FromAddress, To, email.Bytes())
}
