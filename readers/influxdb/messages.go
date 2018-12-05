package influxdb

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/mainflux/mainflux/readers"

	influxdata "github.com/influxdata/influxdb/client/v2"
	"github.com/mainflux/mainflux"
)

const maxLimit = 100

var _ readers.MessageRepository = (*influxRepository)(nil)

type influxRepository struct {
	database string
	client   influxdata.Client
}

// New returns new InfluxDB reader.
func New(client influxdata.Client, database string) (readers.MessageRepository, error) {
	return &influxRepository{database, client}, nil
}

func (repo *influxRepository) ReadAll(chanID string, offset, limit uint64) []mainflux.Message {
	if limit > maxLimit {
		limit = maxLimit
	}

	cmd := fmt.Sprintf(`SELECT * from messages WHERE channel='%s' LIMIT %d OFFSET %d`, chanID, limit, offset)
	q := influxdata.Query{
		Command:  cmd,
		Database: repo.database,
	}

	ret := []mainflux.Message{}

	resp, err := repo.client.Query(q)
	if err != nil || resp.Error() != nil {
		return ret
	}

	if len(resp.Results) < 1 || len(resp.Results[0].Series) < 1 {
		return ret
	}

	result := resp.Results[0].Series[0]
	for _, v := range result.Values {
		ret = append(ret, parseMessage(result.Columns, v))
	}

	return ret
}

// ParseMessage and parseValues are util methods. Since InfluxDB client returns
// results in form of rows and columns, this obscure message conversion is needed
// to return actual []mainflux.Message from the query result.
func parseValues(value interface{}, name string, msg *mainflux.Message) {
	if name == "valueSum" && value != nil {
		if sum, ok := value.(json.Number); ok {
			valSum, err := sum.Float64()
			if err != nil {
				return
			}

			msg.ValueSum = &mainflux.SumValue{Value: valSum}
		}
		return
	}

	if strings.HasSuffix(strings.ToLower(name), "value") {
		switch value.(type) {
		case bool:
			msg.Value = &mainflux.Message_BoolValue{BoolValue: value.(bool)}
		case json.Number:
			num, err := value.(json.Number).Float64()
			if err != nil {
				return
			}

			msg.Value = &mainflux.Message_FloatValue{FloatValue: num}
		case string:
			if strings.HasPrefix(name, "string") {
				msg.Value = &mainflux.Message_StringValue{StringValue: value.(string)}
				return
			}

			if strings.HasPrefix(name, "data") {
				msg.Value = &mainflux.Message_DataValue{DataValue: value.(string)}
			}
		}
	}
}

func parseMessage(names []string, fields []interface{}) mainflux.Message {
	m := mainflux.Message{}
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
			if name == "time" {
				t, err := time.Parse(time.RFC3339, fields[i].(string))
				if err != nil {
					continue
				}

				v := float64(t.Unix())
				msgField.SetFloat(v)
				continue
			}

			val, _ := strconv.ParseFloat(fields[i].(string), 64)
			msgField.SetFloat(val)
		}
	}

	return m
}
