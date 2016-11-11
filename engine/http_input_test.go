package engine

import (
	"net/http/httptest"
	"testing"
	"net/http"
)

func Test_SERVER_REQUEST_URI(t *testing.T) {
	evalAssert(&Context{
		Request: httptest.NewRequest(http.MethodGet, "/hello", nil),
	}, "return $_SERVER['REQUEST_URI'];", func(val evalAssertionArg) {
		if ToString(val.val) != "/hello" {
			t.Fatal(ToString(val.val))
		}
	})
}

func Test_SERVER_QUERY_STRING(t *testing.T) {
	evalAssert(&Context{
		Request: httptest.NewRequest(http.MethodGet, "/hello?qs_arg=qs_value", nil),
	}, "return $_SERVER['QUERY_STRING'];", func(val evalAssertionArg) {
		if ToString(val.val) != "qs_arg=qs_value" {
			t.Fatal(ToString(val.val))
		}
	})
}

func Test_GET(t *testing.T) {
	evalAssert(&Context{
		Request: httptest.NewRequest(http.MethodGet, "/hello?qs_arg=qs_value", nil),
	}, "return $_GET['qs_arg'];", func(val evalAssertionArg) {
		if ToString(val.val) != "qs_value" {
			t.Fatal(ToString(val.val))
		}
	})
}

func Test_SERVER_REQUEST_METHOD(t *testing.T) {
	evalAssert(&Context{
		Request: httptest.NewRequest(http.MethodPost, "/hello", nil),
	}, "return $_SERVER['REQUEST_METHOD'];", func(val evalAssertionArg) {
		if ToString(val.val) != "POST" {
			t.Fatal(ToString(val.val))
		}
	})
}
