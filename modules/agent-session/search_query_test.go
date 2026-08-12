package agentsession

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseQueryOperators(t *testing.T) {
	pq := parseQuery(`customer AND sales NOT draft`)
	assert.Equal(t, []string{"customer", "sales"}, pq.positive)
	assert.ElementsMatch(t, []string{"customer", "sales", "draft"}, pq.terms)
}

func TestPurviewSemantics(t *testing.T) {
	// space = OR
	ok, _ := parseQuery(`customer sales`).match("sales dashboard")
	assert.True(t, ok)

	// AND requires both
	ok, _ = parseQuery(`customer AND sales`).match("sales dashboard")
	assert.False(t, ok)
	ok, _ = parseQuery(`customer AND sales`).match("customer sales report")
	assert.True(t, ok)

	// + is AND, | is OR
	ok, _ = parseQuery(`customer + sales`).match("only customer here")
	assert.False(t, ok)
	ok, _ = parseQuery(`customer | sales`).match("only sales here")
	assert.True(t, ok)

	// NOT excludes
	ok, _ = parseQuery(`customer NOT draft`).match("customer draft copy")
	assert.False(t, ok)
	ok, _ = parseQuery(`customer NOT draft`).match("customer final")
	assert.True(t, ok)

	// phrase must be adjacent, in order
	ok, _ = parseQuery(`"sales report"`).match("the sales report is ready")
	assert.True(t, ok)
	ok, _ = parseQuery(`"sales report"`).match("sales quarterly report")
	assert.False(t, ok)

	// grouping controls precedence
	ok, _ = parseQuery(`(customer OR client) AND sales`).match("client sales deck")
	assert.True(t, ok)
	ok, _ = parseQuery(`(customer OR client) AND sales`).match("client roadmap")
	assert.False(t, ok)

	// match all
	ok, _ = parseQuery(`*`).match("anything")
	assert.True(t, ok)
	assert.True(t, parseQuery(``).empty())
}

func TestMatchFolder(t *testing.T) {
	assert.True(t, matchFolder("/a/b/c", ""))
	assert.True(t, matchFolder("/a/b/c", "/a/b"))
	assert.True(t, matchFolder("/a/b", "/a/b"))
	assert.False(t, matchFolder("/a/bc", "/a/b"))
	assert.False(t, matchFolder("/x/y", "/a/b"))
}

func TestResolveFolder(t *testing.T) {
	assert.Equal(t, "", resolveFolder("", "/home/u"))
	assert.Equal(t, "/home/u", resolveFolder(".", "/home/u"))
	assert.Equal(t, "/abs/path", resolveFolder("/abs/path", "/home/u"))
	assert.Equal(t, "/home/u/sub", resolveFolder("sub", "/home/u"))
}

func TestNextRole(t *testing.T) {
	assert.Equal(t, roleUser, nextRole(roleAll))
	assert.Equal(t, roleAssistant, nextRole(roleUser))
	assert.Equal(t, roleAll, nextRole(roleAssistant))
}

func TestRoleMatches(t *testing.T) {
	assert.True(t, roleMatches(roleAll, "user"))
	assert.True(t, roleMatches("", "assistant"))
	assert.True(t, roleMatches(roleUser, "user"))
	assert.False(t, roleMatches(roleUser, "assistant"))
	assert.False(t, roleMatches(roleAssistant, "user"))
}
