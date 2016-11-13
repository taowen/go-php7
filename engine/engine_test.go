// Copyright 2016 Alexander Palaistras. All rights reserved.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package engine

import (
	"io/ioutil"
	"os"
	"testing"
)

type Script struct {
	*os.File
}

func NewScript(name, contents string) (*Script, error) {
	file, err := ioutil.TempFile("", name)
	if err != nil {
		return nil, err
	}

	if _, err := file.WriteString(contents); err != nil {
		file.Close()
		os.Remove(file.Name())

		return nil, err
	}

	return &Script{file}, nil
}

func (s *Script) Remove() {
	s.Close()
	os.Remove(s.Name())
}

func TestEngineNew(t *testing.T) {
	Initialize()
	if engine.engine == nil || engine.contexts == nil || engine.receivers == nil {
		t.Fatalf("New(): Struct fields are `nil` but no error returned")
	}
}

func TestEngineNewContext(t *testing.T) {
	Initialize()
	c := &Context{}
	err := RequestStartup(c)
	if err != nil {
		t.Errorf("NewContext(): %s", err)
	}
	defer RequestShutdown(c)

	if len(engine.contexts) != 1 {
		t.Errorf("NewContext(): `Engine.contexts` length is %d, should be 1", len(engine.contexts))
	}
}

func TestEngineDefine(t *testing.T) {
	Initialize()
	ctor := func(args []interface{}) interface{} {
		return nil
	}

	if err := Define("TestDefine", ctor); err != nil {
		t.Errorf("Engine.Define(): %s", err)
	}

	if len(engine.receivers) != 1 {
		t.Errorf("Engine.Define(): `Engine.receivers` length is %d, should be 1", len(engine.receivers))
	}

	if err := Define("TestDefine", ctor); err == nil {
		t.Errorf("Engine.Define(): Incorrectly defined duplicate receiver")
	}
}

func TestPhpIni(t *testing.T) {
	ioutil.WriteFile("/tmp/php.ini", []byte("post_max_size = 16M"), 0666)
	PHP_INI_PATH_OVERRIDE = "/tmp/php.ini"
	defer func() {
		PHP_INI_PATH_OVERRIDE = ""
	}()
	Initialize()
	c := &Context{}
	RequestStartup(c)
	defer RequestShutdown(c)
	val, _ := c.Eval("return ini_get('post_max_size');")
	defer DestroyValue(val)
	if ToString(val) != "16M" {
		t.FailNow()
	}
}
