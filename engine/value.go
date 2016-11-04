// Copyright 2016 Alexander Palaistras. All rights reserved.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package engine

// #include <stdlib.h>
// #include <stdbool.h>
// #include <main/php.h>
// #include "value.h"
import "C"

import (
	"fmt"
	"strconv"
	"unsafe"
	"reflect"
)

// ValueKind represents the specific kind of type represented in Value.
type ValueKind int

// ZVal types
const (
	IS_UNDEF ValueKind = 0
	IS_NULL ValueKind = 1
	IS_FALSE ValueKind = 2
	IS_TRUE ValueKind = 3
	IS_LONG ValueKind = 4
	IS_DOUBLE ValueKind = 5
	IS_STRING ValueKind = 6
	IS_ARRAY ValueKind = 7
	IS_OBJECT ValueKind = 8
	IS_RESOURCE ValueKind = 9
	IS_REFERENCE ValueKind = 10
)

//NewValue creates a PHP value representation of a Go value val. Available
//bindings for Go to PHP types are:
//
//int             -> integer
//float64         -> double
//bool            -> boolean
//string          -> string
//slice           -> indexed array
//map[int|string] -> associative array
//struct          -> object
//
//It is only possible to bind maps with integer or string keys. Only exported
//struct fields are passed to the PHP context. Bindings for functions and method
//receivers to PHP functions and classes are only available in the engine scope,
//and must be predeclared before context execution.
func NewValue(val interface{}) (*C.struct__zval_struct, error) {
	zval, err := C.value_new()
	if err != nil {
		return &zval, fmt.Errorf("Unable to instantiate PHP value")
	}

	v := reflect.ValueOf(val)

	// Determine interface value type and create PHP value from the concrete type.
	switch v.Kind() {
	// Bind integer to PHP int type.
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		C.value_set_long(&zval, C.long(v.Int()))
	// Bind floating point number to PHP double type.
	case reflect.Float32, reflect.Float64:
		C.value_set_double(&zval, C.double(v.Float()))
	// Bind boolean to PHP bool type.
	case reflect.Bool:
		C.value_set_bool(&zval, C.bool(v.Bool()))
	// Bind string to PHP string type.
	case reflect.String:
		str := C.CString(v.String())
		defer C.free(unsafe.Pointer(str))

		C.value_set_string(&zval, str)
	// Bind slice to PHP indexed array type.
	case reflect.Slice:
		C.value_set_array(&zval, C.uint(v.Len()))

		for i := 0; i < v.Len(); i++ {
			vs, err := NewValue(v.Index(i).Interface())
			if err != nil {
				C._value_destroy(&zval)
				return nil, err
			}

			C.value_array_next_set(&zval, vs)
		}
	// Bind map (with integer or string keys) to PHP associative array type.
	case reflect.Map:
		kt := v.Type().Key().Kind()

		if kt == reflect.Int || kt == reflect.String {
			C.value_set_array(&zval, C.uint(v.Len()))

			for _, key := range v.MapKeys() {
				kv, err := NewValue(v.MapIndex(key).Interface())
				if err != nil {
					C._value_destroy(&zval)
					return nil, err
				}

				if kt == reflect.Int {
					C.value_array_index_set(&zval, C.ulong(key.Int()), kv)
				} else {
					str := C.CString(key.String())
					C.value_array_key_set(&zval, str, kv)
					C.free(unsafe.Pointer(str))
				}
			}
		} else {
			return nil, fmt.Errorf("Unable to create value of unknown type '%T'", val)
		}
	// Bind struct to PHP object (stdClass) type.
	case reflect.Struct:
		C.value_set_object(&zval)
		vt := v.Type()

		for i := 0; i < v.NumField(); i++ {
			// Skip unexported fields.
			if vt.Field(i).PkgPath != "" {
				continue
			}

			fv, err := NewValue(v.Field(i).Interface())
			if err != nil {
				C._value_destroy(&zval)
				return nil, err
			}
			str := C.CString(vt.Field(i).Name)
			C.value_object_property_set(&zval, str, fv)
			C.free(unsafe.Pointer(str))
			DestroyValue(fv)
		}
	case reflect.Invalid:
		C.value_set_null(&zval)
	default:
		C._value_destroy(&zval)
		return nil, fmt.Errorf("Unable to create value of unknown type '%T'", val)
	}

	return &zval, nil
}

func IsNull(zval *C.struct__zval_struct) bool {
	return zval == nil || GetKind(zval) == IS_NULL
}

// Kind returns the Value's concrete kind of type.
func GetKind(zval *C.struct__zval_struct) ValueKind {
	return (ValueKind)(C.value_kind(zval))
}

// Interface returns the internal PHP value as it lies, with no conversion step.
func ToInterface(zval *C.struct__zval_struct) interface{} {
	switch GetKind(zval) {
	case IS_LONG:
		return ToInt(zval)
	case IS_DOUBLE:
		return ToFloat(zval)
	case IS_TRUE:
		return true
	case IS_FALSE:
		return false
	case IS_STRING:
		return ToString(zval)
	case IS_ARRAY:
		if C.value_array_is_associative(zval) {
			return ToMap(zval)
		} else {
			return ToSlice(zval)
		}
	case IS_OBJECT:
		return ToMap(zval)
	}

	return nil
}

// Int returns the internal PHP value as an integer, converting if necessary.
func ToInt(zval *C.struct__zval_struct) int64 {
	return (int64)(C.value_get_long(zval))
}

// Float returns the internal PHP value as a floating point number, converting
// if necessary.
func ToFloat(zval *C.struct__zval_struct) float64 {
	return (float64)(C.value_get_double(zval))
}

// Bool returns the internal PHP value as a boolean, converting if necessary.
func ToBool(zval *C.struct__zval_struct) bool {
	return (bool)(C.value_get_bool(zval))
}

// String returns the internal PHP value as a string, converting if necessary.
func ToString(zval *C.struct__zval_struct) string {
	str := C.value_get_string(zval)
	defer C.free(unsafe.Pointer(str))

	return C.GoString(str)
}

// Slice returns the internal PHP value as a slice of interface types. Non-array
// values are implicitly converted to single-element slices.
func ToSlice(zval *C.struct__zval_struct) []interface{} {
	size := (int)(C.value_array_size(zval))
	val := make([]interface{}, size)

	C.value_array_reset(zval)

	for i := 0; i < size; i++ {
		zval := C.value_array_next_get(zval)
		val[i] = ToInterface(&zval)
		DestroyValue(&zval)
	}

	return val
}

// Map returns the internal PHP value as a map of interface types, indexed by
// string keys. Non-array values are implicitly converted to single-element maps
// with a key of '0'.
func ToMap(v *C.struct__zval_struct) map[string]interface{} {
	val := make(map[string]interface{})
	keys := C.value_array_keys(v)
	defer DestroyValue(&keys)
	for _, k := range ToSlice(&keys) {
		fillMap(val, k, v)
	}
	return val
}

func fillMap(val map[string]interface{}, k interface{}, v *C.struct__zval_struct) {
	switch key := k.(type) {
	case int64:
		zval := C.value_array_index_get(v, C.ulong(key))
		defer DestroyValue(&zval)
		sk := strconv.Itoa((int)(key))
		val[sk] = ToInterface(&zval)
	case string:
		str := C.CString(key)
		defer C.free(unsafe.Pointer(str))
		zval := C.value_array_key_get(v, str)
		defer DestroyValue(&zval)
		val[key] = ToInterface(&zval)
	}
}

// Destroy removes all active references to the internal PHP value and frees
// any resources used.
func DestroyValue(zval *C.struct__zval_struct) {
	C._value_destroy(zval)
}
