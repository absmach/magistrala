// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package outputs

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"text/template"

	"github.com/absmach/supermq/pkg/errors"
	"github.com/absmach/supermq/pkg/messaging"
	"github.com/jmoiron/sqlx"
)

type Postgres struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	User     string `json:"user"`
	Password string `json:"password"`
	Database string `json:"database"`
	Table    string `json:"table"`
	Mapping  string `json:"mapping"`
}

func (p Postgres) Run(ctx context.Context, msg *messaging.Message, val interface{}) error {
	data := map[string]interface{}{
		LogicRespKey: val,
		MsgKey:       msg,
	}

	tmpl, err := template.New("postgres").Parse(p.Mapping)
	if err != nil {
		return err
	}

	var output bytes.Buffer
	if err := tmpl.Execute(&output, data); err != nil {
		return err
	}

	mapping := output.String()
	var columns map[string]interface{}
	if err = json.Unmarshal([]byte(mapping), &columns); err != nil {
		return err
	}

	connStr := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		p.Host, p.Port, p.User, p.Password, p.Database,
	)

	db, err := sqlx.Open("pgx", connStr)
	if err != nil {
		return err
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		return errors.Wrap(errors.New("failed to connect to DB"), err)
	}

	cols := []string{}
	values := []interface{}{}
	placeholders := []string{}
	i := 1
	for k, v := range data {
		cols = append(cols, k)
		values = append(values, v)
		placeholders = append(placeholders, fmt.Sprintf("$%d", i))
		i++
	}

	q := fmt.Sprintf(
		`INSERT INTO %s (%s) VALUES (%s)`,
		p.Table,
		strings.Join(cols, ", "),
		strings.Join(placeholders, ", "),
	)

	_, err = db.Exec(q, values...)
	if err != nil {
		return errors.Wrap(errors.New("failed to insert data"), err)
	}

	return nil
}
