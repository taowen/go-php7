package engine

import (
	"testing"
	"bytes"
	"net/http"
	"net/http/httptest"
	"reflect"
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



var headerTests = []struct {
	script   string
	expected http.Header
}{
	{
		"header('X-Testing: Hello');",
		http.Header{"X-Testing": []string{"Hello"}},
	},
	{
		"header('X-Testing: World', false);",
		http.Header{"X-Testing": []string{"Hello", "World"}},
	},
	{
		"header_remove('X-Testing');",
		http.Header{},
	},
	{
		"header('X-Testing: Done', false);",
		http.Header{"X-Testing": []string{"Done"}},
	},
}

func Test_write_header_but_do_not_send(t *testing.T) {
	e, _ := New()
	defer e.Destroy()
	recorder := httptest.NewRecorder()
	recorder.Code = 0
	c := &Context{
		ResponseWriter: recorder,
	}
	e.RequestStartup(c)

	for _, tt := range headerTests {
		if _, err := c.Eval(tt.script); err != nil {
			t.Errorf("Context.Eval('%s'): %s", tt.script, err)
			continue
		}

		if reflect.DeepEqual(c.ResponseWriter.Header(), tt.expected) == false {
			t.Errorf("Context.Eval('%s'): expected '%#v', actual '%#v'", tt.script, tt.expected, c.ResponseWriter.Header())
		}
	}
	if recorder.Code != 0 {
		t.FailNow()
	}

	e.RequestShutdown(c)
}

func Test_send_200_by_default(t *testing.T) {
	e, _ := New()
	defer e.Destroy()
	recorder := httptest.NewRecorder()
	recorder.Code = 0
	c := &Context{
		ResponseWriter: recorder,
	}
	e.RequestStartup(c)
	defer e.RequestShutdown(c)
	c.FinishRequest()
	if recorder.Code != 200 {
		t.FailNow()
	}
}

func Test_send_specified_status_code(t *testing.T) {
	e, _ := New()
	defer e.Destroy()
	recorder := httptest.NewRecorder()
	recorder.Code = 0
	c := &Context{
		ResponseWriter: recorder,
	}
	e.RequestStartup(c)
	defer e.RequestShutdown(c)
	c.Eval("http_response_code(400);")
	c.FinishRequest()
	if recorder.Code != 400 {
		t.FailNow()
	}
}

func Test_echo_to_http_response(t *testing.T) {
	e, _ := New()
	defer e.Destroy()
	recorder := httptest.NewRecorder()
	c := &Context{
		ResponseWriter: recorder,
	}
	e.RequestStartup(c)
	defer e.RequestShutdown(c)
	c.Eval("echo('hello');")
	if recorder.Body.String() != "hello" {
		t.FailNow()
	}
}

func Test_echo_will_send_response_code(t *testing.T) {
	e, _ := New()
	defer e.Destroy()
	recorder := httptest.NewRecorder()
	c := &Context{
		ResponseWriter: recorder,
	}
	e.RequestStartup(c)
	defer e.RequestShutdown(c)
	c.Eval("http_response_code(400); echo('hello');")
	if recorder.Code != 400 {
		t.FailNow()
	}
}
