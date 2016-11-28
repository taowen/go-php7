package php

import (
	"testing"
	"github.com/deuill/go-php/engine"
	"fmt"
	"os"
	"net/http/httptest"
	"io/ioutil"
	"bytes"
)

func Test_exec(t *testing.T) {
	engine.Initialize()
	ctx := &engine.Context{
		Output: os.Stdout,
	}
	err := engine.RequestStartup(ctx)
	if err != nil {
		fmt.Println(err)
	}
	defer engine.RequestShutdown(ctx)
	err = ctx.Exec("/tmp/index.php")
	if err != nil {
		fmt.Println(err)
	}
}

func Test_eval(t *testing.T) {
	engine.Initialize()
	ctx := &engine.Context{}
	err := engine.RequestStartup(ctx)
	if err != nil {
		fmt.Println(err)
	}
	defer engine.RequestShutdown(ctx)
	val, err := ctx.Eval("return 'hello';")
	if err != nil {
		fmt.Println(err)
	}
	defer engine.DestroyValue(val)
	if engine.ToString(val) != "hello" {
		t.FailNow()
	}
}

func Test_argument(t *testing.T) {
	engine.Initialize()
	ctx := &engine.Context{}
	err := engine.RequestStartup(ctx)
	if err != nil {
		fmt.Println(err)
	}
	defer engine.RequestShutdown(ctx)
	err = ctx.Bind("greeting", "hello")
	if err != nil {
		fmt.Println(err)
	}
	val, err := ctx.Eval("return $greeting;")
	if err != nil {
		fmt.Println(err)
	}
	defer engine.DestroyValue(val)
	if engine.ToString(val) != "hello" {
		t.FailNow()
	}
}

type greetingProvider struct {
	greeting string
}

func (provider *greetingProvider) GetGreeting() string {
	return provider.greeting
}

func newGreetingProvider(args []interface{}) interface{} {
	return &greetingProvider{
		greeting: args[0].(string),
	}
}

func Test_callback(t *testing.T) {
	engine.Initialize()
	ctx := &engine.Context{}
	err := engine.RequestStartup(ctx)
	if err != nil {
		fmt.Println(err)
	}
	defer engine.RequestShutdown(ctx)
	err = engine.Define("GreetingProvider", newGreetingProvider)
	if err != nil {
		fmt.Println(err)
	}
	val, err := ctx.Eval(`
	$greetingProvider = new GreetingProvider('hello');
	return $greetingProvider->GetGreeting();`)
	if err != nil {
		fmt.Println(err)
	}
	defer engine.DestroyValue(val)
	if engine.ToString(val) != "hello" {
		t.FailNow()
	}
}

func Test_log(t *testing.T) {
	engine.PHP_INI_PATH_OVERRIDE = "/tmp/php.ini"
	engine.Initialize()
	ctx := &engine.Context{
		Log: os.Stderr,
	}
	err := engine.RequestStartup(ctx)
	if err != nil {
		fmt.Println(err)
	}
	defer engine.RequestShutdown(ctx)
	_, err = ctx.Eval("error_log('hello', 4); trigger_error('sent from golang', E_USER_ERROR);")
	if err != nil {
		fmt.Println(err)
	}
}

func Test_http(t *testing.T) {
	engine.Initialize()
	recorder := httptest.NewRecorder()
	ctx := &engine.Context{
		Request: httptest.NewRequest("GET", "/hello", nil),
		ResponseWriter: recorder,
	}
	err := engine.RequestStartup(ctx)
	if err != nil {
		fmt.Println(err)
	}
	defer engine.RequestShutdown(ctx)
	_, err = ctx.Eval("echo($_SERVER['REQUEST_URI']);")
	if err != nil {
		fmt.Println(err)
	}
	body, err := ioutil.ReadAll(recorder.Result().Body)
	if err != nil {
		fmt.Println(err)
	}
	if string(body) != "/hello" {
		t.FailNow()
	}
}


func Test_fastcgi_finish_reqeust(t *testing.T) {
	engine.Initialize()
	buffer := &bytes.Buffer{}
	ctx := &engine.Context{
		Output: buffer,
	}
	err := engine.RequestStartup(ctx)
	if err != nil {
		fmt.Println(err)
	}
	defer engine.RequestShutdown(ctx)
	ctx.Eval("ob_start(); echo ('hello');")
	if buffer.String() != "" {
		t.FailNow()
	}
	ctx.Eval("fastcgi_finish_request();")
	if buffer.String() != "hello" {
		t.FailNow()
	}
}
