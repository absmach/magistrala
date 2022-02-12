package noresponse

import (
	"github.com/plgd-dev/go-coap/v2/message/codes"
)

var (
	resp2XXCodes       = []codes.Code{codes.Created, codes.Deleted, codes.Valid, codes.Changed, codes.Content}
	resp4XXCodes       = []codes.Code{codes.BadRequest, codes.Unauthorized, codes.BadOption, codes.Forbidden, codes.NotFound, codes.MethodNotAllowed, codes.NotAcceptable, codes.PreconditionFailed, codes.RequestEntityTooLarge, codes.UnsupportedMediaType}
	resp5XXCodes       = []codes.Code{codes.InternalServerError, codes.NotImplemented, codes.BadGateway, codes.ServiceUnavailable, codes.GatewayTimeout, codes.ProxyingNotSupported}
	noResponseValueMap = map[uint32][]codes.Code{
		2:  resp2XXCodes,
		8:  resp4XXCodes,
		16: resp5XXCodes,
	}
)

func isSet(n uint32, pos uint32) bool {
	val := n & (1 << pos)
	return (val > 0)
}

func decodeNoResponseOption(v uint32) []codes.Code {
	var codes []codes.Code
	if v == 0 {
		// No suppresed code
		return codes
	}

	var i uint32
	// Max bit value:4; ref:table_2_rfc7967
	for i = 0; i <= 4; i++ {
		if isSet(v, i) {
			index := uint32(1 << i)
			codes = append(codes, noResponseValueMap[index]...)
		}
	}
	return codes
}

// IsNoResponseCode validates response code against NoResponse option from request.
// https://www.rfc-editor.org/rfc/rfc7967.txt
func IsNoResponseCode(code codes.Code, noRespValue uint32) error {
	suppressedCodes := decodeNoResponseOption(noRespValue)

	for _, suppressedCode := range suppressedCodes {
		if suppressedCode == code {
			return ErrMessageNotInterested
		}
	}
	return nil
}
