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
	buffer := &bytes.Buffer{}
	c.Output = buffer
	c.Eval("ob_start(); echo ('hello');")
	if buffer.String() != "" {
		t.FailNow()
	}
	e.RequestShutdown(c)
	if buffer.String() != "hello" {
		t.FailNow()
	}
}
