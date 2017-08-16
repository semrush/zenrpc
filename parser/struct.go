package parser

import (
	"go/ast"
)

func (pi *PackageInfo) fillStructs() {
	for _, s := range pi.Structs {
		if s.TypeSpec != nil {
			continue
		}

		for _, f := range pi.Scopes[s.Namespace] {
			if obj, ok := f.Objects[s.Type]; ok && obj.Decl != nil {
				if ts, ok := obj.Decl.(*ast.TypeSpec); ok {
					s.TypeSpec = ts
					break
				}
			}
		}
	}
}

func (pi *PackageInfo) parseStructs() {
	for _, s := range pi.Structs {
		s.parse()
	}
}

func (s *Struct) parse() {
	//
}
