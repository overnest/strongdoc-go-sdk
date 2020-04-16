package utils

import (
	"fmt"
	"reflect"
	"time"

	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/timestamp"
)

func printValue(v reflect.Value) {
	if v.IsValid() {
		fmt.Printf("Value=%v, Type=%v, Kind=%v, CanAddr=%v, CanSet=%v, IsValid=%v, IsZero=%v, IsNil=%v\n",
			v, v.Type(), v.Kind(), v.CanAddr(), v.CanSet(), v.IsValid(), v.IsZero(), isNil(v))
	} else {
		fmt.Printf("Value=%v, IsValid=%v\n", v, v.IsValid())
	}
}

var scalar = map[reflect.Kind]bool{
	reflect.Bool:       true,
	reflect.Int:        true,
	reflect.Int8:       true,
	reflect.Int16:      true,
	reflect.Int32:      true,
	reflect.Int64:      true,
	reflect.Uint:       true,
	reflect.Uint8:      true,
	reflect.Uint16:     true,
	reflect.Uint32:     true,
	reflect.Uint64:     true,
	reflect.Uintptr:    true,
	reflect.Float32:    true,
	reflect.Float64:    true,
	reflect.Complex64:  true,
	reflect.Complex128: true,
	reflect.String:     true,
}

func isScalar(kind reflect.Kind) (ok bool) {
	_, ok = scalar[kind]
	return
}

func isNil(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice:
		return v.IsNil()
	default:
		return false
	}
}

func getPtrValue(v reflect.Value) reflect.Value {
	if v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface {
		return v
	}
	return v.Addr()
}

// Dereference pointer types until we get the non-pointer type
func getNoPtrType(t reflect.Type) reflect.Type {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t
}

// Dereference pointer until we get the non-pointer value
func getNoPtrValue(v reflect.Value) reflect.Value {
	for v.Kind() == reflect.Ptr {
		v = reflect.Indirect(v)
	}
	return v
}

func copyField(from, to reflect.Value) {
	// The "to" field is a pointer
	if to.Kind() == reflect.Ptr {
		// The "from" field is also a pointer
		if from.Kind() == reflect.Ptr {
			to.Set(from)
		} else { // The "from" field is not a pointer
			to.Set(getPtrValue(from))
		}
	} else { // The "to" field is not a pointer
		// The "from" field is a pointer
		if from.Kind() == reflect.Ptr {
			to.Set(reflect.Indirect(from))
		} else { // The "from" field is not a pointer
			to.Set(from)
		}
	}
}

// ConvertStruct converts a GRPC struct to a given struct
//
// Usage:
//     filledGoClientStruct, err := ConvertStruct(grpcStruct, emptyGoClientStruct)
//     grpcStruct - The struct returned from gRPC calls.
//     emptyGoClientStruct - The empty go client struct. This could be:
//             "(*Foo)(nil)" or "&Foo{}". But this can not be "nil". That's
//             because the code can not tell the type if it's just a nil
//             interface. This struct is never filled in. It's only used
//             to tell the type of the Go client struct so that it can
//             be constructed, filled in, and returned
//     filledGoClientStruct - This is the filled in Go client struct. This
//             is created from the type of emptyGoClientStruct. This is why
//             emptyGoClientStruct can not be just nil. This return value
//             is always a pointer of the base type of emptyGoClientStruct.
//             For example, the return value could be of type "*Foo",
//             "*[]*Foo", or "*map[string]*Foo".
func ConvertStruct(from interface{}, to interface{}) (interface{}, error) {
	fromValue := reflect.ValueOf(from)
	toValue := reflect.ValueOf(to)

	if !fromValue.IsValid() || !toValue.IsValid() ||
		(fromValue.Kind() == reflect.Ptr && fromValue.IsNil()) {
		return to, nil
	}

	// The following will always be converted to the non-pointer type.
	// For example, if "to" is type "**Foo", "toVal" will be the type "Foo"
	toValue = reflect.New(getNoPtrType(toValue.Type())).Elem()
	fromValue = getNoPtrValue(fromValue)

	if isScalar(toValue.Kind()) {
		toValue.Set(fromValue)
	}

	// This is the time type. We need to convert these from gRPC time type
	if reflect.PtrTo(toValue.Type()) == reflect.TypeOf((*time.Time)(nil)) {
		gTime := reflect.Indirect(fromValue).Interface().(timestamp.Timestamp)
		cTime, err := ptypes.Timestamp(&gTime)
		if err != nil {
			return nil, err
		}
		toValue.Set(reflect.ValueOf(cTime))
	}

	if toValue.Kind() == reflect.Struct {
		for i := 0; i < toValue.NumField(); i++ {
			toField := toValue.Field(i)
			fromField := fromValue.FieldByName(toValue.Type().Field(i).Name)
			if !fromField.IsValid() {
				// If gRPC object doesn't have the client field, skip
				continue
			}

			newValue, err := ConvertStruct(fromField.Interface(), toField.Interface())
			if err != nil {
				return nil, err
			}
			copyField(reflect.ValueOf(newValue), toField)
		}
	}

	if toValue.Kind() == reflect.Slice {
		toElemType := reflect.TypeOf(toValue.Interface()).Elem()
		newSlice := toValue
		for i := 0; i < fromValue.Len(); i++ {
			newSliceElem, err := ConvertStruct(
				fromValue.Index(i).Interface(),
				reflect.Zero(toElemType).Interface(),
			)
			if err != nil {
				return nil, err
			}

			if toElemType.Kind() == reflect.Ptr {
				newSlice = reflect.Append(newSlice, reflect.ValueOf(newSliceElem))
			} else {
				newSlice = reflect.Append(newSlice, reflect.ValueOf(newSliceElem).Elem())
			}
		}
		toValue.Set(newSlice)
	}

	if toValue.Kind() == reflect.Map {
		toElemType := reflect.TypeOf(toValue.Interface()).Elem()
		iter := fromValue.MapRange()

		newMap := reflect.MakeMapWithSize(toValue.Type(), fromValue.Len())
		for iter.Next() {
			key := iter.Key()
			value := iter.Value()
			newMapElem, err := ConvertStruct(
				value.Interface(),
				reflect.Zero(toElemType).Interface(),
			)
			if err != nil {
				return nil, err
			}

			if toElemType.Kind() == reflect.Ptr {
				newMap.SetMapIndex(key, reflect.ValueOf(newMapElem))
			} else {
				newMap.SetMapIndex(key, reflect.ValueOf(newMapElem).Elem())
			}
		}
		toValue.Set(newMap)
	}

	// Not gonna do the following kinds:
	// Array
	// Chan
	// Func
	// UnsafePointer

	return toValue.Addr().Interface(), nil
}
