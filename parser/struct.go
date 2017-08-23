package parser

import (
	"go/ast"
	"reflect"
	"strings"
)

func (pi *PackageInfo) parseStructs() {
	for _, s := range pi.Structs {
		s.parse(pi)
	}
}

func (s *Struct) findTypeSpec(pi *PackageInfo) {
	if s.TypeSpec != nil {
		return
	}

	for _, f := range pi.Scopes[s.Namespace] {
		if obj, ok := f.Objects[s.Type]; ok && obj.Decl != nil {
			if ts, ok := obj.Decl.(*ast.TypeSpec); ok {
				s.TypeSpec = ts
				return
			}
		}
	}
}

func (s *Struct) parse(pi *PackageInfo) error {
	s.findTypeSpec(pi)

	if s.TypeSpec == nil || s.Properties != nil {
		// can't find struct implementation
		// or struct already parsed
		return nil
	}

	structType, ok := s.TypeSpec.Type.(*ast.StructType)
	if !ok {
		return nil
	}

	s.Properties = []Property{}
	for _, field := range structType.Fields.List {
		if field.Names == nil {
			continue
		}

		smdType, itemType := parseSMDType(field.Type)

		// field with struct type
		internalS := parseStruct(field.Type)
		var ref string
		if internalS != nil {
			// set right namespace for struct from another package
			if internalS.Namespace == "." && s.Namespace != "." {
				internalS.Namespace = s.Namespace
				internalS.Name = s.Namespace + "." + internalS.Type
			}

			ref = internalS.Name
			if currentS, ok := pi.Structs[internalS.Name]; !ok || currentS.TypeSpec != nil {
				pi.Structs[internalS.Name] = internalS
			}

			if err := internalS.parse(pi); err != nil {
				return err
			}
		}

		tag := parseJsonTag(field.Tag)

		// description
		description := parseCommentGroup(field.Doc)
		comment := parseCommentGroup(field.Comment)
		if description != "" && comment != "" {
			description += "\n"
		}
		description += comment

		// parse names
		for i, name := range field.Names {
			p := Property{
				Name:        name.Name,
				Description: description,
				SMDType: SMDType{
					Type:      smdType,
					ItemsType: itemType,
					Ref:       ref,
				},
			}

			if i == 0 && tag != "" {
				// tag only for first name
				p.Name = tag
			}

			s.Properties = append(s.Properties, p)
		}
	}

	return nil
}

func parseJsonTag(bl *ast.BasicLit) string {
	if bl == nil {
		return ""
	}

	tags := bl.Value[1 : len(bl.Value)-1] // remove quotes ``
	tag := strings.Split(reflect.StructTag(tags).Get("json"), ",")[0]
	if tag == "-" {
		tag = ""
	}

	return tag
}
