package urlstruct

import (
	"fmt"
	"net/url"
	"reflect"
	"sync"
)

var globalMap structInfoMap

func DescribeStruct(typ reflect.Type) *StructInfo {
	return globalMap.DescribeStruct(typ)
}

// Unmarshal unmarshals url values into the struct.
func Unmarshal(values url.Values, strct interface{}) error {
	v := reflect.Indirect(reflect.ValueOf(strct))
	d := newStructDecoder(v)
	return d.Decode(values)
}

type structInfoMap struct {
	m sync.Map
}

func (m *structInfoMap) DescribeStruct(typ reflect.Type) *StructInfo {
	if typ.Kind() != reflect.Struct {
		panic(fmt.Errorf("got %s, wanted %s", typ.Kind(), reflect.Struct))
	}

	if v, ok := m.m.Load(typ); ok {
		return v.(*StructInfo)
	}

	sinfo := newStructInfo(typ)
	if v, loaded := m.m.LoadOrStore(typ, sinfo); loaded {
		return v.(*StructInfo)
	}
	return sinfo
}
