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

var Analyzer = &analysis.Analyzer{
	Name:     "spannercheck",
	Doc:      "checks that persistence packages use Spanner helpers instead of raw client calls",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

const (
	spannerPkg         = "cloud.google.com/go/spanner"
	platformSpannerPkg = "github.com/rai/clean-modularmonolith-go/internal/platform/spanner"
)

type methodKey struct {
	typeName   string
	methodName string
}

var forbiddenMethods = map[methodKey]string{
	{"Client", "Apply"}:                     "use platformspanner.Write",
	{"Client", "Single"}:                    "use platformspanner.Read",
	{"Client", "ReadOnlyTransaction"}:       "use platformspanner.ReadConsistent",
	{"Client", "ReadWriteTransaction"}:      "use platformspanner.Write",
	{"ReadWriteTransaction", "BufferWrite"}: "use DML via platformspanner.Write",
}

var forbiddenFuncs = map[string]string{
	"ReadWriteTxFromContext":     "use platformspanner.Write",
	"ReadTransactionFromContext": "use platformspanner.Read or platformspanner.ReadConsistent",
}

func run(pass *analysis.Pass) (interface{}, error) {
	pkgPath := pass.Pkg.Path()
	if !strings.Contains(pkgPath, "infrastructure/persistence") {
		return nil, nil
	}

	// Build lookup set of forbidden package-level function objects by identity.
	bannedFuncObjs := make(map[types.Object]string)
	for _, imp := range pass.Pkg.Imports() {
		if imp.Path() == platformSpannerPkg {
			for funcName, suggestion := range forbiddenFuncs {
				if obj := imp.Scope().Lookup(funcName); obj != nil {
					bannedFuncObjs[obj] = suggestion
				}
			}
		}
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

		// Check if this is a method call (has a Selection entry).
		if selection, exists := pass.TypesInfo.Selections[sel]; exists {
			checkMethod(pass, call, sel, selection)
			return
		}

		// Otherwise it's a qualified identifier (pkg.Func).
		if obj := pass.TypesInfo.ObjectOf(sel.Sel); obj != nil {
			if suggestion, found := bannedFuncObjs[obj]; found {
				pass.Reportf(call.Pos(),
					"direct call to %s in persistence package; %s",
					obj.Name(), suggestion,
				)
			}
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
