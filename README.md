Original library is at: https://github.com/taowen/go-php7

This fork is for:

* Support PHP7 only, so that we can embrace the value semantics of zval
* Fix memory leaks by using zval instead of engine_value
* Use golang http request as PHP input
* Use golang http response writer as PHP output

# PHP bindings for Go [![API Documentation][godoc-svg]][godoc-url] [![MIT License][license-svg]][license-url]

This package implements support for executing PHP scripts, exporting Go variables for use in PHP contexts, attaching Go method receivers as PHP classes and returning PHP variables for use in Go contexts.

Only PHP 7.x series is supported.

## Usage

### Basic

Executing a script is very simple:

```go
package main

import (
    php "github.com/taowen/go-php7"
    "os"
)

func main() {
    engine, _ := php.New()

	context := &Context{Output: os.Stdout}
    engine.RequestStartup(context)
    defer engine.RequestShutdown(context)

    context.Exec("index.php")
}
```

The above will execute script file `index.php` located in the current folder and will write any output 
to the `io.Writer` assigned to `Context.Output` (in this case, the standard output).

### Binding and returning variables

The following example demonstrates binding a Go variable to the running PHP context, and returning a PHP variable for use in Go:

```go
package main

import (
    "fmt"
    php "github.com/taowen/go-php7"
)

func main() {
    engine, _ := php.New()
	context := &Context{}
    engine.RequestStartup(context)
    defer engine.RequestShutdown(context)

    var str string = "Hello"
    context.Bind("var", str)

    val, _ := context.Eval("return $var.' World';")
    fmt.Printf("%s", val.Interface())
    // Prints 'Hello World' back to the user.
}
```

A string value "Hello" is attached using `Context.Bind` under a name `var` (available in PHP as `$var`). 
A script is executed inline using `Context.Eval`, combinding the attached value with a PHP string and returning it to the user.

Finally, the value is returned as an `interface{}` using `Value.Interface()` (one could also use `Value.String()`, 
though the both are equivalent in this case).

## License

All code in this repository is covered by the terms of the MIT License, the full text of which can be found in the LICENSE file.

[godoc-url]: https://godoc.org/github.com/taowen/go-php7
[godoc-svg]: https://godoc.org/github.com/taowen/go-php7?status.svg

[license-url]: https://github.com/taowen/go-php7/blob/master/LICENSE
[license-svg]: https://img.shields.io/badge/license-MIT-blue.svg

[Context.Exec]: https://godoc.org/github.com/taowen/go-php7/engine#Context.Exec
[Context.Eval]: https://godoc.org/github.com/taowen/go-php7/engine#Context.Eval
[NewValue]:     https://godoc.org/github.com/taowen/go-php7/engine#NewValue
[NewReceiver]:  https://godoc.org/github.com/taowen/go-php7/engine#NewReceiver
