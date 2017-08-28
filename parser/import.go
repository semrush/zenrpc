package parser

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/ioutil"
	"path/filepath"
	"strings"
)

func (pi *PackageInfo) parseImports(imports []*ast.ImportSpec, dir string) error {
	for _, i := range imports {
		path := i.Path.Value[1 : len(i.Path.Value)-1] // remove quotes ""
		name := ""
		if i.Name != nil {
			name = i.Name.Name
		} else {
			name = path[strings.LastIndex(path, "/")+1:]
		}

		realPath := tryFindPath(path, dir)
		// can't find path to package
		if realPath == "" {
			continue
		}

		// read import dir
		files, err := ioutil.ReadDir(realPath)
		if err != nil {
			return err
		}

		// for each file
		pkgImports := []*ast.ImportSpec{}
		for _, f := range files {
			if f.IsDir() {
				continue
			}

			if !strings.HasSuffix(f.Name(), goFileSuffix) ||
				strings.HasSuffix(f.Name(), GenerateFileSuffix) || strings.HasSuffix(f.Name(), testFileSuffix) {
				continue
			}

			filename := filepath.Join(realPath, f.Name())
			fset := token.NewFileSet()
			f, err := parser.ParseFile(fset, filename, nil, parser.ParseComments)
			if err != nil {
				return err
			}

			pi.Scopes[name] = append(pi.Scopes[name], f.Scope)
			pkgImports = append(pkgImports, f.Imports...)
		}

		// collect unique imports from package files and call parseImports once for package
		if err := pi.parseImports(uniqueImports(pkgImports), dir); err != nil {
			return err
		}
	}
	return nil
}

func uniqueImports(in []*ast.ImportSpec) (out []*ast.ImportSpec) {
	set := map[string]struct{}{}
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
