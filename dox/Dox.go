package dox

import (
	"encoding/json"
	"fmt"
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

	return NewPackagesDox(pkgList)
}

type PackagesDox struct {
	Packages []PackageDox `json:"packages"`
}

func (d PackagesDox) Json() (string, error) {
	out, err := json.Marshal(&d)
	return string(out), err
}

func NewPackagesDox(pkgs []ast.Package) (PackagesDox, error) {
	var dox []PackageDox

	for _, pkg := range pkgs {
		p, err := NewPackageDox(pkg)
		if err != nil {
			return PackagesDox{}, err
		}

		dox = append(dox, p)
	}

	return PackagesDox{dox}, nil
}

type PackageDox struct {
	Name  string    `json:"name"`
	Decls []DeclDox `json:"decls"`
	Files []FileDox `json:"files"`
}

func NewPackageDox(pkg ast.Package) (PackageDox, error) {
	var decls []DeclDox
	var files []FileDox

	for _, file := range pkg.Files {
		files = append(files, NewFileDox(*file))

		for _, decl := range file.Decls {
			d, ok, err := NewDeclDox(decl)
			if err != nil {
				return PackageDox{}, err
			}

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
	}, nil
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
	FuncDecl *FuncDox     `json:"func"`
	VarGroup *VarGroupDox `json:"var_group"`
}

type VarGroupDox struct {
	Doc   string   `json:"doc"`
	Items []VarDox `json:"items"`
}

func NewDeclDox(decl ast.Decl) (DeclDox, bool, error) {
	switch decl := decl.(type) {
	case *ast.FuncDecl:
		dox, err := NewFuncDox(*decl)
		if err != nil {
			return DeclDox{}, false, err
		}

		return DeclDox{
			FuncDecl: &dox,
		}, decl.Name.IsExported(), nil
	case *ast.GenDecl:
		if decl.Tok.String() == "var" {
			var vars []VarDox

			for _, spec := range decl.Specs {
				vr, err := NewVarDox(*spec.(*ast.ValueSpec))
				if err != nil {
					return DeclDox{}, false, err
				}

				vars = append(vars, vr)
			}

			group := VarGroupDox{
				Doc:   decl.Doc.Text(),
				Items: vars,
			}

			return DeclDox{
				VarGroup: &group,
			}, true, nil
		}

		return DeclDox{}, false, nil
	default:
		return DeclDox{}, false, nil
	}
}

type FuncDox struct {
	Name     string   `json:"name"`
	Doc      string   `json:"doc"`
	RecvType *TypeDox `json:"recv_type"`
	//FuncType ast.FuncType `json:"type"`
}

func NewFuncDox(decl ast.FuncDecl) (FuncDox, error) {
	recv := new(TypeDox)
	if decl.Recv != nil && len(decl.Recv.List) > 0 {
		typ, err := NewTypeDox(decl.Recv.List[0].Type)
		if err != nil {
			return FuncDox{}, err
		}

		recv = &typ
	}

	return FuncDox{
		Name:     decl.Name.Name,
		Doc:      decl.Doc.Text(),
		RecvType: recv,
		//FuncType: *decl.Type,
	}, nil
}

type VarDox struct {
	Doc   string   `json:"doc"`
	Names []string `json:"names"`
	Type  TypeDox  `json:"type"`
}

func NewVarDox(spec ast.ValueSpec) (VarDox, error) {
	var names []string
	for _, name := range spec.Names {
		if !name.IsExported() {
			continue
		}

		names = append(names, name.Name)
	}

	typ, err := NewTypeDox(spec.Type)
	if err != nil {
		return VarDox{}, err
	}

	return VarDox{
		Doc:   spec.Doc.Text(),
		Names: names,
		Type:  typ,
	}, nil
}

type TypeDox struct {
	Ident *string `json:"ident"`
}

func NewTypeDox(expr ast.Expr) (TypeDox, error) {
	switch expr := expr.(type) {
	case *ast.Ident:
		val := expr.Name

		return TypeDox{
			Ident: &val,
		}, nil
	default:
		return TypeDox{}, fmt.Errorf("Unsupported expr: %+v", expr)
	}
}
