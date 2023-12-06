// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package influxdb

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/absmach/magistrala/pkg/errors"
	jsont "github.com/absmach/magistrala/pkg/transformers/json"
	"github.com/absmach/magistrala/pkg/transformers/senml"
	"github.com/absmach/magistrala/readers"
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
)

const (
	// Measurement for SenML messages.
	defMeasurement = "messages"
)

var _ readers.MessageRepository = (*influxRepository)(nil)

var errResultTime = errors.New("invalid result time")

type RepoConfig struct {
	Bucket string
	Org    string
}
type influxRepository struct {
	cfg    RepoConfig
	client influxdb2.Client
}

// New returns new InfluxDB reader.
func New(client influxdb2.Client, repoCfg RepoConfig) readers.MessageRepository {
	return &influxRepository{
		repoCfg,
		client,
	}
}

func (repo *influxRepository) ReadAll(chanID string, rpm readers.PageMetadata) (readers.MessagesPage, error) {
	format := defMeasurement
	if rpm.Format != "" {
		format = rpm.Format
	}

	queryAPI := repo.client.QueryAPI(repo.cfg.Org)
	condition, timeRange := fmtCondition(chanID, rpm)

	query := fmt.Sprintf(`
	import "influxdata/influxdb/v1"
	import "strings"
	from(bucket: "%s")
	%s
	|> v1.fieldsAsCols()
	|> group()
	|> filter(fn: (r) => r._measurement == "%s")
	%s
	|> sort(columns: ["_time"], desc: true)
	|> limit(n:%d,offset:%d)
	|> yield(name: "sort")`,
		repo.cfg.Bucket,
		timeRange,
		format,
		condition,
		rpm.Limit, rpm.Offset,
	)

	resp, err := queryAPI.Query(context.Background(), query)
	if err != nil {
		return readers.MessagesPage{}, errors.Wrap(readers.ErrReadMessages, err)
	}

	var messages []readers.Message
	var valueMap map[string]interface{}
	for resp.Next() {
		valueMap = resp.Record().Values()
		msg, err := parseMessage(format, valueMap)
		if err != nil {
			return readers.MessagesPage{}, err
		}
		messages = append(messages, msg)
	}
	if resp.Err() != nil {
		return readers.MessagesPage{}, errors.Wrap(readers.ErrReadMessages, resp.Err())
	}

	total, err := repo.count(format, condition, timeRange)
	if err != nil {
		return readers.MessagesPage{}, errors.Wrap(readers.ErrReadMessages, err)
	}

	page := readers.MessagesPage{
		PageMetadata: rpm,
		Total:        total,
		Messages:     messages,
	}

	return page, nil
}

func (repo *influxRepository) count(measurement, condition, timeRange string) (uint64, error) {
	cmd := fmt.Sprintf(`
	import "influxdata/influxdb/v1"
	import "strings"
	from(bucket: "%s")
	%s
	|> v1.fieldsAsCols()
	|> filter(fn: (r) => r._measurement == "%s")
	%s
	|> group()
	|> count(column:"_measurement")
	|> yield(name: "count")
	`,
		repo.cfg.Bucket,
		timeRange,
		measurement,
		condition)
	queryAPI := repo.client.QueryAPI(repo.cfg.Org)
	resp, err := queryAPI.Query(context.Background(), cmd)
	if err != nil {
		return 0, err
	}

	switch resp.Next() {
	case true:
		valueMap := resp.Record().Values()

		val, ok := valueMap["_measurement"].(int64)
		if !ok {
			return 0, nil
		}
		return uint64(val), nil

	default:
		// same as no rows.
		return 0, nil
	}
}

func fmtCondition(chanID string, rpm readers.PageMetadata) (string, string) {
	var timeRange string
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf(`|> filter(fn: (r) => r["channel"] == "%s" )`, chanID))

	var query map[string]interface{}
	meta, err := json.Marshal(rpm)
	if err != nil {
		return sb.String(), timeRange
	}

	if err := json.Unmarshal(meta, &query); err != nil {
		return sb.String(), timeRange
	}

	// range(start:...) is a must for FluxQL syntax.
	from := `start: time(v:0)`
	if value, ok := query["from"]; ok {
		fromValue := int64(value.(float64)*1e9) - 1
		from = fmt.Sprintf(`start: time(v: %d )`, fromValue)
	}
	// range(...,stop:) is an option for FluxQL syntax.
	to := ""
	if value, ok := query["to"]; ok {
		toValue := int64(value.(float64) * 1e9)
		to = fmt.Sprintf(`, stop: time(v: %d )`, toValue)
	}
	// timeRange returned separately because
	// in FluxQL time range must be at the
	// beginning of the query.
	timeRange = fmt.Sprintf(`|> range(%s %s)`, from, to)

	for name, value := range query {
		switch name {
		case
			"channel",
			"subtopic",
			"publisher",
			"name",
			"protocol":
			sb.WriteString(fmt.Sprintf(`|> filter(fn: (r) => r.%s == "%s" )`, name, value))
		case "v":
			comparator := readers.ParseValueComparator(query)
			// flux eq comparator is different
			if comparator == "=" {
				comparator = "=="
			}
			sb.WriteString(`|> filter(fn: (r) => exists r.value)`)
			sb.WriteString(fmt.Sprintf(`|> filter(fn: (r) => r.value %s %v)`, comparator, value))
		case "vb":
			sb.WriteString(`|> filter(fn: (r) => exists r.boolValue)`)
			sb.WriteString(fmt.Sprintf(`|> filter(fn: (r) => r.boolValue == %v)`, value))
		case "vs":
			comparator := readers.ParseValueComparator(query)
			sb.WriteString(`|> filter(fn: (r) => exists r.stringValue)`)
			switch comparator {
			case "=":
				sb.WriteString(fmt.Sprintf(`|> filter(fn: (r) =>  r.stringValue == "%s")`, value))
			case "<":
				sb.WriteString(fmt.Sprintf(`|> filter(fn: (r) => strings.containsStr(v: "%s", substr: r.stringValue) == true)`, value))
				sb.WriteString(fmt.Sprintf(`|> filter(fn: (r) =>  r.stringValue !="%s")`, value))
			case "<=":
				sb.WriteString(fmt.Sprintf(`|> filter(fn: (r) => strings.containsStr(v: "%s", substr: r.stringValue) == true)`, value))
			case ">":
				sb.WriteString(fmt.Sprintf(`|> filter(fn: (r) => strings.containsStr(v: r.stringValue, substr: "%s") == true)`, value))
				sb.WriteString(fmt.Sprintf(`|> filter(fn: (r) =>  r.stringValue != "%s")`, value))
			case ">=":
				sb.WriteString(fmt.Sprintf(`|> filter(fn: (r) => strings.containsStr(v: r.stringValue, substr: "%s") == true)`, value))
			}
		case "vd":
			comparator := readers.ParseValueComparator(query)
			if comparator == "=" {
				comparator = "=="
			}
			sb.WriteString(`|> filter(fn: (r) => exists r.dataValue)`)
			sb.WriteString(fmt.Sprintf(`|> filter(fn: (r) => r.dataValue%s"%s")`, comparator, value))
		}
	}

	return sb.String(), timeRange
}

func parseMessage(measurement string, valueMap map[string]interface{}) (interface{}, error) {
	switch measurement {
	case defMeasurement:
		return parseSenml(valueMap)
	default:
		return parseJSON(valueMap)
	}
}

func underscore(name string) string {
	var buff []rune
	idx := 0
	for i, c := range name {
		if unicode.IsUpper(c) {
			buff = append(buff, []rune(name[idx:i])...)
			buff = append(buff, []rune{'_', unicode.ToLower(c)}...)
			idx = i + 1
			continue
		}
	}
	buff = append(buff, []rune(name[idx:])...)
	return string(buff)
}

func parseSenml(valueMap map[string]interface{}) (interface{}, error) {
	msg := make(map[string]interface{})

	for k, v := range valueMap {
		k = underscore(k)
		if k == "_time" {
			k = "time"
			t, ok := v.(time.Time)
			if !ok {
				return nil, errResultTime
			}
			v := float64(t.UnixNano()) / 1e9
			msg[k] = v
			continue
		}
		msg[k] = v
	}
	data, err := json.Marshal(msg)
	if err != nil {
		return nil, err
	}
	senmlMsg := senml.Message{}
	if err := json.Unmarshal(data, &senmlMsg); err != nil {
		return nil, err
	}
	return senmlMsg, nil
}

func parseJSON(valueMap map[string]interface{}) (interface{}, error) {
	ret := make(map[string]interface{})
	pld := make(map[string]interface{})
	for name, field := range valueMap {
		switch name {
		case "channel", "created", "subtopic", "publisher", "protocol":
			ret[name] = field
		case "_time":
			name = "time"
			t, ok := field.(time.Time)
			if !ok {
				return nil, errResultTime
			}
			v := float64(t.UnixNano()) / 1e9
			ret[name] = v
			continue
		case "table", "_start", "_stop", "result", "_measurement":
		default:
			v := field
			if val, ok := v.(json.Number); ok {
				var err error
				v, err = val.Float64()
				if err != nil {
					return nil, err
				}
			}
			pld[name] = v
		}
	}
	ret["payload"] = jsont.ParseFlat(pld)
	return ret, nil
}
