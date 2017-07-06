package main

import (
	"bytes"
	"errors"
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
	zenrpcMagicPrefix  = "//zenrpc:"
)

func main() {
	var filename string
	if len(os.Args) > 1 {
		filename = os.Args[len(os.Args)-1]
	} else {
		filename = os.Getenv("GOFILE")
	}

	log.Printf("Entrypoint: %s", filename)

	sd := packageInfo{Services: make(map[string]*service)}
	dir, err := sd.parseFiles(filename)
	if err != nil {
		log.Fatal(err)
	}

	if len(sd.Services) == 0 {
		log.Printf("Services not found")
		return
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
	Services    map[string]*service
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
	Args          map[string]*arg
	DefaultValues map[string]*defaultValue
}

type defaultValue struct {
	Name        string
	CapitalName string
	Type        string // without star
	Comment     string // original comment
	Value       string
}

type arg struct {
	Name        string
	Type        string
	CapitalName string
	JsonName    string
	Description string // from comment
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

	if len(files) == 0 {
		return dir, errors.New("Directory is empty")
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
	pi.parseServices(f)

	// get funcs for structs
	if err := pi.parseMethods(f); err != nil {
		return err
	}

	return nil
}

func (pi *packageInfo) parseServices(f *ast.File) {
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
				pi.Services[spec.Name.Name] = &service{gdecl, spec.Name.Name, make(map[string]*method)}
			}
		}
	}
}

func (pi *packageInfo) parseMethods(f *ast.File) error {
	for _, decl := range f.Decls {
		fdecl, ok := decl.(*ast.FuncDecl)
		if !ok || fdecl.Recv == nil {
			continue
		}

		m := method{
			FuncDecl:      fdecl.Type,
			Name:          fdecl.Name.Name,
			LowerCaseName: strings.ToLower(fdecl.Name.Name),
			Args:          make(map[string]*arg),
			DefaultValues: make(map[string]*defaultValue),
		}

		m.linkWithServices(pi, fdecl)

		// parse arguments
		if fdecl.Type.Params == nil || fdecl.Type.Params.List == nil {
			continue
		}

		if err := m.parseArguments(fdecl); err != nil {
			return err
		}

		// parse default values
		m.parseDefaultValues(fdecl.Doc)
	}

	return nil
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

// linkWithServices add method for services
func (m *method) linkWithServices(pi *packageInfo, fdecl *ast.FuncDecl) {
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

		pi.Services[ident.Name].Methods[fdecl.Name.Name] = m
	}
}

func (m *method) parseArguments(fdecl *ast.FuncDecl) error {
	for _, field := range fdecl.Type.Params.List {
		if field.Names == nil {
			continue
		}

		// parse type
		typeName := parseType(field.Type)
		if typeName == "" {
			return errors.New("Can't parse type of argument")
		}

		if typeName == contextTypeName {
			m.HasContext = true
			continue // not add context to arg list
		}

		// parse names
		for _, name := range field.Names {
			m.Args[name.Name] = &arg{
				Name:        name.Name,
				Type:        typeName,
				CapitalName: strings.Title(name.Name),
				JsonName:    lowerFirst(name.Name),
			}
		}
	}

	return nil
}

func (m *method) parseDefaultValues(doc *ast.CommentGroup) {
	if doc == nil {
		return
	}

	for _, comment := range doc.List {
		if !strings.HasPrefix(comment.Text, zenrpcMagicPrefix) {
			continue
		}

		// split by magic path and description
		couple := strings.SplitN(comment.Text, " ", 2)
		if len(couple) == 1 {
			couple = strings.SplitN(comment.Text, "\t", 2)
		}

		couple[0] = strings.TrimPrefix(strings.TrimSpace(couple[0]), zenrpcMagicPrefix)

		// parse arguments
		args := strings.Split(couple[0], ":")
		if len(args) == 2 {
			// default value
			name := args[0]
			value := args[1]

			if _, ok := m.Args[name]; !ok {
				continue
			}

			m.DefaultValues[name] = &defaultValue{
				Name:        name,
				CapitalName: m.Args[name].CapitalName,
				Type:        m.Args[name].Type[1:], // remove star
				Comment:     comment.Text,
				Value:       value,
			}

			if len(couple) == 2 {
				m.Args[name].Description = strings.TrimSpace(couple[1])
			}
		} else {
			// parse error code
		}
	}
}

func parseType(expr ast.Expr) string {
	switch v := expr.(type) {
	case *ast.StarExpr:
		return "*" + parseType(v.X)
	case *ast.SelectorExpr:
		return parseType(v.X) + "." + v.Sel.Name
	case *ast.ArrayType:
		return "[" + parseType(v.Len) + "]" + parseType(v.Elt)
	case *ast.MapType:
		return "map[" + parseType(v.Key) + "]" + parseType(v.Value)
	case *ast.Ident:
		return v.Name
	case *ast.BasicLit:
		// for array size
		return v.Value
	default:
		return ""
	}
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
