package dox

import (
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
)

func LoadPackages(path string) (PackagesDox, error) {
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, path, nil, parser.ParseComments)
	if err != nil {
		return PackagesDox{}, err
	}

	var pkgList []ast.Package
	for _, v := range pkgs {
		pkgList = append(pkgList, *v)
	}

	return NewPackagesDox(pkgList), nil
}

type PackagesDox struct {
	Packages []PackageDox `json:"packages"`
}

func (d PackagesDox) Json() (string, error) {
	out, err := json.Marshal(&d)
	return string(out), err
}

func NewPackagesDox(pkgs []ast.Package) PackagesDox {
	var dox []PackageDox

	for _, pkg := range pkgs {
		dox = append(dox, NewPackageDox(pkg))
	}

	return PackagesDox{dox}
}

type PackageDox struct {
	Name  string    `json:"name"`
	Decls []DeclDox `json:"decls"`
	Files []FileDox `json:"files"`
}

func NewPackageDox(pkg ast.Package) PackageDox {
	var decls []DeclDox
	var files []FileDox

	for _, file := range pkg.Files {
		files = append(files, NewFileDox(*file))

		for _, decl := range file.Decls {
			d, ok := NewDeclDox(decl)
			if !ok {
				continue
			}

			decls = append(decls, d)
		}
	}

	return PackageDox{
		Name:  pkg.Name,
		Decls: decls,
		Files: files,
	}
}

type FileDox struct {
	Name string            `json:"name"`
	Doc  *ast.CommentGroup `json:"doc"`
}

func NewFileDox(file ast.File) FileDox {
	return FileDox{
		Name: file.Name.Name,
		Doc:  file.Doc,
	}
}

type DeclDox struct {
	FuncDecl *FuncDox `json:"func"`
}

func NewDeclDox(decl ast.Decl) (DeclDox, bool) {
	switch decl := decl.(type) {
	case *ast.FuncDecl:
		dox := NewFuncDox(*decl)

		return DeclDox{
			FuncDecl: &dox,
		}, decl.Name.IsExported()
	default:
		return DeclDox{}, false
	}
}

type FuncDox struct {
	Name     string            `json:"name"`
	Doc      *ast.CommentGroup `json:"doc"`
	Recv     *ast.FieldList    `json:"recv"`
	FuncType ast.FuncType      `json:"type"`
}

func NewFuncDox(decl ast.FuncDecl) FuncDox {
	return FuncDox{
		Name:     decl.Name.Name,
		Doc:      decl.Doc,
		Recv:     decl.Recv,
		FuncType: *decl.Type,
	}
}
