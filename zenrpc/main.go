package main

import (
	"bytes"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"log"
	"os"
	"path/filepath"
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	zenrpcComment       = "//zenrpc"
	zenrpcService       = "zenrpc.Service"
	contextTypeName     = "context.Context"
	generateFilePostfix = "_zenrpc.go"
)

func main() {
	var filename string
	if len(os.Args) > 1 {
		filename = os.Args[len(os.Args)-1]
	} else {
		filename = os.Getenv("GOFILE")
	}

	log.Printf("Entrypoint: %s", filename)
	structData, err := parse(filename)
	die(err)

	dir, err := filepath.Abs(filepath.Dir(filename))
	die(err)

	outputFileName := filepath.Join(dir, structData.PackageName+generateFilePostfix)
	file, err := os.Create(outputFileName)
	die(err)
	defer file.Close()

	output := new(bytes.Buffer)
	die(serviceTemplate.Execute(output, structData))

	source, err := format.Source(output.Bytes())
	die(err)

	_, err = file.Write(source)
	die(err)

	log.Printf("Generated: %s", outputFileName)
}

func die(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

// StructData represents struct info for XXX_zenrpc.go file generation.
type StructData struct {
	PackageName string
	Services    map[string]Service
}

type Service struct {
	GenDecl *ast.GenDecl
	Name    string
	Methods map[string]*Method
}

type Method struct {
	FuncDecl      *ast.FuncType
	Name          string
	LowerCaseName string
	HasContext    bool
	Args          []Arg
}

type Arg struct {
	Name        string
	Type        string
	CapitalName string
	JsonName    string
}

func parse(filename string) (StructData, error) {
	data := StructData{}
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filename, nil, parser.ParseComments)
	if err != nil {
		return data, err
	}
	//ast.Print(fset, f) // TODO remove

	data.PackageName = f.Name.Name

	// get structs for zenrpc
	data.Services = make(map[string]Service)
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
				data.Services[spec.Name.Name] = Service{gdecl, spec.Name.Name, make(map[string]*Method)}
			}
		}
	}

	// get funcs for structs
	for _, decl := range f.Decls {
		fdecl, ok := decl.(*ast.FuncDecl)
		if !ok || fdecl.Recv == nil {
			continue
		}

		method := Method{
			FuncDecl:      fdecl.Type,
			Name:          fdecl.Name.Name,
			LowerCaseName: strings.ToLower(fdecl.Name.Name),
			Args:          []Arg{},
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
			if _, ok := data.Services[ident.Name]; !ok {
				continue
			}

			if !ast.IsExported(fdecl.Name.Name) {
				continue
			}

			data.Services[ident.Name].Methods[fdecl.Name.Name] = &method
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
				method.Args = append(method.Args, Arg{
					Name:        name.Name,
					Type:        typeName,
					CapitalName: strings.Title(name.Name),
					JsonName:    lowerFirst(name.Name),
				})
			}
		}
	}

	return data, nil
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
