package main

import (
	"bytes"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	zenrpcComment      = "//zenrpc"
	zenrpcService      = "zenrpc.Service"
	contextTypeName    = "context.Context"
	generateFileSuffix = "_zenrpc.go"
	testFileSuffix     = "_test.go"
)

func main() {
	var filename string
	if len(os.Args) > 1 {
		filename = os.Args[len(os.Args)-1]
	} else {
		filename = os.Getenv("GOFILE")
	}

	log.Printf("Entrypoint: %s", filename)

	sd := packageInfo{Services: make(map[string]service)}
	dir, err := sd.parseFiles(filename)
	if err != nil {
		log.Fatal(err)
	}

	outputFileName, err := sd.generateFile(dir)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Generated: %s", outputFileName)
}

// packageInfo represents struct info for XXX_zenrpc.go file generation
type packageInfo struct {
	PackageName string
	Services    map[string]service
}

type service struct {
	GenDecl *ast.GenDecl
	Name    string
	Methods map[string]*method
}

type method struct {
	FuncDecl      *ast.FuncType
	Name          string
	LowerCaseName string
	HasContext    bool
	Args          []arg
}

type arg struct {
	Name        string
	Type        string
	CapitalName string
	JsonName    string
}

// parseFiles parse all files associated with package from original file
func (pi *packageInfo) parseFiles(filename string) (string, error) {
	dir, err := filepath.Abs(filepath.Dir(filename))
	if err != nil {
		return dir, err
	}

	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return dir, err
	}
	for _, f := range files {
		if f.IsDir() {
			continue
		}

		if strings.HasSuffix(f.Name(), generateFileSuffix) || strings.HasSuffix(f.Name(), testFileSuffix) {
			continue
		}

		if err := pi.parseFile(filepath.Join(dir, f.Name())); err != nil {
			return dir, err
		}
	}

	return dir, nil
}

func (pi *packageInfo) generateFile(dir string) (string, error) {
	outputFileName := filepath.Join(dir, pi.PackageName+generateFileSuffix)
	file, err := os.Create(outputFileName)
	if err != nil {
		return outputFileName, err
	}
	defer file.Close()

	output := new(bytes.Buffer)
	if err := serviceTemplate.Execute(output, pi); err != nil {
		return outputFileName, err
	}

	source, err := format.Source(output.Bytes())
	if err != nil {
		return outputFileName, err
	}

	if _, err = file.Write(source); err != nil {
		return outputFileName, err
	}

	return outputFileName, nil
}

func (pi *packageInfo) parseFile(filename string) error {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filename, nil, parser.ParseComments)
	if err != nil {
		return err
	}
	//ast.Print(fset, f) // TODO remove

	if len(pi.PackageName) == 0 {
		pi.PackageName = f.Name.Name
	} else if pi.PackageName != f.Name.Name {
		return nil
	}

	// get structs for zenrpc
	for _, decl := range f.Decls {
		gdecl, ok := decl.(*ast.GenDecl)
		if !ok || gdecl.Tok != token.TYPE {
			continue
		}

		for _, spec := range gdecl.Specs {
			spec, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}

			if !ast.IsExported(spec.Name.Name) {
				continue
			}

			structType, ok := spec.Type.(*ast.StructType)
			if !ok {
				continue
			}

			// check that struct is our zenrpc struct
			if hasZenrpcComment(spec) || hasZenrpcService(structType) {
				pi.Services[spec.Name.Name] = service{gdecl, spec.Name.Name, make(map[string]*method)}
			}
		}
	}

	// get funcs for structs
	for _, decl := range f.Decls {
		fdecl, ok := decl.(*ast.FuncDecl)
		if !ok || fdecl.Recv == nil {
			continue
		}

		method := method{
			FuncDecl:      fdecl.Type,
			Name:          fdecl.Name.Name,
			LowerCaseName: strings.ToLower(fdecl.Name.Name),
			Args:          []arg{},
		}

		for _, field := range fdecl.Recv.List {
			// field can be pointer or not
			var ident *ast.Ident
			if starExpr, ok := field.Type.(*ast.StarExpr); ok {
				if ident, ok = starExpr.X.(*ast.Ident); !ok {
					continue
				}
			} else if ident, ok = field.Type.(*ast.Ident); !ok {
				continue
			}

			// find service in our service list
			// method can be in several services
			if _, ok := pi.Services[ident.Name]; !ok {
				continue
			}

			if !ast.IsExported(fdecl.Name.Name) {
				continue
			}

			pi.Services[ident.Name].Methods[fdecl.Name.Name] = &method
		}

		// parse arguments
		if fdecl.Type.Params == nil || fdecl.Type.Params.List == nil {
			continue
		}

		for _, field := range fdecl.Type.Params.List {
			if field.Names == nil {
				continue
			}

			// parse type
			typeName := ""
			switch v := field.Type.(type) {
			case *ast.StarExpr:
				// pointer
				typeName += "*" // TODO not implemented
			case *ast.SelectorExpr:
				// struct
				x, ok := v.X.(*ast.Ident)
				if ok && v.Sel != nil { // TODO check it
					typeName = x.Name + "." + v.Sel.Name
				} else {
					continue
				}
			case *ast.Ident:
				// basic types
				typeName = v.Name
			default:
				continue
			}

			if typeName == contextTypeName {
				method.HasContext = true
				continue // not add context to arg list
			}

			// parse names
			for _, name := range field.Names {
				method.Args = append(method.Args, arg{
					Name:        name.Name,
					Type:        typeName,
					CapitalName: strings.Title(name.Name),
					JsonName:    lowerFirst(name.Name),
				})
			}
		}
	}

	return nil
}

func hasZenrpcComment(spec *ast.TypeSpec) bool {
	if spec.Comment != nil && len(spec.Comment.List) > 0 && spec.Comment.List[0].Text == zenrpcComment {
		return true
	}

	return false
}

func hasZenrpcService(structType *ast.StructType) bool {
	if structType.Fields.List == nil {
		return false
	}

	for _, field := range structType.Fields.List {
		selectorExpr, ok := field.Type.(*ast.SelectorExpr)
		if !ok {
			continue
		}

		x, ok := selectorExpr.X.(*ast.Ident)
		if ok && selectorExpr.Sel != nil && x.Name+"."+selectorExpr.Sel.Name == zenrpcService {
			return true
		}
	}

	return false
}

func lowerFirst(s string) string {
	if s == "" {
		return ""
	}
	r, n := utf8.DecodeRuneInString(s)
	return string(unicode.ToLower(r)) + s[n:]
}
