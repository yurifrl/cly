package agentsession

import "strings"

// parsedQuery holds a boolean search expression using Microsoft Purview
// Unified Catalog syntax, plus `+` as AND and `|` as OR:
//
//	customer sales        space = OR (assets matching more rank higher)
//	customer AND sales    both must be present   (also: customer + sales)
//	customer OR sales     either may be present  (also: customer | sales)
//	customer NOT draft    first present, second absent   (also: -draft)
//	"sales report"        exact phrase, in order
//	(a OR b) AND c        grouping controls precedence
//	name:customer         field-scoped (name, description, path, provider)
//	*  or empty           match all
//
// root is the evaluable expression tree (nil = match all). positive lists
// the non-negated leaf terms (for hit counting, snippets, highlighting) and
// terms lists every leaf term (for the body-presence scan).
type parsedQuery struct {
	root     node
	positive []string
	terms    []string
}

func (p parsedQuery) empty() bool { return p.root == nil }

// node is one element of the boolean expression tree.
type node interface {
	eval(present func(field, text string) bool) bool
}

type termNode struct{ field, text string }

func (n termNode) eval(present func(field, text string) bool) bool {
	return present(n.field, n.text)
}

type notNode struct{ child node }

func (n notNode) eval(present func(field, text string) bool) bool {
	return !n.child.eval(present)
}

type andNode struct{ a, b node }

func (n andNode) eval(present func(field, text string) bool) bool {
	return n.a.eval(present) && n.b.eval(present)
}

type orNode struct{ a, b node }

func (n orNode) eval(present func(field, text string) bool) bool {
	return n.a.eval(present) || n.b.eval(present)
}

// evalMatch reports whether the query matches given a presence test.
func (p parsedQuery) evalMatch(present func(field, text string) bool) bool {
	if p.root == nil {
		return true
	}
	return p.root.eval(present)
}

// match evaluates the query against a single lowered text blob and returns
// the total occurrence count of positive terms (for ranking).
func (p parsedQuery) match(lowered string) (ok bool, hits int) {
	present := func(_, t string) bool { return strings.Contains(lowered, t) }
	if !p.evalMatch(present) {
		return false, 0
	}
	for _, t := range p.positive {
		hits += strings.Count(lowered, t)
	}
	return true, hits
}

// parseQuery tokenizes raw and parses it into a boolean expression tree.
func parseQuery(raw string) parsedQuery {
	toks := tokenize(raw)
	pr := &parser{toks: toks}
	root := pr.parseOr()
	pq := parsedQuery{root: root}
	collectTerms(root, false, &pq)
	return pq
}

// collectTerms walks the tree; terms gets every leaf, positive gets leaves
// under an even number of NOTs.
func collectTerms(n node, negated bool, pq *parsedQuery) {
	switch t := n.(type) {
	case termNode:
		pq.terms = append(pq.terms, t.text)
		if !negated {
			pq.positive = append(pq.positive, t.text)
		}
	case notNode:
		collectTerms(t.child, !negated, pq)
	case andNode:
		collectTerms(t.a, negated, pq)
		collectTerms(t.b, negated, pq)
	case orNode:
		collectTerms(t.a, negated, pq)
		collectTerms(t.b, negated, pq)
	}
}

// --- tokenizer ---

type tokKind int

const (
	tTerm tokKind = iota
	tAnd
	tOr
	tNot
	tLParen
	tRParen
	tStar
)

type token struct {
	kind        tokKind
	field, text string
}

// tokenize splits raw into typed tokens. `"quoted phrases"` stay whole;
// `(`, `)`, `+`, `|` are punctuation operators; bare uppercase AND/OR/NOT are
// operators; `*` alone is match-all; `field:value` sets a field on a term.
func tokenize(raw string) []token {
	var out []token
	var buf strings.Builder
	field := ""
	quoted := false
	inQuote := false

	flush := func() {
		text := buf.String()
		buf.Reset()
		f := field
		field = ""
		wasQuoted := quoted
		quoted = false
		if text == "" && f == "" {
			return
		}
		if !wasQuoted && f == "" {
			switch text {
			case "AND":
				out = append(out, token{kind: tAnd})
				return
			case "OR":
				out = append(out, token{kind: tOr})
				return
			case "NOT":
				out = append(out, token{kind: tNot})
				return
			case "*":
				out = append(out, token{kind: tStar})
				return
			}
		}
		out = append(out, token{kind: tTerm, field: f, text: strings.ToLower(text)})
	}

	for _, r := range raw {
		if inQuote {
			if r == '"' {
				inQuote = false
			} else {
				buf.WriteRune(r)
			}
			continue
		}
		switch r {
		case '"':
			inQuote = true
			quoted = true
		case ' ', '\t':
			flush()
		case '(':
			flush()
			out = append(out, token{kind: tLParen})
		case ')':
			flush()
			out = append(out, token{kind: tRParen})
		case '+':
			flush()
			out = append(out, token{kind: tAnd})
		case '|':
			flush()
			out = append(out, token{kind: tOr})
		case ':':
			if field == "" && buf.Len() > 0 && !quoted {
				field = strings.ToLower(buf.String())
				buf.Reset()
			} else {
				buf.WriteRune(r)
			}
		default:
			buf.WriteRune(r)
		}
	}
	flush()
	return out
}

// --- parser (precedence: OR < AND < NOT, parentheses override) ---

type parser struct {
	toks []token
	pos  int
}

func (p *parser) peek() (token, bool) {
	if p.pos < len(p.toks) {
		return p.toks[p.pos], true
	}
	return token{}, false
}

func (p *parser) next() (token, bool) {
	t, ok := p.peek()
	if ok {
		p.pos++
	}
	return t, ok
}

// parseOr := parseAnd ( (OR | implicit) parseAnd )*
func (p *parser) parseOr() node {
	left := p.parseAnd()
	for {
		t, ok := p.peek()
		if !ok || t.kind == tRParen {
			break
		}
		if t.kind == tOr {
			p.next()
		}
		// else: implicit adjacency between operands is also OR (Purview).
		right := p.parseAnd()
		if right == nil {
			break
		}
		if left == nil {
			left = right
		} else {
			left = orNode{a: left, b: right}
		}
	}
	return left
}

// parseAnd := parseNot ( (AND | NOT) parseNot )*  — NOT negates its right.
func (p *parser) parseAnd() node {
	left := p.parseNot()
	for {
		t, ok := p.peek()
		if !ok {
			break
		}
		if t.kind == tAnd {
			p.next()
			right := p.parseNot()
			left = joinAnd(left, right)
			continue
		}
		if t.kind == tNot {
			p.next()
			right := p.parseNot()
			if right != nil {
				left = joinAnd(left, notNode{child: right})
			}
			continue
		}
		break
	}
	return left
}

func joinAnd(left, right node) node {
	if right == nil {
		return left
	}
	if left == nil {
		return right
	}
	return andNode{a: left, b: right}
}

// parseNot := ('-' | NOT) parseNot | primary
func (p *parser) parseNot() node {
	t, ok := p.peek()
	if ok && t.kind == tNot {
		p.next()
		child := p.parseNot()
		if child == nil {
			return nil
		}
		return notNode{child: child}
	}
	return p.parsePrimary()
}

// primary := '(' parseOr ')' | STAR | term  ('-term' prefix negates)
func (p *parser) parsePrimary() node {
	t, ok := p.next()
	if !ok {
		return nil
	}
	switch t.kind {
	case tLParen:
		inner := p.parseOr()
		if nt, ok := p.peek(); ok && nt.kind == tRParen {
			p.next()
		}
		return inner
	case tStar:
		return nil // match all
	case tTerm:
		return termNode{field: t.field, text: t.text}
	default:
		return nil
	}
}
