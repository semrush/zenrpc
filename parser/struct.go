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

func (s *Struct) findTypeSpec(pi *PackageInfo) bool {
	if s.StructType != nil {
		return true
	}

	for _, f := range pi.Scopes[s.Namespace] {
		if obj, ok := f.Objects[s.Type]; ok && obj.Decl != nil {
			if ts, ok := obj.Decl.(*ast.TypeSpec); ok {
				if st, ok := ts.Type.(*ast.StructType); ok {
					s.StructType = st
					return true
				}
			}
		}
	}

	return false
}

func (s *Struct) parse(pi *PackageInfo) error {
	if !s.findTypeSpec(pi) || s.Properties != nil {
		// can't find struct implementation
		// or struct already parsed
		return nil
	}

	s.Properties = []Property{}
	for _, field := range s.StructType.Fields.List {
		if field.Names == nil {
			continue
		}

		smdType, itemType := parseSMDType(field.Type)

		var ref string
		// field with struct type
		if internalS := parseStruct(field.Type); internalS != nil {
			// set right namespace for struct from another package
			if internalS.Namespace == "." && s.Namespace != "." {
				internalS.Namespace = s.Namespace
				internalS.Name = s.Namespace + "." + internalS.Type
			}

			ref = internalS.Name
			if currentS, ok := pi.Structs[internalS.Name]; !ok || currentS.StructType != nil {
				pi.Structs[internalS.Name] = internalS
			}

			if err := internalS.parse(pi); err != nil {
				return err
			}
		}

		// parse inline struct
		if inlineStructType, ok := field.Type.(*ast.StructType); ok {
			// call struct by first property name
			inlineS := &Struct{
				Name:       s.Name + "_" + field.Names[0].Name,
				Namespace:  s.Namespace,
				Type:       s.Type + "_" + field.Names[0].Name,
				StructType: inlineStructType,
			}

			pi.Structs[inlineS.Name] = inlineS
			ref = inlineS.Name
			if err := inlineS.parse(pi); err != nil {
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
