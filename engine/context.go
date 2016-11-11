//home/xiaoju/workspace/go-php/ Copyright 2016 Alexander Palaistras. All rights reserved.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package engine

// #include <stdlib.h>
// #include <main/php.h>
// #include "context.h"
import "C"

import (
	"fmt"
	"io"
	"net/http"
	"unsafe"
	"bytes"
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

	context *C.struct__engine_context
	serverValues *C.struct__zval_struct
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
	return c.writeResponse()
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
	err = c.writeResponse()
	return &result, err
}

func (ctx *Context) writeResponse() error {
	if ctx.ResponseWriter == nil {
		return nil
	}
	response_code := int(C.context_get_response_code(ctx.context))
	if response_code == 0 {
		ctx.ResponseWriter.WriteHeader(200);
	} else {
		ctx.ResponseWriter.WriteHeader(response_code);
	}
	outputBuffer := ctx.Output.(*bytes.Buffer)
	outputBytes := outputBuffer.Bytes()
	writeOut, err := ctx.ResponseWriter.Write(outputBytes)
	if err != nil {
		return errors.New(fmt.Sprintf("failed to write output: %s", err.Error()))
	}
	if writeOut != len(outputBytes) {
		return errors.New("not all output write out")
	}
	return nil
}


type evalAssertionArg struct {
	val *C.struct__zval_struct
}
type evalAssertion func(val evalAssertionArg)

func evalAssert(ctx *Context, script string, assertion evalAssertion) {
	theEngine, err := New()
	if err != nil {
		panic(err)
	}
	defer theEngine.Destroy()
	ctx.Output = os.Stdout
	err = engine.RequestStartup(ctx)
	if err != nil {
		panic(err)
	}
	defer engine.RequestShutdown(ctx)
	val, err := ctx.Eval(script)
	if err != nil {
		panic(err)
	}
	defer DestroyValue(val)
	assertion(evalAssertionArg{val})
}
