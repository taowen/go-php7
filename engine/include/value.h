// Copyright 2016 Alexander Palaistras. All rights reserved.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

#ifndef __VALUE_H__
#define __VALUE_H__

//typedef struct _zval {
//	zval *internal;
//	int  kind;
//} zval;

enum {
	KIND_NULL,
	KIND_LONG,
	KIND_DOUBLE,
	KIND_BOOL,
	KIND_STRING,
	KIND_ARRAY,
	KIND_MAP,
	KIND_OBJECT
};

zval value_new();
void value_copy(zval *dst, zval *src);
int value_kind(zval *val);

void value_set_null(zval *val);
void value_set_long(zval *val, long int num);
void value_set_double(zval *val, double num);
void value_set_bool(zval *val, bool status);
void value_set_string(zval *val, char *str);
void value_set_array(zval *val, unsigned int size);
void value_set_object(zval *val);
void value_set_zval(zval *val, zval *src);

void value_array_next_set(zval *arr, zval *val);
void value_array_index_set(zval *arr, unsigned long idx, zval *val);
void value_array_key_set(zval *arr, const char *key, zval *val);
void value_object_property_set(zval *obj, const char *key, zval *val);

int value_get_long(zval *val);
double value_get_double(zval *val);
bool value_get_bool(zval *val);
char *value_get_string(zval *val);

unsigned int value_array_size(zval *arr);
zval value_array_keys(zval *arr);
void value_array_reset(zval *arr);
zval value_array_next_get(zval *arr);
zval value_array_index_get(zval *arr, unsigned long idx);
zval value_array_key_get(zval *arr, char *key);
bool value_array_is_associative(zval *src);

#include "_value.h"

#endif
