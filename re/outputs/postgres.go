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
	_ "github.com/jackc/pgx/v5/stdlib" // required for SQL access
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

func (p *Postgres) Run(ctx context.Context, msg *messaging.Message, val any) error {
	templData := templateVal{
		Message: msg,
		Result:  val,
	}

	tmpl, err := template.New("postgres").Parse(p.Mapping)
	if err != nil {
		return err
	}

	var output bytes.Buffer
	if err := tmpl.Execute(&output, templData); err != nil {
		return err
	}

	mapping := output.String()
	var columns map[string]any
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

	var (
		cols         []string
		values       []any
		placeholders []string
	)

	i := 1
	for k, v := range columns {
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

func (p *Postgres) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]any{
		"type":     SaveRemotePgType.String(),
		"host":     p.Host,
		"port":     p.Port,
		"user":     p.User,
		"password": p.Password,
		"database": p.Database,
		"table":    p.Table,
		"mapping":  p.Mapping,
	})
}
