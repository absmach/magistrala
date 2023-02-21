package things

type CreateThingEvent struct {
	ID       string                 `mapstructure:"id"`
	Owner    string                 `mapstructure:"owner"`
	Name     string                 `mapstructure:"name"`
	Metadata map[string]interface{} `mapstructure:"metadata"`
}

type UpdateThingEvent struct {
	ID       string                 `mapstructure:"id"`
	Name     string                 `mapstructure:"name"`
	Metadata map[string]interface{} `mapstructure:"metadata"`
}

type RemoveThingEvent struct {
	ID string `mapstructure:"id"`
}

type CreateChannelEvent struct {
	ID       string                 `mapstructure:"id"`
	Owner    string                 `mapstructure:"owner"`
	Name     string                 `mapstructure:"name"`
	Metadata map[string]interface{} `mapstructure:"metadata"`
}

type UpdateChannelEvent struct {
	ID       string                 `mapstructure:"id"`
	Name     string                 `mapstructure:"name"`
	Metadata map[string]interface{} `mapstructure:"metadata"`
}

type RemoveChannelEvent struct {
	ID string `mapstructure:"id"`
}

type ConnectThingEvent struct {
	ChanID  string `mapstructure:"chan_id"`
	ThingID string `mapstructure:"thing_id"`
}

type DisconnectThingEvent struct {
	ChanID  string `mapstructure:"chan_id"`
	ThingID string `mapstructure:"thing_id"`
}

type Type interface {
	CreateThingEvent | UpdateThingEvent | RemoveThingEvent | CreateChannelEvent | UpdateChannelEvent | RemoveChannelEvent | ConnectThingEvent | DisconnectThingEvent
}
