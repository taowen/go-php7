// Copyright 2016 Alexander Palaistras. All rights reserved.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

#include <errno.h>
#include <stdbool.h>
#include <main/php.h>

#include "value.h"

// Creates a new value and initializes type to null.
zval value_new() {
	return _value_init();
}

// Creates a complete copy of a zval.
// The destination zval needs to be correctly initialized before use.
void value_copy(zval *dst, zval *src) {
	ZVAL_COPY_VALUE(dst, src);
	zval_copy_ctor(dst);
}

// Returns engine value type. Usually compared against KIND_* constants, defined
// in the `value.h` header file.
int value_kind(zval *val) {
	return Z_TYPE((*val));
}

// Set type and value to null.
void value_set_null(zval *val) {
	ZVAL_NULL(val);
}

// Set type and value to integer.
void value_set_long(zval *val, long int num) {
	ZVAL_LONG(val, num);
}

// Set type and value to floating point.
void value_set_double(zval *val, double num) {
	ZVAL_DOUBLE(val, num);
}

// Set type and value to boolean.
void value_set_bool(zval *val, bool status) {
	ZVAL_BOOL(val, status);
}

// Set type and value to string.
void value_set_string(zval *val, char *str) {
	_value_set_string(val, str);
}

// Set type and value to array with a preset initial size.
void value_set_array(zval *val, unsigned int size) {
	array_init_size(val, size);
}

// Set type and value to object.
void value_set_object(zval *val) {
	object_init(val);
}

// Set type and value from zval. The source zval is copied and is otherwise not
// affected.
void value_set_zval(zval *val, zval *src) {
	int kind;

	// Determine concrete type from source zval.
	switch (Z_TYPE_P(src)) {
	case IS_NULL:
		kind = KIND_NULL;
		break;
	case IS_LONG:
		kind = KIND_LONG;
		break;
	case IS_DOUBLE:
		kind = KIND_DOUBLE;
		break;
	case IS_STRING:
		kind = KIND_STRING;
		break;
	case IS_OBJECT:
		kind = KIND_OBJECT;
		break;
	case IS_ARRAY:
		kind = KIND_ARRAY;
		HashTable *h = (Z_ARRVAL_P(src));

		// Determine if array is associative or indexed. In the simplest case, a
		// associative array will have different values for the number of elements
		// and the index of the next free element. In cases where the number of
		// elements and the next free index is equal, we must iterate through
		// the hash table and check the keys themselves.
		if (h->nNumOfElements != h->nNextFreeElement) {
			kind = KIND_MAP;
			break;
		}

		unsigned long i = 0;

		for (zend_hash_internal_pointer_reset(h); i < h->nNumOfElements; i++) {
			unsigned long index;
			int type = _value_current_key_get(h, NULL, &index);

			if (type == HASH_KEY_IS_STRING || index != i) {
				kind = KIND_MAP;
				break;
			}

			zend_hash_move_forward(h);
		}

		break;
	default:
		// Booleans need special handling for different PHP versions.
		if (_value_truth(src) != -1) {
			kind = KIND_BOOL;
			break;
		}

		errno = 1;
		return;
	}

	value_copy(val, src);

	errno = 0;
}

// Set next index of array or map value.
void value_array_next_set(zval *arr, zval *val) {
	add_next_index_zval(arr, val);
}

void value_array_index_set(zval *arr, unsigned long idx, zval *val) {
	add_index_zval(arr, idx, val);
}

void value_array_key_set(zval *arr, const char *key, zval *val) {
	add_assoc_zval(arr, key, val);
}

void value_object_property_set(zval *obj, const char *key, zval *val) {
	add_property_zval(obj, key, val);
}

int value_get_long(zval *val) {
	zval tmp;

	// Return value directly if already in correct type.
	if (Z_TYPE_P(val) == IS_LONG) {
		return Z_LVAL_P(val);
	}

	value_copy(&tmp, val);
	convert_to_long(&tmp);

	return Z_LVAL(tmp);
}

double value_get_double(zval *val) {
	zval tmp;

	// Return value directly if already in correct type.
	if (Z_TYPE_P(val) == IS_DOUBLE) {
		return Z_DVAL_P(val);
	}

	value_copy(&tmp, val);
	convert_to_double(&tmp);

	return Z_DVAL(tmp);
}

bool value_get_bool(zval *val) {
	zval tmp;

	// Return value directly if already in correct type.
	if (Z_TYPE_P(val) == IS_TRUE || Z_TYPE_P(val) == IS_FALSE) {
		return _value_truth(val);
	}

	value_copy(&tmp, val);
	convert_to_boolean(&tmp);

	return _value_truth(&tmp);
}

char *value_get_string(zval *val) {
	zval tmp;
	int result;

	switch (Z_TYPE_P(val)) {
	case IS_STRING:
		value_copy(&tmp, val);
		break;
	case IS_OBJECT:
		result = zend_std_cast_object_tostring(val, &tmp, IS_STRING);
		if (result == FAILURE) {
			ZVAL_EMPTY_STRING(&tmp);
		}

		break;
	default:
		value_copy(&tmp, val);
		convert_to_cstring(&tmp);
	}

	int len = Z_STRLEN(tmp) + 1;
	char *str = malloc(len);
	memcpy(str, Z_STRVAL(tmp), len);

	zval_dtor(&tmp);

	return str;
}

unsigned int value_array_size(zval *arr) {
	switch (Z_TYPE_P(arr)) {
	case IS_ARRAY:
		return Z_ARRVAL_P(arr)->nNumOfElements;
	case IS_OBJECT:
		// Object size is determined by the number of properties, regardless of
		// visibility.
		return Z_OBJPROP_P(arr)->nNumOfElements;
	case IS_NULL:
		// Null values are considered empty.
		return 0;
	}

	// Non-array or object values are considered to be single-value arrays.
	return 1;
}

zval value_array_keys(zval *arr) {
	HashTable *h = NULL;
	zval keys = value_new();

	value_set_array(&keys, value_array_size(arr));

	switch (Z_TYPE_P(arr)) {
	case IS_ARRAY:
	case IS_OBJECT:
		if (Z_TYPE_P(arr) == IS_OBJECT) {
			h = Z_OBJPROP_P(arr);
		} else {
			h = Z_ARRVAL_P(arr);
		}

		unsigned long i = 0;

		for (zend_hash_internal_pointer_reset(h); i < h->nNumOfElements; i++) {
			_value_current_key_set(h, &keys);
			zend_hash_move_forward(h);
		}

		break;
	case IS_NULL:
		// Null values are considered empty.
		break;
	default:
		// Non-array or object values are considered to contain a single key, '0'.
		add_next_index_long(&keys, 0);
	}

	return keys;
}

void value_array_reset(zval *arr) {
	HashTable *h = NULL;

	switch (Z_TYPE_P(arr)) {
	case IS_ARRAY:
		h = Z_ARRVAL_P(arr);
		break;
	case IS_OBJECT:
		h = Z_OBJPROP_P(arr);
		break;
	default:
		return;
	}

	zend_hash_internal_pointer_reset(h);
}

zval value_array_next_get(zval *arr) {
	HashTable *ht = NULL;
	zval val = value_new();

	switch (Z_TYPE_P(arr)) {
	case IS_ARRAY:
		ht = Z_ARRVAL_P(arr);
		break;
	case IS_OBJECT:
		ht = Z_OBJPROP_P(arr);
		break;
	default:
		// Attempting to return the next index of a non-array value will return
		// the value itself, allowing for implicit conversions of scalar values
		// to arrays.
		value_set_zval(&val, arr);
		return val;
	}

	_value_array_next_get(ht, &val);
	return val;
}

zval value_array_index_get(zval *arr, unsigned long idx) {
	HashTable *ht = NULL;
	zval val = value_new();

	switch (Z_TYPE(val)) {
	case IS_ARRAY:
		ht = Z_ARRVAL_P(arr);
		break;
	case IS_OBJECT:
		ht = Z_OBJPROP_P(arr);
		break;
	default:
		// Attempting to return the first index of a non-array value will return
		// the value itself, allowing for implicit conversions of scalar values
		// to arrays.
		if (idx == 0) {
			value_set_zval(&val, arr);
			return val;
		}

		return val;
	}

	_value_array_index_get(ht, idx, &val);
	return val;
}

zval value_array_key_get(zval *arr, char *key) {
	HashTable *ht = NULL;
	zval val = value_new();

	switch (Z_TYPE_P(arr)) {
	case IS_ARRAY:
		ht = Z_ARRVAL_P(arr);
		break;
	case IS_OBJECT:
		ht = Z_OBJPROP_P(arr);
		break;
	default:
		return val;
	}

	_value_array_key_get(ht, key, &val);
	return val;
}

#include "_value.c"
