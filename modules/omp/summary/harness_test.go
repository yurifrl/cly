package ompsummary

import "github.com/yurifrl/cly/pkg/cmux"

// Thin aliases so test fixtures read like the JSON they mirror.
type (
	cmuxRow   = cmux.SessionRow
	treeShape = cmux.Tree
	treeWin   = cmux.TreeWindow
	treeWs    = cmux.TreeWorkspace
	treePane  = cmux.TreePane
	treeSurf  = cmux.TreeSurface
)

// go.mod directive is 1.25 (new(expr) needs 1.26): helper form retained per
// the go-new-expr rule's compatibility note.
func strPtr(s string) *string { return &s }

func mergeFixture(tree *treeShape, rows []cmuxRow, now, maxAge float64) []Target {
	return merge(tree, rows, now, maxAge)
}
