// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package mocks

import (
	"encoding/json"
	"strconv"
	"time"

	authmocks "github.com/absmach/magistrala/auth/mocks"
	"github.com/absmach/magistrala/pkg/messaging"
	"github.com/absmach/magistrala/pkg/uuid"
	"github.com/absmach/magistrala/twins"
	"github.com/absmach/senml"
)

const publisher = "twins"

var id = 0

// NewService use mock dependencies to create real twins service.
func NewService() (twins.Service, *authmocks.Service) {
	auth := new(authmocks.Service)
	twinsRepo := NewTwinRepository()
	twinCache := NewTwinCache()
	statesRepo := NewStateRepository()
	idProvider := uuid.NewMock()
	subs := map[string]string{"chanID": "chanID"}
	broker := NewBroker(subs)

	return twins.New(broker, auth, twinsRepo, twinCache, statesRepo, idProvider, "chanID", nil), auth
}

// CreateDefinition creates twin definition.
func CreateDefinition(channels, subtopics []string) twins.Definition {
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

// CreateTwin creates twin.
func CreateTwin(channels, subtopics []string) twins.Twin {
	id++
	return twins.Twin{
		ID:          strconv.Itoa(id),
		Definitions: []twins.Definition{CreateDefinition(channels, subtopics)},
	}
}

// CreateSenML creates SenML record array.
func CreateSenML(recs []senml.Record) {
	for i, rec := range recs {
		rec.BaseTime = float64(time.Now().Unix())
		rec.Time = float64(i)
		rec.Value = nil
	}
}

// CreateMessage creates Magistrala message using SenML record array.
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
