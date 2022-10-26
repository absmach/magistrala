package message

import (
	"errors"
	"strings"
)

// Options Container of COAP Options, It must be always sort'ed after modification.
type Options []Option

const maxPathValue = 255

// GetPathBufferSize gets the size of the buffer required to store path in URI-Path options.
//
// If the path cannot be stored an error is returned.
func GetPathBufferSize(path string) (int, error) {
	size := 0
	for start := 0; start < len(path); {
		subPath := path[start:]
		segmentSize := strings.Index(subPath, "/")
		if segmentSize == 0 {
			start = start + 1
			continue
		}
		if segmentSize < 0 {
			segmentSize = len(subPath)
		}
		if segmentSize > maxPathValue {
			return -1, ErrInvalidValueLength
		}
		size = size + segmentSize
		start = start + segmentSize + 1
	}
	return size, nil
}

func setPath(options Options, optionID OptionID, buf []byte, path string) (Options, int, error) {
	if len(path) == 0 {
		return options, 0, nil
	}
	o := options.Remove(optionID)
	if path[0] == '/' {
		path = path[1:]
	}
	requiredSize, err := GetPathBufferSize(path)
	if err != nil {
		return options, -1, err
	}
	if requiredSize > len(buf) {
		return options, -1, ErrTooSmall
	}
	encoded := 0
	for start := 0; start < len(path); {
		subPath := path[start:]
		end := strings.Index(subPath, "/")
		if end == 0 {
			start = start + 1
			continue
		}
		if end < 0 {
			end = len(subPath)
		}
		data := buf[encoded:]
		var enc int
		var err error
		o, enc, err = o.AddString(data, optionID, subPath[:end])
		if err != nil {
			return o, -1, err
		}
		encoded += enc
		start = start + end + 1
	}
	return o, encoded, nil
}

// SetPath splits path by '/' to URIPath options and copies it to buffer.
//
// Returns modified options, number of used buf bytes and error if occurs.
//
// @note the url encoded into URIHost, URIPort, URIPath is expected to be
// absolute (https://www.rfc-editor.org/rfc/rfc7252.txt)
func (options Options) SetPath(buf []byte, path string) (Options, int, error) {
	return setPath(options, URIPath, buf, path)
}

// SetLocationPath splits path by '/' to LocationPath options and copies it to buffer.
//
// Returns modified options, number of used buf bytes and error if occurs.
//
// @note the url encoded into LocationPath is expected to be
// absolute (https://www.rfc-editor.org/rfc/rfc7252.txt)
func (options Options) SetLocationPath(buf []byte, path string) (Options, int, error) {
	return setPath(options, LocationPath, buf, path)
}

func (options Options) path(buf []byte, id OptionID) (int, error) {
	firstIdx, lastIdx, err := options.Find(id)
	if err != nil {
		return -1, err
	}
	var needed int
	for i := firstIdx; i < lastIdx; i++ {
		needed += len(options[i].Value)
		needed++
	}

	if len(buf) < needed {
		return needed, ErrTooSmall
	}
	for i := firstIdx; i < lastIdx; i++ {
		buf[0] = '/'
		buf = buf[1:]

		copy(buf, options[i].Value)
		buf = buf[len(options[i].Value):]
	}
	return needed, nil
}

// Path joins URIPath options by '/' to the buf.
//
// Returns number of used buf bytes or error when occurs.
func (options Options) Path() (string, error) {
	buf := make([]byte, 32)
	m, err := options.path(buf, URIPath)
	if errors.Is(err, ErrTooSmall) {
		buf = append(buf, make([]byte, m)...)
		m, err = options.path(buf, URIPath)
	}
	if err != nil {
		return "", err
	}
	buf = buf[:m]
	return string(buf), nil
}

// LocationPath joins Location-Path options by '/' to the buf.
//
// Returns number of used buf bytes or error when occurs.
func (options Options) LocationPath() (string, error) {
	buf := make([]byte, 32)
	m, err := options.path(buf, LocationPath)
	if errors.Is(err, ErrTooSmall) {
		buf = append(buf, make([]byte, m)...)
		m, err = options.path(buf, LocationPath)
	}
	if err != nil {
		return "", err
	}
	buf = buf[:m]
	return string(buf), nil
}

// SetString replaces/stores string option to options.
//
// Returns modified options, number of used buf bytes and error if occurs.
func (options Options) SetString(buf []byte, id OptionID, str string) (Options, int, error) {
	data := []byte(str)
	return options.SetBytes(buf, id, data)
}

// AddString appends string option to options.
//
// Returns modified options, number of used buf bytes and error if occurs.
func (options Options) AddString(buf []byte, id OptionID, str string) (Options, int, error) {
	data := []byte(str)
	return options.AddBytes(buf, id, data)
}

// HasOption returns true is option exist in options.
func (options Options) HasOption(id OptionID) bool {
	_, _, err := options.Find(id)
	return err == nil
}

// GetUint32s gets all options with same id.
func (options Options) GetUint32s(id OptionID, r []uint32) (int, error) {
	firstIdx, lastIdx, err := options.Find(id)
	if err != nil {
		return 0, err
	}
	if len(r) < lastIdx-firstIdx {
		return lastIdx - firstIdx, ErrTooSmall
	}
	var idx int
	for i := firstIdx; i <= lastIdx; i++ {
		val, _, err := DecodeUint32(options[i].Value)
		if err == nil {
			r[idx] = val
			idx++
		}
	}

	return idx, nil
}

// GetUint32 gets the uin32 value of the first option with the given ID.
func (options Options) GetUint32(id OptionID) (uint32, error) {
	firstIdx, _, err := options.Find(id)
	if err != nil {
		return 0, err
	}
	val, _, err := DecodeUint32(options[firstIdx].Value)
	return val, err
}

// ContentFormat gets the content format of body.
func (options Options) ContentFormat() (MediaType, error) {
	v, err := options.GetUint32(ContentFormat)
	return MediaType(v), err
}

// GetString gets the string value of the first option with the given ID.
func (options Options) GetString(id OptionID) (string, error) {
	firstIdx, _, err := options.Find(id)
	if err != nil {
		return "", err
	}
	return string(options[firstIdx].Value), nil
}

// GetStrings gets string array of all options with the given id.
func (options Options) GetStrings(id OptionID, r []string) (int, error) {
	firstIdx, lastIdx, err := options.Find(id)
	if err != nil {
		return 0, err
	}
	if len(r) < lastIdx-firstIdx {
		return lastIdx - firstIdx, ErrTooSmall
	}
	var idx int
	for i := firstIdx; i < lastIdx; i++ {
		r[idx] = string(options[i].Value)
		idx++
	}

	return idx, nil
}

// Queries gets URIQuery parameters.
func (options Options) Queries() ([]string, error) {
	q := make([]string, 4)
	n, err := options.GetStrings(URIQuery, q)
	if errors.Is(err, ErrTooSmall) {
		q = append(q, make([]string, n-len(q))...)
		n, err = options.GetStrings(URIQuery, q)
	}
	if err != nil {
		return nil, err
	}
	return q[:n], nil
}

// SetBytes replaces/stores bytes of a option to options.
//
// Returns modified options, number of used buf bytes and error if occurs.
func (options Options) SetBytes(buf []byte, id OptionID, data []byte) (Options, int, error) {
	if len(buf) < len(data) {
		return options, len(data), ErrTooSmall
	}
	if id == URIPath && len(data) > maxPathValue {
		return options, -1, ErrInvalidValueLength
	}
	copy(buf, data)
	return options.Set(Option{ID: id, Value: buf[:len(data)]}), len(data), nil
}

// AddBytes appends bytes of a option option to options.
//
// Returns modified options, number of used buf bytes and error if occurs.
func (options Options) AddBytes(buf []byte, id OptionID, data []byte) (Options, int, error) {
	if len(buf) < len(data) {
		return options, len(data), ErrTooSmall
	}
	if id == URIPath && len(data) > maxPathValue {
		return options, -1, ErrInvalidValueLength
	}
	copy(buf, data)
	return options.Add(Option{ID: id, Value: buf[:len(data)]}), len(data), nil
}

// GetBytes gets bytes of the first option with given id.
func (options Options) GetBytes(id OptionID) ([]byte, error) {
	firstIdx, _, err := options.Find(id)
	if err != nil {
		return nil, err
	}
	return options[firstIdx].Value, nil
}

// GetBytess gets array of bytes of all options with the same id.
func (options Options) GetBytess(id OptionID, r [][]byte) (int, error) {
	firstIdx, lastIdx, err := options.Find(id)
	if err != nil {
		return 0, err
	}
	if len(r) < lastIdx-firstIdx {
		return lastIdx - firstIdx, ErrTooSmall
	}
	var idx int
	for i := firstIdx; i < lastIdx; i++ {
		r[idx] = options[i].Value
		idx++
	}

	return idx, nil
}

// AddUint32 appends uint32 option to options.
//
// Returns modified options, number of used buf bytes and error if occurs.
func (options Options) AddUint32(buf []byte, id OptionID, value uint32) (Options, int, error) {
	enc, err := EncodeUint32(buf, value)
	if err != nil {
		return options, enc, err
	}
	o := options.Add(Option{ID: id, Value: buf[:enc]})
	return o, enc, err
}

// SetUint32  replaces/stores uint32 option to options.
//
// Returns modified options, number of used buf bytes and error if occurs.
func (options Options) SetUint32(buf []byte, id OptionID, value uint32) (Options, int, error) {
	enc, err := EncodeUint32(buf, value)
	if err != nil {
		return options, enc, err
	}
	o := options.Set(Option{ID: id, Value: buf[:enc]})
	return o, enc, err
}

// SetContentFormat sets ContentFormat option.
func (options Options) SetContentFormat(buf []byte, contentFormat MediaType) (Options, int, error) {
	return options.SetUint32(buf, ContentFormat, uint32(contentFormat))
}

// SetObserve sets ContentFormat option.
func (options Options) SetObserve(buf []byte, observe uint32) (Options, int, error) {
	return options.SetUint32(buf, Observe, observe)
}

// Observe gets observe option.
func (options Options) Observe() (uint32, error) {
	return options.GetUint32(Observe)
}

// SetAccept sets accept option.
func (options Options) SetAccept(buf []byte, contentFormat MediaType) (Options, int, error) {
	return options.SetUint32(buf, Accept, uint32(contentFormat))
}

// Accept gets accept option.
func (options Options) Accept() (MediaType, error) {
	v, err := options.GetUint32(Accept)
	return MediaType(v), err
}

// Find returns range of type options. First number is index and second number is index of next option type.
func (options Options) Find(ID OptionID) (int, int, error) {
	idxPre, idxPost := options.findPosition(ID)
	if idxPre == -1 && idxPost == 0 {
		return -1, -1, ErrOptionNotFound
	}
	if idxPre == len(options)-1 && idxPost == -1 {
		return -1, -1, ErrOptionNotFound
	}
	if idxPre < idxPost && idxPost-idxPre == 1 {
		return -1, -1, ErrOptionNotFound
	}
	idxPre = idxPre + 1
	if idxPost < 0 {
		idxPost = len(options)
	}
	return idxPre, idxPost, nil
}

// findPosition returns opened interval, -1 at means minIdx insert at 0, -1 maxIdx at maxIdx means append.
func (options Options) findPosition(ID OptionID) (minIdx int, maxIdx int) {
	if len(options) == 0 {
		return -1, 0
	}
	pivot := 0
	maxIdx = len(options)
	minIdx = 0
	for {
		switch {
		case ID == options[pivot].ID || (maxIdx-minIdx)/2 == 0:
			for maxIdx = pivot; maxIdx < len(options) && options[maxIdx].ID <= ID; maxIdx++ {
			}
			if maxIdx == len(options) {
				maxIdx = -1
			}
			for minIdx = pivot; minIdx >= 0 && options[minIdx].ID >= ID; minIdx-- {
			}
			return minIdx, maxIdx
		case ID < options[pivot].ID:
			maxIdx = pivot
			pivot = maxIdx - (maxIdx-minIdx)/2
		case ID > options[pivot].ID:
			minIdx = pivot
			pivot = minIdx + (maxIdx-minIdx)/2
		}
	}
}

// Set replaces/stores option at options.
//
// Returns modified options.
func (options Options) Set(opt Option) Options {
	idxPre, idxPost := options.findPosition(opt.ID)
	if idxPre == -1 && idxPost == -1 {
		//append
		options = append(options[:0], opt)
		return options
	}
	var insertPosition int
	var updateTo int
	var updateFrom int
	optsLength := len(options)
	switch {
	case idxPre == -1 && idxPost >= 0:
		insertPosition = 0
		updateTo = 1
		updateFrom = idxPost
	case idxPre == idxPost:
		insertPosition = idxPre
		updateFrom = idxPre
		updateTo = idxPre + 1
	case idxPre >= 0:
		insertPosition = idxPre + 1
		updateTo = idxPre + 2
		updateFrom = idxPost
		if updateFrom < 0 {
			updateFrom = len(options)
		}
		if updateTo == updateFrom {
			options[insertPosition] = opt
			return options
		}
	}
	if len(options) == cap(options) {
		options = append(options, Option{})
	} else {
		options = options[:len(options)+1]
	}
	//replace + move
	updateIdx := updateTo
	if updateFrom < updateTo {
		for i := optsLength; i > updateFrom; i-- {
			options[i] = options[i-1]
			updateIdx++
		}
	} else {
		for i := updateFrom; i < optsLength; i++ {
			options[updateIdx] = options[i]
			updateIdx++
		}
	}
	options[insertPosition] = opt
	options = options[:updateIdx]

	return options
}

// Add appends option to options.
func (options Options) Add(opt Option) Options {
	_, idxPost := options.findPosition(opt.ID)
	if idxPost == -1 {
		idxPost = len(options)
	}
	if len(options) == cap(options) {
		options = append(options, Option{})
	} else {
		options = options[:len(options)+1]
	}
	for i := len(options) - 1; i > idxPost; i-- {
		options[i] = options[i-1]
	}
	options[idxPost] = opt
	return options
}

// Remove removes all options with ID.
func (options Options) Remove(ID OptionID) Options {
	idxPre, idxPost, err := options.Find(ID)
	if err != nil {
		return options
	}
	updateIdx := idxPre
	for i := idxPost; i < len(options); i++ {
		options[updateIdx] = options[i]
		updateIdx++
	}
	length := len(options) - (idxPost - idxPre)
	options = options[:length]

	return options
}

// Marshal marshals options to buf.
//
// Returns the number of used buf bytes.
func (options Options) Marshal(buf []byte) (int, error) {
	previousID := OptionID(0)
	length := 0

	for _, o := range options {
		//return coap.error but calculate length
		if length > len(buf) {
			buf = nil
		}

		var optionLength int
		var err error

		if buf != nil {
			optionLength, err = o.Marshal(buf[length:], previousID)
		} else {
			optionLength, err = o.Marshal(nil, previousID)
		}
		previousID = o.ID

		switch {
		case err == nil:
		case errors.Is(err, ErrTooSmall):
			buf = nil
		default:
			return -1, err
		}
		length = length + optionLength
	}
	if buf == nil {
		return length, ErrTooSmall
	}
	return length, nil
}

// Unmarshal unmarshals data bytes to options and returns the number of consumed bytes.
func (options *Options) Unmarshal(data []byte, optionDefs map[OptionID]OptionDef) (int, error) {
	prev := 0
	processed := 0
	for len(data) > 0 {
		if data[0] == 0xff {
			processed++
			break
		}

		delta := int(data[0] >> 4)
		length := int(data[0] & 0x0f)

		if delta == ExtendOptionError || length == ExtendOptionError {
			return -1, ErrOptionUnexpectedExtendMarker
		}

		data = data[1:]
		processed++

		proc, delta, err := parseExtOpt(data, delta)
		if err != nil {
			return -1, err
		}
		processed += proc
		data = data[proc:]
		proc, length, err = parseExtOpt(data, length)
		if err != nil {
			return -1, err
		}
		processed += proc
		data = data[proc:]

		if len(data) < length {
			return -1, ErrOptionTruncated
		}

		option := Option{}
		oid := OptionID(prev + delta)
		proc, err = option.Unmarshal(data[:length], optionDefs, oid)
		if err != nil {
			return -1, err
		}

		if cap(*options) == len(*options) {
			return -1, ErrOptionsTooSmall
		}
		if option.ID != 0 {
			(*options) = append(*options, option)
		}

		processed += proc
		data = data[proc:]
		prev = int(oid)
	}

	return processed, nil
}

// ResetOptionsTo resets options to in options.
//
// Returns modified options, number of used buf bytes and error if occurs.
func (options Options) ResetOptionsTo(buf []byte, in Options) (Options, int, error) {
	opts := options[:0]
	used := 0
	for idx, o := range in {
		if len(buf) < len(o.Value) {
			for i := idx; i < len(in); i++ {
				used += len(in[i].Value)
			}
			return options, used, ErrTooSmall
		}
		copy(buf, o.Value)
		used += len(o.Value)
		opts = opts.Add(Option{
			ID:    o.ID,
			Value: buf[:len(o.Value)],
		})
		buf = buf[len(o.Value):]
	}
	return opts, used, nil
}

// Clone create duplicates of options.
func (options Options) Clone() (Options, error) {
	opts := make(Options, 0, len(options))
	buf := make([]byte, 64)
	opts, used, err := opts.ResetOptionsTo(buf, options)
	if errors.Is(err, ErrTooSmall) {
		buf = append(buf, make([]byte, used-len(buf))...)
		opts, _, err = opts.ResetOptionsTo(buf, options)
	}
	if err != nil {
		return nil, err
	}
	return opts, nil
}
