package urlstruct

import (
	"net/url"
	"reflect"
	"strings"
)

type structDecoder struct {
	v             reflect.Value
	sinfo         *StructInfo
	unknownFields *fieldMap
}

func newStructDecoder(v reflect.Value) *structDecoder {
	return &structDecoder{
		v:     v,
		sinfo: DescribeStruct(v.Type()),
	}
}

func (d *structDecoder) Decode(values url.Values) error {
	var maps map[string][]string
	for name, values := range values {
		name = strings.TrimPrefix(name, ":")
		name = strings.TrimSuffix(name, "[]")

		if name, key, ok := mapKey(name); ok {
			if mdec := d.mapDecoder(name); mdec != nil {
				if err := mdec.DecodeField(key, values); err != nil {
					return err
				}
				continue
			}

			if maps == nil {
				maps = make(map[string][]string)
			}
			maps[name] = append(maps[name], key, values[0])
			continue
		}

		if err := d.DecodeField(name, values); err != nil {
			return err
		}
	}

	for name, values := range maps {
		if err := d.DecodeField(name, values); err != nil {
			return nil
		}
	}

	for _, idx := range d.sinfo.unmarshalerIndexes {
		fv := d.v.FieldByIndex(idx)
		if fv.Kind() == reflect.Struct {
			fv = fv.Addr()
		} else if fv.IsNil() {
			fv.Set(reflect.New(fv.Type().Elem()))
		}

		u := fv.Interface().(Unmarshaler)
		if err := u.UnmarshalValues(values); err != nil {
			return err
		}
	}

	if d.sinfo.isUnmarshaler {
		return d.v.Addr().Interface().(Unmarshaler).UnmarshalValues(values)
	}

	return nil
}

func (d *structDecoder) mapDecoder(name string) *structDecoder {
	if idx, ok := d.sinfo.structs[name]; ok {
		return newStructDecoder(d.v.FieldByIndex(idx))
	}
	return nil
}

func (d *structDecoder) DecodeField(name string, values []string) error {
	if field := d.sinfo.Field(name); field != nil && !field.noDecode {
		return field.scanValue(field.Value(d.v), values)
	}

	if d.sinfo.unknownFieldsIndex == nil {
		return nil
	}

	if d.unknownFields == nil {
		d.unknownFields = newFieldMap(d.v.FieldByIndex(d.sinfo.unknownFieldsIndex))
	}
	return d.unknownFields.Decode(name, values)
}

func mapKey(s string) (name string, key string, ok bool) {
	ind := strings.IndexByte(s, '[')
	if ind == -1 || s[len(s)-1] != ']' {
		return "", "", false
	}
	key = s[ind+1 : len(s)-1]
	if key == "" {
		return "", "", false
	}
	name = s[:ind]
	return name, key, true
}
