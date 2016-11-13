// Copyright 2016 Alexander Palaistras. All rights reserved.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

// Package engine provides methods allowing for the initialization and teardown
// of PHP engine bindings, off which execution contexts can be launched.
package engine

// #include <stdlib.h>
// #include <main/php.h>
// #include "receiver.h"
// #include "context.h"
// #include "engine.h"
import "C"

import (
	"fmt"
	"io"
	"strings"
	"unsafe"
	"errors"
	"strconv"
	"path/filepath"
)

// Engine represents the core PHP engine bindings.
type Engine struct {
	engine    *C.struct__php_engine
	contexts  map[*C.struct__engine_context]*Context
	receivers map[string]*Receiver
}

// This contains a reference to the active engine, if any.
var engine *Engine
var PHP_INI_PATH_OVERRIDE string

// New initializes a PHP engine instance on which contexts can be executed. It
// corresponds to PHP's MINIT (module init) phase.
func Initialize() error {
	if engine != nil {
		return fmt.Errorf("Cannot activate multiple engine instances")
	}

	var phpInitPathOverride *C.char
	if PHP_INI_PATH_OVERRIDE != "" {
		phpInitPathOverride = C.CString(PHP_INI_PATH_OVERRIDE)
	}
	ptr, err := C.engine_init(phpInitPathOverride)
	if err != nil {
		return fmt.Errorf("PHP engine failed to initialize")
	}

	engine = &Engine{
		engine:    ptr,
		contexts:  make(map[*C.struct__engine_context]*Context),
		receivers: make(map[string]*Receiver),
	}

	return nil
}

// NewContext creates a new execution context for the active engine and returns
// an error if the execution context failed to initialize at any point. This
// corresponds to PHP's RINIT (request init) phase.
func RequestStartup(ctx *Context) error {
	var serverValues *C.struct__zval_struct
	if ctx.Request != nil {
		scriptName, err := filepath.Rel(ctx.DocumentRoot, ctx.ScriptFileName)
		if err != nil {
			scriptName = ""
		} else {
			scriptName = "/" + scriptName
		}
		remoteAddrParts := strings.SplitN(ctx.Request.RemoteAddr, ":", 2)
		remoteAddr := remoteAddrParts[0]
		remotePort := ""
		if len(remoteAddrParts) == 2 {
			remotePort = remoteAddrParts[1]
		}
		httpHostParts := strings.SplitN(ctx.Request.Host, ":", 2)
		serverName := httpHostParts[0]
		serverPort := ""
		if len(httpHostParts) == 2 {
			serverPort = httpHostParts[1]
		}
		serverValues_ := map[string]interface{}{
			"REQUEST_URI": ctx.Request.RequestURI,
			"QUERY_STRING": ctx.Request.URL.RawQuery,
			"REQUEST_METHOD": ctx.Request.Method,
			"DOCUMENT_ROOT": ctx.DocumentRoot,
			"SCRIPT_FILENAME": ctx.ScriptFileName,
			"SCRIPT_NAME": scriptName,
			"PHP_SELF": scriptName,
			"REMOTE_ADDR": remoteAddr,
			"REMOTE_PORT": remotePort,
			"HTTP_HOST": ctx.Request.Host,
			"SERVER_NAME": serverName,
			"SERVER_PORT": serverPort,
		}
		for k, v := range ctx.Request.Header {
			serverValues_["HTTP_" + strings.Replace(strings.ToUpper(k), "-", "_", -1)] = v[0]
		}
		contentLength, isContentLengthSet := serverValues_["HTTP_CONTENT_LENGTH"].(string)
		if isContentLengthSet {
			contentLengthAsInt, err := strconv.Atoi(contentLength)
			if err != nil {
				serverValues_["HTTP_CONTENT_LENGTH"] = 0
			} else {
				serverValues_["HTTP_CONTENT_LENGTH"] = contentLengthAsInt
			}
		}
		serverValues, err = NewValue(serverValues_)
		if err != nil {
			return errors.New(fmt.Sprintf("failed to create server values: %s", err.Error()))
		}
	}
	ptr, err := C.context_new(serverValues)
	if err != nil {
		// serverValues is not owned by context now, need to free it here
		DestroyValue(serverValues)
		return fmt.Errorf("failed to new context: %s", err.Error())
	}
	// passed serverValues ownership to context
	ctx.context = ptr
	// Store reference to context, using pointer as key.
	engine.contexts[ptr] = ctx
	if ctx.ResponseWriter != nil {
		if ctx.Output != nil {
			return errors.New("can not set Output when ResponseWriter is specified")
		}
		ctx.Output = ctx.ResponseWriter
	}
	_, err = C.context_startup(ptr)
	if err != nil {
		return fmt.Errorf("failed to startup context: %s", err.Error())
	}
	return nil
}

// Destroy tears down the current execution context
// corresponds to PHP's RSHUTDOWN phase
func RequestShutdown(ctx *Context) {
	if ctx.context == nil {
		return
	}
	delete(engine.contexts, ctx.context)
	C.context_destroy(ctx.context)
	ctx.context = nil
}

// Define registers a PHP class for the name passed, using function fn as
// constructor for individual object instances as needed by the PHP context.
//
// The class name registered is assumed to be unique for the active engine.
//
// The constructor function accepts a slice of arguments, as passed by the PHP
// context, and should return a method receiver instance, or nil on error (in
// which case, an exception is thrown on the PHP object constructor).
func Define(name string, fn func(args []interface{}) interface{}) error {
	if _, exists := engine.receivers[name]; exists {
		return fmt.Errorf("Failed to define duplicate receiver '%s'", name)
	}

	rcvr := &Receiver{
		name:    name,
		create:  fn,
		objects: make(map[*C.struct__engine_receiver]*ReceiverObject),
	}

	n := C.CString(name)
	defer C.free(unsafe.Pointer(n))

	C.receiver_define(n)
	engine.receivers[name] = rcvr

	return nil
}

func write(w io.Writer, buffer unsafe.Pointer, length C.uint) C.int {
	// Do not return error if writer is unavailable.
	if w == nil {
		return C.int(length)
	}

	written, err := w.Write(C.GoBytes(buffer, C.int(length)))
	if err != nil {
		return -1
	}

	return C.int(written)
}

//export engineWriteOut
func engineWriteOut(ctx *C.struct__engine_context, buffer unsafe.Pointer, length C.uint) C.int {
	if engine == nil || engine.contexts[ctx] == nil {
		return -1
	}

	return write(engine.contexts[ctx].Output, buffer, length)
}

//export engineWriteLog
func engineWriteLog(ctx *C.struct__engine_context, buffer unsafe.Pointer, length C.uint) C.int {
	if engine == nil || engine.contexts[ctx] == nil {
		return -1
	}

	return write(engine.contexts[ctx].Log, buffer, length)
}

//export engineSetHeader
func engineSetHeader(ctx *C.struct__engine_context, operation C.uint, buffer unsafe.Pointer, length C.uint) {
	if engine == nil || engine.contexts[ctx] == nil {
		return
	}

	header := (string)(C.GoBytes(buffer, C.int(length)))
	split := strings.SplitN(header, ":", 2)

	for i := range split {
		split[i] = strings.TrimSpace(split[i])
	}

	context := engine.contexts[ctx]
	if context.ResponseWriter == nil {
		return
	}
	httpHeader := context.ResponseWriter.Header()
	switch operation {
	case 0: // Replace header.
		if len(split) == 2 && split[1] != "" {
			httpHeader.Set(split[0], split[1])
		}
	case 1: // Append header.
		if len(split) == 2 && split[1] != "" {
			httpHeader.Add(split[0], split[1])
		}
	case 2: // Delete header.
		if split[0] != "" {
			httpHeader.Del(split[0])
		}
	}
}

//export engineReceiverNew
func engineReceiverNew(rcvr *C.struct__engine_receiver, args *C.struct__zval_struct) C.int {
	n := C.GoString(C._receiver_get_name(rcvr))
	if engine == nil || engine.receivers[n] == nil {
		return 1
	}

	obj, err := engine.receivers[n].NewObject(ToSlice(args))
	if err != nil {
		return 1
	}

	engine.receivers[n].objects[rcvr] = obj

	return 0
}

//export engineReceiverGet
func engineReceiverGet(rcvr *C.struct__engine_receiver, name *C.char) C.struct__zval_struct {
	n := C.GoString(C._receiver_get_name(rcvr))
	if engine == nil || engine.receivers[n].objects[rcvr] == nil {
		zvalNull, _:= NewValue(nil)
		return *zvalNull
	}

	val, err := engine.receivers[n].objects[rcvr].Get(C.GoString(name))
	if err != nil {
		zvalNull, _:= NewValue(nil)
		return *zvalNull
	}

	return *val
}

//export engineReceiverSet
func engineReceiverSet(rcvr *C.struct__engine_receiver, name *C.char, val *C.struct__zval_struct) {
	n := C.GoString(C._receiver_get_name(rcvr))
	if engine == nil || engine.receivers[n].objects[rcvr] == nil {
		return
	}

	engine.receivers[n].objects[rcvr].Set(C.GoString(name), ToInterface(val))
}

//export engineReceiverExists
func engineReceiverExists(rcvr *C.struct__engine_receiver, name *C.char) C.int {
	n := C.GoString(C._receiver_get_name(rcvr))
	if engine == nil || engine.receivers[n].objects[rcvr] == nil {
		return 0
	}

	if engine.receivers[n].objects[rcvr].Exists(C.GoString(name)) {
		return 1
	}

	return 0
}

//export engineReceiverCall
func engineReceiverCall(rcvr *C.struct__engine_receiver, name *C.char, args *C.struct__zval_struct) C.struct__zval_struct {
	n := C.GoString(C._receiver_get_name(rcvr))
	if engine == nil || engine.receivers[n].objects[rcvr] == nil {
		zvalNull, _:= NewValue(nil)
		return *zvalNull
	}

	val := engine.receivers[n].objects[rcvr].Call(C.GoString(name), ToSlice(args))

	if val == nil {
		zvalNull, _:= NewValue(nil)
		return *zvalNull
	}

	return *val
}


//export engineReadPost
func engineReadPost(ctx *C.struct__engine_context, buffer unsafe.Pointer, length C.uint) C.int {
	context := engine.contexts[ctx]
	if (context == nil || context.Request == nil) {
		return 0
	}
	bufferAsGoSlice := (*[1<<30]byte)(buffer)[:int(length):int(length)]
	readBytesCount, err := context.Request.Body.Read(bufferAsGoSlice)
	if (err != nil && err != io.EOF) {
		return -1
	}
	return C.int(readBytesCount)
}

//export engineSendHeaders
func engineSendHeaders(ctx *C.struct__engine_context, httpResponseCode_ C.int) {
	context := engine.contexts[ctx]
	if (context == nil || context.ResponseWriter == nil) {
		return
	}
	httpResponseCode := int(httpResponseCode_)
	if httpResponseCode == 0 {
		httpResponseCode = 200
	}
	context.ResponseWriter.WriteHeader(httpResponseCode)
}
