// Copyright 2016 Alexander Palaistras. All rights reserved.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

#include <errno.h>
#include <stdbool.h>

#include <main/php.h>
#include <main/php_main.h>

#include "value.h"
#include "context.h"

engine_context *context_new(zval *server_values) {
	engine_context *context;

	// Initialize context.
	context = malloc((sizeof(engine_context)));
	if (context == NULL) {
		errno = 1;
		return NULL;
	}
	context->is_finished = 0;

	if (server_values) {
		zval query_string = value_array_key_get(server_values, "QUERY_STRING");
		SG(request_info).query_string = Z_STRVAL(query_string);
		context->query_string = query_string;
		zval request_method = value_array_key_get(server_values, "REQUEST_METHOD");
		SG(request_info).request_method = Z_STRVAL(request_method);
		context->request_method = request_method;
		zval content_type = value_array_key_get(server_values, "HTTP_CONTENT_TYPE");
		SG(request_info).content_type = Z_STRVAL(content_type);
		context->content_type = content_type;
		zval content_length = value_array_key_get(server_values, "HTTP_CONTENT_LENGTH");
		SG(request_info).content_length = Z_LVAL(content_length);
		context->server_values = *server_values;
		context->http_cookie = value_array_key_get(server_values, "HTTP_COOKIE");
	} else {
		ZVAL_NULL(&context->server_values);
	}
	SG(server_context) = context;
	errno = 0;
	return context;
}

void context_dtor(engine_context *context) {
	zval_dtor(&context->server_values);
	zval_dtor(&context->query_string);
	zval_dtor(&context->request_method);
	zval_dtor(&context->content_type);
	zval_dtor(&context->http_cookie);
}

void context_startup(engine_context *context) {
	// Initialize request lifecycle.
	if (php_request_startup() == FAILURE) {
		context_dtor(context);
		SG(server_context) = NULL;
		free(context);
		errno = 1;
	}
	errno = 0;
}

void context_exec(engine_context *context, char *filename) {
	int ret;

	// Attempt to execute script file.
	zend_first_try {
		zend_file_handle script;

		script.type = ZEND_HANDLE_FILENAME;
		script.filename = filename;
		script.opened_path = NULL;
		script.free_filename = 0;

		ret = php_execute_script(&script);
	} zend_catch {
		errno = 1;
		return;
	} zend_end_try();

	if (ret == FAILURE) {
		errno = 1;
		return;
	}

	errno = 0;
	return;
}

zval context_eval(engine_context *context, char *script) {
	zval str = _value_init();
	ZVAL_STRING(&str, script);

	// Compile script value.
	uint32_t compiler_options = CG(compiler_options);

	CG(compiler_options) = ZEND_COMPILE_DEFAULT_FOR_EVAL;
	zend_op_array *op = zend_compile_string(&str, "gophp-engine");
	CG(compiler_options) = compiler_options;

	zval_dtor(&str);

	zval result;
	ZVAL_NULL(&result);
	// Return error if script failed to compile.
	if (!op) {
		errno = 1;
		return result;
	}

	// Attempt to execute compiled string.
	_context_eval(op, &result);

	errno = 0;
	return result;
}

void context_bind(engine_context *context, char *name, zval *value) {
	_context_bind(name, value);
}

void context_destroy(engine_context *context) {
	context_dtor(context);
	php_request_shutdown(NULL);

	SG(server_context) = NULL;
	free(context);
}

#include "_context.c"
