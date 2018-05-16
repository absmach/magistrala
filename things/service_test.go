package things_test

import (
	"fmt"
	"testing"

	"github.com/mainflux/mainflux/things"
	"github.com/mainflux/mainflux/things/mocks"
	"github.com/stretchr/testify/assert"
)

const (
	wrong = "wrong-value"
	email = "user@example.com"
	token = "token"
)

var (
	thing   = things.Thing{Type: "app", Name: "test"}
	channel = things.Channel{Name: "test", Things: []things.Thing{}}
)

func newService(tokens map[string]string) things.Service {
	users := mocks.NewUsersService(tokens)
	thingsRepo := mocks.NewThingRepository()
	channelsRepo := mocks.NewChannelRepository(thingsRepo)
	idp := mocks.NewIdentityProvider()

	return things.New(users, thingsRepo, channelsRepo, idp)
}

func TestAddThing(t *testing.T) {
	svc := newService(map[string]string{token: email})

	cases := map[string]struct {
		thing things.Thing
		key   string
		err   error
	}{
		"add new app":                      {things.Thing{Type: "app", Name: "a"}, token, nil},
		"add new device":                   {things.Thing{Type: "device", Name: "b"}, token, nil},
		"add thing with wrong credentials": {things.Thing{Type: "app", Name: "d"}, wrong, things.ErrUnauthorizedAccess},
	}

	for desc, tc := range cases {
		_, err := svc.AddThing(tc.key, tc.thing)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestUpdateThing(t *testing.T) {
	svc := newService(map[string]string{token: email})
	saved, _ := svc.AddThing(token, thing)

	cases := map[string]struct {
		thing things.Thing
		key   string
		err   error
	}{
		"update existing thing":               {saved, token, nil},
		"update thing with wrong credentials": {saved, wrong, things.ErrUnauthorizedAccess},
		"update non-existing thing":           {things.Thing{ID: "2", Type: "app", Key: "x"}, token, things.ErrNotFound},
	}

	for desc, tc := range cases {
		err := svc.UpdateThing(tc.key, tc.thing)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestViewThing(t *testing.T) {
	svc := newService(map[string]string{token: email})
	saved, _ := svc.AddThing(token, thing)

	cases := map[string]struct {
		id  string
		key string
		err error
	}{
		"view existing thing":               {saved.ID, token, nil},
		"view thing with wrong credentials": {saved.ID, wrong, things.ErrUnauthorizedAccess},
		"view non-existing thing":           {wrong, token, things.ErrNotFound},
	}

	for desc, tc := range cases {
		_, err := svc.ViewThing(tc.key, tc.id)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestListThings(t *testing.T) {
	svc := newService(map[string]string{token: email})

	n := 10
	for i := 0; i < n; i++ {
		svc.AddThing(token, thing)
	}

	cases := map[string]struct {
		key    string
		offset int
		limit  int
		size   int
		err    error
	}{
		"list all things":             {token, 0, n, n, nil},
		"list subset":                 {token, 1, 3, 3, nil},
		"list half":                   {token, n / 2, n, n / 2, nil},
		"list last thing":             {token, n - 1, n, 1, nil},
		"list empty set":              {token, n + 1, n, 0, nil},
		"list with negative offset":   {token, -1, n, 0, nil},
		"list with negative limit":    {token, 1, -n, 0, nil},
		"list with zero limit":        {token, 1, 0, 0, nil},
		"list with wrong credentials": {wrong, 0, 0, 0, things.ErrUnauthorizedAccess},
	}

	for desc, tc := range cases {
		ts, err := svc.ListThings(tc.key, tc.offset, tc.limit)
		size := len(ts)
		assert.Equal(t, tc.size, size, fmt.Sprintf("%s: expected %d got %d\n", desc, tc.size, size))
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestRemoveThing(t *testing.T) {
	svc := newService(map[string]string{token: email})
	saved, _ := svc.AddThing(token, thing)

	cases := map[string]struct {
		id  string
		key string
		err error
	}{
		"remove thing with wrong credentials": {saved.ID, "?", things.ErrUnauthorizedAccess},
		"remove existing thing":               {saved.ID, token, nil},
		"remove removed thing":                {saved.ID, token, nil},
		"remove non-existing thing":           {"?", token, nil},
	}

	for desc, tc := range cases {
		err := svc.RemoveThing(tc.key, tc.id)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestCreateChannel(t *testing.T) {
	svc := newService(map[string]string{token: email})

	cases := map[string]struct {
		channel things.Channel
		key     string
		err     error
	}{
		"create channel":                        {things.Channel{}, token, nil},
		"create channel with wrong credentials": {things.Channel{}, wrong, things.ErrUnauthorizedAccess},
	}

	for desc, tc := range cases {
		_, err := svc.CreateChannel(tc.key, tc.channel)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestUpdateChannel(t *testing.T) {
	svc := newService(map[string]string{token: email})
	saved, _ := svc.CreateChannel(token, channel)

	cases := map[string]struct {
		channel things.Channel
		key     string
		err     error
	}{
		"update existing channel":               {saved, token, nil},
		"update channel with wrong credentials": {saved, wrong, things.ErrUnauthorizedAccess},
		"update non-existing channel":           {things.Channel{ID: "2", Name: "test"}, token, things.ErrNotFound},
	}

	for desc, tc := range cases {
		err := svc.UpdateChannel(tc.key, tc.channel)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestViewChannel(t *testing.T) {
	svc := newService(map[string]string{token: email})
	saved, _ := svc.CreateChannel(token, channel)

	cases := map[string]struct {
		id  string
		key string
		err error
	}{
		"view existing channel":               {saved.ID, token, nil},
		"view channel with wrong credentials": {saved.ID, wrong, things.ErrUnauthorizedAccess},
		"view non-existing channel":           {wrong, token, things.ErrNotFound},
	}

	for desc, tc := range cases {
		_, err := svc.ViewChannel(tc.key, tc.id)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestListChannels(t *testing.T) {
	svc := newService(map[string]string{token: email})

	n := 10
	for i := 0; i < n; i++ {
		svc.CreateChannel(token, channel)
	}
	cases := map[string]struct {
		key    string
		offset int
		limit  int
		size   int
		err    error
	}{
		"list first 5 channels":                {token, 0, 5, 5, nil},
		"list channels 5-10 channels":          {token, 5, 10, 5, nil},
		"list last channel":                    {token, 6, 10, 4, nil},
		"list offset < 0":                      {token, -1, 10, 0, nil},
		"list limit < 0":                       {token, 1, -10, 0, nil},
		"list limit = 0":                       {token, 1, 0, 0, nil},
		"list channels with wrong credentials": {wrong, 0, 0, 0, things.ErrUnauthorizedAccess},
	}

	for desc, tc := range cases {
		ch, err := svc.ListChannels(tc.key, tc.offset, tc.limit)
		size := len(ch)
		assert.Equal(t, tc.size, size, fmt.Sprintf("%s: expected %d got %d\n", desc, tc.size, size))
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestRemoveChannel(t *testing.T) {
	svc := newService(map[string]string{token: email})
	saved, _ := svc.CreateChannel(token, channel)

	cases := map[string]struct {
		id  string
		key string
		err error
	}{
		"remove channel with wrong credentials": {saved.ID, wrong, things.ErrUnauthorizedAccess},
		"remove existing channel":               {saved.ID, token, nil},
		"remove removed channel":                {saved.ID, token, nil},
		"remove non-existing channel":           {saved.ID, token, nil},
	}

	for desc, tc := range cases {
		err := svc.RemoveChannel(tc.key, tc.id)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestConnect(t *testing.T) {
	svc := newService(map[string]string{token: email})

	sth, _ := svc.AddThing(token, thing)
	sch, _ := svc.CreateChannel(token, channel)

	cases := map[string]struct {
		key     string
		chanID  string
		thingID string
		err     error
	}{
		"connect thing":                         {token, sch.ID, sth.ID, nil},
		"connect thing with wrong credentials":  {wrong, sch.ID, sth.ID, things.ErrUnauthorizedAccess},
		"connect thing to non-existing channel": {token, wrong, sth.ID, things.ErrNotFound},
	}

	for desc, tc := range cases {
		err := svc.Connect(tc.key, tc.chanID, tc.thingID)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestDisconnect(t *testing.T) {
	svc := newService(map[string]string{token: email})

	sth, _ := svc.AddThing(token, thing)
	sch, _ := svc.CreateChannel(token, channel)
	svc.Connect(token, sch.ID, sth.ID)

	cases := []struct {
		desc    string
		key     string
		chanID  string
		thingID string
		err     error
	}{
		{"disconnect connected thing", token, sch.ID, sth.ID, nil},
		{"disconnect disconnected thing", token, sch.ID, sth.ID, things.ErrNotFound},
		{"disconnect thing with wrong credentials", wrong, sch.ID, sth.ID, things.ErrUnauthorizedAccess},
		{"disconnect thing from non-existing channel", token, wrong, sth.ID, things.ErrNotFound},
		{"disconnect non-existing thing", token, sch.ID, wrong, things.ErrNotFound},
	}

	for _, tc := range cases {
		err := svc.Disconnect(tc.key, tc.chanID, tc.thingID)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}

}

func TestCanAccess(t *testing.T) {
	svc := newService(map[string]string{token: email})

	sth, _ := svc.AddThing(token, thing)
	sch, _ := svc.CreateChannel(token, channel)
	svc.Connect(token, sch.ID, sth.ID)

	cases := map[string]struct {
		key     string
		channel string
		err     error
	}{
		"allowed access":              {sth.Key, sch.ID, nil},
		"not-connected cannot access": {"", sch.ID, things.ErrUnauthorizedAccess},
		"access non-existing channel": {sth.Key, wrong, things.ErrUnauthorizedAccess},
	}

	for desc, tc := range cases {
		_, err := svc.CanAccess(tc.key, tc.channel)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}
