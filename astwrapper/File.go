package astwrapper

import (
	"go/ast"
)

type File struct {
	ast.File
}

func (file File) GetTypeSpecs() []ast.TypeSpec {
	var specs []ast.TypeSpec
	for _, decl := range file.Decls {
		switch decl := decl.(type) {
		case *ast.GenDecl:
			// typeのときはspecは必ず長さ1？
			if decl.Tok.String() == "type" {
				specs = append(specs, *decl.Specs[0].(*ast.TypeSpec))
			}
		default:
			continue
		}
	}

	return specs
}

func (file File) GetFuncDecls() []ast.FuncDecl {
	var decls []ast.FuncDecl
	for _, decl := range file.Decls {
		switch decl := decl.(type) {
		case *ast.FuncDecl:
			decls = append(decls, *decl)
		default:
			continue
		}
	}

	return decls
}

func (file File) GetValueGroups() []ValueGroup {
	var groups []ValueGroup
	for _, decl := range file.Decls {
		switch decl := decl.(type) {
		case *ast.GenDecl:
			if decl.Tok.String() == "var" {
				var specs []ast.ValueSpec
				for _, spec := range decl.Specs {
					specs = append(specs, *spec.(*ast.ValueSpec))
				}

				groups = append(groups, ValueGroup{
					Doc:   decl.Doc,
					Specs: specs,
				})
			}
		default:
			continue
		}
	}

	return groups
}
