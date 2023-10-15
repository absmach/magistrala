package groups

import "github.com/mainflux/mainflux/pkg/clients"

// PageMeta contains page metadata that helps navigation.
type PageMeta struct {
	Total    uint64           `json:"total"`
	Offset   uint64           `json:"offset"`
	Limit    uint64           `json:"limit"`
	Name     string           `json:"name,omitempty"`
	OwnerID  string           `json:"identity,omitempty"`
	Tag      string           `json:"tag,omitempty"`
	Metadata clients.Metadata `json:"metadata,omitempty"`
	Status   clients.Status   `json:"status,omitempty"`
}
