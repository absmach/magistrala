package influxdb

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/mainflux/mainflux/pkg/errors"
	"github.com/mainflux/mainflux/readers"

	influxdata "github.com/influxdata/influxdb/client/v2"
	jsont "github.com/mainflux/mainflux/pkg/transformers/json"
	"github.com/mainflux/mainflux/pkg/transformers/senml"
)

const (
	countCol = "count_protocol"
	// Measurement for SenML messages
	defMeasurement = "messages"
)

var _ readers.MessageRepository = (*influxRepository)(nil)

type influxRepository struct {
	database string
	client   influxdata.Client
}

// New returns new InfluxDB reader.
func New(client influxdata.Client, database string) readers.MessageRepository {
	return &influxRepository{
		database,
		client,
	}
}

func (repo *influxRepository) ReadAll(chanID string, rpm readers.PageMetadata) (readers.MessagesPage, error) {
	format := defMeasurement
	if rpm.Format != "" {
		format = rpm.Format
	}

	condition := fmtCondition(chanID, rpm)

	cmd := fmt.Sprintf(`SELECT * FROM %s WHERE %s ORDER BY time DESC LIMIT %d OFFSET %d`, format, condition, rpm.Limit, rpm.Offset)
	q := influxdata.Query{
		Command:  cmd,
		Database: repo.database,
	}

	resp, err := repo.client.Query(q)
	if err != nil {
		return readers.MessagesPage{}, errors.Wrap(readers.ErrReadMessages, err)
	}
	if resp.Error() != nil {
		return readers.MessagesPage{}, errors.Wrap(readers.ErrReadMessages, resp.Error())
	}

	if len(resp.Results) == 0 || len(resp.Results[0].Series) == 0 {
		return readers.MessagesPage{}, nil
	}

	var messages []readers.Message
	result := resp.Results[0].Series[0]
	for _, v := range result.Values {
		msg, err := parseMessage(format, result.Columns, v)
		if err != nil {
			return readers.MessagesPage{}, err
		}
		messages = append(messages, msg)
	}

	total, err := repo.count(format, condition)
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

func (repo *influxRepository) count(measurement, condition string) (uint64, error) {
	cmd := fmt.Sprintf(`SELECT COUNT(*) FROM %s WHERE %s`, measurement, condition)
	q := influxdata.Query{
		Command:  cmd,
		Database: repo.database,
	}

	resp, err := repo.client.Query(q)
	if err != nil {
		return 0, err
	}
	if resp.Error() != nil {
		return 0, resp.Error()
	}

	if len(resp.Results) == 0 ||
		len(resp.Results[0].Series) == 0 ||
		len(resp.Results[0].Series[0].Values) == 0 {
		return 0, nil
	}

	countIndex := 0
	for i, col := range resp.Results[0].Series[0].Columns {
		if col == countCol {
			countIndex = i
			break
		}
	}

	result := resp.Results[0].Series[0].Values[0]
	if len(result) < countIndex+1 {
		return 0, nil
	}

	count, ok := result[countIndex].(json.Number)
	if !ok {
		return 0, nil
	}
	return strconv.ParseUint(count.String(), 10, 64)
}

func fmtCondition(chanID string, rpm readers.PageMetadata) string {
	condition := fmt.Sprintf(`channel='%s'`, chanID)

	var query map[string]interface{}
	meta, err := json.Marshal(rpm)
	if err != nil {
		return condition
	}

	if err := json.Unmarshal(meta, &query); err != nil {
		return condition
	}

	for name, value := range query {
		switch name {
		case
			"channel",
			"subtopic",
			"publisher",
			"name",
			"protocol":
			condition = fmt.Sprintf(`%s AND "%s"='%s'`, condition, name, value)
		case "v":
			comparator := readers.ParseValueComparator(query)
			condition = fmt.Sprintf(`%s AND value %s %f`, condition, comparator, value)
		case "vb":
			condition = fmt.Sprintf(`%s AND boolValue = %t`, condition, value)
		case "vs":
			condition = fmt.Sprintf(`%s AND stringValue = '%s'`, condition, value)
		case "vd":
			condition = fmt.Sprintf(`%s AND dataValue = '%s'`, condition, value)
		case "from":
			iVal := int64(value.(float64) * 1e9)
			condition = fmt.Sprintf(`%s AND time >= %d`, condition, iVal)
		case "to":
			iVal := int64(value.(float64) * 1e9)
			condition = fmt.Sprintf(`%s AND time < %d`, condition, iVal)
		}
	}
	return condition
}

// ParseMessage and parseValues are util methods. Since InfluxDB client returns
// results in form of rows and columns, this obscure message conversion is needed
// to return actual []broker.Message from the query result.
func parseValues(value interface{}, name string, msg *senml.Message) {
	if name == "sum" && value != nil {
		if valSum, ok := value.(json.Number); ok {
			sum, err := valSum.Float64()
			if err != nil {
				return
			}

			msg.Sum = &sum
		}
		return
	}

	if strings.HasSuffix(strings.ToLower(name), "value") {
		switch value.(type) {
		case bool:
			v := value.(bool)
			msg.BoolValue = &v
		case json.Number:
			num, err := value.(json.Number).Float64()
			if err != nil {
				return
			}
			msg.Value = &num
		case string:
			if strings.HasPrefix(name, "string") {
				v := value.(string)
				msg.StringValue = &v
				return
			}

			if strings.HasPrefix(name, "data") {
				v := value.(string)
				msg.DataValue = &v
			}
		}
	}
}

func parseMessage(measurement string, names []string, fields []interface{}) (interface{}, error) {
	switch measurement {
	case defMeasurement:
		return parseSenml(names, fields), nil
	default:
		return parseJSON(names, fields)
	}
}

func parseSenml(names []string, fields []interface{}) interface{} {
	m := senml.Message{}
	v := reflect.ValueOf(&m).Elem()
	for i, name := range names {
		parseValues(fields[i], name, &m)
		msgField := v.FieldByName(strings.Title(name))
		if !msgField.IsValid() {
			continue
		}

		f := msgField.Interface()
		switch f.(type) {
		case string:
			if s, ok := fields[i].(string); ok {
				msgField.SetString(s)
			}
		case float64:
			fs, ok := fields[i].(string)
			if !ok {
				continue
			}

			if name == "time" {
				t, err := time.Parse(time.RFC3339Nano, fs)
				if err != nil {
					continue
				}

				v := float64(t.UnixNano()) / 1e9
				msgField.SetFloat(v)
				continue
			}

			v, err := strconv.ParseFloat(fs, 64)
			if err != nil {
				continue
			}
			msgField.SetFloat(v)
		}
	}

	return m
}

func parseJSON(names []string, fields []interface{}) (interface{}, error) {
	ret := make(map[string]interface{})
	pld := make(map[string]interface{})
	for i, n := range names {
		switch n {
		case "channel", "created", "subtopic", "publisher", "protocol", "time":
			ret[n] = fields[i]
		default:
			v := fields[i]
			if val, ok := v.(json.Number); ok {
				var err error
				v, err = val.Float64()
				if err != nil {
					return nil, err
				}
			}
			pld[n] = v
		}
	}
	ret["payload"] = jsont.ParseFlat(pld)
	return ret, nil
}
