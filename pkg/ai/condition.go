package ai

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

type condExpr interface{ eval(ctx *Context) bool }

type orExpr struct{ l, r condExpr }

func (e orExpr) eval(c *Context) bool { return e.l.eval(c) || e.r.eval(c) }

type andExpr struct{ l, r condExpr }

func (e andExpr) eval(c *Context) bool { return e.l.eval(c) && e.r.eval(c) }

type notExpr struct{ e condExpr }

func (e notExpr) eval(c *Context) bool { return !e.e.eval(c) }

type truthyExpr struct{ field string }

func (e truthyExpr) eval(c *Context) bool {
	v, _ := c.lookup(e.field)
	return v != ""
}

type cmpExpr struct {
	op   string // "==", "!=", "=~", "!~"
	l, r operand
}

func (e cmpExpr) eval(c *Context) bool {
	lv := e.l.value(c)
	rv := e.r.value(c)
	switch e.op {
	case "==":
		return lv == rv
	case "!=":
		return lv != rv
	case "=~":
		return globMatch(rv, lv)
	case "!~":
		return !globMatch(rv, lv)
	}
	return false
}

type operand struct {
	field   string // non-empty when this operand is a field reference
	literal string // used when field is empty
}

func (o operand) value(c *Context) string {
	if o.field != "" {
		v, _ := c.lookup(o.field)
		return v
	}
	return o.literal
}

// globMatch matches value against a glob pattern where * and ? match any
// characters (including path separators). A leading ~ expands to $HOME.
func globMatch(pattern, value string) bool {
	if strings.HasPrefix(pattern, "~") {
		if home, err := os.UserHomeDir(); err == nil {
			pattern = home + strings.TrimPrefix(pattern, "~")
		}
	}
	var b strings.Builder
	b.WriteString("^")
	for _, r := range pattern {
		switch r {
		case '*':
			b.WriteString(".*")
		case '?':
			b.WriteString(".")
		default:
			b.WriteString(regexp.QuoteMeta(string(r)))
		}
	}
	b.WriteString("$")
	re, err := regexp.Compile(b.String())
	if err != nil {
		return false
	}
	return re.MatchString(value)
}

// --- tokenizer ---

type token struct {
	kind string // "ident", "string", "op", "(", ")", "!"
	val  string
	pos  int
}

func tokenize(s string) ([]token, error) {
	var toks []token
	i := 0
	for i < len(s) {
		ch := s[i]
		switch {
		case ch == ' ' || ch == '\t':
			i++
		case ch == '(' || ch == ')':
			toks = append(toks, token{kind: string(ch), val: string(ch), pos: i})
			i++
		case ch == '!':
			if i+1 < len(s) && (s[i+1] == '=' || s[i+1] == '~') {
				toks = append(toks, token{kind: "op", val: s[i : i+2], pos: i})
				i += 2
			} else {
				toks = append(toks, token{kind: "!", val: "!", pos: i})
				i++
			}
		case ch == '=' && i+1 < len(s) && (s[i+1] == '=' || s[i+1] == '~'):
			toks = append(toks, token{kind: "op", val: s[i : i+2], pos: i})
			i += 2
		case ch == '&' && i+1 < len(s) && s[i+1] == '&':
			toks = append(toks, token{kind: "op", val: "&&", pos: i})
			i += 2
		case ch == '|' && i+1 < len(s) && s[i+1] == '|':
			toks = append(toks, token{kind: "op", val: "||", pos: i})
			i += 2
		case ch == '\'' || ch == '"':
			end := strings.IndexByte(s[i+1:], ch)
			if end < 0 {
				return nil, fmt.Errorf("unterminated string at position %d", i)
			}
			toks = append(toks, token{kind: "string", val: s[i+1 : i+1+end], pos: i})
			i += end + 2
		case ch == '=' || ch == '&' || ch == '|' || ch == '~':
			return nil, fmt.Errorf("invalid operator at position %d", i)
		default:
			j := i
			for j < len(s) && (s[j] == '.' || s[j] == '_' || s[j] == '-' ||
				(s[j] >= 'a' && s[j] <= 'z') || (s[j] >= 'A' && s[j] <= 'Z') ||
				(s[j] >= '0' && s[j] <= '9')) {
				j++
			}
			if j == i {
				return nil, fmt.Errorf("unexpected character %q at position %d", ch, i)
			}
			toks = append(toks, token{kind: "ident", val: s[i:j], pos: i})
			i = j
		}
	}
	return toks, nil
}

// --- parser ---

var knownFields = map[string]bool{
	"user": true, "host": true, "arch": true, "os": true, "dir": true,
}

type condParser struct {
	toks []token
	pos  int
}

func parseCondition(s string) (condExpr, error) {
	toks, err := tokenize(s)
	if err != nil {
		return nil, err
	}
	if len(toks) == 0 {
		return nil, fmt.Errorf("empty condition")
	}
	p := &condParser{toks: toks}
	e, err := p.parseOr()
	if err != nil {
		return nil, err
	}
	if p.pos != len(p.toks) {
		return nil, fmt.Errorf("unexpected trailing input at position %d", p.toks[p.pos].pos)
	}
	return e, nil
}

func (p *condParser) peek() *token {
	if p.pos < len(p.toks) {
		return &p.toks[p.pos]
	}
	return nil
}

func (p *condParser) next() *token {
	t := p.peek()
	if t != nil {
		p.pos++
	}
	return t
}

func (p *condParser) parseOr() (condExpr, error) {
	l, err := p.parseAnd()
	if err != nil {
		return nil, err
	}
	for t := p.peek(); t != nil && t.kind == "op" && t.val == "||"; t = p.peek() {
		p.next()
		r, err := p.parseAnd()
		if err != nil {
			return nil, err
		}
		l = orExpr{l, r}
	}
	return l, nil
}

func (p *condParser) parseAnd() (condExpr, error) {
	l, err := p.parseUnary()
	if err != nil {
		return nil, err
	}
	for t := p.peek(); t != nil && t.kind == "op" && t.val == "&&"; t = p.peek() {
		p.next()
		r, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		l = andExpr{l, r}
	}
	return l, nil
}

func (p *condParser) parseUnary() (condExpr, error) {
	t := p.peek()
	if t == nil {
		return nil, fmt.Errorf("unexpected end of condition")
	}
	switch t.kind {
	case "!":
		p.next()
		e, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		return notExpr{e}, nil
	case "(":
		p.next()
		e, err := p.parseOr()
		if err != nil {
			return nil, err
		}
		closing := p.next()
		if closing == nil || closing.kind != ")" {
			return nil, fmt.Errorf("missing closing paren (opened at position %d)", t.pos)
		}
		return e, nil
	}
	return p.parseComparison()
}

func (p *condParser) parseComparison() (condExpr, error) {
	l, err := p.parseOperand()
	if err != nil {
		return nil, err
	}
	t := p.peek()
	if t == nil || t.kind != "op" || (t.val != "==" && t.val != "!=" && t.val != "=~" && t.val != "!~") {
		if l.field == "" {
			return nil, fmt.Errorf("bare string literal is not a valid condition")
		}
		return truthyExpr{l.field}, nil
	}
	op := p.next().val
	r, err := p.parseOperand()
	if err != nil {
		return nil, err
	}
	return cmpExpr{op: op, l: l, r: r}, nil
}

func (p *condParser) parseOperand() (operand, error) {
	t := p.next()
	if t == nil {
		return operand{}, fmt.Errorf("missing operand")
	}
	if t.kind == "string" {
		return operand{literal: t.val}, nil
	}
	if t.kind == "ident" {
		if t.val == "env." {
			return operand{}, fmt.Errorf("env. requires a variable name at position %d", t.pos)
		}
		if !knownFields[t.val] && !strings.HasPrefix(t.val, "env.") {
			return operand{}, fmt.Errorf("unknown field %q at position %d (known: user, host, arch, os, dir, env.NAME)", t.val, t.pos)
		}
		return operand{field: t.val}, nil
	}
	return operand{}, fmt.Errorf("unexpected token %q at position %d", t.val, t.pos)
}
