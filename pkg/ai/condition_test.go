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
		{`env.CLY_TEST_COND`, true},         // truthy when set non-empty
		{`env.CLY_DEFINITELY_UNSET`, false}, // truthy check on unset
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
		`user ==`,         // missing operand
		`user === "x"`,    // bad operator
		`foo == "x"`,      // unknown field
		`env. == "x"`,     // empty env name
		`(user == "x"`,    // unbalanced paren
		`user == "x" &&`,  // dangling operator
		`user == "x" "y"`, // trailing tokens
		`dir ~ "x"`,       // =~ is the only match op
	} {
		t.Run(expr, func(t *testing.T) {
			_, err := parseCondition(expr)
			assert.Error(t, err, expr)
		})
	}
}
