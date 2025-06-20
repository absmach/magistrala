// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package re

import (
	"bytes"
	"context"
	"encoding/gob"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"text/template"

	"github.com/absmach/magistrala/alarms"
	"github.com/absmach/senml"
	"github.com/absmach/supermq/pkg/errors"
	"github.com/absmach/supermq/pkg/messaging"
	_ "github.com/jackc/pgx/v5/stdlib" // required for SQL access
	"github.com/jmoiron/sqlx"
	lua "github.com/yuin/gopher-lua"
)

const (
	msgKey       = "message"
	LogicRespKey = "result"
)

func luaEncrypt(l *lua.LState) int {
	key, iv, data, err := decodeParams(l)
	if err != nil {
		l.Push(lua.LNil)
		l.Push(lua.LString(fmt.Sprintf("failed to decode params: %v", err)))
		return 2
	}

	enc, err := encrypt(key, iv, data)
	if err != nil {
		l.Push(lua.LNil)
		l.Push(lua.LString(fmt.Sprintf("failed to encrypt: %v", err)))
		return 2
	}
	l.Push(lua.LString(hex.EncodeToString(enc)))

	return 1
}

func luaDecrypt(l *lua.LState) int {
	key, iv, data, err := decodeParams(l)
	if err != nil {
		l.Push(lua.LNil)
		l.Push(lua.LString(fmt.Sprintf("failed to decode params: %v", err)))
		return 2
	}

	dec, err := decrypt(key, iv, data)
	if err != nil {
		l.Push(lua.LNil)
		l.Push(lua.LString(fmt.Sprintf("failed to decrypt: %v", err)))
		return 2
	}

	l.Push(lua.LString(hex.EncodeToString(dec)))

	return 1
}

func decodeParams(l *lua.LState) (key, iv, data []byte, err error) {
	keyStr := l.ToString(1)
	ivStr := l.ToString(2)
	dataStr := l.ToString(3)

	key, err = hex.DecodeString(keyStr)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to decode key: %v", err)
	}

	iv, err = hex.DecodeString(ivStr)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to decode IV: %v", err)
	}

	data, err = hex.DecodeString(dataStr)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to decode data: %v", err)
	}

	return key, iv, data, nil
}

func (re *re) sendEmail(rule Rule, val interface{}, msg *messaging.Message) error {
	if rule.Outputs.EmailOutput == nil {
		return errors.New("missing email output")
	}

	data := map[string]interface{}{
		LogicRespKey: val,
		msgKey:       msg,
	}

	tmpl, err := template.New("email").Parse(rule.Outputs.EmailOutput.Content)
	if err != nil {
		return err
	}

	var output bytes.Buffer
	if err := tmpl.Execute(&output, data); err != nil {
		return err
	}

	content := output.String()

	if err := re.email.SendEmailNotification(rule.Outputs.EmailOutput.To, "", rule.Outputs.EmailOutput.Subject, "", "", content, "", make(map[string][]byte)); err != nil {
		return err
	}

	return nil
}

func (re *re) sendAlarm(ctx context.Context, ruleID string, val interface{}, msg *messaging.Message) error {
	data, err := json.Marshal(val)
	if err != nil {
		return err
	}

	alarm := alarms.Alarm{
		RuleID:    ruleID,
		DomainID:  msg.Domain,
		ClientID:  msg.Publisher,
		ChannelID: msg.Channel,
		Subtopic:  msg.Subtopic,
	}
	if err := json.Unmarshal(data, &alarm); err != nil {
		return err
	}

	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(alarm); err != nil {
		return err
	}

	m := &messaging.Message{
		Domain:    msg.Domain,
		Publisher: msg.Publisher,
		Created:   msg.Created,
		Channel:   msg.Channel,
		Subtopic:  msg.Subtopic,
		Protocol:  msg.Protocol,
		Payload:   buf.Bytes(),
	}

	topic := messaging.EncodeMessageTopic(msg)
	if err := re.alarmsPub.Publish(ctx, topic, m); err != nil {
		return err
	}
	return nil
}

func (re *re) saveSenml(ctx context.Context, val interface{}, msg *messaging.Message) error {
	// In case there is a single SenML value, convert to slice so we can decode.
	if _, ok := val.([]any); !ok {
		val = []any{val}
	}
	data, err := json.Marshal(val)
	if err != nil {
		return err
	}
	if _, err := senml.Decode(data, senml.JSON); err != nil {
		return err
	}

	m := &messaging.Message{
		Domain:    msg.Domain,
		Publisher: msg.Publisher,
		Created:   msg.Created,
		Channel:   msg.Channel,
		Subtopic:  msg.Subtopic,
		Protocol:  msg.Protocol,
		Payload:   data,
	}
	topic := messaging.EncodeMessageTopic(msg)
	if err := re.writersPub.Publish(ctx, topic, m); err != nil {
		return err
	}

	return nil
}

func (re *re) publishChannel(ctx context.Context, val interface{}, channel, subtopic string, msg *messaging.Message) error {
	data, err := json.Marshal(val)
	if err != nil {
		return err
	}

	m := &messaging.Message{
		Domain:    msg.Domain,
		Publisher: msg.Publisher,
		Created:   msg.Created,
		Channel:   channel,
		Subtopic:  subtopic,
		Protocol:  protocol,
		Payload:   data,
	}

	topic := messaging.EncodeTopicSuffix(msg.Domain, channel, subtopic)
	if err := re.rePubSub.Publish(ctx, topic, m); err != nil {
		return err
	}

	return nil
}

func (re *re) saveRemotePg(rule Rule, val interface{}, msg *messaging.Message) error {
	if rule.Outputs.PosgresDBOutput == nil {
		return errors.New("missing postgresDB output")
	}

	data := map[string]interface{}{
		LogicRespKey: val,
		msgKey:       msg,
	}

	tmpl, err := template.New("postgres").Parse(rule.Outputs.PosgresDBOutput.Mapping)
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

	cfg := rule.Outputs.PosgresDBOutput
	connStr := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.Database,
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
		cfg.Table,
		strings.Join(cols, ", "),
		strings.Join(placeholders, ", "),
	)

	_, err = db.Exec(q, values...)
	if err != nil {
		return errors.Wrap(errors.New("failed to insert data"), err)
	}

	return nil
}
