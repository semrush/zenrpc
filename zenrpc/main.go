package main

import (
	"bytes"
	"errors"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime/debug"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

const (
	zenrpcComment      = "//zenrpc"
	zenrpcService      = "zenrpc.Service"
	contextTypeName    = "context.Context"
	generateFileSuffix = "_zenrpc.go"
	testFileSuffix     = "_test.go"
	goFileSuffix       = ".go"
	zenrpcMagicPrefix  = "//zenrpc:"

	openIssueURL = "https://github.com/sergeyfast/zenrpc/issues/new"
	githubURL    = "https://github.com/sergeyfast/zenrpc"
)

func main() {
	start := time.Now()
	var filename string
	if len(os.Args) > 1 {
		filename = os.Args[len(os.Args)-1]
	} else {
		filename = os.Getenv("GOFILE")
	}

	fmt.Printf("Entrypoint: %s\n", filename)

	pi := packageInfo{Services: []*service{}, Errors: make(map[int]string)}
	dir, err := pi.parseFiles(filename)
	if err != nil {
		printError(err)
	}

	if len(pi.Services) == 0 {
		fmt.Printf("Services not found")
		return
	}

	outputFileName, err := pi.generateFile(dir)
	if err != nil {
		printError(err)
	}

	fmt.Printf("Generated: %s\n", outputFileName)
	fmt.Printf("Duration: %s\n", time.Since(start))
	pi.PrintInfo()
}

func printError(err error) {
	// print error wish stack trace to stderr
	fmt.Fprintf(os.Stderr, "\nError: %s\n", err)
	fmt.Fprint(os.Stderr, string(debug.Stack()))

	// print contact information to stdout
	fmt.Println("\nYou may help us and create issue:")
	fmt.Printf("\t%s\n", openIssueURL)
	fmt.Println("For more information, see:")
	fmt.Printf("\t%s\n\n", githubURL)

	os.Exit(1)
}

// packageInfo represents struct info for XXX_zenrpc.go file generation
type packageInfo struct {
	PackageName string
	Services    []*service
	Errors      map[int]string // errors map for documentation in SMD
}

type service struct {
	GenDecl     *ast.GenDecl
	Name        string
	Methods     []*method
	Description string
}

type method struct {
	FuncDecl      *ast.FuncType
	Name          string
	LowerCaseName string
	HasContext    bool
	Args          []*arg
	DefaultValues []*defaultValue
	Description   string
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
	HasStar     bool
	Description string // from magic comment
	SMDType     string
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

		if !strings.HasSuffix(f.Name(), goFileSuffix) ||
			strings.HasSuffix(f.Name(), generateFileSuffix) || strings.HasSuffix(f.Name(), testFileSuffix) {
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
	//ast.Print(fset, f) // for debug

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
				pi.Services = append(pi.Services, &service{
					GenDecl:     gdecl,
					Name:        spec.Name.Name,
					Methods:     []*method{},
					Description: parseCommentGroup(spec.Doc),
				})
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
			Args:          []*arg{},
			DefaultValues: []*defaultValue{},
			Description:   parseCommentGroup(fdecl.Doc),
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
		m.parseComments(fdecl.Doc, pi)
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

func (pi packageInfo) PrintInfo() {
	fmt.Printf("\nGenerated services for package %s:\n", pi.PackageName)
	for _, s := range pi.Services {
		fmt.Printf("- %s\n", s.Name)
		for _, m := range s.Methods {
			fmt.Printf("  • %s", m.Name)
			fmt.Printf("(")
			for i, a := range m.Args {
				if i != 0 {
					fmt.Printf(", ")
				}

				fmt.Printf("%s %s", a.Name, a.Type)
			}
			fmt.Printf(")\n")
		}
	}

	fmt.Println()
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

		if !ast.IsExported(fdecl.Name.Name) {
			continue
		}

		// find service in our service list
		// method can be in several services
		for _, s := range pi.Services {
			if s.Name == ident.Name {
				s.Methods = append(s.Methods, m)
				break
			}
		}
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

		hasStar := hasStar(typeName) // check for pointer
		smdType := parseSMDType(field.Type)

		// parse names
		for _, name := range field.Names {
			m.Args = append(m.Args, &arg{
				Name:        name.Name,
				Type:        typeName,
				CapitalName: strings.Title(name.Name),
				JsonName:    lowerFirst(name.Name),
				HasStar:     hasStar,
				SMDType:     smdType,
			})
		}
	}

	return nil
}

// parseComments parse method comments and
// fill default values, description for params and user errors map
func (m *method) parseComments(doc *ast.CommentGroup, pi *packageInfo) {
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

			for _, a := range m.Args {
				if a.Name == name {
					m.DefaultValues = append(m.DefaultValues, &defaultValue{
						Name:        name,
						CapitalName: a.CapitalName,
						Type:        a.Type[1:], // remove star
						Comment:     comment.Text,
						Value:       value,
					})

					if len(couple) == 2 {
						a.Description = strings.TrimSpace(couple[1])
					}

					break
				}
			}
		} else if i, err := strconv.Atoi(args[0]); err == nil && len(couple) == 2 {
			// add error code
			// example: //zenrpc:-32603		divide by zero
			pi.Errors[i] = strings.TrimSpace(couple[1])
		}
	}
}

func parseCommentGroup(doc *ast.CommentGroup) string {
	if doc == nil {
		return ""
	}

	result := ""
	for _, comment := range doc.List {
		if strings.HasPrefix(comment.Text, zenrpcMagicPrefix) {
			continue
		}

		if len(result) > 0 {
			result += "\n"
		}
		result += strings.TrimSpace(strings.TrimPrefix(comment.Text, "//"))
	}

	return result
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

func parseSMDType(expr ast.Expr) string {
	switch v := expr.(type) {
	case *ast.StarExpr:
		return parseSMDType(v.X)
	case *ast.SelectorExpr, *ast.MapType:
		return "Object"
	case *ast.ArrayType:
		return "Array"
	case *ast.Ident:
		switch v.Name {
		case "bool":
			return "Boolean"
		case "string":
			return "String"
		case "int", "int8", "int16", "int32", "int64", "uint", "uint8", "uint16", "uint32", "uint64", "uintptr", "byte", "rune":
			return "Integer"
		case "float32", "float64", "complex64", "complex128":
			return "Float"
		default:
			return "Object" // *ast.Ident contain type name, if type not basic then it struct or alias
		}
	default:
		return "Object" // default complex type is object
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

func hasStar(s string) bool {
	if s[:1] == "*" {
		return true
	}

	return false
}
