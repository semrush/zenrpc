package parser

import (
	"go/ast"
	"strings"
)

func importNameOrAliasAndPath(i *ast.ImportSpec) (name, path string) {
	path = i.Path.Value[1 : len(i.Path.Value)-1] // remove quotes ""
	if i.Name != nil {
		name = i.Name.Name
	} else {
		name = path[strings.LastIndex(path, "/")+1:]
	}

	return
}

func uniqueImports(in []*ast.ImportSpec) (out []*ast.ImportSpec) {
	set := make(map[string]struct{})
	for _, i := range in {
		key := i.Path.Value
		if i.Name != nil {
			key += "|" + i.Name.Name
		}

		if _, ok := set[key]; !ok {
			out = append(out, i)
			set[key] = struct{}{}
		}
	}

	return
}

// filterImports filter imports by namespace in structs
func filterImports(in []*ast.ImportSpec, names map[string]struct{}) (out []*ast.ImportSpec) {
	for _, i := range in {
		name, _ := importNameOrAliasAndPath(i)
		if _, ok := names[name]; ok {
			out = append(out, i)
		}
	}

	return
}
