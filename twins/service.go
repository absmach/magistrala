// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package twins

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"time"

	"github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/pkg/errors"
	"github.com/mainflux/mainflux/pkg/messaging"

	"github.com/mainflux/mainflux"
	"github.com/mainflux/senml"
)

const publisher = "twins"

// Service specifies an API that must be fullfiled by the domain service
// implementation, and all of its decorators (e.g. logging & metrics).
type Service interface {
	// AddTwin adds new twin related to user identified by the provided key.
	AddTwin(ctx context.Context, token string, twin Twin, def Definition) (tw Twin, err error)

	// UpdateTwin updates twin identified by the provided Twin that
	// belongs to the user identified by the provided key.
	UpdateTwin(ctx context.Context, token string, twin Twin, def Definition) (err error)

	// ViewTwin retrieves data about twin with the provided
	// ID belonging to the user identified by the provided key.
	ViewTwin(ctx context.Context, token, twinID string) (tw Twin, err error)

	// RemoveTwin removes the twin identified with the provided ID, that
	// belongs to the user identified by the provided key.
	RemoveTwin(ctx context.Context, token, twinID string) (err error)

	// ListTwins retrieves data about subset of twins that belongs to the
	// user identified by the provided key.
	ListTwins(ctx context.Context, token string, offset uint64, limit uint64, name string, metadata Metadata) (Page, error)

	// ListStates retrieves data about subset of states that belongs to the
	// twin identified by the id.
	ListStates(ctx context.Context, token string, offset uint64, limit uint64, twinID string) (StatesPage, error)

	// SaveStates persists states into database
	SaveStates(msg *messaging.Message) error
}

const (
	noop = iota
	update
	save
	millisec         = 1e6
	nanosec          = 1e9
	SubtopicWildcard = ">"
)

var crudOp = map[string]string{
	"createSucc": "create.success",
	"createFail": "create.failure",
	"updateSucc": "update.success",
	"updateFail": "update.failure",
	"getSucc":    "get.success",
	"getFail":    "get.failure",
	"removeSucc": "remove.success",
	"removeFail": "remove.failure",
	"stateSucc":  "save.success",
	"stateFail":  "save.failure",
}

type twinsService struct {
	publisher  messaging.Publisher
	auth       mainflux.AuthServiceClient
	twins      TwinRepository
	states     StateRepository
	idProvider mainflux.IDProvider
	channelID  string
	twinCache  TwinCache
	logger     logger.Logger
}

var _ Service = (*twinsService)(nil)

// New instantiates the twins service implementation.
func New(publisher messaging.Publisher, auth mainflux.AuthServiceClient, twins TwinRepository, tcache TwinCache, sr StateRepository, idp mainflux.IDProvider, chann string, logger logger.Logger) Service {
	return &twinsService{
		publisher:  publisher,
		auth:       auth,
		twins:      twins,
		twinCache:  tcache,
		states:     sr,
		idProvider: idp,
		channelID:  chann,
		logger:     logger,
	}
}

func (ts *twinsService) AddTwin(ctx context.Context, token string, twin Twin, def Definition) (tw Twin, err error) {
	var id string
	var b []byte
	defer ts.publish(&id, &err, crudOp["createSucc"], crudOp["createFail"], &b)

	res, err := ts.auth.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return Twin{}, err
	}

	twin.ID, err = ts.idProvider.ID()
	if err != nil {
		return Twin{}, err
	}

	twin.Owner = res.GetEmail()

	t := time.Now()
	twin.Created = t
	twin.Updated = t

	if def.Attributes == nil {
		def.Attributes = []Attribute{}
	}
	if def.Delta == 0 {
		def.Delta = millisec
	}

	def.Created = time.Now()
	def.ID = 0
	twin.Definitions = append(twin.Definitions, def)

	twin.Revision = 0
	if _, err = ts.twins.Save(ctx, twin); err != nil {
		return Twin{}, err
	}

	id = twin.ID
	b, err = json.Marshal(twin)

	return twin, ts.twinCache.Save(ctx, twin)
}

func (ts *twinsService) UpdateTwin(ctx context.Context, token string, twin Twin, def Definition) (err error) {
	var b []byte
	var id string
	defer ts.publish(&id, &err, crudOp["updateSucc"], crudOp["updateFail"], &b)

	_, err = ts.auth.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return errors.ErrAuthentication
	}

	tw, err := ts.twins.RetrieveByID(ctx, twin.ID)
	if err != nil {
		return err
	}

	revision := false

	if twin.Name != "" {
		revision = true
		tw.Name = twin.Name
	}

	if len(def.Attributes) > 0 {
		revision = true
		def.Created = time.Now()
		def.ID = tw.Definitions[len(tw.Definitions)-1].ID + 1
		tw.Definitions = append(tw.Definitions, def)
	}

	if len(twin.Metadata) > 0 {
		revision = true
		tw.Metadata = twin.Metadata
	}

	if !revision {
		return errors.ErrMalformedEntity
	}

	tw.Updated = time.Now()
	tw.Revision++

	if err := ts.twins.Update(ctx, tw); err != nil {
		return err
	}

	id = twin.ID
	b, err = json.Marshal(tw)

	return ts.twinCache.Update(ctx, twin)
}

func (ts *twinsService) ViewTwin(ctx context.Context, token, twinID string) (tw Twin, err error) {
	var b []byte
	defer ts.publish(&twinID, &err, crudOp["getSucc"], crudOp["getFail"], &b)

	_, err = ts.auth.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return Twin{}, err
	}

	twin, err := ts.twins.RetrieveByID(ctx, twinID)
	if err != nil {
		return Twin{}, err
	}

	b, err = json.Marshal(twin)

	return twin, nil
}

func (ts *twinsService) RemoveTwin(ctx context.Context, token, twinID string) (err error) {
	var b []byte
	defer ts.publish(&twinID, &err, crudOp["removeSucc"], crudOp["removeFail"], &b)

	_, err = ts.auth.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return errors.ErrAuthentication
	}

	if err := ts.twins.Remove(ctx, twinID); err != nil {
		return err
	}

	return ts.twinCache.Remove(ctx, twinID)
}

func (ts *twinsService) ListTwins(ctx context.Context, token string, offset uint64, limit uint64, name string, metadata Metadata) (Page, error) {
	res, err := ts.auth.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return Page{}, errors.ErrAuthentication
	}

	return ts.twins.RetrieveAll(ctx, res.GetEmail(), offset, limit, name, metadata)
}

func (ts *twinsService) ListStates(ctx context.Context, token string, offset uint64, limit uint64, twinID string) (StatesPage, error) {
	_, err := ts.auth.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return StatesPage{}, errors.ErrAuthentication
	}

	return ts.states.RetrieveAll(ctx, offset, limit, twinID)
}

func (ts *twinsService) SaveStates(msg *messaging.Message) error {
	var ids []string

	ctx := context.TODO()
	channel, subtopic := msg.Channel, msg.Subtopic
	ids, err := ts.twinCache.IDs(ctx, channel, subtopic)
	if err != nil {
		return err
	}
	if len(ids) < 1 {
		ids, err = ts.twins.RetrieveByAttribute(ctx, channel, subtopic)
		if err != nil {
			return err
		}
		if len(ids) < 1 {
			return nil
		}
		if err := ts.twinCache.SaveIDs(ctx, channel, subtopic, ids); err != nil {
			return err
		}
	}

	for _, id := range ids {
		if err := ts.saveState(msg, id); err != nil {
			return err
		}
	}

	return nil
}

func (ts *twinsService) saveState(msg *messaging.Message, twinID string) error {
	var b []byte
	var err error
	defer ts.publish(&twinID, &err, crudOp["stateSucc"], crudOp["stateFail"], &b)

	ctx := context.TODO()
	tw, err := ts.twins.RetrieveByID(ctx, twinID)
	if err != nil {
		return fmt.Errorf("Retrieving twin for %s failed: %s", msg.Publisher, err)
	}

	var recs []senml.Record
	if err := json.Unmarshal(msg.Payload, &recs); err != nil {
		return fmt.Errorf("Unmarshal payload for %s failed: %s", msg.Publisher, err)
	}

	st, err := ts.states.RetrieveLast(ctx, tw.ID)
	if err != nil {
		return fmt.Errorf("Retrieve last state for %s failed: %s", msg.Publisher, err)
	}

	for _, rec := range recs {
		action := ts.prepareState(&st, &tw, rec, msg)
		switch action {
		case noop:
			return nil
		case update:
			if err := ts.states.Update(ctx, st); err != nil {
				return fmt.Errorf("Update state for %s failed: %s", msg.Publisher, err)
			}
		case save:
			if err := ts.states.Save(ctx, st); err != nil {
				return fmt.Errorf("Save state for %s failed: %s", msg.Publisher, err)
			}
		}
	}

	twinID = msg.Publisher
	b = msg.Payload

	return nil
}

func (ts *twinsService) prepareState(st *State, tw *Twin, rec senml.Record, msg *messaging.Message) int {
	def := tw.Definitions[len(tw.Definitions)-1]
	st.TwinID = tw.ID
	st.Definition = def.ID

	if st.Payload == nil {
		st.Payload = make(map[string]interface{})
		st.ID = -1 // state is incremented on save -> zero-based index
	} else {
		for k := range st.Payload {
			idx := findAttribute(k, def.Attributes)
			if idx < 0 || !def.Attributes[idx].PersistState {
				delete(st.Payload, k)
			}
		}
	}

	recSec := rec.BaseTime + rec.Time
	recNano := recSec * nanosec
	sec, dec := math.Modf(recSec)
	recTime := time.Unix(int64(sec), int64(dec*nanosec))

	action := noop
	for _, attr := range def.Attributes {
		if !attr.PersistState {
			continue
		}
		if attr.Channel == msg.Channel && (attr.Subtopic == SubtopicWildcard || attr.Subtopic == msg.Subtopic) {
			action = update
			delta := math.Abs(float64(st.Created.UnixNano()) - recNano)
			if recNano == 0 || delta > float64(def.Delta) {
				action = save
				st.ID++
				st.Created = time.Now()
				if recNano != 0 {
					st.Created = recTime
				}
			}
			val := findValue(rec)
			st.Payload[attr.Name] = val

			break
		}
	}

	return action
}

func findValue(rec senml.Record) interface{} {
	if rec.Value != nil {
		return rec.Value
	}
	if rec.StringValue != nil {
		return rec.StringValue
	}
	if rec.DataValue != nil {
		return rec.DataValue
	}
	if rec.BoolValue != nil {
		return rec.BoolValue
	}
	if rec.Sum != nil {
		return rec.Sum
	}
	return nil
}

func findAttribute(name string, attrs []Attribute) (idx int) {
	for idx, attr := range attrs {
		if attr.Name == name {
			return idx
		}
	}
	return -1
}

func (ts *twinsService) publish(twinID *string, err *error, succOp, failOp string, payload *[]byte) {
	if ts.channelID == "" {
		return
	}

	op := succOp
	if *err != nil {
		op = failOp
		esb := []byte((*err).Error())
		payload = &esb
	}

	pl := *payload
	if pl == nil {
		pl = []byte(fmt.Sprintf("{\"deleted\":\"%s\"}", *twinID))
	}

	msg := messaging.Message{
		Channel:   ts.channelID,
		Subtopic:  op,
		Payload:   pl,
		Publisher: publisher,
		Created:   time.Now().UnixNano(),
	}

	if err := ts.publisher.Publish(msg.Channel, &msg); err != nil {
		ts.logger.Warn(fmt.Sprintf("Failed to publish notification on Message Broker: %s", err))
	}
}
