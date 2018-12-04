package redis

type createThingEvent struct {
	id       string
	kind     string
	metadata string
}

type updateThingEvent struct {
	id       string
	kind     string
	metadata string
}

type removeThingEvent struct {
	id string
}

type createChannelEvent struct {
	id       string
	metadata string
}

type updateChannelEvent struct {
	id       string
	metadata string
}

type removeChannelEvent struct {
	id string
}
