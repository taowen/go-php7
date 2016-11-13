//home/xiaoju/workspace/go-php/ Copyright 2016 Alexander Palaistras. All rights reserved.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package engine

// #include <stdlib.h>
// #include <main/php.h>
// #include "context.h"
// #include "engine.h"
import "C"

import (
	"fmt"
	"io"
	"net/http"
	"unsafe"
	"errors"
	"os"
)

// Context represents an individual execution context.
type Context struct {
	// Output and Log are unbuffered writers used for regular and debug output,
	// respectively. If left unset, any data written into either by the calling
	// context will be lost.
	Output io.Writer
	Log    io.Writer

	// Http Input/Output
	ResponseWriter http.ResponseWriter
	Request *http.Request

	// Other variables in $_SERVER
	DocumentRoot string
	ScriptFileName string

	context *C.struct__engine_context
}

// Bind allows for binding Go values into the current execution context under
// a certain name. Bind returns an error if attempting to bind an invalid value
// (check the documentation for NewValue for what is considered to be a "valid"
// value).
func (c *Context) Bind(name string, val interface{}) error {
	v, err := NewValue(val)
	if err != nil {
		return err
	}

	n := C.CString(name)
	defer C.free(unsafe.Pointer(n))

	C.context_bind(c.context, n, v)

	return nil
}

// Exec executes a PHP script pointed to by filename in the current execution
// context, and returns an error, if any. Output produced by the script is
// written to the context's pre-defined io.Writer instance.
func (c *Context) Exec(filename string) error {
	f := C.CString(filename)
	defer C.free(unsafe.Pointer(f))

	_, err := C.context_exec(c.context, f)
	if err != nil {
		return fmt.Errorf("Error executing script '%s' in context", filename)
	}
	return nil
}

// Eval executes the PHP expression contained in script, and returns a Value
// containing the PHP value returned by the expression, if any. Any output
// produced is written context's pre-defined io.Writer instance.
func (c *Context) Eval(script string) (*C.struct__zval_struct, error) {
	s := C.CString(script)
	defer C.free(unsafe.Pointer(s))

	result, err := C.context_eval(c.context, s)
	if err != nil {
		return nil, fmt.Errorf("Error executing script '%s' in context", script)
	}
	return &result, nil
}

func (ctx *Context) FinishRequest() error {
	result, err := ctx.Eval("return fastcgi_finish_request();")
	if err != nil {
		return err
	}
	defer DestroyValue(result)
	if ToBool(result) {
		return nil
	} else {
		return errors.New("failed to finish request")
	}
}

type evalAssertionArg struct {
	val *C.struct__zval_struct
}
type evalAssertion func(val evalAssertionArg)

func evalAssert(ctx *Context, script string, assertion evalAssertion) {
	Initialize()
	ctx.Output = os.Stdout
	err := RequestStartup(ctx)
	if err != nil {
		panic(err)
	}
	defer RequestShutdown(ctx)
	val, err := ctx.Eval(script)
	if err != nil {
		panic(err)
	}
	defer DestroyValue(val)
	assertion(evalAssertionArg{val})
}
