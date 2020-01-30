package main

import (
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"html/template"
	"net/http"
	"strings"
)

// Define configurations
var (
	// Path to template file
	TemplatePath = "./template/index.html"
)

type TypeWrapper struct {
	ast.Expr
}

func (t *TypeWrapper) Text() (string, error) {
	switch t := t.Expr.(type) {
	case *ast.ArrayType:
		elt := &TypeWrapper{Expr: t.Elt}
		r, err := elt.Text()
		if err != nil {
			return "", err
		}

		return fmt.Sprintf("[]%s", r), nil
	case *ast.Ident:
		return t.Name, nil
	case *ast.StarExpr:
		r, err := (&TypeWrapper{Expr: t.X}).Text()
		if err != nil {
			return "", err
		}

		return fmt.Sprintf("*%s", r), nil
	case *ast.SelectorExpr:
		r, err := (&TypeWrapper{Expr: t.X}).Text()
		if err != nil {
			return "", err
		}

		return fmt.Sprintf("%s.%s", r, t.Sel.Name), nil
	default:
		return "", fmt.Errorf("Not yet implemented: %+v", t)
	}
}

type FieldListWrapper struct {
	*ast.FieldList
}

func (t *FieldListWrapper) Text() ([]string, error) {
	if t == nil || t.FieldList == nil {
		return nil, nil
	}

	var rep []string
	for _, field := range t.List {
		t := TypeWrapper{Expr: field.Type}
		r, err := t.Text()
		if err != nil {
			return nil, err
		}

		if len(field.Names) == 0 {
			rep = append(rep, r)
		} else {
			rep = append(rep, fmt.Sprintf("%s %s", strings.Join(ShowNames(field.Names), ","), r))
		}
	}

	return rep, nil
}

// A helper function that shows names of idents.
func ShowNames(idents []*ast.Ident) []string {
	var names []string
	for _, ident := range idents {
		if ident == nil {
			continue
		}

		names = append(names, ident.Name)
	}

	return names
}

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

type ValueGroup struct {
	Doc   *ast.CommentGroup
	Specs []ast.ValueSpec
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

func prints(pkgs Packages) error {
	fmt.Printf("%+v\n", pkgs.CollectTypes())

	for _, pkg := range pkgs {
		fmt.Printf("\nPackage %s\n=====\n", pkg.Name)

		fmt.Printf("\n\nFunctions\n-----\n")
		for _, decl := range pkg.GetFuncDecls() {
			fmt.Printf("%s\t", decl.Name.String())
		}

		fmt.Printf("\n\nTypes\n-----\n")
		for _, spec := range pkg.GetTypeSpecs() {
			fmt.Printf("%s\t", spec.Name.String())
		}

		fmt.Printf("\n\nVariables\n-----\n")
		for _, vg := range pkg.GetValueGroups() {
			for _, spec := range vg.Specs {
				fmt.Printf("%+v\n", spec.Names)
			}
		}
	}

	return nil
}

func serves(pkgs Packages) error {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		tpl := template.Must(template.ParseFiles(TemplatePath))

		tpl.Execute(w, map[string]interface{}{
			"Packages": pkgs,
		})
	})
	println("Listening on http://localhost:8080...")

	return http.ListenAndServe(":8080", nil)
}

func run(path string, serveFlag bool) error {
	pkgs, err := LoadPackages(path)
	if err != nil {
		return err
	}

	if serveFlag {
		if err := prints(pkgs); err != nil {
			return err
		}
	} else {
		if err := serves(pkgs); err != nil {
			return err
		}
	}

	return nil
}

func main() {
	serveFlag := flag.Bool("s", false, "serve a web server")
	flag.Parse()
	args := flag.Args()

	if err := run(args[0], *serveFlag); err != nil {
		panic(err)
	}
}
