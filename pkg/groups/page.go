package groups

import "github.com/mainflux/mainflux/pkg/clients"

// Page contains page metadata that helps navigation.
type Page struct {
	Total        uint64           `json:"total"`
	Offset       uint64           `json:"offset"`
	Limit        uint64           `json:"limit"`
	Name         string           `json:"name,omitempty"`
	OwnerID      string           `json:"identity,omitempty"`
	Tag          string           `json:"tag,omitempty"`
	Metadata     clients.Metadata `json:"metadata,omitempty"`
	SharedBy     string           `json:"shared_by,omitempty"`
	Status       clients.Status   `json:"status,omitempty"`
	Subject      string           `json:"subject,omitempty"`
	Action       string           `json:"action,omitempty"`
	Disconnected bool             `json:"disconnected,omitempty"` // Used for connected or disconnected lists
}
