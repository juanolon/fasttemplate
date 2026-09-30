fasttemplate
============

Simple and fast template engine for Go.

Fasttemplate performs only a single task - it substitutes template placeholders
with user-defined values. At high speed :)
Now supporting conditional branches in templates.

Take a look at [quicktemplate](https://github.com/valyala/quicktemplate) if you  need fast yet powerful html template engine.

*Please note that fasttemplate doesn't do any escaping on template values
unlike [html/template](http://golang.org/pkg/html/template/) do. So values
must be properly escaped before passing them to fasttemplate.*

Fasttemplate is faster than [text/template](http://golang.org/pkg/text/template/),
[strings.Replace](http://golang.org/pkg/strings/#Replace),
[strings.Replacer](http://golang.org/pkg/strings/#Replacer)
and [fmt.Fprintf](https://golang.org/pkg/fmt/#Fprintf) on placeholders' substitution.

Below are benchmark results comparing fasttemplate performance to text/template,
strings.Replace, strings.Replacer and fmt.Fprintf:

```
$ go test -bench=. -benchmem
PASS
BenchmarkFmtFprintf-4                   	 2000000	       790 ns/op	       0 B/op	       0 allocs/op
BenchmarkStringsReplace-4               	  500000	      3474 ns/op	    2112 B/op	      14 allocs/op
BenchmarkStringsReplacer-4              	  500000	      2657 ns/op	    2256 B/op	      23 allocs/op
BenchmarkTextTemplate-4                 	  500000	      3333 ns/op	     336 B/op	      19 allocs/op
BenchmarkFastTemplateExecuteFunc-4      	 5000000	       349 ns/op	       0 B/op	       0 allocs/op
BenchmarkFastTemplateExecute-4          	 3000000	       383 ns/op	       0 B/op	       0 allocs/op
BenchmarkFastTemplateExecuteFuncString-4	 3000000	       549 ns/op	     144 B/op	       1 allocs/op
BenchmarkFastTemplateExecuteString-4    	 3000000	       572 ns/op	     144 B/op	       1 allocs/op
BenchmarkFastTemplateExecuteTagFunc-4   	 2000000	       743 ns/op	     144 B/op	       3 allocs/op
```


Docs
====

See http://godoc.org/github.com/valyala/fasttemplate .


Usage
=====

```go
	template := "http://{{host}}/?q={{query}}&foo={{bar}}{{bar}}"
	t := fasttemplate.New(template, "{{", "}}")
	s := t.ExecuteString(map[string]interface{}{
		"host":  "google.com",
		"query": url.QueryEscape("hello=world"),
		"bar":   "foobar",
	})
	fmt.Printf("%s", s)

	// Output:
	// http://google.com/?q=hello%3Dworld&foo=foobarfoobar
```


Conditions
==========

Use `if`, `else` and `end` to include parts of a template depending on values
in the substitution map:

```go
	template := "Hello, {{if registered}}{{name}}{{else}}guest{{end}}!"
	t := fasttemplate.New(template, "{{", "}}")
	s := t.ExecuteString(map[string]interface{}{
		"registered": true,
		"name":       "John",
	})
	fmt.Printf("%s", s)

	// Output:
	// Hello, John!
```

Conditions can be nested and support `==`, `!=`, `<`, `>`, `<=`, `>=`, `!`, `&&`, `||` and
parentheses, with Go operator precedence. For example:
```
{{if enabled && (count >= 2 || admin)}}Hello {{name}}{{else}}Hello guest{{end}}
```

False values:
* Missing or nil values,
* empty strings or byte slices,
* boolean `false`
All other values (including zero and the string `"false"`) are true.
Only tags in the selected branch will execute their TagFunc value.


Conditions may contain variable names, quoted strings, booleans and decimal
numbers. Values aren't converted between types, so `"1" == 1` is false.
Arithmetic and function calls aren't supported.

Conditions are evaluated by `Execute`, `ExecuteStd`, `ExecuteString` and
`ExecuteStringStd`. The `ExecuteFunc` variants treat `if`, `else` and `end` as
ordinary tags and pass them to the callback.

Below are benchmark results for reusable (`Frozen`) and one-shot templates,
with and without a true condition:

```
$ go test -run '^$' -bench '^BenchmarkConditioningOverhead$/(Frozen|OneShot)/(CurrentNoCondition|ConditionTrue)$' -benchmem
BenchmarkConditioningOverhead/Frozen/CurrentNoCondition-16     7045305    168.9 ns/op    0 B/op    0 allocs/op
BenchmarkConditioningOverhead/Frozen/ConditionTrue-16          6184298    192.8 ns/op    0 B/op    0 allocs/op
BenchmarkConditioningOverhead/OneShot/CurrentNoCondition-16    3613738    334.5 ns/op    0 B/op    0 allocs/op
BenchmarkConditioningOverhead/OneShot/ConditionTrue-16         2563436    471.4 ns/op    0 B/op    0 allocs/op
PASS
```

Advanced usage
==============

```go
	template := "Hello, [user]! You won [prize]!!! [foobar]"
	t, err := fasttemplate.NewTemplate(template, "[", "]")
	if err != nil {
		log.Fatalf("unexpected error when parsing template: %s", err)
	}
	s := t.ExecuteFuncString(func(w io.Writer, tag string) (int, error) {
		switch tag {
		case "user":
			return w.Write([]byte("John"))
		case "prize":
			return w.Write([]byte("$100500"))
		default:
			return w.Write([]byte(fmt.Sprintf("[unknown tag %q]", tag)))
		}
	})
	fmt.Printf("%s", s)

	// Output:
	// Hello, John! You won $100500!!! [unknown tag "foobar"]
```
