package fasttemplate

import (
	"io"
	"testing"
)

type conditionBenchmarkWriter int

func (w *conditionBenchmarkWriter) Write(p []byte) (int, error) {
	*w += conditionBenchmarkWriter(len(p))
	return len(p), nil
}

func BenchmarkConditioningOverhead(b *testing.B) {
	m := map[string]interface{}{
		"cb":      []byte("1234"),
		"width":   []byte("1232"),
		"height":  []byte("123"),
		"timeout": []byte("123123"),
		"uid":     []byte("aaasdf"),
		"subid":   []byte("asdfds"),
		"ref":     []byte("http://google.com/aaa/bbb/ccc"),
		"enabled": true,
	}
	conditionSource := "{{if enabled}}" + source + "{{end}}"
	plain := New(source, "{{", "}}")
	conditional := New(conditionSource, "{{", "}}")

	b.Run("Frozen", func(b *testing.B) {
		b.Run("CurrentNoCondition", func(b *testing.B) {
			benchmarkCondition(b, func(w io.Writer) (int64, error) {
				return plain.Execute(w, m)
			})
		})
		b.Run("ConditionTrue", func(b *testing.B) {
			benchmarkCondition(b, func(w io.Writer) (int64, error) {
				return conditional.Execute(w, m)
			})
		})
	})

	b.Run("OneShot", func(b *testing.B) {
		b.Run("CurrentNoCondition", func(b *testing.B) {
			benchmarkCondition(b, func(w io.Writer) (int64, error) {
				return Execute(source, "{{", "}}", w, m)
			})
		})
		b.Run("ConditionTrue", func(b *testing.B) {
			benchmarkCondition(b, func(w io.Writer) (int64, error) {
				return Execute(conditionSource, "{{", "}}", w, m)
			})
		})
	})
}

func benchmarkCondition(b *testing.B, execute func(io.Writer) (int64, error)) {
	var w conditionBenchmarkWriter
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w = 0
		if _, err := execute(&w); err != nil {
			b.Fatalf("unexpected error: %s", err)
		}
	}
}
