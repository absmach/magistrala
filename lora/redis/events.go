package redis

type createThingEvent struct {
	id       string
	metadata thingMetadata
}

type updateThingEvent struct {
	id       string
	metadata thingMetadata
}

type removeThingEvent struct {
	id string
}

type thingMetadata struct {
	DevEUI string `json:"devEUI"`
}

type createChannelEvent struct {
	id       string
	metadata channelMetadata
}

type updateChannelEvent struct {
	id       string
	metadata channelMetadata
}

type removeChannelEvent struct {
	id string
}

type channelMetadata struct {
	AppID string `json:"appID"`
}
