package urlstruct

import (
	"fmt"
	"net/url"
	"reflect"

	"github.com/codemodus/kace"
	"github.com/vmihailenco/tagparser"
)

type Unmarshaler interface {
	UnmarshalValues(url.Values) error
}

//------------------------------------------------------------------------------

type StructInfo struct {
	TableName string
	Fields    []*Field
	structs   map[string][]int

	isUnmarshaler      bool
	unknownFieldsIndex []int
	unmarshalerIndexes [][]int
}

func newStructInfo(typ reflect.Type) *StructInfo {
	sinfo := &StructInfo{
		Fields:        make([]*Field, 0, typ.NumField()),
		isUnmarshaler: isUnmarshaler(reflect.PtrTo(typ)),
	}
	addFields(sinfo, typ, nil)
	return sinfo
}

func (s *StructInfo) Field(name string) *Field {
	col, op := splitColumnOperator(name, "__")
	for _, f := range s.Fields {
		if f.Column == col && f.Op == op {
			return f
		}
	}
	return nil
}

func addFields(sinfo *StructInfo, typ reflect.Type, baseIndex []int) {
	for i := 0; i < typ.NumField(); i++ {
		sf := typ.Field(i)
		if sf.Anonymous {
			tag := sf.Tag.Get("urlstruct")
			if tag == "-" {
				continue
			}

			sfType := sf.Type
			if sfType.Kind() == reflect.Ptr {
				sfType = sfType.Elem()
			}
			if sfType.Kind() != reflect.Struct {
				continue
			}

			addFields(sinfo, sfType, sf.Index)

			if isUnmarshaler(reflect.PtrTo(sfType)) {
				index := joinIndex(baseIndex, sf.Index)
				sinfo.unmarshalerIndexes = append(sinfo.unmarshalerIndexes, index)
			}
		} else {
			addField(sinfo, sf, baseIndex)
		}
	}
}

func addField(sinfo *StructInfo, sf reflect.StructField, baseIndex []int) {
	tagStr := sf.Tag.Get("urlstruct")
	if tagStr == "-" {
		return
	}
	tag := tagparser.Parse(tagStr)
	if tag.Name == "-" {
		return
	}

	if sf.Name == "tableName" {
		name, _ := tagparser.Unquote(tag.Name)
		sinfo.TableName = name
		return
	}

	if _, ok := tag.Options["unknown"]; ok {
		if sf.Type != mapType {
			err := fmt.Errorf("urlstruct: got %s for unknown fields, wanted %s",
				sf.Type, mapType)
			panic(err)
		}
		sinfo.unknownFieldsIndex = joinIndex(baseIndex, sf.Index)
		return
	}

	name := tag.Name
	if name == "" {
		name = sf.Name
	}
	index := joinIndex(baseIndex, sf.Index)

	if sf.Type.Kind() == reflect.Struct {
		if sinfo.structs == nil {
			sinfo.structs = make(map[string][]int)
		}
		sinfo.structs[kace.Snake(name)] = append(baseIndex, sf.Index...)
	}

	if isUnmarshaler(reflect.PtrTo(sf.Type)) {
		sinfo.unmarshalerIndexes = append(sinfo.unmarshalerIndexes, index)
	}

	f := &Field{
		Type:  sf.Type,
		Name:  name,
		Index: index,
		Tag:   tag,
	}
	f.init()
	if f.scanValue != nil && f.isZeroValue != nil {
		sinfo.Fields = append(sinfo.Fields, f)
	}
}

func joinIndex(base, idx []int) []int {
	if len(base) == 0 {
		return idx
	}
	r := make([]int, 0, len(base)+len(idx))
	r = append(r, base...)
	r = append(r, idx...)
	return r
}

//------------------------------------------------------------------------------

var (
	urlValuesType = reflect.TypeOf((*url.Values)(nil)).Elem()
	errorType     = reflect.TypeOf((*error)(nil)).Elem()
)

func isUnmarshaler(typ reflect.Type) bool {
	for i := 0; i < typ.NumMethod(); i++ {
		meth := typ.Method(i)
		if meth.Name == "UnmarshalValues" &&
			meth.Type.NumIn() == 2 &&
			meth.Type.NumOut() == 1 &&
			meth.Type.In(1) == urlValuesType &&
			meth.Type.Out(0) == errorType {
			return true
		}
	}
	return false
}

//------------------------------------------------------------------------------

var mapType = reflect.TypeOf((*map[string][]string)(nil)).Elem()

type fieldMap struct {
	m map[string][]string
}

func newFieldMap(v reflect.Value) *fieldMap {
	if v.IsNil() {
		v.Set(reflect.MakeMap(mapType))
	}
	return &fieldMap{
		m: v.Interface().(map[string][]string),
	}
}

func (fm fieldMap) Decode(name string, values []string) error {
	fm.m[name] = values
	return nil
}
