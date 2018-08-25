package influxdb

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"

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

type fields map[string]interface{}
type tags map[string]string

// New returns new InfluxDB reader.
func New(client influxdata.Client, database string) (readers.MessageRepository, error) {
	return &influxRepository{database, client}, nil
}

func (repo *influxRepository) ReadAll(chanID, offset, limit uint64) []mainflux.Message {
	if limit > maxLimit {
		limit = maxLimit
	}
	cmd := fmt.Sprintf(`SELECT * from messages WHERE Channel='%d' LIMIT %d OFFSET %d`, chanID, limit, offset)
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
		ret = append(ret, genMessage(result.Columns, v))
	}

	return ret
}

// GenMessage and parseFloat are util methods. Since InfluxDB client returns
// results in some proprietary from, this obscure message conversion is needed
// to return actual []mainflux.Message from the query result.
func parseFloat(value interface{}) float64 {
	switch value.(type) {
	case string:
		ret, _ := strconv.ParseFloat(value.(string), 64)
		return ret
	case json.Number:
		ret, _ := strconv.ParseFloat((value.(json.Number)).String(), 64)
		return ret
	}
	return 0
}

func genMessage(names []string, fields []interface{}) mainflux.Message {
	m := mainflux.Message{}
	v := reflect.ValueOf(&m).Elem()
	for i, name := range names {
		msgField := v.FieldByName(name)
		if !msgField.IsValid() {
			continue
		}
		f := msgField.Interface()
		switch f.(type) {
		case string:
			if s, ok := fields[i].(string); ok {
				msgField.SetString(s)
			}
		case uint64:
			u, _ := strconv.ParseUint(fields[i].(string), 10, 64)
			msgField.SetUint(u)
		case float64:
			msgField.SetFloat(parseFloat(fields[i]))
		case bool:
			if b, ok := fields[i].(bool); ok {
				msgField.SetBool(b)
			}
		}
	}

	return m
}
