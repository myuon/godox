package astwrapper

import (
	"go/ast"
	"go/parser"
	"go/token"
)

type Package struct {
	ast.Package
}

func (pkg Package) Files() []File {
	var files []File
	for _, file := range pkg.Package.Files {
		files = append(files, File{*file})
	}

	return files
}

func (pkg Package) GetFuncDecls() []ast.FuncDecl {
	var decls []ast.FuncDecl
	for _, v := range pkg.Package.Files {
		decls = append(decls, File{*v}.GetFuncDecls()...)
	}

	return decls
}

func (pkg Package) GetTypeSpecs() []ast.TypeSpec {
	var specs []ast.TypeSpec
	for _, v := range pkg.Package.Files {
		specs = append(specs, File{*v}.GetTypeSpecs()...)
	}

	return specs
}

func (pkg Package) GetValueGroups() []ValueGroup {
	var groups []ValueGroup
	for _, v := range pkg.Package.Files {
		groups = append(groups, File{*v}.GetValueGroups()...)
	}

	return groups
}

type Packages []Package

func LoadPackages(path string) (Packages, error) {
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, path, nil, parser.ParseComments)
	if err != nil {
		return Packages{}, err
	}

	var pkgList []Package
	for _, v := range pkgs {
		pkgList = append(pkgList, Package{*v})
	}

	return Packages(pkgList), nil
}

func (pkgs Packages) CollectTypes() map[string]string {
	typs := map[string]string{}
	for _, pkg := range pkgs {
		for _, file := range pkg.Files() {
			for _, t := range file.GetTypeSpecs() {
				typs[t.Name.String()] = file.Name.String()
			}
		}
	}

	return typs
}
