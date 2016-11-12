package engine

import (
	"net/http/httptest"
	"testing"
	"net/http"
	"net/url"
	"bytes"
	"strconv"
	"mime/multipart"
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

func Test_SERVER_HTTP_CONTENT_TYPE(t *testing.T) {
	body := url.Values{}
	body.Set("form_arg", "form_value")
	bodyBytes := body.Encode()
	req := httptest.NewRequest(http.MethodPost, "/hello", bytes.NewBufferString(bodyBytes))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Content-Length", strconv.Itoa(len(bodyBytes)))
	evalAssert(&Context{
		Request: req,
	}, "return $_SERVER['HTTP_CONTENT_TYPE'];", func(val evalAssertionArg) {
		if ToString(val.val) != "application/x-www-form-urlencoded" {
			t.Fatal(ToString(val.val))
		}
	})
}

func Test_SERVER_HTTP_CONTENT_LENGTH(t *testing.T) {
	body := url.Values{}
	body.Set("form_arg", "form_value")
	bodyBytes := body.Encode()
	req := httptest.NewRequest(http.MethodPost, "/hello", bytes.NewBufferString(bodyBytes))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Content-Length", strconv.Itoa(len(bodyBytes)))
	evalAssert(&Context{
		Request: req,
	}, "return $_SERVER['HTTP_CONTENT_LENGTH'];", func(val evalAssertionArg) {
		if ToInt(val.val) != int64(19) {
			t.Fatal(ToInt(val.val))
		}
	})
}

func Test_POST_form_urlencoded(t *testing.T) {
	body := url.Values{}
	body.Set("form_arg", "form_value")
	bodyBytes := body.Encode()
	req := httptest.NewRequest(http.MethodPost, "/hello", bytes.NewBufferString(bodyBytes))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Content-Length", strconv.Itoa(len(bodyBytes)))
	evalAssert(&Context{
		Request: req,
	}, "return $_POST['form_arg'];", func(val evalAssertionArg) {
		if ToString(val.val) != "form_value" {
			t.Fatal(ToString(val.val))
		}
	})
}

func Test_POST_multipart_value(t *testing.T) {
	b := bytes.Buffer{}
	w := multipart.NewWriter(&b)
	fw, err := w.CreateFormField("mp_arg")
	if err != nil {
		t.Fatal(err)
	}
	_, err = fw.Write([]byte("mp_value"))
	if err != nil {
		t.Fatal(err)
	}
	w.Close()
	req := httptest.NewRequest(http.MethodPost, "/hello", &b)
	req.Header.Add("Content-Type", w.FormDataContentType())
	req.Header.Add("Content-Length", strconv.Itoa(b.Len()))
	evalAssert(&Context{
		Request: req,
	}, "return $_POST['mp_arg'];", func(val evalAssertionArg) {
		if ToString(val.val) != "mp_value" {
			t.Fatal(ToString(val.val))
		}
	})
}
