package dox

import (
	"encoding/json"
	"fmt"
	"github.com/myuon/godox/parserExtra"
	"go/ast"
	"go/parser"
	"go/token"
)

func LoadPackages(path string) (PackagesDox, error) {
	fset := token.NewFileSet()
	pkgs, err := parserExtra.ParseDirRecursively(fset, path, nil, parser.ParseComments)
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
	out, err := json.MarshalIndent(&d, "", "  ")
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
	Name string `json:"name"`
	Doc  string `json:"doc,omitempty"`
}

func NewFileDox(file ast.File) FileDox {
	return FileDox{
		Name: file.Name.Name,
		Doc:  file.Doc.Text(),
	}
}

type DeclDox struct {
	FuncDecl *FuncDox     `json:"func,omitempty"`
	VarGroup *VarGroupDox `json:"var_group,omitempty"`
}

type VarGroupDox struct {
	Doc   string   `json:"doc,omitempty"`
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
	Name       string    `json:"name"`
	Doc        string    `json:"doc,omitempty"`
	RecvType   *TypeDox  `json:"recv_type,omitempty"`
	ParamTypes []TypeDox `json:"param_types,omitempty"`
}

func NewFuncDox(decl ast.FuncDecl) (FuncDox, error) {
	recv, err := (func() (*TypeDox, error) {
		if decl.Recv != nil && len(decl.Recv.List) > 0 {
			typ, err := NewTypeDox(decl.Recv.List[0].Type)
			if err != nil {
				return nil, err
			}

			return &typ, err
		}

		return nil, nil
	})()
	if err != nil {
		return FuncDox{}, err
	}

	var params []TypeDox
	for _, field := range decl.Type.Params.List {
		ty, err := NewTypeDox(field.Type)
		if err != nil {
			return FuncDox{}, err
		}

		params = append(params, ty)
	}

	return FuncDox{
		Name:       decl.Name.Name,
		Doc:        decl.Doc.Text(),
		RecvType:   recv,
		ParamTypes: params,
	}, nil
}

type VarDox struct {
	Doc   string   `json:"doc,omitempty"`
	Names []string `json:"names"`
	Type  *TypeDox `json:"type,omitempty"`
}

func NewVarDox(spec ast.ValueSpec) (VarDox, error) {
	var names []string
	for _, name := range spec.Names {
		if !name.IsExported() {
			continue
		}

		names = append(names, name.Name)
	}

	typ, err := func() (*TypeDox, error) {
		if spec.Type == nil {
			return nil, nil
		}

		typ, err := NewTypeDox(spec.Type)
		if err != nil {
			return nil, err
		}

		return &typ, nil
	}()
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
	Ident        *string          `json:"ident,omitempty"`
	ArrayType    *TypeDox         `json:"array,omitempty"`
	SelectorType *SelectorTypeDox `json:"selector,omitempty"`
	PointerType  *TypeDox         `json:"pointer,omitempty"`
}

type SelectorTypeDox struct {
	Expr   TypeDox `json:"expr"`
	Select string  `json:"select"`
}

func NewTypeDox(expr ast.Expr) (TypeDox, error) {
	switch expr := expr.(type) {
	case *ast.Ident:
		val := expr.Name

		return TypeDox{
			Ident: &val,
		}, nil
	case *ast.ArrayType:
		val, err := NewTypeDox(expr.Elt)
		if err != nil {
			return TypeDox{}, err
		}

		return TypeDox{
			ArrayType: &val,
		}, nil
	case *ast.SelectorExpr:
		body, err := NewTypeDox(expr.X)
		if err != nil {
			return TypeDox{}, err
		}

		sel := SelectorTypeDox{
			Expr:   body,
			Select: expr.Sel.Name,
		}

		return TypeDox{
			SelectorType: &sel,
		}, nil
	case *ast.StarExpr:
		val, err := NewTypeDox(expr.X)
		if err != nil {
			return TypeDox{}, err
		}

		return TypeDox{
			PointerType: &val,
		}, nil
	default:
		return TypeDox{}, fmt.Errorf("Unsupported expr: %+v", expr)
	}
}
