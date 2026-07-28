package api

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/danielgtaylor/huma/v2"
)

// Parameters keyed by operation ID go stale silently: rename a route and the
// entry stops reaching the page with nothing to say so.
func TestOpenAPIParametersMatchLiveOperations(t *testing.T) {
	if err := checkParameterisedOperations(New(Options{}).document); err != nil {
		t.Fatal(err)
	}
}

// A declared parameter has to survive into the document. Path parameters are
// set at the registration site and must not be displaced by the table.
func TestOpenAPIDeclaredParametersReachTheDocument(t *testing.T) {
	document := New(Options{}).document

	found := map[string]map[string]bool{}
	forEachOperation(document, func(operation *huma.Operation) {
		names := map[string]bool{}
		for _, parameter := range operation.Parameters {
			names[parameter.In+" "+parameter.Name] = true
		}
		found[operation.OperationID] = names
	})

	for id, declared := range operationQueryParameters {
		for _, parameter := range declared {
			if !found[id]["query "+parameter.Name] {
				t.Errorf("operation %q lost query parameter %q",
					id, parameter.Name)
			}
		}
	}

	// /sde/types is registered with a path-free operation literal, and
	// /sde/types/{id} declares its id at the site. Both must still hold.
	if !found["sde-type"]["path id"] {
		t.Error("sde-type lost its path parameter")
	}
	if !found["sde-types"]["query category_id"] {
		t.Error("sde-types lost its declared query parameter")
	}
}

// Every parameter the document declares needs a schema and a description, or
// the reference page renders a nameless box the caller has to guess at.
func TestOpenAPIQueryParametersAreDescribed(t *testing.T) {
	forEachOperation(New(Options{}).document, func(operation *huma.Operation) {
		for _, parameter := range operation.Parameters {
			if parameter.In != "query" {
				continue
			}
			if parameter.Description == "" {
				t.Errorf("%s: query parameter %q has no description",
					operation.OperationID, parameter.Name)
			}
			if parameter.Schema == nil || parameter.Schema.Type == "" {
				t.Errorf("%s: query parameter %q has no schema type",
					operation.OperationID, parameter.Name)
			}
		}
	})
}

// The reference page switches a parameter to its entity search on x-entity,
// and it recognises a fixed set of kinds. A typo here is invisible in the
// document and shows up only as a picker that never renders.
func TestOpenAPIEntityParametersNameAKnownKind(t *testing.T) {
	known := map[string]bool{
		"character": true, "corporation": true, "alliance": true,
		"faction": true, "ship": true, "item": true,
		"system": true, "region": true, "constellation": true,
	}
	forEachOperation(New(Options{}).document, func(operation *huma.Operation) {
		for _, parameter := range operation.Parameters {
			kind, ok := parameter.Extensions[entityExtension]
			if !ok {
				continue
			}
			name, isString := kind.(string)
			if !isString || !known[name] {
				t.Errorf("%s: parameter %q names unknown entity kind %v",
					operation.OperationID, parameter.Name, kind)
			}
		}
	})
}

func TestQuerySchemasMatchNumericHandlerInputs(t *testing.T) {
	document := New(Options{}).document
	cases := []struct {
		path, operation, name, schemaType string
	}{
		{"/conflicts/battles", "conflict-battles", "minKills", huma.TypeInteger},
		{"/conflicts/battles", "conflict-battles", "minIsk", huma.TypeNumber},
		{"/conflicts/battles", "conflict-battles", "regionId", huma.TypeInteger},
		{"/conflicts/battles", "conflict-battles", "systemId", huma.TypeInteger},
		{"/stats", "global-stats", "days", huma.TypeNumber},
		{"/location", "location", "x", huma.TypeNumber},
		{"/location", "location", "y", huma.TypeNumber},
		{"/location", "location", "z", huma.TypeNumber},
	}

	for _, test := range cases {
		operation := document.Paths[test.path].Get
		if operation == nil || operation.OperationID != test.operation {
			t.Fatalf("%s: operation = %#v", test.path, operation)
		}
		found := false
		for _, parameter := range operation.Parameters {
			if parameter.In != "query" || parameter.Name != test.name {
				continue
			}
			found = true
			if parameter.Schema == nil ||
				parameter.Schema.Type != test.schemaType {
				t.Errorf("%s %s type = %v, want %s",
					test.operation, test.name, parameter.Schema,
					test.schemaType)
			}
		}
		if !found {
			t.Errorf("%s is missing query parameter %s",
				test.operation, test.name)
		}
	}
}

func TestQueriesRejectedWhenOmittedAreRequired(t *testing.T) {
	document := New(Options{}).document
	cases := map[string][]string{
		"/fittings/search": {"ship"},
		"/stats":           {"dataType"},
		"/stats/rankings":  {"section"},
		"/search":          {"q"},
		"/matchup":         {"attacker", "victim"},
		"/location":        {"system_id", "x", "y", "z"},
	}
	for path, names := range cases {
		operation := document.Paths[path].Get
		if operation == nil {
			t.Fatalf("%s has no GET operation", path)
		}
		for _, name := range names {
			found := false
			for _, parameter := range operation.Parameters {
				if parameter.In == "query" && parameter.Name == name {
					found = true
					if !parameter.Required {
						t.Errorf("%s query %s is not required", path, name)
					}
				}
			}
			if !found {
				t.Errorf("%s is missing query %s", path, name)
			}
		}
	}
}

// The table describes handlers, so it goes stale the moment a handler reads a
// name nobody wrote down. Reading the package source is the only check that
// notices: the parameters are documentation, so no request can fail and no
// response can change when one is missing.
//
// This is deliberately package-wide rather than per-operation. Handlers are
// shared across routes and reached through dispatch tables, so tying a name to
// one operation from the syntax alone would report matches that are not there.
// Package-wide still catches the case that matters — a parameter added to a
// handler and never documented anywhere.
func TestEveryQueryParameterHandlersReadIsDeclared(t *testing.T) {
	read, err := queryNamesReadInPackage(".")
	if err != nil {
		t.Fatalf("scan package source: %v", err)
	}
	// A scanner that silently found nothing would pass this test forever.
	// These three cover the three shapes it has to recognise: a direct
	// req.Query.Get, a name passed to a bounded helper, and a name read
	// through a url.Values parameter.
	for _, name := range []string{"victimFactions", "limit", "warSideCorps"} {
		if !read[name] {
			t.Fatalf("scanner missed %q; it is no longer reading the source",
				name)
		}
	}

	declared := declaredQueryNames()
	var missing []string
	for name := range read {
		if !declared[name] {
			missing = append(missing, name)
		}
	}
	sort.Strings(missing)
	if len(missing) > 0 {
		t.Errorf(
			"handlers read query parameters that no operation declares: %v",
			missing,
		)
	}
}

// queryNamesReadInPackage reports every literal query parameter name the
// non-test sources in dir pass to a query lookup.
func queryNamesReadInPackage(dir string) (map[string]bool, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	// Helpers that take the parameter name as an argument rather than reading
	// req.Query directly, mapped to the index of that argument.
	namedHelpers := map[string]int{
		"boundedQueryInt":         1,
		"optionalQueryNumber":     1,
		"finiteQueryNumber":       1,
		"parseConflictBoundedInt": 1,
		"parseConflictOptionalID": 1,
		"fittingSearchPageNumber": 1,
	}

	names := map[string]bool{}
	fset := token.NewFileSet()
	for _, entry := range entries {
		file := entry.Name()
		if !strings.HasSuffix(file, ".go") ||
			strings.HasSuffix(file, "_test.go") {
			continue
		}
		parsed, err := parser.ParseFile(
			fset, filepath.Join(dir, file), nil, 0,
		)
		if err != nil {
			return nil, err
		}
		ast.Inspect(parsed, func(node ast.Node) bool {
			switch value := node.(type) {
			case *ast.CallExpr:
				collectQueryCall(value, namedHelpers, names)
			case *ast.IndexExpr:
				if isQueryExpression(value.X) {
					if literal, ok := stringLiteral(value.Index); ok {
						names[literal] = true
					}
				}
			}
			return true
		})
	}
	return names, nil
}

func collectQueryCall(
	call *ast.CallExpr,
	namedHelpers map[string]int,
	names map[string]bool,
) {
	callee := ""
	switch fun := call.Fun.(type) {
	case *ast.SelectorExpr:
		callee = fun.Sel.Name
		if (callee == "Get" || callee == "Has") && isQueryExpression(fun.X) {
			if len(call.Args) > 0 {
				if literal, ok := stringLiteral(call.Args[0]); ok {
					names[literal] = true
				}
			}
		}
	case *ast.Ident:
		callee = fun.Name
	}
	index, ok := namedHelpers[callee]
	if !ok || index >= len(call.Args) {
		return
	}
	if literal, ok := stringLiteral(call.Args[index]); ok {
		names[literal] = true
	}
}

// isQueryExpression reports whether an expression names the request's parsed
// query values. Both `req.Query` and a `url.Values` named after it qualify.
func isQueryExpression(expr ast.Expr) bool {
	switch value := expr.(type) {
	case *ast.Ident:
		return strings.Contains(strings.ToLower(value.Name), "query")
	case *ast.SelectorExpr:
		return value.Sel.Name == "Query" || isQueryExpression(value.X)
	}
	return false
}

func stringLiteral(expr ast.Expr) (string, bool) {
	literal, ok := expr.(*ast.BasicLit)
	if !ok || literal.Kind != token.STRING {
		return "", false
	}
	value, err := strconv.Unquote(literal.Value)
	if err != nil {
		return "", false
	}
	return value, true
}
