package noresponse

import "errors"

// ErrMessageNotInterested message is not of interest to the client
var ErrMessageNotInterested = errors.New("message not to be sent due to disinterest")
