package fasttemplate

import (
	"bytes"
	"io"
	"testing"
)

func TestCondition(t *testing.T) {
	m := map[string]interface{}{
		"emptyString": "",
		"emptyBytes":  []byte{},
		"false":       false,
		"true":        true,
		"zero":        0,
		"name":        "John",
	}

	testCondition(t, "{{if missing}}true{{else}}false{{end}}", "false", m)
	testCondition(t, "{{if emptyString}}true{{else}}false{{end}}", "false", m)
	testCondition(t, "{{if emptyBytes}}true{{else}}false{{end}}", "false", m)
	testCondition(t, "{{if false}}true{{else}}false{{end}}", "false", m)
	testCondition(t, "{{if true}}Hello, {{name}}!{{else}}Hello, guest!{{end}}", "Hello, John!", m)
	testCondition(t, "{{if zero}}true{{else}}false{{end}}", "true", m)
	testCondition(t, "{{if missing}}hidden{{end}}visible", "visible", m)
}

func TestConditionExpression(t *testing.T) {
	m := map[string]interface{}{
		"enabled": true,
		"admin":   false,
		"count":   3,
		"name":    "John",
	}

	testCondition(t, "{{if enabled && (count >= 2 || admin)}}yes{{else}}no{{end}}", "yes", m)
	testCondition(t, "{{if !enabled || count < 3}}yes{{else}}no{{end}}", "no", m)
	testCondition(t, `{{if name == "John"}}yes{{else}}no{{end}}`, "yes", m)
	testCondition(t, `{{if name != "John"}}yes{{else}}no{{end}}`, "no", m)
	testCondition(t, `{{if "1" == 1}}yes{{else}}no{{end}}`, "no", m)
}

func TestNestedCondition(t *testing.T) {
	m := map[string]interface{}{
		"registered": true,
		"admin":      false,
	}
	template := "{{if registered}}registered {{if admin}}admin{{else}}user{{end}}{{else}}guest{{end}}"
	testCondition(t, template, "registered user", m)
}

func TestConditionStd(t *testing.T) {
	template := "{{if enabled}}{{unknown}}{{else}}hidden{{end}}"
	m := map[string]interface{}{"enabled": true}

	s := ExecuteStringStd(template, "{{", "}}", m)
	result := "{{unknown}}"
	if s != result {
		t.Fatalf("unexpected template value %q. Expected %q", s, result)
	}

	tpl := New(template, "{{", "}}")
	s = tpl.ExecuteStringStd(m)
	if s != result {
		t.Fatalf("unexpected template value %q. Expected %q", s, result)
	}
}

func TestConditionDoesNotExecuteUnselectedTag(t *testing.T) {
	called := false
	m := map[string]interface{}{
		"enabled": false,
		"value": TagFunc(func(w io.Writer, tag string) (int, error) {
			called = true
			return w.Write([]byte("unexpected"))
		}),
	}

	testCondition(t, "{{if enabled}}{{value}}{{else}}disabled{{end}}", "disabled", m)
	if called {
		t.Fatalf("TagFunc in unselected branch was called")
	}
}

func TestConditionErrors(t *testing.T) {
	templates := []string{
		"{{if enabled}}missing end",
		"{{else}}",
		"{{end}}",
		"{{if enabled}}{{else}}{{else}}{{end}}",
		"{{if enabled + 1}}{{end}}",
		"{{if enabled &&}}{{end}}",
	}
	for _, template := range templates {
		if _, err := NewTemplate(template, "{{", "}}"); err == nil {
			t.Fatalf("expected error for template %q", template)
		}
	}
}

func TestExecuteConditionError(t *testing.T) {
	var bb bytes.Buffer
	n, err := Execute("foo{{if enabled &&}}bar{{end}}", "{{", "}}", &bb, nil)
	if err == nil {
		t.Fatalf("expected non-nil error. got nil")
	}
	if n != 3 || bb.String() != "foo" {
		t.Fatalf("unexpected output %q with length %d", bb.String(), n)
	}
}

func TestExecuteFuncConditionTags(t *testing.T) {
	template := "{{if enabled}}yes{{else}}no{{end}}"
	s := ExecuteFuncString(template, "{{", "}}", func(w io.Writer, tag string) (int, error) {
		return w.Write([]byte("[" + tag + "]"))
	})
	result := "[if enabled]yes[else]no[end]"
	if s != result {
		t.Fatalf("unexpected template value %q. Expected %q", s, result)
	}
}

func TestDeepNestedCondition(t *testing.T) {
	template := ""
	for i := 0; i < 17; i++ {
		template += "{{if enabled}}"
	}
	template += "yes"
	for i := 0; i < 17; i++ {
		template += "{{end}}"
	}
	testCondition(t, template, "yes", map[string]interface{}{"enabled": true})
}

func testCondition(t *testing.T, template, expectedOutput string, m map[string]interface{}) {
	output := ExecuteString(template, "{{", "}}", m)
	if output != expectedOutput {
		t.Fatalf("unexpected output for template=%q: %q. Expected %q", template, output, expectedOutput)
	}

	tpl := New(template, "{{", "}}")
	output = tpl.ExecuteString(m)
	if output != expectedOutput {
		t.Fatalf("unexpected output for frozen template=%q: %q. Expected %q", template, output, expectedOutput)
	}
}
