package gosenml

import "testing"

var singleDatapoint = string(`{"e":[{ "n": "urn:dev:ow:10e2073a01080063", "v":23.5 }]}`)

var multipleDatapoints1 = string(
	`{"e":[
	        { "n": "voltage", "t": 0, "u": "V", "v": 120.1 },
	        { "n": "current", "t": 0, "u": "A", "v": 1.2 }],
	    "bn": "urn:dev:mac:0024befffe804ff1/"}`)

var multipleDatapoints2 = string(
	`{"e":[
	        { "n": "voltage", "u": "V", "v": 120.1 },
	        { "n": "current", "t": -5, "v": 1.2 },
	        { "n": "current", "t": -4, "v": 1.30 },
	        { "n": "current", "t": -3, "v": 0.14e1 },
	        { "n": "current", "t": -2, "v": 1.5 },
	        { "n": "current", "t": -1, "v": 1.6 },
	        { "n": "current", "t": 0,   "v": 1.7 }],
	    "bn": "urn:dev:mac:0024befffe804ff1/",
	    "bt": 1276020076,
	    "ver": 1,
	    "bu": "A"}`)

var multipleMeasurements = string(
	`{"e":[
        { "v": 20.0, "t": 0 },
        { "sv": "E 24' 30.621", "u": "lon", "t": 0 },
        { "sv": "N 60' 7.965", "u": "lat", "t": 0 },
        { "v": 20.3, "t": 60 },
        { "sv": "E 24' 30.622", "u": "lon", "t": 60 },
        { "sv": "N 60' 7.965", "u": "lat", "t": 60 },
        { "v": 20.7, "t": 120 },
        { "sv": "E 24' 30.623", "u": "lon", "t": 120 },
        { "sv": "N 60' 7.966", "u": "lat", "t": 120 },
        { "v": 98.0, "u": "%EL", "t": 150 },
        { "v": 21.2, "t": 180 },
        { "sv": "E 24' 30.628", "u": "lon", "t": 180 },
        { "sv": "N 60' 7.967", "u": "lat", "t": 180 }],
    "bn": "http://[2001:db8::1]",
    "bt": 1320067464,
    "bu": "%RH"}`)

var collectionOfResources = string(
	`{"e":[
        { "n": "temperature", "v": 27.2, "u": "Cel" },
        { "n": "humidity", "v": 80, "u": "%RH" }],
    "bn": "http://[2001:db8::2]/",
    "bt": 1320078429,
    "ver": 1}`)

var sumAndValue = string(
	`{"e":[
        { "n": "capacity", "s": 52.4 },
        { "n": "capacityA", "s": 14.6, "v": 7.3 },
        { "n": "capacityB", "s": 37.8, "v": 18.9 }]}`)

var testSuiteValid = []string{
	singleDatapoint,
	multipleDatapoints1,
	multipleDatapoints2,
	multipleMeasurements,
	collectionOfResources,
	sumAndValue,
}

var emptyEntries = string(`{"e":[]}`)

var multipleValues = string(
	`{"e":[
        { "n": "temperature", "v": 27.2, "sv": "normal" },
        { "n": "humidity", "v": 80, "bv": true }]}`)

var noValues = string(
	`{"e":[
        { "n": "temperature" },
        { "n": "humidity", "v": 80 }]}`)

var testSuiteInvalid = []string{
	emptyEntries,
	multipleValues,
	noValues,
}

func TestDecodeValid(t *testing.T) {
	decoder := NewJSONDecoder()
	for _, d := range testSuiteValid {
		_, err := decoder.DecodeMessage([]byte(d))
		if err != nil {
			t.Errorf("%v", err)
		}
	}
}

func TestDecodeInvalid(t *testing.T) {
	decoder := NewJSONDecoder()
	for _, d := range testSuiteInvalid {
		_, err := decoder.DecodeMessage([]byte(d))
		if err == nil {
			t.Fail()
		}
	}
}

func TestEncodeValid(t *testing.T) {
	encoder := NewJSONEncoder()
	decoder := NewJSONDecoder()
	for _, d := range testSuiteValid {
		m, err := decoder.DecodeMessage([]byte(d))
		if err != nil {
			t.Errorf("Failed to decode: %v", err)
		}
		_, err = encoder.EncodeMessage(&m)
		if err != nil {
			t.Errorf("%v", err)
		}
	}
}
