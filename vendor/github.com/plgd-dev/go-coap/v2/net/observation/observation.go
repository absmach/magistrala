package observation

import (
	"time"
)

// ObservationSequenceTimeout defines how long is sequence number is valid. https://tools.ietf.org/html/rfc7641#section-3.4
const ObservationSequenceTimeout = 128 * time.Second

// ValidSequenceNumber implements conditions in https://tools.ietf.org/html/rfc7641#section-3.4
func ValidSequenceNumber(oldValue, newValue uint32, lastEventOccurs time.Time, now time.Time) bool {
	if oldValue < newValue && (newValue-oldValue) < (1<<23) {
		return true
	}
	if oldValue > newValue && (oldValue-newValue) > (1<<23) {
		return true
	}
	if now.Sub(lastEventOccurs) > ObservationSequenceTimeout {
		return true
	}
	return false
}
