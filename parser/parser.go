package parser

import (
	"fmt"
	"go/ast"
	"go/token"
	"path/filepath"
	"regexp"
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
	errorTypeName     = "zenrpc.Error"
	testFileSuffix    = "_test.go"
	goFileSuffix      = ".go"
	zenrpcMagicPrefix = "//zenrpc:"
)

var errorCommentRegexp = regexp.MustCompile("^(-?\\d+)\\s*(.*)$")
var returnCommentRegexp = regexp.MustCompile("return\\s*(.*)")
var argumentCommentRegexp = regexp.MustCompile("([^=( ]+)\\s*(\\(\\s*([^ )]+)\\s*\\))?(\\s*=\\s*((`([^`]+)`)|([^ ]+)))?\\s*(.*)")

// PackageInfo represents struct info for XXX_zenrpc.go file generation
type PackageInfo struct {
	EntryPoint  string
	Dir         string
	PackageName string
	PackagePath string

	Services []*Service

	Scopes  map[string][]*ast.Scope // key - import name, value - array of scopes from each package file
	Structs map[string]*Struct
	Imports []*ast.ImportSpec

	PackageNamesAndAliasesUsedInServices map[string]struct{} // set of structs names from arguments for printing imports
	ImportsIncludedToGeneratedCode       []*ast.ImportSpec
}

type Service struct {
	GenDecl     *ast.GenDecl
	Name        string
	Methods     []*Method
	Description string
}

type Method struct {
	FuncDecl      *ast.FuncType
	Name          string
	LowerCaseName string
	HasContext    bool
	Args          []Arg
	DefaultValues map[string]DefaultValue
	Returns       []Return
	SMDReturn     *SMDReturn // return for generate smd schema; pointer for nil check
	Description   string

	Errors []SMDError // errors for documentation in SMD
}

type DefaultValue struct {
	Name        string
	CapitalName string
	Type        string // without star
	Comment     string // original comment
	Value       string
}

type Arg struct {
	Name            string
	Type            string
	CapitalName     string
	JsonName        string
	HasStar         bool
	HasDefaultValue bool
	Description     string // from magic comment
	SMDType         SMDType
}

type Return struct {
	Name string
	Type string
}

type SMDReturn struct {
	Name        string
	HasStar     bool
	Description string
	SMDType     SMDType
}

type Struct struct {
	Name       string // key in map, Ref in arguments and returns
	Namespace  string
	Type       string
	StructType *ast.StructType
	Properties []Property // array because order is important
}

type Property struct {
	Name        string
	Description string
	SMDType     SMDType
}

// SMDType is a type representation for SMD generation
type SMDType struct {
	Type      string
	ItemsType string // for array
	Ref       string // for object and also if array item is object
}

type SMDError struct {
	Code        int
	Description string
}

func NewPackageInfo(filename string) (*PackageInfo, error) {
	dir, err := filepath.Abs(filepath.Dir(filename))
	if err != nil {
		return nil, err
	}

	packageName, packagePath, err := EntryPointPackageName(filename)
	if err != nil {
		return nil, err
	}

	return &PackageInfo{
		EntryPoint:  filename,
		Dir:         dir,
		PackageName: packageName,
		PackagePath: packagePath,
		Services:    []*Service{},

		Scopes:  make(map[string][]*ast.Scope),
		Structs: make(map[string]*Struct),
		Imports: []*ast.ImportSpec{},

		PackageNamesAndAliasesUsedInServices: make(map[string]struct{}),
		ImportsIncludedToGeneratedCode:       []*ast.ImportSpec{},
	}, nil
}

// ParseFiles parse all files associated with package from original file
func (pi *PackageInfo) Parse(filename string) error {
	pfs, err := GetDependenciesAstFiles(filename)
	if err != nil {
		return err
	}

	for _, pkg := range pfs {
		for _, astFile := range pkg.AstFiles {
			if pkg.PackagePath == pi.PackagePath {
				// get structs for zenrpc only for root package
				pi.collectServices(astFile)
			}
			// collect scopes
			pi.collectScopes(astFile)
			// get imports
			pi.collectImports(astFile)
		}
	}

	// second loop: parse methods. It runs in separate loop because we need all services to be collected for this parsing
	for _, pkg := range pfs {
		for _, f := range pkg.AstFiles {
			if err := pi.parseMethods(f); err != nil {
				return err
			}
		}
	}

	// collect imports for generated code - only include imports that are explicitly imported in service code (all imports with definitions are more)
	pi.collectImportsForGeneratedCode()

	pi.parseStructs()

	return nil
}

func (pi *PackageInfo) collectScopes(astFile *ast.File) {
	if pi.PackageName != astFile.Name.Name {
		pi.Scopes[astFile.Name.Name] = append(pi.Scopes[astFile.Name.Name], astFile.Scope) // collect other package scopes
	} else {
		pi.Scopes["."] = append(pi.Scopes["."], astFile.Scope) // collect current package scopes
	}
}

func (pi *PackageInfo) collectImports(astFile *ast.File) {
	pi.Imports = append(pi.Imports, astFile.Imports...) // collect imports
}

func (pi *PackageInfo) collectImportsForGeneratedCode() {
	// collect scopes from imported packages
	pi.ImportsIncludedToGeneratedCode = filterImports(uniqueImports(pi.Imports), pi.PackageNamesAndAliasesUsedInServices)
}

func (pi *PackageInfo) collectServices(f *ast.File) {
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
			Args:          []Arg{},
			DefaultValues: make(map[string]DefaultValue),
			Returns:       []Return{},
			Description:   parseCommentGroup(fdecl.Doc),
			Errors:        []SMDError{},
		}

		serviceNames := m.linkWithServices(pi, fdecl)

		// services not found
		if len(serviceNames) == 0 {
			continue
		}

		if err := m.parseArguments(pi, fdecl, serviceNames); err != nil {
			return err
		}

		if err := m.parseReturns(pi, fdecl, serviceNames); err != nil {
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

func (pi PackageInfo) OutputFilename() string {
	return filepath.Join(pi.Dir, pi.PackageName+GenerateFileSuffix)
}

// HasErrorVariable define adding err variable to generated Invoke function
func (s Service) HasErrorVariable() bool {
	for _, m := range s.Methods {
		if len(m.Args) > 0 {
			return true
		}
	}
	return false
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

func (m *Method) parseArguments(pi *PackageInfo, fdecl *ast.FuncDecl, serviceNames []string) error {
	if fdecl.Type.Params == nil || fdecl.Type.Params.List == nil {
		return nil
	}

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
			return fmt.Errorf("Can't parse type of argument %s in %s", strings.Join(fields, ", "), strings.Join(methods, ", "))
		}

		if typeName == contextTypeName {
			m.HasContext = true
			continue // not add context to arg list
		}

		hasStar := hasStar(typeName) // check for pointer
		smdType, itemType := parseSMDType(field.Type)

		// find and save struct
		s := parseStruct(field.Type)
		var ref string
		if s != nil {
			ref = s.Name

			// collect namespaces (imports)
			if s.Namespace != "" {
				if _, ok := pi.PackageNamesAndAliasesUsedInServices[s.Namespace]; !ok {
					pi.PackageNamesAndAliasesUsedInServices[s.Namespace] = struct{}{}
				}
			}

			if currentS, ok := pi.Structs[s.Name]; !ok || (currentS.StructType == nil && s.StructType != nil) {
				pi.Structs[s.Name] = s
			}
		}

		// parse names
		for _, name := range field.Names {
			m.Args = append(m.Args, Arg{
				Name:        name.Name,
				Type:        typeName,
				CapitalName: strings.Title(name.Name),
				JsonName:    lowerFirst(name.Name),
				HasStar:     hasStar,
				SMDType: SMDType{
					Type:      smdType,
					ItemsType: itemType,
					Ref:       ref,
				},
			})
		}
	}

	return nil
}

func (m *Method) parseReturns(pi *PackageInfo, fdecl *ast.FuncDecl, serviceNames []string) error {
	if fdecl.Type.Results == nil || fdecl.Type.Results.List == nil {
		return nil
	}

	// get Service.Method list
	methods := func() string {
		methods := []string{}
		for _, s := range serviceNames {
			methods = append(methods, s+"."+m.Name)
		}
		return strings.Join(methods, ", ")
	}

	hasError := false
	for _, field := range fdecl.Type.Results.List {
		if len(field.Names) > 1 {
			return fmt.Errorf("%s contain more than one return arguments with same type", methods())
		}

		// parse type
		typeName := parseType(field.Type)
		if typeName == "" {
			return fmt.Errorf("Can't parse type of return value in %s on position %d", methods(), len(m.Returns)+1)
		}

		var fieldName string
		// get names if exist
		if field.Names != nil {
			fieldName = field.Names[0].Name
		}

		m.Returns = append(m.Returns, Return{
			Type: typeName,
			Name: fieldName,
		})

		if typeName == "error" || typeName == errorTypeName || typeName == "*"+errorTypeName {
			if hasError {
				return fmt.Errorf("%s contain more than one error return arguments", methods())
			}
			hasError = true
			continue
		}

		if m.SMDReturn != nil {
			return fmt.Errorf("%s contain more than one variable return argument", methods())
		}

		hasStar := hasStar(typeName) // check for pointer
		smdType, itemType := parseSMDType(field.Type)

		// find and save struct
		s := parseStruct(field.Type)
		var ref string
		if s != nil {
			ref = s.Name

			if currentS, ok := pi.Structs[s.Name]; !ok || (currentS.StructType == nil && s.StructType != nil) {
				pi.Structs[s.Name] = s
			}
		}

		m.SMDReturn = &SMDReturn{
			Name:    fieldName,
			HasStar: hasStar,
			SMDType: SMDType{
				Type:      smdType,
				ItemsType: itemType,
				Ref:       ref,
			},
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

		line := strings.TrimPrefix(strings.TrimSpace(comment.Text), zenrpcMagicPrefix)
		switch parseCommentType(line) {
		case "argument":
			name, alias, hasDefault, defaultValue, description := parseArgumentComment(line)
			for i, a := range m.Args {
				if a.Name == name {
					m.Args[i].Description = description

					if hasDefault {
						m.DefaultValues[name] = DefaultValue{
							Name:        name,
							CapitalName: a.CapitalName,
							Type:        strings.TrimPrefix(a.Type, "*"), // remove star
							Comment:     comment.Text,
							Value:       defaultValue,
						}

						m.Args[i].HasDefaultValue = true
					}

					if alias != "" {
						m.Args[i].JsonName = alias
					}
				}
			}
		case "return":
			m.SMDReturn.Description = parseReturnComment(line)
		case "error":
			code, description := parseErrorComment(line)
			m.Errors = append(m.Errors, SMDError{code, description})
		}
	}
}

func parseCommentType(line string) string {
	if strings.HasPrefix(line, "return") {
		return "return"
	}

	if errorCommentRegexp.MatchString(line) {
		return "error"
	}

	return "argument"
}

func parseReturnComment(line string) string {
	matches := returnCommentRegexp.FindStringSubmatch(line)
	if len(matches) < 2 {
		return ""
	}

	return matches[1]
}

func parseErrorComment(line string) (int, string) {
	matches := errorCommentRegexp.FindStringSubmatch(line)
	if len(matches) < 3 {
		// should not be here
		return 0, ""
	}

	code, err := strconv.Atoi(matches[1])
	if err != nil {
		return 0, ""
	}

	return code, matches[2]
}

func parseArgumentComment(line string) (name, alias string, hasDefault bool, defaultValue, description string) {
	matches := argumentCommentRegexp.FindStringSubmatch(line)

	if len(matches) < 10 {
		return
	}

	// name index = 1
	name = matches[1]
	// alias index = 3
	alias = matches[3]
	// has default index = 4
	hasDefault = matches[4] != ""
	// default index = 5
	defaultValue = matches[5]
	// default quoted index = 7 can override non quoted string
	if matches[7] != "" {
		defaultValue = matches[7]
	}
	// description index = 9
	description = strings.TrimSpace(matches[9])

	return
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

// Returned value will be used as smd.{Value} variable from smd package
func parseSMDType(expr ast.Expr) (string, string) {
	switch v := expr.(type) {
	case *ast.StarExpr:
		return parseSMDType(v.X)
	case *ast.SelectorExpr, *ast.MapType, *ast.InterfaceType:
		return "Object", ""
	case *ast.ArrayType:
		mainType, itemType := parseSMDType(v.Elt)
		if itemType != "" {
			return "Array", itemType
		}

		return "Array", mainType
	case *ast.Ident:
		switch v.Name {
		case "bool":
			return "Boolean", ""
		case "string":
			return "String", ""
		case "int", "int8", "int16", "int32", "int64", "uint", "uint8", "uint16", "uint32", "uint64", "uintptr", "byte", "rune":
			return "Integer", ""
		case "float32", "float64", "complex64", "complex128":
			return "Float", ""
		default:
			return "Object", "" // *ast.Ident contain type name, if type not basic then it struct or alias
		}
	default:
		return "Object", "" // default complex type is object
	}
}

// parseStruct find struct in type for display in SMD
func parseStruct(expr ast.Expr) *Struct {
	switch v := expr.(type) {
	case *ast.StarExpr:
		return parseStruct(v.X)
	case *ast.SelectorExpr:
		namespace := v.X.(*ast.Ident).Name
		return &Struct{
			Name:      namespace + "." + v.Sel.Name,
			Namespace: namespace,
			Type:      v.Sel.Name,
		}
	case *ast.ArrayType:
		// will get last type
		return parseStruct(v.Elt)
	case *ast.MapType:
		// will get last type
		return parseStruct(v.Value)
	case *ast.Ident:
		switch v.Name {
		case "bool", "string",
			"int", "int8", "int16", "int32", "int64", "uint", "uint8", "uint16", "uint32", "uint64", "uintptr", "byte", "rune",
			"float32", "float64", "complex64", "complex128":
			return nil
		}

		s := &Struct{
			Name:      v.Name,
			Namespace: ".",
			Type:      v.Name,
		}

		if v.Obj != nil && v.Obj.Decl != nil {
			if ts, ok := v.Obj.Decl.(*ast.TypeSpec); ok {
				if st, ok := ts.Type.(*ast.StructType); ok {
					s.StructType = st
				}
			}
		}

		return s
	default:
		return nil
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
