package redis

type createThingEvent struct {
	id       string
	metadata map[string]interface{}
}

type removeThingEvent struct {
	id string
}

type createChannelEvent struct {
	id       string
	metadata map[string]interface{}
}

type removeChannelEvent struct {
	id string
}
