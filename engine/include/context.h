// Copyright 2016 Alexander Palaistras. All rights reserved.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

#ifndef __CONTEXT_H__
#define __CONTEXT_H__

typedef struct _engine_context {
	zval server_values;
	zval query_string;
	zval request_method;
	zval content_type;
	zval http_cookie;
} engine_context;

engine_context *context_new(zval *server_values);
void context_startup(engine_context *context);
void context_exec(engine_context *context, char *filename);
zval context_eval(engine_context *context, char *script);
void context_bind(engine_context *context, char *name, zval *value);
void context_destroy(engine_context *context);
int context_get_response_code(engine_context *context);
void context_finish_request(engine_context *context);

#include "_context.h"

#endif
