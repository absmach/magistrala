package redis

type createThingEvent struct {
	id         string
	loraDevEUI string
}

type removeThingEvent struct {
	id string
}

type createChannelEvent struct {
	id        string
	loraAppID string
}

type removeChannelEvent struct {
	id string
}

type connectionThingEvent struct {
	chanID  string
	thingID string
}
