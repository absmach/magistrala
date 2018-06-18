package influxdb

import (
	"strconv"
	"time"

	"github.com/mainflux/mainflux/writers"

	influxdata "github.com/influxdata/influxdb/client/v2"
	"github.com/mainflux/mainflux"
)

const pointName = "messages"

var _ writers.MessageRepository = (*influxRepo)(nil)

type influxRepo struct {
	database string
	client   influxdata.Client
}

type fields map[string]interface{}
type tags map[string]string

// New returns new InfluxDB writer.
func New(client influxdata.Client, database string) (writers.MessageRepository, error) {
	return &influxRepo{database, client}, nil
}

func (repo *influxRepo) Save(msg mainflux.Message) error {
	bp, err := influxdata.NewBatchPoints(influxdata.BatchPointsConfig{
		Database: repo.database,
	})
	if err != nil {
		return err
	}

	tags, fields := repo.tagsOf(&msg), repo.fieldsOf(&msg)
	pt, err := influxdata.NewPoint(pointName, tags, fields, time.Now())
	if err != nil {
		return err
	}

	bp.AddPoint(pt)
	return repo.client.Write(bp)
}

func (repo *influxRepo) tagsOf(msg *mainflux.Message) tags {
	time := strconv.FormatFloat(msg.Time, 'f', -1, 64)
	update := strconv.FormatFloat(msg.UpdateTime, 'f', -1, 64)
	channel := strconv.FormatUint(msg.Channel, 10)
	publisher := strconv.FormatUint(msg.Publisher, 10)
	return tags{
		"Channel":    channel,
		"Publisher":  publisher,
		"Protocol":   msg.Protocol,
		"Name":       msg.Name,
		"Unit":       msg.Unit,
		"Link":       msg.Link,
		"Time":       time,
		"UpdateTime": update,
	}
}

func (repo *influxRepo) fieldsOf(msg *mainflux.Message) fields {
	return fields{
		"Value":       msg.Value,
		"ValueSum":    msg.ValueSum,
		"BoolValue":   msg.BoolValue,
		"StringValue": msg.StringValue,
		"DataValue":   msg.DataValue,
	}
}
