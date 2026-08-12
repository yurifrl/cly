# AI Provider Conditions Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the map-shaped `ai.providers` config with a list of named entries supporting OpenAI-compatible endpoints, `condition` expressions, `weight`, and `default`, with always-on selection logging.

**Architecture:** All selection logic lives in `pkg/ai` (core infra, not a module). A tiny built-in expression evaluator (no new deps) parses conditions at config resolution; a pure selection function picks an entry and returns a decision record; `resolve()` in `ai.go` switches from map lookup to list selection. A new `cly ai status` command prints the decision.

**Tech Stack:** Go, Cobra, testify. No new dependencies.

## Global Constraints

- Spec: `docs/superpowers/specs/2026-08-12-ai-provider-conditions-design.md`
- Old `ai.provider` key and map-shaped `ai.providers` are REMOVED — no compat shim.
- No new third-party dependencies.
- Never print resolved secret values anywhere — env var NAMES and set/unset only.
- Verbose dump goes to **stderr**, stdout stays clean for piping.
- Config errors fail fast with entry name + message; `HasAPIKey` must never error (returns false).
- Tests: `go test ./pkg/ai/... ./modules/ai/...` must pass; `go build ./...` must pass.

## Locked Interfaces (used across tasks)

```go
// pkg/ai/condition.go
type condExpr interface{ eval(ctx *Context) bool }
func parseCondition(s string) (condExpr, error)

// pkg/ai/context.go
type Context struct{ User, Host, Arch, OS, Dir string }
func buildContext() *Context
func (c *Context) lookup(field string) (string, bool) // user|host|arch|os|dir|env.NAME

// pkg/ai/providers.go
type Entry struct {
	Name      string
	Provider  string
	Model     string
	APIKey    string
	APIKeyEnv string
	BaseURL   string
	Weight    int
	Condition string
	Default   bool
	cond      condExpr
}
func parseProviders(global map[string]interface{}) ([]Entry, error)

// pkg/ai/select.go
type EntryResult struct {
	Name      string
	Condition string
	Matched   bool
	Weight    int
	Note      string // "picked", "default fallback", "first entry fallback", ""
}
type Decision struct {
	Context *Context
	EnvRefs map[string]bool // env var names referenced by any condition -> set?
	Entries []EntryResult
	Picked  string
	Reason  string // "condition match", "default", "first entry"
}
func selectProvider(entries []Entry, ctx *Context) (Entry, *Decision)

// pkg/ai/ai.go (modified)
func LastDecision() *Decision          // most recent selection, nil before first
func LoadConfigWith(override map[string]interface{}) *Resolved        // unchanged sig; nil on config error
func NewClientWith(override map[string]interface{}) (llm.Client, error) // surfaces config error
```

---

### Task 1: Condition parser + evaluator

**Files:**
- Create: `pkg/ai/condition.go`
- Test: `pkg/ai/condition_test.go`

**Interfaces:**
- Consumes: `Context` (defined inline in this task's test via a stub; real one lands in Task 3 — the stub type must match: `type Context struct{ User, Host, Arch, OS, Dir string }` with `lookup(field string) (string, bool)`).
- Produces: `condExpr`, `parseCondition(s string) (condExpr, error)`.

Note: `Context` lives in `pkg/ai/context.go` (Task 3). To keep Task 1 self-contained, define `Context` + `lookup` in `condition.go` in this task; Task 3 MOVES it to `context.go` unchanged and adds `buildContext`.

- [ ] **Step 1: Write the failing test**

`pkg/ai/condition_test.go`:

```go
package ai

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testCtx() *Context {
	return &Context{
		User: "yuri",
		Host: "yuris-mac",
		Arch: "arm64",
		OS:   "darwin",
		Dir:  "/Users/yuri/Workdir/Yuri/cly",
	}
}

func TestConditionEval(t *testing.T) {
	t.Setenv("CLY_TEST_COND", "1")
	tests := []struct {
		expr string
		want bool
	}{
		{`user == "yuri"`, true},
		{`user == "root"`, false},
		{`user != "root"`, true},
		{`arch == "arm64" && os == "darwin"`, true},
		{`arch == "amd64" || os == "darwin"`, true},
		{`arch == "amd64" || os == "linux"`, false},
		{`!(arch == "amd64")`, true},
		{`(user == "yuri" && arch == "arm64") || host == "x"`, true},
		{`user == "yuri" && (arch == "amd64" || host == "x")`, false},
		// glob on strings; * crosses path separators
		{`dir =~ "/Users/yuri/Workdir/Yuri/*"`, true},
		{`dir =~ "/Users/yuri/*"`, true},
		{`dir =~ "/opt/*"`, false},
		{`dir !~ "/opt/*"`, true},
		// ~ expands to home dir in patterns
		{`dir =~ "~/Workdir/Yuri/*"`, true},
		// env lookup
		{`env.CLY_TEST_COND == "1"`, true},
		{`env.CLY_TEST_COND`, true},          // truthy when set non-empty
		{`env.CLY_DEFINITELY_UNSET`, false},  // truthy check on unset
		{`env.CLY_DEFINITELY_UNSET == ""`, true},
		// precedence: && binds tighter than ||
		{`host == "x" || user == "yuri" && arch == "arm64"`, true},
	}
	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			expr, err := parseCondition(tt.expr)
			require.NoError(t, err)
			assert.Equal(t, tt.want, expr.eval(testCtx()))
		})
	}
}

func TestConditionParseErrors(t *testing.T) {
	for _, expr := range []string{
		`user ==`,               // missing operand
		`user === "x"`,          // bad operator
		`foo == "x"`,            // unknown field
		`env. == "x"`,           // empty env name
		`(user == "x"`,          // unbalanced paren
		`user == "x" &&`,        // dangling operator
		`user == "x" "y"`,       // trailing tokens
		`dir ~ "x"`,             // =~ is the only match op
	} {
		t.Run(expr, func(t *testing.T) {
			_, err := parseCondition(expr)
			assert.Error(t, err, expr)
		})
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/ai/ -run TestCondition -v`
Expected: FAIL — `undefined: parseCondition` / `undefined: Context`.

- [ ] **Step 3: Implement**

`pkg/ai/condition.go` — tokenizer + recursive-descent parser + evaluator. Grammar:

```
expr    := or
or      := and ("||" and)*
and     := unary ("&&" unary)*
unary   := "!" unary | "(" expr ")" | comparison
comparison := operand (("==" | "!=" | "=~" | "!~") operand)?   // bare operand = truthy
operand := field | string_literal
field   := "user" | "host" | "arch" | "os" | "dir" | "env." NAME
```

Key semantics:
- `=~` / `!~`: glob where `*` matches ANY characters including `/` (translate pattern to regex: escape regex metachars except `*`/`?`, then `*`→`.*`, `?`→`.`, anchor with `^...$`). A leading `~` in the pattern expands to the current user's home dir.
- Bare field operand: truthy when its value is non-empty.
- Unknown field or syntax error → error with position info.

```go
package ai

import (
	"fmt"
	"os"
	"os/user"
	"regexp"
	"strings"
)

// Context is the runtime environment conditions evaluate against.
type Context struct {
	User, Host, Arch, OS, Dir string
}

// lookup resolves a condition field to its value. env.NAME reads the
// process environment. bool reports whether the field name is known.
func (c *Context) lookup(field string) (string, bool) {
	switch field {
	case "user":
		return c.User, true
	case "host":
		return c.Host, true
	case "arch":
		return c.Arch, true
	case "os":
		return c.OS, true
	case "dir":
		return c.Dir, true
	}
	if strings.HasPrefix(field, "env.") && len(field) > 4 {
		return os.Getenv(field[4:]), true
	}
	return "", false
}

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
	op    string // "==", "!=", "=~", "!~"
	l, r  operand
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
		case ch == '(' || ch == ')' || ch == '!':
			if ch == '!' && i+1 < len(s) && (s[i+1] == '=' || s[i+1] == '~') {
				toks = append(toks, token{kind: "op", val: s[i : i+2], pos: i})
				i += 2
			} else {
				toks = append(toks, token{kind: string(ch), val: string(ch), pos: i})
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
		// bare operand: truthy check; only valid on field refs
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

func (p *condParser) parseOperand() (condExprOperand, error) {
	t := p.next()
	if t == nil {
		return condExprOperand{}, fmt.Errorf("missing operand")
	}
	if t.kind == "string" {
		return condExprOperand{literal: t.val}, nil
	}
	if t.kind == "ident" {
		if !knownFields[t.val] && !strings.HasPrefix(t.val, "env.") {
			return condExprOperand{}, fmt.Errorf("unknown field %q at position %d (known: user, host, arch, os, dir, env.NAME)", t.val, t.pos)
		}
		if t.val == "env." {
			return condExprOperand{}, fmt.Errorf("env. requires a variable name at position %d", t.pos)
		}
		return condExprOperand{field: t.val}, nil
	}
	return condExprOperand{}, fmt.Errorf("unexpected token %q at position %d", t.val, t.pos)
}

// condExprOperand is exported within the package as operand for cmpExpr.
type condExprOperand = operand

// ensure os/user import is used (home dir fallback for ~ expansion tests)
var _ = user.Current
```

Note for the implementer: `condExprOperand`/`operand` naming — keep ONE name (`operand`) and drop the alias if it adds confusion; tests only touch `parseCondition` and `eval`.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./pkg/ai/ -run TestCondition -v`
Expected: PASS (all table rows).

- [ ] **Step 5: Commit**

```bash
git add pkg/ai/condition.go pkg/ai/condition_test.go
git commit -m "ai: add condition expression parser and evaluator"
```

---

### Task 2: Provider list parsing + validation

**Files:**
- Create: `pkg/ai/providers.go`
- Test: `pkg/ai/providers_test.go`

**Interfaces:**
- Consumes: `parseCondition` (Task 1).
- Produces: `Entry`, `parseProviders(global map[string]interface{}) ([]Entry, error)`.

- [ ] **Step 1: Write the failing test**

`pkg/ai/providers_test.go`:

```go
package ai

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseProviders(t *testing.T) {
	global := map[string]interface{}{
		"providers": []interface{}{
			map[string]interface{}{
				"name":      "aihub",
				"provider":  "openai",
				"base_url":  "https://aihub-gateway.fbrai.dev/v1",
				"api_key":   "$AIHUB_API_KEY",
				"model":     "aihub/claude-sonnet-5",
				"weight":    10,
				"condition": `user == "yuri" && dir =~ "~/Workdir/Yuri/*"`,
				"default":   true,
			},
			map[string]interface{}{
				"name":     "bedrock",
				"provider": "bedrock",
				"model":    "us.anthropic.claude-sonnet-4-5-20250929-v1:0",
			},
		},
	}
	entries, err := parseProviders(global)
	require.NoError(t, err)
	require.Len(t, entries, 2)

	assert.Equal(t, "aihub", entries[0].Name)
	assert.Equal(t, "openai", entries[0].Provider)
	assert.Equal(t, "https://aihub-gateway.fbrai.dev/v1", entries[0].BaseURL)
	assert.Equal(t, "", entries[0].APIKey)
	assert.Equal(t, "AIHUB_API_KEY", entries[0].APIKeyEnv) // $ENV expanded to env name
	assert.Equal(t, "aihub/claude-sonnet-5", entries[0].Model)
	assert.Equal(t, 10, entries[0].Weight)
	assert.True(t, entries[0].Default)
	require.NotNil(t, entries[0].cond)

	assert.Equal(t, "bedrock", entries[1].Name)
	assert.Equal(t, 0, entries[1].Weight)
	assert.False(t, entries[1].Default)
	assert.Nil(t, entries[1].cond)
}

func TestParseProvidersDefaults(t *testing.T) {
	// entry with no provider type defaults to anthropic; no api_key gets
	// the provider's conventional env var name
	global := map[string]interface{}{
		"providers": []interface{}{
			map[string]interface{}{"name": "x", "model": "m"},
		},
	}
	entries, err := parseProviders(global)
	require.NoError(t, err)
	assert.Equal(t, "anthropic", entries[0].Provider)
	assert.Equal(t, "ANTHROPIC_API_KEY", entries[0].APIKeyEnv)
}

func TestParseProvidersErrors(t *testing.T) {
	tests := []struct {
		name   string
		global map[string]interface{}
	}{
		{"empty list", map[string]interface{}{"providers": []interface{}{}}},
		{"missing name", map[string]interface{}{"providers": []interface{}{
			map[string]interface{}{"model": "m"}}}},
		{"duplicate name", map[string]interface{}{"providers": []interface{}{
			map[string]interface{}{"name": "x"},
			map[string]interface{}{"name": "x"}}}},
		{"bad condition", map[string]interface{}{"providers": []interface{}{
			map[string]interface{}{"name": "x", "condition": "user =="}}}},
		{"unknown condition field", map[string]interface{}{"providers": []interface{}{
			map[string]interface{}{"name": "x", "condition": `foo == "bar"`}}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseProviders(tt.global)
			assert.Error(t, err)
		})
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/ai/ -run TestParseProviders -v`
Expected: FAIL — `undefined: parseProviders`.

- [ ] **Step 3: Implement**

`pkg/ai/providers.go`:

```go
package ai

import (
	"fmt"
	"strings"
)

// Entry is one named provider in the ai.providers list.
type Entry struct {
	Name      string
	Provider  string
	Model     string
	APIKey    string
	APIKeyEnv string
	BaseURL   string
	Weight    int
	Condition string
	Default   bool
	cond      condExpr
}

// parseProviders validates the ai.providers list from raw config.
// providerEnv comes from ai.go (provider -> conventional env var name).
func parseProviders(global map[string]interface{}) ([]Entry, error) {
	raw, ok := global["providers"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("ai.providers must be a list of named entries (map form was removed; see docs/superpowers/specs/2026-08-12-ai-provider-conditions-design.md)")
	}
	if len(raw) == 0 {
		return nil, fmt.Errorf("ai.providers is empty: define at least one entry")
	}
	seen := map[string]bool{}
	entries := make([]Entry, 0, len(raw))
	for i, item := range raw {
		m, ok := item.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("ai.providers[%d]: must be a map", i)
		}
		e := Entry{Provider: defaultProvider}
		e.Name, _ = m["name"].(string)
		if e.Name == "" {
			return nil, fmt.Errorf("ai.providers[%d]: name is required", i)
		}
		if seen[e.Name] {
			return nil, fmt.Errorf("ai.providers[%d]: duplicate name %q", i, e.Name)
		}
		seen[e.Name] = true
		if v, ok := m["provider"].(string); ok && v != "" {
			e.Provider = v
		}
		e.Model, _ = m["model"].(string)
		e.BaseURL, _ = m["base_url"].(string)
		if v, ok := m["weight"].(int); ok {
			e.Weight = v
		}
		e.Default, _ = m["default"].(bool)
		e.Condition, _ = m["condition"].(string)
		if e.Condition != "" {
			cond, err := parseCondition(e.Condition)
			if err != nil {
				return nil, fmt.Errorf("ai.providers[%d] (%s): invalid condition: %w", i, e.Name, err)
			}
			e.cond = cond
		}
		if v, ok := m["api_key"].(string); ok && v != "" {
			setKeyOrEnvStr(v, &e)
		}
		if v, ok := m["api_key_env"].(string); ok && v != "" {
			e.APIKey = ""
			e.APIKeyEnv = v
		}
		if e.APIKey == "" && e.APIKeyEnv == "" {
			e.APIKeyEnv = providerEnv[e.Provider]
		}
		entries = append(entries, e)
	}
	return entries, nil
}

// setKeyOrEnvStr mirrors setKeyOrEnv in ai.go but targets Entry fields.
func setKeyOrEnvStr(s string, e *Entry) {
	s = strings.TrimSpace(s)
	switch {
	case strings.HasPrefix(s, "${") && strings.HasSuffix(s, "}"):
		e.APIKey = ""
		e.APIKeyEnv = s[2 : len(s)-1]
	case strings.HasPrefix(s, "$") && len(s) > 1:
		e.APIKey = ""
		e.APIKeyEnv = s[1:]
	default:
		e.APIKey = s
		e.APIKeyEnv = ""
	}
}
```

Note for the implementer: YAML numbers may decode as `int` OR `float64` depending on the loader — handle weight as: try `int`, then `float64` (truncate). Check what viper produces (`m["weight"].(int)`); if the test fails with weight 0, add the float64 case.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./pkg/ai/ -run TestParseProviders -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add pkg/ai/providers.go pkg/ai/providers_test.go
git commit -m "ai: parse named provider list with validation"
```

---

### Task 3: Runtime context + selection algorithm

**Files:**
- Create: `pkg/ai/context.go`, `pkg/ai/select.go`
- Test: `pkg/ai/select_test.go`
- Modify: `pkg/ai/condition.go` (move `Context` + `lookup` out to `context.go`)

**Interfaces:**
- Consumes: `Entry`, `condExpr`, `Context` (Tasks 1-2).
- Produces: `buildContext() *Context`, `selectProvider(entries []Entry, ctx *Context) (Entry, *Decision)`, `Decision`, `EntryResult`.

- [ ] **Step 1: Write the failing test**

`pkg/ai/select_test.go`:

```go
package ai

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mkEntry(name string, weight int, def bool, cond string) Entry {
	e := Entry{Name: name, Provider: "openai", Weight: weight, Default: def, Condition: cond}
	if cond != "" {
		c, err := parseCondition(cond)
		if err != nil {
			panic(err)
		}
		e.cond = c
	}
	return e
}

func selCtx() *Context {
	return &Context{User: "yuri", Host: "mac", Arch: "arm64", OS: "darwin", Dir: "/Users/yuri/Workdir/Yuri/cly"}
}

func TestSelectConditionMatch(t *testing.T) {
	entries := []Entry{
		mkEntry("first", 0, true, ""),
		mkEntry("matched", 0, false, `user == "yuri"`),
	}
	e, d := selectProvider(entries, selCtx())
	assert.Equal(t, "matched", e.Name)
	assert.Equal(t, "matched", d.Picked)
	assert.Equal(t, "condition match", d.Reason)
	assert.Equal(t, "picked", d.Entries[1].Note)
	assert.False(t, d.Entries[0].Matched)
}

func TestSelectWeightWins(t *testing.T) {
	entries := []Entry{
		mkEntry("low", 1, false, `user == "yuri"`),
		mkEntry("high", 10, false, `user == "yuri"`),
	}
	e, d := selectProvider(entries, selCtx())
	assert.Equal(t, "high", e.Name)
	assert.Equal(t, "condition match", d.Reason)
}

func TestSelectWeightTiePicksFirst(t *testing.T) {
	entries := []Entry{
		mkEntry("a", 5, false, `user == "yuri"`),
		mkEntry("b", 5, false, `user == "yuri"`),
	}
	e, _ := selectProvider(entries, selCtx())
	assert.Equal(t, "a", e.Name)
}

func TestSelectZeroWeightTiePicksFirst(t *testing.T) {
	entries := []Entry{
		mkEntry("a", 0, false, `user == "yuri"`),
		mkEntry("b", 0, false, `user == "yuri"`),
	}
	e, _ := selectProvider(entries, selCtx())
	assert.Equal(t, "a", e.Name)
}

func TestSelectDefaultFallback(t *testing.T) {
	entries := []Entry{
		mkEntry("nomatch", 0, false, `user == "root"`),
		mkEntry("fallback", 0, true, ""),
		mkEntry("last", 0, false, ""),
	}
	e, d := selectProvider(entries, selCtx())
	assert.Equal(t, "fallback", e.Name)
	assert.Equal(t, "default", d.Reason)
	assert.Equal(t, "default fallback", d.Entries[1].Note)
}

func TestSelectMultipleDefaultsPicksFirst(t *testing.T) {
	entries := []Entry{
		mkEntry("d1", 0, true, ""),
		mkEntry("d2", 0, true, ""),
	}
	e, _ := selectProvider(entries, selCtx())
	assert.Equal(t, "d1", e.Name)
}

func TestSelectNoDefaultPicksFirstEntry(t *testing.T) {
	entries := []Entry{
		mkEntry("first", 0, false, `user == "root"`),
		mkEntry("second", 0, false, ""),
	}
	e, d := selectProvider(entries, selCtx())
	assert.Equal(t, "first", e.Name)
	assert.Equal(t, "first entry", d.Reason)
	assert.Equal(t, "first entry fallback", d.Entries[0].Note)
}

func TestSelectEntryWithoutConditionNeverMatches(t *testing.T) {
	entries := []Entry{
		mkEntry("nocond", 100, false, ""), // weight irrelevant: no condition = no match
		mkEntry("cond", 0, false, `user == "yuri"`),
	}
	e, _ := selectProvider(entries, selCtx())
	assert.Equal(t, "cond", e.Name)
}

func TestDecisionRecordsEnvRefs(t *testing.T) {
	t.Setenv("CLY_SEL_TEST", "x")
	entries := []Entry{
		mkEntry("a", 0, false, `env.CLY_SEL_TEST && env.CLY_SEL_UNSET`),
	}
	_, d := selectProvider(entries, selCtx())
	assert.Equal(t, map[string]bool{"CLY_SEL_TEST": true, "CLY_SEL_UNSET": false}, d.EnvRefs)
}

func TestBuildContext(t *testing.T) {
	c := buildContext()
	assert.NotEmpty(t, c.User)
	assert.NotEmpty(t, c.Host)
	assert.NotEmpty(t, c.Arch)
	assert.NotEmpty(t, c.OS)
	assert.NotEmpty(t, c.Dir)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/ai/ -run 'TestSelect|TestDecision|TestBuildContext' -v`
Expected: FAIL — `undefined: selectProvider`, `undefined: buildContext`.

- [ ] **Step 3: Implement**

Move `Context` + `lookup` from `condition.go` into new `pkg/ai/context.go` (verbatim move, no behavior change), then add `buildContext`:

`pkg/ai/context.go`:

```go
package ai

import (
	"os"
	"os/user"
	"runtime"
	"strings"
)

// Context is the runtime environment conditions evaluate against.
type Context struct {
	User, Host, Arch, OS, Dir string
}

// lookup resolves a condition field to its value. env.NAME reads the
// process environment. bool reports whether the field name is known.
func (c *Context) lookup(field string) (string, bool) {
	// (verbatim from condition.go)
}

// buildContext captures the current process environment for selection.
func buildContext() *Context {
	c := &Context{Arch: runtime.GOARCH, OS: runtime.GOOS}
	if u, err := user.Current(); err == nil {
		c.User = u.Username
	}
	if h, err := os.Hostname(); err == nil {
		c.Host = h
	}
	if d, err := os.Getwd(); err == nil {
		c.Dir = d
	}
	return c
}
```

`pkg/ai/select.go`:

```go
package ai

import "strings"

// EntryResult records how one entry fared in selection.
type EntryResult struct {
	Name      string
	Condition string
	Matched   bool
	Weight    int
	Note      string
}

// Decision is the full record of a provider selection: what context was
// used, how each entry evaluated, and why the winner won.
type Decision struct {
	Context *Context
	EnvRefs map[string]bool
	Entries []EntryResult
	Picked  string
	Reason  string
}

// selectProvider picks the winning entry:
//  1. highest weight among condition matches (ties: list order)
//  2. first entry with default: true
//  3. first entry
func selectProvider(entries []Entry, ctx *Context) (Entry, *Decision) {
	d := &Decision{Context: ctx, EnvRefs: map[string]bool{}}
	d.Entries = make([]EntryResult, len(entries))
	for i, e := range entries {
		matched := e.cond != nil && e.cond.eval(ctx)
		d.Entries[i] = EntryResult{
			Name: e.Name, Condition: e.Condition, Matched: matched, Weight: e.Weight,
		}
		collectEnvRefs(e.Condition, d.EnvRefs)
	}

	best := -1
	for i, e := range entries {
		if !d.Entries[i].Matched {
			continue
		}
		if best == -1 || e.Weight > entries[best].Weight {
			best = i
		}
	}
	if best >= 0 {
		d.Entries[best].Note = "picked"
		d.Picked = entries[best].Name
		d.Reason = "condition match"
		return entries[best], d
	}

	for i, e := range entries {
		if e.Default {
			d.Entries[i].Note = "default fallback"
			d.Picked = e.Name
			d.Reason = "default"
			return entries[i], d
		}
	}

	d.Entries[0].Note = "first entry fallback"
	d.Picked = entries[0].Name
	d.Reason = "first entry"
	return entries[0], d
}

// collectEnvRefs records env var names mentioned in a condition and
// whether each is currently set (never records values).
func collectEnvRefs(cond string, refs map[string]bool) {
	for cond != "" {
		i := strings.Index(cond, "env.")
		if i < 0 {
			return
		}
		rest := cond[i+4:]
		j := 0
		for j < len(rest) && (rest[j] == '_' || (rest[j] >= 'A' && rest[j] <= 'Z') ||
			(rest[j] >= 'a' && rest[j] <= 'z') || (rest[j] >= '0' && rest[j] <= '9')) {
			j++
		}
		if j > 0 {
			name := rest[:j]
			if _, ok := refs[name]; !ok {
				refs[name] = os.Getenv(name) != ""
			}
		}
		cond = rest[j:]
	}
}
```

(`collectEnvRefs` needs `os` import — merge imports properly.)

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./pkg/ai/ -v`
Expected: PASS — all condition, providers, and select tests.

- [ ] **Step 5: Commit**

```bash
git add pkg/ai/context.go pkg/ai/select.go pkg/ai/select_test.go pkg/ai/condition.go
git commit -m "ai: runtime context and weighted provider selection"
```

---

### Task 4: Wire selection into resolve() + debug logging

**Files:**
- Modify: `pkg/ai/ai.go`
- Test: `pkg/ai/ai_test.go` (new; covers resolution end-to-end)

**Interfaces:**
- Consumes: `parseProviders`, `selectProvider`, `buildContext`, `Decision` (Tasks 1-3).
- Produces: `LastDecision() *Decision`; changed `resolve` behavior. `Resolved` struct UNCHANGED. `LoadConfig`, `LoadConfigWith`, `NewClient*`, `HasAPIKey*` signatures UNCHANGED.

Behavior rules:
- No `ai:` block in config at all → library defaults (anthropic + defaultModel + ANTHROPIC_API_KEY), as today. This keeps zero-config usage working.
- `ai:` present → `providers` list required; parse errors make resolution fail.
- Resolution failure: `LoadConfigWith` returns nil; `NewClientWith` returns the error; `HasAPIKey` returns false; the error is retrievable via `LastSelectionError() error`.
- Debug logging: on the FIRST successful selection per process (sync.Once), if `pkgconfig.Get().App.Debug`, print to stderr the one-liner `ai: picked provider "X" (condition match, weight 10)` followed by the full context + per-entry table. Later selections in the same process print nothing (selection result can't change within a process).

- [ ] **Step 1: Write the failing test**

`pkg/ai/ai_test.go`:

```go
package ai

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func listConfig(entries ...map[string]interface{}) map[string]interface{} {
	list := make([]interface{}, len(entries))
	for i, e := range entries {
		list[i] = e
	}
	return map[string]interface{}{"providers": list}
}

func TestResolveListSelection(t *testing.T) {
	global := listConfig(
		map[string]interface{}{"name": "work", "provider": "openai",
			"base_url": "https://gw.example/v1", "api_key": "$WORK_KEY",
			"model": "work-model", "weight": 10,
			"condition": `user == "` + buildContext().User + `"`},
		map[string]interface{}{"name": "personal", "provider": "anthropic",
			"model": "claude-sonnet-4-5-20250929", "default": true},
	)
	r, err := resolveE(global, nil)
	require.NoError(t, err)
	assert.Equal(t, "openai", r.Provider)
	assert.Equal(t, "work-model", r.Model)
	assert.Equal(t, "https://gw.example/v1", r.BaseURL)
	assert.Equal(t, "WORK_KEY", r.APIKeyEnv)
	require.NotNil(t, LastDecision())
	assert.Equal(t, "work", LastDecision().Picked)
}

func TestResolveDefaultFallback(t *testing.T) {
	global := listConfig(
		map[string]interface{}{"name": "nomatch", "provider": "openai",
			"model": "x", "condition": `user == "definitely-not-me-zzz"`},
		map[string]interface{}{"name": "fb", "provider": "bedrock",
			"model": "us.anthropic.claude-sonnet-4-5-20250929-v1:0", "default": true},
	)
	r, err := resolveE(global, nil)
	require.NoError(t, err)
	assert.Equal(t, "bedrock", r.Provider)
	assert.Equal(t, "us.anthropic.claude-sonnet-4-5-20250929-v1:0", r.Model)
}

func TestResolveNoAIBlockKeepsLibraryDefaults(t *testing.T) {
	r, err := resolveE(nil, nil)
	require.NoError(t, err)
	assert.Equal(t, defaultProvider, r.Provider)
	assert.Equal(t, defaultModel, r.Model)
	assert.Equal(t, defaultAPIKeyEnv, r.APIKeyEnv)
}

func TestResolveBadConfig(t *testing.T) {
	global := listConfig(map[string]interface{}{"model": "m"}) // no name
	r, err := resolveE(global, nil)
	assert.Error(t, err)
	assert.Nil(t, r)
	// LoadConfigWith-style behavior: nil on error
	assert.Nil(t, resolve(global, nil))
	// error retrievable for NewClientWith path
	require.Error(t, LastSelectionError())
}

func TestResolveModuleOverrideOnPickedEntry(t *testing.T) {
	global := listConfig(
		map[string]interface{}{"name": "only", "provider": "openai",
			"model": "base-model", "api_key": "$ONLY_KEY"},
	)
	override := map[string]interface{}{"model": "override-model"}
	r, err := resolveE(global, override)
	require.NoError(t, err)
	assert.Equal(t, "openai", r.Provider)      // provider from picked entry
	assert.Equal(t, "override-model", r.Model) // model from override
	assert.Equal(t, "ONLY_KEY", r.APIKeyEnv)   // key from picked entry
}

func TestResolveModuleOverrideSwitchesProviderType(t *testing.T) {
	// override provider: bedrock re-bases to the first bedrock entry
	global := listConfig(
		map[string]interface{}{"name": "oa", "provider": "openai", "model": "m1"},
		map[string]interface{}{"name": "br", "provider": "bedrock", "model": "m2"},
	)
	override := map[string]interface{}{"provider": "bedrock"}
	r, err := resolveE(global, override)
	require.NoError(t, err)
	assert.Equal(t, "bedrock", r.Provider)
	assert.Equal(t, "m2", r.Model)
}

func TestResolveOverrideDisabled(t *testing.T) {
	global := listConfig(map[string]interface{}{"name": "x", "provider": "openai", "model": "m"})
	r, err := resolveE(global, map[string]interface{}{"enabled": false})
	require.NoError(t, err)
	assert.Nil(t, r)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/ai/ -run TestResolve -v`
Expected: FAIL — `undefined: resolveE`, `undefined: LastSelectionError`.

- [ ] **Step 3: Implement**

Rewrite the resolution core of `pkg/ai/ai.go`:

1. Replace the package doc comment's "Config layout" section with the new list shape (copy the YAML example from the spec).
2. Delete: `defaultProvider`-keyed map lookup in `resolve`, `applyProviderBlock`, `descendMap` (if unused elsewhere — grep first), and the old "Config layout" doc.
3. New resolution core:

```go
var (
	lastDecision *Decision
	lastSelErr   error
	loggedOnce   sync.Once
)

// LastDecision returns the most recent provider selection record, or nil
// if no selection has run yet this process.
func LastDecision() *Decision { return lastDecision }

// LastSelectionError returns the most recent resolution error, or nil.
func LastSelectionError() error { return lastSelErr }

// resolveE is resolve with an error return for paths that can surface
// config problems (NewClientWith, cly ai status).
func resolveE(global, override map[string]interface{}) (*Resolved, error) {
	if global == nil {
		// zero-config: library defaults
		return &Resolved{
			Provider: defaultProvider, Model: defaultModel,
			APIKeyEnv: defaultAPIKeyEnv, Enabled: true,
		}, nil
	}
	entries, err := parseProviders(global)
	if err != nil {
		lastSelErr = err
		return nil, err
	}
	entry, decision := selectProvider(entries, buildContext())
	lastDecision = decision
	lastSelErr = nil
	logSelection(decision, entry)

	r := &Resolved{
		Provider:  entry.Provider,
		Model:     entry.Model,
		APIKey:    entry.APIKey,
		APIKeyEnv: entry.APIKeyEnv,
		BaseURL:   entry.BaseURL,
		Enabled:   true,
	}

	if override != nil {
		if v, ok := override["enabled"].(bool); ok {
			r.Enabled = v
		}
		if v, ok := override["provider"].(string); ok && v != "" && v != r.Provider {
			r.Provider = v
			r.APIKey = ""
			r.APIKeyEnv = providerEnv[v]
			r.BaseURL = ""
			for _, e := range entries {
				if e.Provider == v {
					r.Model = e.Model
					r.APIKey = e.APIKey
					r.APIKeyEnv = e.APIKeyEnv
					r.BaseURL = e.BaseURL
					break
				}
			}
		}
		applyOverrideBlock(override, r)
	}
	if !r.Enabled {
		return nil, nil
	}
	return r, nil
}

// resolve keeps the historical signature: nil on error or disabled.
func resolve(global, override map[string]interface{}) *Resolved {
	r, _ := resolveE(global, override)
	return r
}

func logSelection(d *Decision, picked Entry) {
	if !pkgconfig.Get().App.Debug {
		return
	}
	loggedOnce.Do(func() {
		w := os.Stderr
		fmt.Fprintf(w, "ai: picked provider %q (%s, weight %d)\n", picked.Name, d.Reason, picked.Weight)
		fmt.Fprintf(w, "ai: context: user=%s host=%s arch=%s os=%s dir=%s\n",
			d.Context.User, d.Context.Host, d.Context.Arch, d.Context.OS, d.Context.Dir)
		for name, set := range d.EnvRefs {
			fmt.Fprintf(w, "ai: context: env.%s=%s\n", name, setUnset(set))
		}
		for _, e := range d.Entries {
			fmt.Fprintf(w, "ai:   %-20s matched=%-5v weight=%d %s\n", e.Name, e.Matched, e.Weight, e.Note)
		}
	})
}

func setUnset(b bool) string {
	if b {
		return "(set)"
	}
	return "(unset)"
}
```

4. `NewClientWith` switches to `resolveE` and returns the error:

```go
func NewClientWith(override map[string]interface{}) (llm.Client, error) {
	r, err := resolveE(pkgconfig.Get().AI, override)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, nil
	}
	cfg := llm.Config{...unchanged...}
	return llm.NewClient(cfg)
}
```

5. Imports to add in ai.go: `sync`, `fmt` (already?), keep `os`. Remove now-unused helpers.

- [ ] **Step 4: Run tests**

Run: `go test ./pkg/ai/... -v && go build ./...`
Expected: PASS; build clean.

- [ ] **Step 5: Commit**

```bash
git add pkg/ai/ai.go pkg/ai/ai_test.go
git commit -m "ai: select provider by condition, weight, default; log decision"
```

---

### Task 5: `cly ai status` command + default config migration

**Files:**
- Create: `modules/ai/cmd.go`
- Modify: `cmd/root.go` (register), `modules/config/config.yaml` (new list shape)
- Test: manual run (command output), plus existing tests stay green

**Interfaces:**
- Consumes: `ai.LastDecision()`, `ai.LoadConfigWith`, `ai.LastSelectionError()`, `pkgconfig.Get().App.Debug`.

- [ ] **Step 1: Write the command**

`modules/ai/cmd.go`:

```go
// Package ai exposes the `cly ai` command group: visibility into which
// provider cly's AI features will use and why.
package ai

import (
	"fmt"
	"os"
	"sort"

	"github.com/spf13/cobra"
	coreai "github.com/yurifrl/cly/pkg/ai"
	pkgconfig "github.com/yurifrl/cly/pkg/config"
)

var Cmd = &cobra.Command{
	Use:   "ai",
	Short: "AI provider inspection",
}

func Register(parent *cobra.Command) {
	parent.AddCommand(Cmd)
}

func init() {
	Cmd.AddCommand(&cobra.Command{
		Use:   "status",
		Short: "Show which AI provider is selected and why",
		RunE: func(cmd *cobra.Command, args []string) error {
			return status()
		},
	})
}

func status() error {
	if err := coreai.LastSelectionError(); err != nil {
		return fmt.Errorf("ai config error: %w", err)
	}
	// Force a resolution so the decision exists even if nothing else ran.
	r := coreai.LoadConfigWith(nil)
	d := coreai.LastDecision()
	if d == nil {
		// zero-config library defaults path: no decision recorded
		fmt.Println("ai: no providers configured; using library defaults (anthropic)")
		return nil
	}
	if r == nil {
		fmt.Println("ai: disabled")
		return nil
	}
	fmt.Printf("picked: %s (%s)\n", d.Picked, d.Reason)
	fmt.Printf("provider: %s  model: %s\n", r.Provider, r.Model)
	if r.BaseURL != "" {
		fmt.Printf("base_url: %s\n", r.BaseURL)
	}
	if r.APIKeyEnv != "" {
		fmt.Printf("api_key: $%s %s\n", r.APIKeyEnv, setUnset(os.Getenv(r.APIKeyEnv) != ""))
	} else if r.APIKey != "" {
		fmt.Println("api_key: (literal, set)")
	}
	fmt.Println()
	fmt.Println("context:")
	fmt.Printf("  user=%s host=%s arch=%s os=%s dir=%s\n",
		d.Context.User, d.Context.Host, d.Context.Arch, d.Context.OS, d.Context.Dir)
	if len(d.EnvRefs) > 0 {
		names := make([]string, 0, len(d.EnvRefs))
		for n := range d.EnvRefs {
			names = append(names, n)
		}
		sort.Strings(names)
		for _, n := range names {
			fmt.Printf("  env.%s=%s\n", n, setUnset(d.EnvRefs[n]))
		}
	}
	fmt.Println()
	fmt.Println("entries:")
	for _, e := range d.Entries {
		note := e.Note
		if note == "" {
			note = "-"
		}
		fmt.Printf("  %-20s matched=%-5v weight=%-3d %s\n", e.Name, e.Matched, e.Weight, note)
	}
	if !pkgconfig.Get().App.Debug {
		fmt.Println("\n(debug off: run with CLY_APP_DEBUG=true to log selection on every AI call)")
	}
	return nil
}

func setUnset(b bool) string {
	if b {
		return "(set)"
	}
	return "(unset)"
}
```

`cmd/root.go`: add `aimod.Register(RootCmd)` following the existing module registration pattern (import alias `aimod "github.com/yurifrl/cly/modules/ai"`).

`modules/config/config.yaml`: replace the `ai:` block (lines 12-26) with:

```yaml
ai:
  providers:
    - name: anthropic          # default fallback
      provider: anthropic
      model: claude-sonnet-4-5-20250929
      api_key: $ANTHROPIC_API_KEY
      default: true
    - name: openai
      provider: openai
      model: gpt-4o-mini
      api_key: $OPENAI_API_KEY
    - name: openrouter         # OpenAI-compatible gateway (base_url preset)
      provider: openrouter
      model: anthropic/claude-3.5-sonnet
      api_key: $OPENROUTER_API_KEY
    - name: bedrock            # Anthropic models via AWS Bedrock
      provider: bedrock
      model: us.anthropic.claude-sonnet-4-5-20250929-v1:0
      # no api_key: uses AWS_BEARER_TOKEN_BEDROCK or AWS creds/profile
```

Also grep for other references to the old shape and update them:
Run: `grep -rn "providers:" --include="*.yaml" --include="*.md" . | grep -v node_modules | grep -v .pi/`
Update README.md / docs examples to the list shape if present.

- [ ] **Step 2: Build and verify manually**

```bash
go build ./... && go test ./pkg/ai/... ./modules/...
go run . ai status
```

Expected: builds; tests pass; `ai status` prints the picked entry (anthropic default from the shipped config), context line, and entries table. Then with a debug flag:

```bash
CLY_APP_DEBUG=true go run . ai status
```

Expected: no debug line from status itself about the hint; selection one-liner appears on stderr.

Also verify a condition-driven pick locally: temporarily point `~/.config/cly/config.local.yaml` (or env) at a config with `condition: 'dir =~ "*/cly"'` and confirm `go run . ai status` picks it. Do NOT commit any local config.

- [ ] **Step 3: Commit**

```bash
git add modules/ai/cmd.go cmd/root.go modules/config/config.yaml
git commit -m "ai: add cly ai status command; migrate default config to provider list"
```

---

### Task 6: Docs + final verification

**Files:**
- Modify: `pkg/ai/ai.go` package doc (if not fully done in Task 4), README.md AI section (if present)

- [ ] **Step 1: Update README/docs references**

Run: `grep -n "ai:" -A 8 README.md docs/**/*.md 2>/dev/null | head -40`
Replace any old-shape YAML snippets with the new list shape from the spec.

- [ ] **Step 2: Full test + build + lint**

```bash
go build ./... && go test ./...
task lint:isolation 2>/dev/null || true
```

Expected: all green (lint:isolation unaffected — pkg/ai imports stay stdlib+pkg/config+pkg/llm).

- [ ] **Step 3: Commit and push**

```bash
git add -A
git commit -m "docs: update ai config examples to provider list"
git pull --rebase && git push
```

---

## Self-Review Notes

- Spec coverage: conditions (T1), list parsing + default/weight fields (T2), context incl. env refs (T3), selection precedence (T3), decision logging + verbose dump (T4), `cly ai status` (T5), old-shape removal + config migration (T4/T5), module-override compatibility (T4 tests), error cases (T2/T4).
- Type consistency: `Entry`, `Decision`, `EntryResult`, `resolveE`, `LastDecision`, `LastSelectionError` used consistently across tasks; `Resolved` untouched.
- Known risk flagged in T2: YAML number decoding for `weight` (int vs float64) — implementer must check and handle both.
