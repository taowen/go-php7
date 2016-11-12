// Copyright 2016 Alexander Palaistras. All rights reserved.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

#include <stdio.h>
#include <errno.h>

#include <main/php.h>
#include <main/SAPI.h>
#include <main/php_main.h>
#include <main/php_variables.h>

#include "context.h"
#include "engine.h"
#include "_cgo_export.h"

// The php.ini defaults for the Go-PHP engine.
const char engine_ini_defaults[] = {
	"expose_php = 0\n"
	"default_mimetype =\n"
	"html_errors = 0\n"
	"register_argc_argv = 1\n"
	"implicit_flush = 1\n"
	"output_buffering = 0\n"
	"max_execution_time = 0\n"
	"log_errors = 1\n"
	"error_log = \"/tmp/php-error.log\"\n"
	"max_input_time = -1\n\0"
};
static zend_module_entry engine_module_entry;

static int engine_ub_write(const char *str, uint len) {
	engine_context *context = SG(server_context);

	int written = engineWriteOut(context, (void *) str, len);
	if (written != len) {
		php_handle_aborted_connection();
	}

	return len;
}

static int engine_header_handler(sapi_header_struct *sapi_header, sapi_header_op_enum op, sapi_headers_struct *sapi_headers) {
	engine_context *context = SG(server_context);

	switch (op) {
	case SAPI_HEADER_REPLACE:
	case SAPI_HEADER_ADD:
	case SAPI_HEADER_DELETE:
		engineSetHeader(context, op, (void *) sapi_header->header, sapi_header->header_len);
		break;
	}

	return 0;
}

static void engine_send_header(sapi_header_struct *sapi_header, void *server_context) {
	// Do nothing.
}

static size_t engine_read_post(char *buffer, size_t count_bytes) {
	return engineReadPost(SG(server_context), buffer, count_bytes);
}

static char *engine_read_cookies() {
	engine_context *context = SG(server_context);
	if (Z_TYPE(context->http_cookie) == IS_STRING) {
		return Z_STRVAL(context->http_cookie);
	}
	return NULL;
}

static void engine_register_variables(zval *track_vars_array) {
	engine_context *context = SG(server_context);
	if (Z_TYPE(context->server_values) == IS_ARRAY) {
		zval_dtor(track_vars_array);
		ZVAL_DUP(track_vars_array, &context->server_values);
	}
}

static void engine_log_message(char *str) {
	engine_context *context = SG(server_context);

	engineWriteLog(context, (void *) str, strlen(str));
}

PHP_FUNCTION(fastcgi_finish_request) /* {{{ */
{
	engine_context *context = SG(server_context);
	if (context->is_finished) {
		RETURN_FALSE;
	}
	sapi_send_headers();
	php_output_end_all();
	context->is_finished = 1;
	RETURN_TRUE;
}

static const zend_function_entry engine_sapi_functions[] = {
	PHP_FE(fastcgi_finish_request,              NULL)
	{NULL, NULL, NULL}
};

static sapi_module_struct engine_module = {
	"gophp-engine",              // Name
	"Go PHP Engine Library",     // Pretty Name

	NULL,                        // Startup
	php_module_shutdown_wrapper, // Shutdown

	NULL,                        // Activate
	NULL,                        // Deactivate

	_engine_ub_write,            // Unbuffered Write
	NULL,                        // Flush
	NULL,                        // Get UID
	NULL,                        // Getenv

	php_error,                   // Error Handler

	engine_header_handler,       // Header Handler
	NULL,                        // Send Headers Handler
	engine_send_header,          // Send Header Handler

	engine_read_post,            // Read POST Data
	engine_read_cookies,         // Read Cookies

	engine_register_variables,   // Register Server Variables
	engine_log_message,          // Log Message
	NULL,                        // Get Request Time
	NULL,                        // Child Terminate

	STANDARD_SAPI_MODULE_PROPERTIES
};

php_engine *engine_init(void) {
	php_engine *engine;

	#ifdef HAVE_SIGNAL_H
		#if defined(SIGPIPE) && defined(SIG_IGN)
			signal(SIGPIPE, SIG_IGN);
		#endif
	#endif

	sapi_startup(&engine_module);

	engine_module.ini_entries = malloc(sizeof(engine_ini_defaults));
	memcpy(engine_module.ini_entries, engine_ini_defaults, sizeof(engine_ini_defaults));
	engine_module.additional_functions = engine_sapi_functions;

	if (php_module_startup(&engine_module, NULL, 0) == FAILURE) {
		sapi_shutdown();

		errno = 1;
		return NULL;
	}

	engine = malloc((sizeof(php_engine)));

	errno = 0;
	return engine;
}

void engine_shutdown(php_engine *engine) {
	php_module_shutdown();
	sapi_shutdown();

	free(engine_module.ini_entries);
	free(engine);
}

#include "_engine.c"
