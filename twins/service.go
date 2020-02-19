// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package twins

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/mainflux/mainflux"
	nats "github.com/mainflux/mainflux/twins/nats/publisher"
	"github.com/mainflux/senml"
)

var (
	// ErrMalformedEntity indicates malformed entity specification (e.g.
	// invalid username or password).
	ErrMalformedEntity = errors.New("malformed entity specification")

	// ErrUnauthorizedAccess indicates missing or invalid credentials provided
	// when accessing a protected resource.
	ErrUnauthorizedAccess = errors.New("missing or invalid credentials provided")

	// ErrNotFound indicates a non-existent entity request.
	ErrNotFound = errors.New("non-existent entity")

	// ErrConflict indicates that entity already exists.
	ErrConflict = errors.New("entity already exists")
)

// Service specifies an API that must be fullfiled by the domain service
// implementation, and all of its decorators (e.g. logging & metrics).
type Service interface {
	// AddTwin adds new twin related to user identified by the provided key.
	AddTwin(context.Context, string, Twin, Definition) (Twin, error)

	// UpdateTwin updates twin identified by the provided Twin that
	// belongs to the user identified by the provided key.
	UpdateTwin(context.Context, string, Twin, Definition) error

	// ViewTwin retrieves data about twin with the provided
	// ID belonging to the user identified by the provided key.
	ViewTwin(context.Context, string, string) (Twin, error)

	// ListTwins retrieves data about subset of twins that belongs to the
	// user identified by the provided key.
	ListTwins(context.Context, string, uint64, uint64, string, Metadata) (TwinsPage, error)

	// ListStates retrieves data about subset of states that belongs to the
	// twin identified by the id.
	ListStates(context.Context, string, uint64, uint64, string) (StatesPage, error)

	// SaveStates persists states into database
	SaveStates(*mainflux.Message) error

	// ListTwinsByThing retrieves data about subset of twins that represent
	// specified thing belong to the user identified by
	// the provided key.
	ViewTwinByThing(context.Context, string, string) (Twin, error)

	// RemoveTwin removes the twin identified with the provided ID, that
	// belongs to the user identified by the provided key.
	RemoveTwin(context.Context, string, string) error
}

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
	auth   mainflux.AuthNServiceClient
	twins  TwinRepository
	states StateRepository
	idp    IdentityProvider
	nats   *nats.Publisher
}

var _ Service = (*twinsService)(nil)

// New instantiates the twins service implementation.
func New(auth mainflux.AuthNServiceClient, twins TwinRepository, sr StateRepository, idp IdentityProvider, n *nats.Publisher) Service {
	return &twinsService{
		auth:   auth,
		twins:  twins,
		states: sr,
		idp:    idp,
		nats:   n,
	}
}

func (ts *twinsService) AddTwin(ctx context.Context, token string, twin Twin, def Definition) (tw Twin, err error) {
	var id string
	var b []byte
	defer ts.nats.Publish(&id, &err, crudOp["createSucc"], crudOp["createFail"], &b)

	res, err := ts.auth.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return Twin{}, ErrUnauthorizedAccess
	}

	twin.ID, err = ts.idp.ID()
	if err != nil {
		return Twin{}, err
	}

	twin.Owner = res.GetValue()

	twin.Created = time.Now()
	twin.Updated = time.Now()

	if len(def.Attributes) == 0 {
		def = Definition{}
		def.Attributes = []Attribute{}
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

	return twin, nil
}

func (ts *twinsService) UpdateTwin(ctx context.Context, token string, twin Twin, def Definition) (err error) {
	var b []byte
	var id string
	defer ts.nats.Publish(&id, &err, crudOp["updateSucc"], crudOp["updateFail"], &b)

	_, err = ts.auth.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return ErrUnauthorizedAccess
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

	if twin.ThingID != "" {
		revision = true
		tw.ThingID = twin.ThingID
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
		return ErrMalformedEntity
	}

	tw.Updated = time.Now()
	tw.Revision++

	if err := ts.twins.Update(ctx, tw); err != nil {
		return err
	}

	id = twin.ID
	b, err = json.Marshal(tw)

	return nil
}

func (ts *twinsService) ViewTwin(ctx context.Context, token, id string) (tw Twin, err error) {
	var b []byte
	defer ts.nats.Publish(&id, &err, crudOp["getSucc"], crudOp["getFail"], &b)

	_, err = ts.auth.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return Twin{}, ErrUnauthorizedAccess
	}

	twin, err := ts.twins.RetrieveByID(ctx, id)
	if err != nil {
		return Twin{}, err
	}

	b, err = json.Marshal(twin)

	return twin, nil
}

func (ts *twinsService) ViewTwinByThing(ctx context.Context, token, thingid string) (Twin, error) {
	_, err := ts.auth.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return Twin{}, ErrUnauthorizedAccess
	}

	return ts.twins.RetrieveByThing(ctx, thingid)
}

func (ts *twinsService) RemoveTwin(ctx context.Context, token, id string) (err error) {
	var b []byte
	defer ts.nats.Publish(&id, &err, crudOp["removeSucc"], crudOp["removeFail"], &b)

	_, err = ts.auth.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return ErrUnauthorizedAccess
	}

	if err := ts.twins.Remove(ctx, id); err != nil {
		return err
	}

	return nil
}

func (ts *twinsService) ListTwins(ctx context.Context, token string, offset uint64, limit uint64, name string, metadata Metadata) (TwinsPage, error) {
	res, err := ts.auth.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return TwinsPage{}, ErrUnauthorizedAccess
	}

	return ts.twins.RetrieveAll(ctx, res.GetValue(), offset, limit, name, metadata)
}

func (ts *twinsService) ListStates(ctx context.Context, token string, offset uint64, limit uint64, id string) (StatesPage, error) {
	_, err := ts.auth.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return StatesPage{}, ErrUnauthorizedAccess
	}

	return ts.states.RetrieveAll(ctx, offset, limit, id)
}

func (ts *twinsService) SaveStates(msg *mainflux.Message) error {
	ids, err := ts.twins.RetrieveByAttribute(context.TODO(), msg.Channel, msg.Subtopic)
	if err != nil {
		return err
	}

	for _, id := range ids {
		if err := ts.saveState(msg, id); err != nil {
			return err
		}
	}

	return nil
}

func (ts *twinsService) saveState(msg *mainflux.Message, id string) error {
	var b []byte
	var err error
	defer ts.nats.Publish(&id, &err, crudOp["stateSucc"], crudOp["stateFail"], &b)

	tw, err := ts.twins.RetrieveByID(context.TODO(), id)
	if err != nil {
		return fmt.Errorf("Retrieving twin for %s failed: %s", msg.Publisher, err)
	}

	var recs []senml.Record
	if err := json.Unmarshal(msg.Payload, &recs); err != nil {
		return fmt.Errorf("Unmarshal payload for %s failed: %s", msg.Publisher, err)
	}

	st, err := ts.states.RetrieveLast(context.TODO(), tw.ID)
	if err != nil {
		return fmt.Errorf("Retrieve last state for %s failed: %s", msg.Publisher, err)
	}

	if save := prepareState(&st, &tw, recs, msg); !save {
		return nil
	}

	if err := ts.states.Save(context.TODO(), st); err != nil {
		return fmt.Errorf("Updating state for %s failed: %s", msg.Publisher, err)
	}

	id = msg.Publisher
	b = msg.Payload

	return nil
}

func prepareState(st *State, tw *Twin, recs []senml.Record, msg *mainflux.Message) bool {
	def := tw.Definitions[len(tw.Definitions)-1]
	st.TwinID = tw.ID
	st.ID++
	st.Created = time.Now()
	st.Definition = def.ID

	if st.Payload == nil {
		st.Payload = make(map[string]interface{})
	} else {
		for k := range st.Payload {
			idx := findAttribute(k, def.Attributes)
			if idx < 0 || !def.Attributes[idx].PersistState {
				delete(st.Payload, k)
			}
		}
	}

	save := false
	for _, attr := range def.Attributes {
		if !attr.PersistState {
			continue
		}
		if attr.Channel == msg.Channel && attr.Subtopic == msg.Subtopic {
			val := findValue(recs[0])
			st.Payload[attr.Name] = val
			save = true
			break
		}
	}

	return save
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
