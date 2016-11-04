// Copyright 2016 Alexander Palaistras. All rights reserved.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package engine

// #cgo CFLAGS: -I/usr/include/php -I/usr/include/php/main -I/usr/include/php/TSRM
// #cgo CFLAGS: -I/usr/include/php/Zend -Iinclude
//
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

// PHP types representable in Go.
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
	Map =11
)

// Value represents a PHP value.
type Value struct {
	value *C.struct__zval_struct
}

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
func NewValue(val interface{}) (Value, error) {
	zval, err := C.value_new()
	ptr := &zval
	if err != nil {
		return Value{value: ptr}, fmt.Errorf("Unable to instantiate PHP value")
	}

	v := reflect.ValueOf(val)

	// Determine interface value type and create PHP value from the concrete type.
	switch v.Kind() {
	// Bind integer to PHP int type.
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		C.value_set_long(ptr, C.long(v.Int()))
	// Bind floating point number to PHP double type.
	case reflect.Float32, reflect.Float64:
		C.value_set_double(ptr, C.double(v.Float()))
	// Bind boolean to PHP bool type.
	case reflect.Bool:
		C.value_set_bool(ptr, C.bool(v.Bool()))
	// Bind string to PHP string type.
	case reflect.String:
		str := C.CString(v.String())
		defer C.free(unsafe.Pointer(str))

		C.value_set_string(ptr, str)
	// Bind slice to PHP indexed array type.
	case reflect.Slice:
		C.value_set_array(ptr, C.uint(v.Len()))

		for i := 0; i < v.Len(); i++ {
			vs, err := NewValue(v.Index(i).Interface())
			if err != nil {
				C._value_destroy(ptr)
				return Value{}, err
			}

			C.value_array_next_set(ptr, vs.value)
		}
	// Bind map (with integer or string keys) to PHP associative array type.
	case reflect.Map:
		kt := v.Type().Key().Kind()

		if kt == reflect.Int || kt == reflect.String {
			C.value_set_array(ptr, C.uint(v.Len()))

			for _, key := range v.MapKeys() {
				kv, err := NewValue(v.MapIndex(key).Interface())
				if err != nil {
					C._value_destroy(ptr)
					return Value{}, err
				}

				if kt == reflect.Int {
					C.value_array_index_set(ptr, C.ulong(key.Int()), kv.value)
				} else {
					str := C.CString(key.String())
					defer C.free(unsafe.Pointer(str))
					C.value_array_key_set(ptr, str, kv.value)
				}
			}
		} else {
			return Value{}, fmt.Errorf("Unable to create value of unknown type '%T'", val)
		}
	// Bind struct to PHP object (stdClass) type.
	case reflect.Struct:
		C.value_set_object(ptr)
		vt := v.Type()

		for i := 0; i < v.NumField(); i++ {
			// Skip unexported fields.
			if vt.Field(i).PkgPath != "" {
				continue
			}

			fv, err := NewValue(v.Field(i).Interface())
			if err != nil {
				C._value_destroy(ptr)
				return Value{}, err
			}
			defer fv.Destroy()

			str := C.CString(vt.Field(i).Name)
			defer C.free(unsafe.Pointer(str))

			C.value_object_property_set(ptr, str, fv.value)
		}
	case reflect.Invalid:
		C.value_set_null(ptr)
	default:
		C._value_destroy(ptr)
		return Value{value: ptr}, fmt.Errorf("Unable to create value of unknown type '%T'", val)
	}

	return Value{value: ptr}, nil
}

// NewValueFromPtr creates a Value type from an existing PHP value pointer.
func NewValueFromPtr(val unsafe.Pointer) (Value, error) {
	if val == nil {
		return Value{}, fmt.Errorf("Cannot create value from 'nil' pointer")
	}

	zval, err := C.value_new()
	ptr := &zval
	if err != nil {
		return Value{}, fmt.Errorf("Unable to create new PHP value")
	}

	if _, err := C.value_set_zval(ptr, (*C.zval)(val)); err != nil {
		return Value{}, fmt.Errorf("Unable to set PHP value from pointer")
	}

	return Value{value: ptr}, nil
}

func (v Value) IsNull() bool {
	return v.value == nil || v.Kind() == IS_NULL
}

// Kind returns the Value's concrete kind of type.
func (v Value) Kind() ValueKind {
	return (ValueKind)(C.value_kind(v.value))
}

// Interface returns the internal PHP value as it lies, with no conversion step.
func (v Value) Interface() interface{} {
	switch v.Kind() {
	case IS_LONG:
		return v.Int()
	case IS_DOUBLE:
		return v.Float()
	case IS_TRUE:
		return true
	case IS_FALSE:
		return false
	case IS_STRING:
		return v.String()
	case IS_ARRAY:
		if C.value_array_is_associative(v.value) {
			return v.Map()
		} else {
			return v.Slice()
		}
	case IS_OBJECT:
		return v.Map()
	}

	return nil
}

// Int returns the internal PHP value as an integer, converting if necessary.
func (v Value) Int() int64 {
	return (int64)(C.value_get_long(v.value))
}

// Float returns the internal PHP value as a floating point number, converting
// if necessary.
func (v Value) Float() float64 {
	return (float64)(C.value_get_double(v.value))
}

// Bool returns the internal PHP value as a boolean, converting if necessary.
func (v Value) Bool() bool {
	return (bool)(C.value_get_bool(v.value))
}

// String returns the internal PHP value as a string, converting if necessary.
func (v Value) String() string {
	str := C.value_get_string(v.value)
	defer C.free(unsafe.Pointer(str))

	return C.GoString(str)
}

// Slice returns the internal PHP value as a slice of interface types. Non-array
// values are implicitly converted to single-element slices.
func (v Value) Slice() []interface{} {
	size := (int)(C.value_array_size(v.value))
	val := make([]interface{}, size)

	C.value_array_reset(v.value)

	for i := 0; i < size; i++ {
		zval := C.value_array_next_get(v.value)
		t := &Value{value: &zval}

		val[i] = t.Interface()
		t.Destroy()
	}

	return val
}

// Map returns the internal PHP value as a map of interface types, indexed by
// string keys. Non-array values are implicitly converted to single-element maps
// with a key of '0'.
func (v Value) Map() map[string]interface{} {
	val := make(map[string]interface{})
	zval := C.value_array_keys(v.value)
	keys := &Value{value: &zval}
	fmt.Println(reflect.TypeOf(keys.Slice()[0]))
	for _, k := range keys.Slice() {
		switch key := k.(type) {
		case int64:
			zval := C.value_array_index_get(v.value, C.ulong(key))
			t := &Value{value: &zval}
			sk := strconv.Itoa((int)(key))

			val[sk] = t.Interface()
			t.Destroy()
		case string:
			str := C.CString(key)
			zval := C.value_array_key_get(v.value, str)
			t := &Value{value: &zval}
			C.free(unsafe.Pointer(str))

			val[key] = t.Interface()
			t.Destroy()
		}
	}

	keys.Destroy()
	return val
}

// Ptr returns a pointer to the internal PHP value, and is mostly used for
// passing to C functions.
func (v Value) Ptr() unsafe.Pointer {
	return unsafe.Pointer(v.value)
}

// Destroy removes all active references to the internal PHP value and frees
// any resources used.
func (v Value) Destroy() {
	if v.value == nil {
		return
	}

	C._value_destroy(v.value)
	v.value = nil
}
