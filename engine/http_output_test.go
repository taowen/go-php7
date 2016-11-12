package engine

import (
	"testing"
	"bytes"
)

func Test_finish_request(t *testing.T) {
	e, _ := New()
	defer e.Destroy()
	c := &Context{}
	e.RequestStartup(c)
	defer e.RequestShutdown(c)
	buffer := &bytes.Buffer{}
	c.Output = buffer
	c.Eval("ob_start(); echo ('hello');")
	if buffer.String() != "" {
		t.FailNow()
	}
	err := c.FinishRequest()
	if err != nil {
		t.Fatal(err)
	}
	err = c.FinishRequest()
	if err == nil {
		t.FailNow()
	}
	if buffer.String() != "hello" {
		t.FailNow()
	}
}

func Test_finish_request_from_php(t *testing.T) {
	e, _ := New()
	defer e.Destroy()
	c := &Context{}
	e.RequestStartup(c)
	defer e.RequestShutdown(c)
	buffer := &bytes.Buffer{}
	c.Output = buffer
	c.Eval("ob_start(); echo ('hello');")
	if buffer.String() != "" {
		t.FailNow()
	}
	c.Eval("fastcgi_finish_request();")
	if buffer.String() != "hello" {
		t.FailNow()
	}
}
