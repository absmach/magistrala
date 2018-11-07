// SenML encoder and decoder to pare Sensor Markup Language
package senml

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/ugorji/go/codec"
)

type Format int

const (
	JSON Format = 1 + iota
	XML
	CBOR
	CSV
	MPACK
	LINEP
	JSONLINE
)

type OutputOptions struct {
	PrettyPrint bool
	Topic       string
}

type SenMLRecord struct {
	XMLName *bool `json:"_,omitempty" xml:"senml"`

	BaseName    string  `json:"bn,omitempty"  xml:"bn,attr,omitempty"`
	BaseTime    float64 `json:"bt,omitempty"  xml:"bt,attr,omitempty"`
	BaseUnit    string  `json:"bu,omitempty"  xml:"bu,attr,omitempty"`
	BaseVersion int     `json:"bver,omitempty"  xml:"bver,attr,omitempty"`

	Link string `json:"l,omitempty"  xml:"l,attr,omitempty"`

	Name       string  `json:"n,omitempty"  xml:"n,attr,omitempty"`
	Unit       string  `json:"u,omitempty"  xml:"u,attr,omitempty"`
	Time       float64 `json:"t,omitempty"  xml:"t,attr,omitempty"`
	UpdateTime float64 `json:"ut,omitempty"  xml:"ut,attr,omitempty"`

	Value       *float64 `json:"v,omitempty"  xml:"v,attr,omitempty"`
	StringValue string   `json:"vs,omitempty"  xml:"vs,attr,omitempty"`
	DataValue   string   `json:"vd,omitempty"  xml:"vd,attr,omitempty"`
	BoolValue   *bool    `json:"vb,omitempty"  xml:"vb,attr,omitempty"`

	Sum *float64 `json:"s,omitempty"  xml:"s,attr,omitempty"`
}

type SenML struct {
	XMLName *bool  `json:"_,omitempty" xml:"sensml"`
	Xmlns   string `json:"_,omitempty" xml:"xmlns,attr"`

	Records []SenMLRecord ` xml:"senml"`
}

// Decode takes a SenML message in the given format and parses it and decodes it
// into the returned SenML record.
func Decode(msg []byte, format Format) (SenML, error) {
	var s SenML
	var err error

	s.XMLName = nil
	s.Xmlns = "urn:ietf:params:xml:ns:senml"

	switch {
	case format == JSON:
		// parse the input JSON stream
		err = json.Unmarshal(msg, &s.Records)
		if err != nil {
			//fmt.Println("error parsing JSON SenML Stream: ", err)
			//fmt.Println("msg=", msg)
			return s, err
		}

	case format == JSONLINE:
		// parse the input JSON line
		lines := strings.Split(string(msg), "\n")
		for _, line := range lines {
			r := new(SenMLRecord)
			if len(line) > 5 {
				err = json.Unmarshal([]byte(line), r)
				if err != nil {
					//fmt.Println("error parsing JSON SenML Line: ", err)
					return s, err
				}
				s.Records = append(s.Records, *r)
			}
		}

	case format == XML:
		// parse the input XML
		err = xml.Unmarshal(msg, &s)
		if err != nil {
			//fmt.Println("error parsing XML SenML", err)
			return s, err
		}

	case format == CBOR:
		// parse the input CBOR
		var cborHandle codec.Handle = new(codec.CborHandle)
		var decoder *codec.Decoder = codec.NewDecoderBytes(msg, cborHandle)
		err = decoder.Decode(&s.Records)
		if err != nil {
			//fmt.Println("error parsing CBOR SenML", err)
			return s, err
		}

	case format == MPACK:
		// parse the input MPACK
		// spec for MessagePack is at https://github.com/msgpack/msgpack/
		var mpackHandle codec.Handle = new(codec.MsgpackHandle)
		var decoder *codec.Decoder = codec.NewDecoderBytes(msg, mpackHandle)
		err = decoder.Decode(&s.Records)
		if err != nil {
			//fmt.Println("error parsing MPACK SenML", err)
			return s, err
		}

	}

	if !IsValid(s) {
		return s, errors.New("SenML record not valid")
	}

	return s, nil
}

// Encode takes a SenML record, and encodes it using the given format.
func Encode(s SenML, format Format, options OutputOptions) ([]byte, error) {
	var data []byte
	var err error

	if options.Topic == "" {
		options.Topic = "senml"
	}

	s.Xmlns = "urn:ietf:params:xml:ns:senml"

	switch {

	case format == JSON:
		// ouput JSON version
		if options.PrettyPrint {
			// data, err = json.MarshalIndent(s.Records, "", "  ")
			var lines string
			lines += fmt.Sprintf("[\n  ")
			for i, r := range s.Records {
				if i != 0 {
					lines += ",\n  "
				}
				recData, err := json.Marshal(r)
				if err != nil {
					//fmt.Println("error encoding JSON SenML", err)
					return nil, err
				}
				lines += fmt.Sprintf("%s", recData)
			}
			lines += fmt.Sprintf("\n]\n")
			data = []byte(lines)
		} else {
			data, err = json.Marshal(s.Records)
		}
		if err != nil {
			//fmt.Println("error encoding JSON SenML", err)
			return nil, err
		}

	case format == XML:
		// output a XML version
		if options.PrettyPrint {
			data, err = xml.MarshalIndent(s, "", "  ")
		} else {
			data, err = xml.Marshal(s)
		}
		if err != nil {
			//fmt.Println("error encoding XML SenML", err)
			return nil, err
		}

	case format == CSV:
		// output a CSV version
		var lines string
		for _, r := range s.Records {
			if r.Value != nil {
				// TODO - replace sprintf with bytes.Buffer
				lines += fmt.Sprintf("%s,", r.Name)
				// excell time in days since 1900, unix seconds since 1970
				// ( 1970 is 25569 days after 1900 )
				lines += fmt.Sprintf("%f,", (r.Time/(24.0*3600.0))+25569.0)
				lines += fmt.Sprintf("%f", *r.Value)
				if len(r.Unit) > 0 {
					lines += fmt.Sprintf(",%s", r.Unit)
				}
				lines += fmt.Sprintf("\r\n")
			}
		}
		data = []byte(lines)
		if err != nil {
			//fmt.Println("error encoding CSV SenML", err)
			return nil, err
		}

	case format == CBOR:
		// output a CBOR version
		var cborHandle codec.Handle = new(codec.CborHandle)
		var encoder *codec.Encoder = codec.NewEncoderBytes(&data, cborHandle)
		err = encoder.Encode(s.Records)
		if err != nil {
			//fmt.Println("error encoding CBOR SenML", err)
			return nil, err
		}

	case format == MPACK:
		// output a MPACK version
		var mpackHandle codec.Handle = new(codec.MsgpackHandle)
		var encoder *codec.Encoder = codec.NewEncoderBytes(&data, mpackHandle)
		err = encoder.Encode(s.Records)
		if err != nil {
			//fmt.Println("error encoding MPACK SenML", err)
			return nil, err
		}

	case format == LINEP:
		// ouput Line Protocol
		var buf bytes.Buffer
		for _, r := range s.Records {
			if r.Value != nil {
				buf.WriteString(options.Topic)
				buf.WriteString(",n=")
				buf.WriteString(r.Name)
				buf.WriteString(",u=")
				buf.WriteString(r.Unit)
				buf.WriteString(" v=")
				buf.WriteString(strconv.FormatFloat(*r.Value, 'f', -1, 64))
				buf.WriteString(" ")
				buf.WriteString(strconv.FormatInt(int64(r.Time*1.0e9), 10))
				buf.WriteString("\n")
			}
		}
		data = buf.Bytes()

	case format == JSONLINE:
		// ouput Line Protocol
		var buf bytes.Buffer
		for _, r := range s.Records {
			if r.Value != nil {
				data, err = json.Marshal(r)
				if err != nil {
					//fmt.Println("error encoding JSONLINE SenML", err)
					return nil, err
				}
				buf.Write(data)
				buf.WriteString("\n")
			}
		}
		data = buf.Bytes()
	}

	return data, nil
}

// Removes all the base items and expands records to have items that include
// what previosly in base iterms. Convets relative times to absoltue times.
func Normalize(senml SenML) SenML {
	var bname string = ""
	var btime float64 = 0
	var bunit string = ""
	var ver = 5
	var ret SenML

	var totalRecords int = 0
	for _, r := range senml.Records {
		if (r.Value != nil) || (len(r.StringValue) > 0) || (len(r.DataValue) > 0) || (r.BoolValue != nil) {
			totalRecords += 1
		}
	}

	ret.XMLName = senml.XMLName
	ret.Xmlns = senml.Xmlns
	ret.Records = make([]SenMLRecord, totalRecords)
	var numRecords = 0

	for _, r := range senml.Records {
		if r.BaseTime != 0 {
			btime = r.BaseTime
		}
		if r.BaseVersion != 0 {
			ver = r.BaseVersion
		}
		if len(r.BaseUnit) > 0 {
			bunit = r.BaseUnit
		}
		if len(r.BaseName) > 0 {
			bname = r.BaseName
		}
		r.BaseTime = 0
		r.BaseUnit = ""
		r.BaseName = ""
		r.Name = bname + r.Name
		r.Time = btime + r.Time
		if len(r.Unit) == 0 {
			r.Unit = bunit
		}
		r.BaseVersion = ver

		if r.Time <= 0 {
			// convert to absolute time
			var now int64 = time.Now().UnixNano()
			var t int64 = now / 1000000000.0
			r.Time = float64(t) + r.Time
		}

		if (r.Value != nil) || (len(r.StringValue) > 0) || (len(r.DataValue) > 0) || (r.BoolValue != nil) {
			ret.Records[numRecords] = r
			numRecords += 1
		}
	}

	return ret
}

// Test if SenML is valid
func IsValid(senml SenML) bool {
	var bname string = ""
	var bver = -1

	//fmt.Println("In Validate")

	for _, r := range senml.Records {

		// Check version is same for all records
		if bver == -1 {
			// set the bver the first time it is seen
			if r.BaseVersion != 0 {
				bver = r.BaseVersion
			}
		} else {
			if r.BaseVersion != 0 {
				// next time a version in seen, check it has not changed
				if r.BaseVersion != bver {
					//fmt.Println("unallowed version change ")
					return false
				}
			}
		}

		// Check name
		if len(r.BaseName) > 0 {
			bname = r.BaseName
		}
		name := bname + r.Name
		if len(name) == 0 {
			//fmt.Println("empty name")

			return false
		}
		if (name[0] == '-') || (name[0] == ':') || (name[0] == '.') || (name[0] == '/') || (name[0] == '_') {
			//fmt.Println("Bad first char in name")
			return false
		}
		for _, l := range name {
			if (l < 'a' || l > 'z') && (l < 'A' || l > 'Z') && (l < '0' || l > '9') && (l != '-') && (l != ':') && (l != '.') && (l != '/') && (l != '_') {
				//fmt.Println("Bad char in name")
				return false
			}
		}

		valueCount := 0
		if r.Value != nil {
			valueCount = valueCount + 1
		}
		if r.BoolValue != nil {
			valueCount = valueCount + 1
		}
		if len(r.DataValue) > 0 {
			valueCount = valueCount + 1
		}
		if len(r.StringValue) > 0 {
			valueCount = valueCount + 1
		}
		if valueCount > 1 {
			//fmt.Println("Too many values ")
			return false
		}
		if r.Sum != nil {
			valueCount = valueCount + 1
		}
		if valueCount < 1 {
			//fmt.Println("No value or sum")
			return false
		}

		// Check if name is known Mandatory To Understand
		//for k :=  r {
		// 	fmt.Println( "key=" , k  )
		//         if k[ len(k)-1 ] == '_' {
		//         	fmt.Println("unknown MTU in record")
		//		return false
		//        }
		// }
	}

	return true
}
