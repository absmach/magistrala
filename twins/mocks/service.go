package mocks

import (
	"encoding/json"
	"time"

	"github.com/mainflux/mainflux/messaging"
	"github.com/mainflux/mainflux/twins"
	"github.com/mainflux/mainflux/twins/uuid"
	"github.com/mainflux/senml"
)

const (
	publisher = "twins"
)

// NewService use mock dependencies to create real twins service
func NewService(tokens map[string]string) twins.Service {
	auth := NewAuthNServiceClient(tokens)
	twinsRepo := NewTwinRepository()
	statesRepo := NewStateRepository()
	idp := NewIdentityProvider()
	subs := map[string]string{"chanID": "chanID"}
	broker := NewBroker(subs)
	return twins.New(broker, auth, twinsRepo, statesRepo, idp, "chanID", nil)
}

// CreateDefinition creates twin definition
func CreateDefinition(names []string, subtopics []string) twins.Definition {
	var def twins.Definition
	for i, v := range names {
		id, _ := uuid.New().ID()
		attr := twins.Attribute{
			Name:         v,
			Channel:      id,
			Subtopic:     subtopics[i],
			PersistState: true,
		}
		def.Attributes = append(def.Attributes, attr)
	}
	return def
}

// CreateSenML creates SenML record array
func CreateSenML(n int, bn string) []senml.Record {
	var recs []senml.Record
	for i := 0; i < n; i++ {
		rec := senml.Record{
			BaseName: bn,
			BaseTime: float64(time.Now().Unix()),
			Time:     float64(i),
			Value:    nil,
		}
		recs = append(recs, rec)
	}
	return recs
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
