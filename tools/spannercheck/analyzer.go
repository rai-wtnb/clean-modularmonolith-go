// Package spannercheck defines an analyzer that ensures persistence packages
// use Spanner helper functions instead of raw Spanner client calls.
package spannercheck

import (
	"go/ast"
	"go/types"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// NOTE: All context-accessor helpers (readWriteTxFromContext, readTransactionFromContext)
// are unexported in the platform/spanner package. If exported accessors are ever added,
// add them to a forbiddenFuncs check here.

var Analyzer = &analysis.Analyzer{
	Name:     "spannercheck",
	Doc:      "checks that persistence packages use Spanner helpers instead of raw client calls",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

const spannerPkg = "cloud.google.com/go/spanner"

type methodKey struct {
	typeName   string
	methodName string
}

var forbiddenMethods = map[methodKey]string{
	{"Client", "Apply"}:                     "use platformspanner.Write",
	{"Client", "Single"}:                    "use platformspanner.SingleRead",
	{"Client", "ReadOnlyTransaction"}:       "use platformspanner.ConsistentRead",
	{"Client", "ReadWriteTransaction"}:      "use platformspanner.Write",
	{"ReadWriteTransaction", "BufferWrite"}: "use DML via platformspanner.Write",
}

func run(pass *analysis.Pass) (interface{}, error) {
	pkgPath := pass.Pkg.Path()
	if !strings.Contains(pkgPath, "infrastructure/persistence") {
		return nil, nil
	}

	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	nodeFilter := []ast.Node{
		(*ast.CallExpr)(nil),
	}

	insp.Preorder(nodeFilter, func(n ast.Node) {
		call := n.(*ast.CallExpr)

		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return
		}

		if selection, exists := pass.TypesInfo.Selections[sel]; exists {
			checkMethod(pass, call, sel, selection)
		}
	})

	return nil, nil
}

func checkMethod(pass *analysis.Pass, call *ast.CallExpr, sel *ast.SelectorExpr, selection *types.Selection) {
	recv := selection.Recv()

	// Dereference pointer.
	if ptr, ok := recv.(*types.Pointer); ok {
		recv = ptr.Elem()
	}

	named, ok := recv.(*types.Named)
	if !ok {
		return
	}

	obj := named.Obj()
	if obj.Pkg() == nil || obj.Pkg().Path() != spannerPkg {
		return
	}

	key := methodKey{obj.Name(), sel.Sel.Name}
	suggestion, found := forbiddenMethods[key]
	if !found {
		return
	}

	pass.Reportf(call.Pos(),
		"direct call to (*spanner.%s).%s in persistence package; %s",
		obj.Name(), sel.Sel.Name, suggestion,
	)
}
