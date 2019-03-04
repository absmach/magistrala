package redis

type removeEvent struct {
	id string
}

type updateChannelEvent struct {
	id       string
	name     string
	metadata string
}

// Connection event is either connect or disconnect event.
type disconnectEvent struct {
	thingID   string
	channelID string
}
