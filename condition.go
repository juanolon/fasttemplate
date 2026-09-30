package fasttemplate

import (
	"fmt"
	"go/token"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// conditionTag stores a condition and its branch destination.
type conditionTag struct {
	expr *conditionExpr
	jump int
}

type conditionExpr struct {
	op          token.Token
	name        string
	value       conditionValue
	left, right *conditionExpr
}

type conditionValue struct {
	kind    byte
	text    string
	number  float64
	boolean bool
}

func conditionMapValue(v interface{}) conditionValue {
	switch v := v.(type) {
	case nil:
		return conditionValue{}
	case string:
		return conditionValue{kind: 's', text: v}
	case []byte:
		return conditionValue{kind: 's', text: unsafeBytes2String(v)}
	case bool:
		return conditionValue{kind: 'b', boolean: v}
	case int:
		return conditionValue{kind: 'n', number: float64(v)}
	case int8:
		return conditionValue{kind: 'n', number: float64(v)}
	case int16:
		return conditionValue{kind: 'n', number: float64(v)}
	case int32:
		return conditionValue{kind: 'n', number: float64(v)}
	case int64:
		return conditionValue{kind: 'n', number: float64(v)}
	case uint:
		return conditionValue{kind: 'n', number: float64(v)}
	case uint8:
		return conditionValue{kind: 'n', number: float64(v)}
	case uint16:
		return conditionValue{kind: 'n', number: float64(v)}
	case uint32:
		return conditionValue{kind: 'n', number: float64(v)}
	case uint64:
		return conditionValue{kind: 'n', number: float64(v)}
	case float32:
		return conditionValue{kind: 'n', number: float64(v)}
	case float64:
		return conditionValue{kind: 'n', number: v}
	default:
		return conditionValue{kind: 'v'}
	}
}

func (v conditionValue) truth() bool {
	switch v.kind {
	case 0:
		return false
	case 's':
		return len(v.text) != 0
	case 'b':
		return v.boolean
	default:
		return true
	}
}

func (e *conditionExpr) eval(m map[string]interface{}) conditionValue {
	switch e.op {
	case token.IDENT:
		return conditionMapValue(m[e.name])
	case token.ILLEGAL:
		return e.value
	case token.NOT:
		return conditionValue{kind: 'b', boolean: !e.left.eval(m).truth()}
	case token.LAND:
		return conditionValue{kind: 'b', boolean: e.left.eval(m).truth() && e.right.eval(m).truth()}
	case token.LOR:
		return conditionValue{kind: 'b', boolean: e.left.eval(m).truth() || e.right.eval(m).truth()}
	default:
		return compareCondition(e.op, e.left.eval(m), e.right.eval(m))
	}
}

func compareCondition(op token.Token, a, b conditionValue) conditionValue {
	var equal, less, greater bool
	if a.kind == b.kind {
		switch a.kind {
		case 0:
			equal = true
		case 's':
			equal, less, greater = a.text == b.text, a.text < b.text, a.text > b.text
		case 'n':
			equal, less, greater = a.number == b.number, a.number < b.number, a.number > b.number
		case 'b':
			equal = a.boolean == b.boolean
		}
	}
	var result bool
	switch op {
	case token.EQL:
		result = equal
	case token.NEQ:
		result = !equal
	case token.LSS:
		result = less
	case token.GTR:
		result = greater
	case token.LEQ:
		result = less || (equal && (a.kind == 's' || a.kind == 'n'))
	case token.GEQ:
		result = greater || (equal && (a.kind == 's' || a.kind == 'n'))
	}
	return conditionValue{kind: 'b', boolean: result}
}

// conditionParser compiles expressions or evaluates them during streaming.
// Streaming evaluation does not construct an expression tree. Skipped operands
// are checked for syntax without reading map values.
type conditionParser struct {
	s             string
	pos           int
	op            token.Token
	text          string
	value         conditionValue
	err           error
	depth, tokens int
	compile       bool
	m             map[string]interface{}
}

type conditionOperand struct {
	value   conditionValue
	expr    *conditionExpr
	literal bool
}

func parseCondition(s string, m map[string]interface{}, compile, evaluate bool) (conditionOperand, error) {
	p := conditionParser{s: s, m: m, compile: compile}
	p.next()
	v := p.parse(1, evaluate)
	if p.err == nil && p.op != token.EOF {
		p.fail("unexpected token")
	}
	return v, p.err
}

func (p *conditionParser) fail(message string) {
	if p.err == nil {
		p.err = fmt.Errorf("invalid condition %q at byte %d: %s", p.s, p.pos, message)
	}
}

func (p *conditionParser) next() {
	p.op = token.EOF
	if p.err != nil {
		return
	}
	p.tokens++
	if p.tokens > 1024 {
		p.fail("too many tokens")
		return
	}
	for p.pos < len(p.s) && strings.IndexByte(" \t\r\n", p.s[p.pos]) >= 0 {
		p.pos++
	}
	if p.pos == len(p.s) {
		return
	}
	start := p.pos
	c := p.s[p.pos]
	p.pos++
	switch c {
	case '(':
		p.op = token.LPAREN
	case ')':
		p.op = token.RPAREN
	case '+':
		p.op = token.ADD
	case '-':
		p.op = token.SUB
	case '!', '<', '>', '=':
		equal := p.pos < len(p.s) && p.s[p.pos] == '='
		if equal {
			p.pos++
		}
		switch c {
		case '!':
			p.op = token.NOT
			if equal {
				p.op = token.NEQ
			}
		case '<':
			p.op = token.LSS
			if equal {
				p.op = token.LEQ
			}
		case '>':
			p.op = token.GTR
			if equal {
				p.op = token.GEQ
			}
		case '=':
			if !equal {
				p.fail("expected ==")
			}
			p.op = token.EQL
		}
	case '&', '|':
		if p.pos == len(p.s) || p.s[p.pos] != c {
			p.fail("expected && or ||")
			return
		}
		p.pos++
		p.op = token.LAND
		if c == '|' {
			p.op = token.LOR
		}
	case '"', '`':
		for p.pos < len(p.s) {
			n := p.s[p.pos]
			p.pos++
			if n == c {
				s, err := strconv.Unquote(p.s[start:p.pos])
				if err != nil {
					p.fail("invalid string literal")
				}
				p.op, p.value = token.STRING, conditionValue{kind: 's', text: s}
				return
			}
			if n == '\\' && c == '"' && p.pos < len(p.s) {
				p.pos++
			}
		}
		p.fail("unterminated string literal")
	default:
		if c >= '0' && c <= '9' || c == '.' {
			for p.pos < len(p.s) {
				n := p.s[p.pos]
				if n >= '0' && n <= '9' || n == '.' {
					p.pos++
					continue
				}
				if n == 'e' || n == 'E' {
					p.pos++
					if p.pos < len(p.s) && (p.s[p.pos] == '+' || p.s[p.pos] == '-') {
						p.pos++
					}
					continue
				}
				break
			}
			n, err := strconv.ParseFloat(p.s[start:p.pos], 64)
			if err != nil {
				p.fail("invalid decimal number")
			}
			p.op, p.value = token.FLOAT, conditionValue{kind: 'n', number: n}
			return
		}
		p.pos = start
		for p.pos < len(p.s) {
			r, size := utf8.DecodeRuneInString(p.s[p.pos:])
			if r != '_' && !unicode.IsLetter(r) && !(p.pos > start && unicode.IsDigit(r)) {
				break
			}
			p.pos += size
		}
		if p.pos == start {
			p.fail("unsupported character")
			return
		}
		p.op, p.text = token.IDENT, p.s[start:p.pos]
	}
}

func (p *conditionParser) parse(min int, evaluate bool) conditionOperand {
	p.depth++
	defer func() { p.depth-- }()
	if p.depth > 128 {
		p.fail("expression nested too deeply")
		return conditionOperand{}
	}
	left := p.unary(evaluate)
	for p.err == nil {
		op := p.op
		precedence := op.Precedence()
		if precedence < min || precedence > token.EQL.Precedence() {
			break
		}
		p.next()
		run := evaluate
		if op == token.LAND && !left.value.truth() || op == token.LOR && left.value.truth() {
			run = false
		}
		right := p.parse(precedence+1, run)
		if p.compile {
			left = conditionOperand{expr: &conditionExpr{op: op, left: left.expr, right: right.expr}}
		} else if evaluate {
			switch op {
			case token.LAND:
				left.value = conditionValue{kind: 'b', boolean: left.value.truth() && right.value.truth()}
			case token.LOR:
				left.value = conditionValue{kind: 'b', boolean: left.value.truth() || right.value.truth()}
			default:
				left.value = compareCondition(op, left.value, right.value)
			}
			left.literal = false
		} else {
			left = conditionOperand{}
		}
	}
	return left
}

func (p *conditionParser) unary(evaluate bool) conditionOperand {
	if p.err != nil {
		return conditionOperand{}
	}
	op := p.op
	switch op {
	case token.NOT, token.ADD, token.SUB:
		p.next()
		v := p.parse(token.HighestPrec, evaluate)
		if op == token.NOT {
			if p.compile {
				return conditionOperand{expr: &conditionExpr{op: op, left: v.expr}}
			}
			if !evaluate {
				return conditionOperand{}
			}
			return conditionOperand{value: conditionValue{kind: 'b', boolean: !v.value.truth()}}
		}
		if !v.literal || v.value.kind != 'n' {
			p.fail("sign requires a numeric literal")
			return v
		}
		if op == token.SUB {
			v.value.number = -v.value.number
		}
		if p.compile {
			v.expr.value = v.value
		}
		return v
	case token.LPAREN:
		p.next()
		v := p.parse(1, evaluate)
		if p.op != token.RPAREN {
			p.fail("expected )")
			return v
		}
		p.next()
		return v
	case token.IDENT:
		name := p.text
		var v conditionOperand
		if name == "true" || name == "false" {
			v.value = conditionValue{kind: 'b', boolean: name == "true"}
			v.literal = true
			if p.compile {
				v.expr = &conditionExpr{value: v.value}
			}
		} else if p.compile {
			v.expr = &conditionExpr{op: token.IDENT, name: name}
		} else if evaluate {
			v.value = conditionMapValue(p.m[name])
		}
		p.next()
		return v
	case token.STRING, token.FLOAT:
		v := conditionOperand{value: p.value, literal: true}
		if p.compile {
			v.expr = &conditionExpr{value: v.value}
		}
		p.next()
		return v
	default:
		p.fail("expected operand")
		return conditionOperand{}
	}
}

// mayBeControlTag reports whether tag has a possible control-tag prefix.
func mayBeControlTag(tag string) bool {
	return len(tag) > 1 && (tag[0] == 'i' || tag[0] == 'e' || tag[0] <= ' ' || tag[0] >= utf8.RuneSelf)
}

func controlTag(tag string) (string, string) {
	if !mayBeControlTag(tag) {
		return "", ""
	}
	tag = strings.TrimSpace(tag)
	if tag == "else" || tag == "end" || tag == "if" {
		return tag, ""
	}
	if len(tag) > 2 && tag[:2] == "if" && strings.IndexByte(" \t\r\n", tag[2]) >= 0 {
		return "if", tag[3:]
	}
	return "", ""
}

// compileConditionTag compiles tag i and updates the stack of open blocks.
// It allocates space for count control-tag entries when needed.
func (t *Template) compileConditionTag(i, count int, stack []int) ([]int, error) {
	kind, s := controlTag(t.tags[i])
	if kind == "" {
		return stack, nil
	}
	if t.conditions == nil {
		t.conditions = make([]conditionTag, count)
	}
	switch kind {
	case "if":
		v, err := parseCondition(s, nil, true, false)
		if err != nil {
			return stack, err
		}
		t.conditions[i].expr = v.expr
		stack = append(stack, i)
	case "else":
		if len(stack) == 0 || t.conditions[stack[len(stack)-1]].expr == nil {
			return stack, fmt.Errorf("unexpected else tag")
		}
		j := stack[len(stack)-1]
		t.conditions[j].jump = i + 1
		stack[len(stack)-1] = i
	case "end":
		if len(stack) == 0 {
			return stack, fmt.Errorf("unexpected end tag")
		}
		j := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		t.conditions[j].jump = i + 1
		t.conditions[i].jump = i + 1
	}
	return stack, nil
}

type conditionFrame struct {
	parent, matched, otherwise bool
}

// conditionState tracks active branches during streaming execution.
// It stores the first 16 nested blocks inline and allocates space for deeper
// nesting in overflow.
type conditionState struct {
	m        map[string]interface{}
	frames   [16]conditionFrame
	overflow []conditionFrame
	depth    int
	active   bool
}

func (s *conditionState) top() *conditionFrame {
	if s.depth == 0 {
		return nil
	}
	if s.depth <= len(s.frames) {
		return &s.frames[s.depth-1]
	}
	return &s.overflow[len(s.overflow)-1]
}

func (s *conditionState) tag(tag string) (bool, error) {
	kind, expr := controlTag(tag)
	switch kind {
	case "if":
		v, err := parseCondition(expr, s.m, false, s.active)
		if err != nil {
			return true, err
		}
		matched := v.value.truth()
		f := conditionFrame{parent: s.active, matched: matched}
		if s.depth < len(s.frames) {
			s.frames[s.depth] = f
		} else {
			s.overflow = append(s.overflow, f)
		}
		s.depth++
		s.active = s.active && matched
	case "else":
		f := s.top()
		if f == nil || f.otherwise {
			return true, fmt.Errorf("unexpected else tag")
		}
		f.otherwise = true
		s.active = f.parent && !f.matched
	case "end":
		f := s.top()
		if f == nil {
			return true, fmt.Errorf("unexpected end tag")
		}
		s.active = f.parent
		if s.depth > len(s.frames) {
			s.overflow = s.overflow[:len(s.overflow)-1]
		}
		s.depth--
	default:
		return false, nil
	}
	return true, nil
}
