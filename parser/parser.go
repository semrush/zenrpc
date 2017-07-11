package parser

import (
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/ioutil"
	"path/filepath"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	GenerateFileSuffix = "_zenrpc.go"

	zenrpcComment     = "//zenrpc"
	zenrpcService     = "zenrpc.Service"
	contextTypeName   = "context.Context"
	testFileSuffix    = "_test.go"
	goFileSuffix      = ".go"
	zenrpcMagicPrefix = "//zenrpc:"
)

// PackageInfo represents struct info for XXX_zenrpc.go file generation
type PackageInfo struct {
	PackageName string
	Services    []*Service
	Errors      map[int]string // errors map for documentation in SMD
}

type Service struct {
	GenDecl     *ast.GenDecl
	Name        string
	Methods     []*Method
	Description string
}

type Method struct {
	FuncDecl          *ast.FuncType
	Name              string
	LowerCaseName     string
	HasContext        bool
	Args              []*Arg
	DefaultValues     []*DefaultValue
	Returns           []*Return
	ReturnDescription string
	Description       string
}

type DefaultValue struct {
	Name        string
	CapitalName string
	Type        string // without star
	Comment     string // original comment
	Value       string
}

type Arg struct {
	Name        string
	Type        string
	CapitalName string
	JsonName    string
	HasStar     bool
	Description string // from magic comment
	SMDType     string
}

type Return struct {
	Name        string
	Type        string
	HasStar     bool
	SMDType     string
}

// ParseFiles parse all files associated with package from original file
func (pi *PackageInfo) ParseFiles(filename string) (string, error) {
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
			strings.HasSuffix(f.Name(), GenerateFileSuffix) || strings.HasSuffix(f.Name(), testFileSuffix) {
			continue
		}

		if err := pi.parseFile(filepath.Join(dir, f.Name())); err != nil {
			return dir, err
		}
	}

	return dir, nil
}

func (pi *PackageInfo) parseFile(filename string) error {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filename, nil, parser.ParseComments)
	if err != nil {
		return err
	}

	// for debug
	//ast.Print(fset, f)

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

func (pi *PackageInfo) parseServices(f *ast.File) {
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
				pi.Services = append(pi.Services, &Service{
					GenDecl:     gdecl,
					Name:        spec.Name.Name,
					Methods:     []*Method{},
					Description: parseCommentGroup(spec.Doc),
				})
			}
		}
	}
}

func (pi *PackageInfo) parseMethods(f *ast.File) error {
	for _, decl := range f.Decls {
		fdecl, ok := decl.(*ast.FuncDecl)
		if !ok || fdecl.Recv == nil {
			continue
		}

		m := Method{
			FuncDecl:      fdecl.Type,
			Name:          fdecl.Name.Name,
			LowerCaseName: strings.ToLower(fdecl.Name.Name),
			Args:          []*Arg{},
			DefaultValues: []*DefaultValue{},
			Description:   parseCommentGroup(fdecl.Doc),
		}

		serviceNames := m.linkWithServices(pi, fdecl)

		// parse arguments
		if len(serviceNames) == 0 || fdecl.Type.Params == nil || fdecl.Type.Params.List == nil {
			continue
		}

		if err := m.parseArguments(fdecl, serviceNames); err != nil {
			return err
		}

		if err := m.parseReturns(fdecl, serviceNames); err != nil {
			return err
		}

		// parse default values
		m.parseComments(fdecl.Doc, pi)
	}

	return nil
}

func (pi PackageInfo) String() string {
	result := fmt.Sprintf("Generated services for package %s:\n", pi.PackageName)
	for _, s := range pi.Services {
		result += fmt.Sprintf("- %s\n", s.Name)
		for _, m := range s.Methods {
			result += fmt.Sprintf("  • %s", m.Name)

			// args
			result += "("
			for i, a := range m.Args {
				if i != 0 {
					result += ", "
				}

				result += fmt.Sprintf("%s %s", a.Name, a.Type)
			}
			result += ") "

			// no return args
			if len(m.Returns) == 0 {
				result += "\n"
				continue
			}

			// only one return arg without name
			if len(m.Returns) == 1 && len(m.Returns[0].Name) == 0 {
				result += m.Returns[0].Type + "\n"
				continue
			}

			// return
			result += "("
			for i, a := range m.Returns {
				if i != 0 {
					result += fmt.Sprintf(", ")
				}

				if len(a.Name) == 0 {
					result += a.Type
				} else {
					result += fmt.Sprintf("%s %s", a.Name, a.Type)
				}
			}
			result += ")\n"
		}
	}

	return result
}

// linkWithServices add method for services
func (m *Method) linkWithServices(pi *PackageInfo, fdecl *ast.FuncDecl) (names []string) {
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
				names = append(names, s.Name)
				s.Methods = append(s.Methods, m)
				break
			}
		}
	}

	return
}

func (m *Method) parseReturns(fdecl *ast.FuncDecl, serviceNames []string) error {
	for _, field := range fdecl.Type.Results.List {
		// parse type
		typeName := parseType(field.Type)
		if typeName == "" {
			// get Service.Method list
			methods := []string{}
			for _, s := range serviceNames {
				methods = append(methods, s+"."+m.Name)
			}
			return errors.New(fmt.Sprintf("Can't parse type of return value in %s on position %d", strings.Join(methods, ", "), len(m.Returns)+1))
		}

		hasStar := hasStar(typeName) // check for pointer
		smdType := parseSMDType(field.Type)

		// parse names if exist and add item to list
		if field.Names == nil {
			m.Returns = append(m.Returns, &Return{
				Type:    typeName,
				HasStar: hasStar,
				SMDType: smdType,
			})
		} else {
			for _, name := range field.Names {
				m.Returns = append(m.Returns, &Return{
					Name:    name.Name,
					Type:    typeName,
					HasStar: hasStar,
					SMDType: smdType,
				})
			}
		}
	}

	return nil
}

func (m *Method) parseArguments(fdecl *ast.FuncDecl, serviceNames []string) error {
	for _, field := range fdecl.Type.Params.List {
		if field.Names == nil {
			continue
		}

		// parse type
		typeName := parseType(field.Type)
		if typeName == "" {
			// get argument names
			fields := []string{}
			for _, name := range field.Names {
				fields = append(fields, name.Name)
			}

			// get Service.Method list
			methods := []string{}
			for _, s := range serviceNames {
				methods = append(methods, s+"."+m.Name)
			}
			return errors.New(fmt.Sprintf("Can't parse type of argument %s in %s", strings.Join(fields, ", "), strings.Join(methods, ", ")))
		}

		if typeName == contextTypeName {
			m.HasContext = true
			continue // not add context to arg list
		}

		hasStar := hasStar(typeName) // check for pointer
		smdType := parseSMDType(field.Type)

		// parse names
		for _, name := range field.Names {
			m.Args = append(m.Args, &Arg{
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
func (m *Method) parseComments(doc *ast.CommentGroup, pi *PackageInfo) {
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
					m.DefaultValues = append(m.DefaultValues, &DefaultValue{
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
	case *ast.InterfaceType:
		return "interface{}"
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
	case *ast.SelectorExpr, *ast.MapType, *ast.InterfaceType:
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
