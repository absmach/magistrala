// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package events_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/absmach/magistrala/twins"
	"github.com/absmach/magistrala/twins/events"
	"github.com/absmach/magistrala/twins/mocks"
	"github.com/stretchr/testify/assert"
)

var (
	subtopics = []string{"engine", "chassis", "wheel_2"}
	channels  = []string{"01ec3c3e-0e66-4e69-9751-a0545b44e08f", "48061e4f-7c23-4f5c-9012-0f9b7cd9d18d", "5b2180e4-e96b-4469-9dc1-b6745078d0b6"}
)

func TestTwinSave(t *testing.T) {
	redisClient.FlushAll(context.Background())
	twinCache := events.NewTwinCache(redisClient)

	twin1 := mocks.CreateTwin(channels[0:2], subtopics[0:2])
	twin2 := mocks.CreateTwin(channels[1:3], subtopics[1:3])

	cases := []struct {
		desc string
		twin twins.Twin
		err  error
	}{
		{
			desc: "Save twin to cache",
			twin: twin1,
			err:  nil,
		},
		{
			desc: "Save already cached twin to cache",
			twin: twin1,
			err:  nil,
		},
		{
			desc: "Save another twin to cache",
			twin: twin2,
			err:  nil,
		},
		{
			desc: "Save already cached twin to cache",
			twin: twin2,
			err:  nil,
		},
	}

	for _, tc := range cases {
		ctx := context.Background()
		err := twinCache.Save(ctx, tc.twin)
		assert.Nil(t, err, fmt.Sprintf("%s: expected %s got %s", tc.desc, tc.err, err))

		def := tc.twin.Definitions[len(tc.twin.Definitions)-1]
		for _, attr := range def.Attributes {
			ids, err := twinCache.IDs(ctx, attr.Channel, attr.Subtopic)
			assert.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
			assert.Contains(t, ids, tc.twin.ID, fmt.Sprintf("%s: id %s not found in %v", tc.desc, tc.twin.ID, ids))
		}
	}
}

func TestTwinSaveIDs(t *testing.T) {
	redisClient.FlushAll(context.Background())
	twinCache := events.NewTwinCache(redisClient)

	twinIDs := []string{"7956f132-0b42-488d-9bd1-0f6dd9d77f98", "a2210c42-1eaf-41ad-b8c1-813317719ed9", "6e815c79-a159-41b0-9ff0-cfa14430e07e"}

	cases := []struct {
		desc     string
		channel  string
		subtopic string
		ids      []string
		err      error
	}{
		{
			desc:     "Save ids to cache",
			channel:  channels[0],
			subtopic: subtopics[0],
			ids:      twinIDs,
			err:      nil,
		},
		{
			desc:     "Save empty ids array to cache",
			channel:  channels[2],
			subtopic: subtopics[2],
			ids:      []string{},
			err:      nil,
		},
		{
			desc:     "Save already saved ids to cache",
			channel:  channels[0],
			subtopic: subtopics[0],
			ids:      twinIDs,
			err:      nil,
		},
		{
			desc:     "Save ids to cache",
			channel:  channels[1],
			subtopic: subtopics[1],
			ids:      twinIDs[0:2],
			err:      nil,
		},
	}

	for _, tc := range cases {
		ctx := context.Background()
		err := twinCache.SaveIDs(ctx, tc.channel, tc.subtopic, tc.ids)
		assert.Nil(t, err, fmt.Sprintf("%s: expected %s got %s", tc.desc, tc.err, err))

		ids, err := twinCache.IDs(ctx, tc.channel, tc.subtopic)
		assert.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
		assert.ElementsMatch(t, ids, tc.ids, fmt.Sprintf("%s: got incorrect ids", tc.desc))
	}
}

func TestTwinUpdate(t *testing.T) {
	redisClient.FlushAll(context.Background())
	twinCache := events.NewTwinCache(redisClient)
	ctx := context.Background()

	var tws []twins.Twin
	for i := range channels {
		tw := mocks.CreateTwin(channels[i:i+1], subtopics[i:i+1])
		tws = append(tws, tw)
	}
	err := twinCache.Save(ctx, tws[0])
	assert.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	tws[1].ID = tws[0].ID

	cases := []struct {
		desc   string
		twinID string
		twin   twins.Twin
		err    error
	}{
		{
			desc:   "Update saved twin",
			twinID: tws[0].ID,
			twin:   tws[1],
			err:    nil,
		},
		{
			desc:   "Update twin with same definition",
			twinID: tws[0].ID,
			twin:   tws[1],
			err:    nil,
		},
		{
			desc:   "Update unsaved twin definition",
			twinID: tws[2].ID,
			twin:   tws[2],
			err:    nil,
		},
	}

	for _, tc := range cases {
		err := twinCache.Update(ctx, tc.twin)
		assert.Nil(t, err, fmt.Sprintf("%s: expected %s got %s", tc.desc, tc.err, err))

		attr := tc.twin.Definitions[0].Attributes[0]
		ids, err := twinCache.IDs(ctx, attr.Channel, attr.Subtopic)
		assert.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
		assert.Contains(t, ids, tc.twinID, fmt.Sprintf("%s: the list doesn't contain the correct elements", tc.desc))
	}
}

func TestTwinIDs(t *testing.T) {
	redisClient.FlushAll(context.Background())
	twinCache := events.NewTwinCache(redisClient)
	ctx := context.Background()

	var tws []twins.Twin
	for i := 0; i < len(channels); i++ {
		tw := mocks.CreateTwin(channels[0:1], subtopics[0:1])
		err := twinCache.Save(ctx, tw)
		assert.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
		tws = append(tws, tw)
	}
	for i := 0; i < len(channels); i++ {
		tw := mocks.CreateTwin(channels[1:2], subtopics[1:2])
		err := twinCache.Save(ctx, tw)
		assert.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
		tws = append(tws, tw)
	}
	twEmptySubt := mocks.CreateTwin(channels[0:1], []string{""})
	err := twinCache.Save(ctx, twEmptySubt)
	assert.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	twSubtWild := mocks.CreateTwin(channels[0:1], []string{twins.SubtopicWildcard})
	err = twinCache.Save(ctx, twSubtWild)
	assert.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	nonExistAttr := twins.Attribute{
		Channel:      channels[2],
		Subtopic:     subtopics[2],
		PersistState: true,
	}

	cases := []struct {
		desc string
		ids  []string
		attr twins.Attribute
		err  error
	}{
		{
			desc: "Get twin IDs from cache for empty subtopic attribute",
			ids:  []string{twEmptySubt.ID, twSubtWild.ID},
			attr: twEmptySubt.Definitions[0].Attributes[0],
			err:  nil,
		},
		{
			desc: "Get twin IDs from cache for subset of ids",
			ids:  []string{tws[0].ID, tws[1].ID, tws[2].ID, twSubtWild.ID},
			attr: tws[0].Definitions[0].Attributes[0],
			err:  nil,
		},
		{
			desc: "Get twin IDs from cache for subset of ids",
			ids:  []string{tws[3].ID, tws[4].ID, tws[5].ID},
			attr: tws[3].Definitions[0].Attributes[0],
			err:  nil,
		},
		{
			desc: "Get twin IDs from cache for non existing attribute",
			ids:  []string{},
			attr: nonExistAttr,
			err:  nil,
		},
	}

	for _, tc := range cases {
		ids, err := twinCache.IDs(ctx, tc.attr.Channel, tc.attr.Subtopic)
		assert.Nil(t, err, fmt.Sprintf("%s: expected %s got %s", tc.desc, tc.err, err))
		assert.ElementsMatch(t, ids, tc.ids, fmt.Sprintf("%s: got unexpected list of IDs", tc.desc))
	}
}

func TestTwinRemove(t *testing.T) {
	redisClient.FlushAll(context.Background())
	twinCache := events.NewTwinCache(redisClient)
	ctx := context.Background()

	var tws []twins.Twin
	for i := range channels {
		tw := mocks.CreateTwin(channels[i:i+1], subtopics[i:i+1])
		err := twinCache.Save(ctx, tw)
		assert.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
		tws = append(tws, tw)
	}

	cases := []struct {
		desc string
		twin twins.Twin
		err  error
	}{
		{
			desc: "Remove twin from cache",
			twin: tws[0],
			err:  nil,
		},
		{
			desc: "Remove already removed twin from cache",
			twin: tws[0],
			err:  nil,
		},
		{
			desc: "Remove another twin from cache",
			twin: tws[1],
			err:  nil,
		},
	}

	for _, tc := range cases {
		err := twinCache.Remove(ctx, tc.twin.ID)
		assert.Nil(t, err, fmt.Sprintf("%s: expected %s got %s", tc.desc, tc.err, err))

		def := tc.twin.Definitions[len(tc.twin.Definitions)-1]
		for _, attr := range def.Attributes {
			ids, err := twinCache.IDs(ctx, attr.Channel, attr.Subtopic)
			assert.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
			assert.NotContains(t, ids, tc.twin.ID, fmt.Sprintf("%s: found unexpected ID in the list", tc.desc))
		}
	}
}
