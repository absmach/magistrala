package dtls

import (
	"context"
)

// Parse received handshakes and return next flightVal
type flightParser func(context.Context, flightConn, *State, *handshakeCache, *handshakeConfig) (flightVal, *alert, error)

// Generate flights
type flightGenerator func(flightConn, *State, *handshakeCache, *handshakeConfig) ([]*packet, *alert, error)

func (f flightVal) getFlightParser() (flightParser, error) {
	switch f {
	case flight0:
		return flight0Parse, nil
	case flight1:
		return flight1Parse, nil
	case flight2:
		return flight2Parse, nil
	case flight3:
		return flight3Parse, nil
	case flight4:
		return flight4Parse, nil
	case flight5:
		return flight5Parse, nil
	case flight6:
		return flight6Parse, nil
	default:
		return nil, errInvalidFlight
	}
}

func (f flightVal) getFlightGenerator() (flightGenerator, error) {
	switch f {
	case flight0:
		return flight0Generate, nil
	case flight1:
		return flight1Generate, nil
	case flight2:
		return flight2Generate, nil
	case flight3:
		return flight3Generate, nil
	case flight4:
		return flight4Generate, nil
	case flight5:
		return flight5Generate, nil
	case flight6:
		return flight6Generate, nil
	default:
		return nil, errInvalidFlight
	}
}
