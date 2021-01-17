package mocks

import (
	"encoding/json"
	"strconv"
	"time"

	"github.com/mainflux/mainflux/pkg/messaging"
	"github.com/mainflux/mainflux/pkg/uuid"
	"github.com/mainflux/mainflux/twins"
	"github.com/mainflux/senml"
)

const publisher = "twins"

var id = 0

// NewService use mock dependencies to create real twins service
func NewService(tokens map[string]string) twins.Service {
	auth := NewAuthServiceClient(tokens)
	twinsRepo := NewTwinRepository()
	twinCache := NewTwinCache()
	statesRepo := NewStateRepository()
	idProvider := uuid.NewMock()
	subs := map[string]string{"chanID": "chanID"}
	broker := NewBroker(subs)

	return twins.New(broker, auth, twinsRepo, twinCache, statesRepo, idProvider, "chanID", nil)
}

// CreateDefinition creates twin definition
func CreateDefinition(channels []string, subtopics []string) twins.Definition {
	var def twins.Definition
	for i := range channels {
		attr := twins.Attribute{
			Channel:      channels[i],
			Subtopic:     subtopics[i],
			PersistState: true,
		}
		def.Attributes = append(def.Attributes, attr)
	}
	return def
}

// CreateTwin creates twin
func CreateTwin(channels []string, subtopics []string) twins.Twin {
	id++
	return twins.Twin{
		ID:          strconv.Itoa(id),
		Definitions: []twins.Definition{CreateDefinition(channels, subtopics)},
	}
}

// CreateSenML creates SenML record array
func CreateSenML(n int, recs []senml.Record) {
	for i, rec := range recs {
		rec.BaseTime = float64(time.Now().Unix())
		rec.Time = float64(i)
		rec.Value = nil
	}
}

// CreateMessage creates Mainflux message using SenML record array
func CreateMessage(attr twins.Attribute, recs []senml.Record) (*messaging.Message, error) {
	mRecs, err := json.Marshal(recs)
	if err != nil {
		return nil, err
	}
	return &messaging.Message{
		Channel:   attr.Channel,
		Subtopic:  attr.Subtopic,
		Payload:   mRecs,
		Publisher: publisher,
	}, nil
}
