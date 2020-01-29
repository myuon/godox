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

// Function declaration
type FuncDecl struct {
	Name    string
	Params  FieldListWrapper
	Results FieldListWrapper
	Doc     string
}

// Make a new FuncDecl from ast.FuncDecl
func NewFuncDecl(val ast.FuncDecl) FuncDecl {
	return FuncDecl{
		Name:    val.Name.Name,
		Params:  FieldListWrapper{FieldList: val.Type.Params},
		Results: FieldListWrapper{FieldList: val.Type.Results},
		Doc:     val.Doc.Text(),
	}
}

// Struct declaration
type StructDecl struct {
	Name string
	Doc  string
}

func NewStructDecl(val ast.TypeSpec, doc *ast.CommentGroup) StructDecl {
	return StructDecl{
		Name: val.Name.Name,
		Doc:  doc.Text(),
	}
}

// Var declaration
type VarDecl struct {
	Doc  string
	Vars []VarLine
}

// Var line
type VarLine struct {
	Names []string
	Doc   string
}

func NewVarDecl(vals []*ast.ValueSpec, doc *ast.CommentGroup) VarDecl {
	var vs []VarLine
	for _, val := range vals {
		vs = append(vs, VarLine{
			Names: ShowNames(val.Names),
			Doc:   val.Doc.Text(),
		})
	}

	return VarDecl{
		Doc:  doc.Text(),
		Vars: vs,
	}
}

// Decl is a large union of possible declarations.
// It has many pointers but only one could be non-nil at the same time.
type Decl struct {
	Func   *FuncDecl
	Struct *StructDecl
	Var    *VarDecl
}

// Make a new decl from FuncDecl
func NewDeclFromFunc(val FuncDecl) Decl {
	return Decl{
		Func: &val,
	}
}

// Make a new decl from StructDecl
func NewDeclFromStruct(val StructDecl) Decl {
	return Decl{
		Struct: &val,
	}
}

// Make a new decl from VarDecl
func NewDeclFromVar(val VarDecl) Decl {
	return Decl{
		Var: &val,
	}
}

// FileDox represents an analyzed result of an ast.File
type FileDox struct {
	Package string
	Name    string
	FileDoc string
	Decls   []Decl
}

// Create an array of FileDox
func Run(pkgName string, files []*ast.File) ([]FileDox, error) {
	var doxs []FileDox

	for _, file := range files {
		if file == nil {
			continue
		}

		var decls []Decl
		for _, decl := range file.Decls {
			switch decl := decl.(type) {
			case *ast.FuncDecl:
				decls = append(decls, NewDeclFromFunc(NewFuncDecl(*decl)))
			case *ast.GenDecl:
				if decl.Tok.String() == "type" {
					decls = append(decls, NewDeclFromStruct(NewStructDecl(*decl.Specs[0].(*ast.TypeSpec), decl.Doc)))
				}

				if decl.Tok.String() == "var" {
					var specs []*ast.ValueSpec
					for _, spec := range decl.Specs {
						specs = append(specs, spec.(*ast.ValueSpec))
					}

					decls = append(decls, NewDeclFromVar(NewVarDecl(specs, decl.Doc)))
				}
			default:
				continue
			}
		}

		dox := FileDox{
			Package: pkgName,
			Name:    file.Name.String(),
			FileDoc: file.Doc.Text(),
			Decls:   decls,
		}
		doxs = append(doxs, dox)
	}

	return doxs, nil
}

type Content struct {
	Name string
	Type string
	Doc  string
}

type NamesContent struct {
	Names []string
	Doc   string
}

type Section struct {
	Doc      string
	Contents []NamesContent
}

// Stat for templates
type Stat struct {
	Package string
	Index   []string
	Funcs   []Content
	Types   []Content
	Vars    []Section
}

// Calculate Stat from FileDox
func (dox *FileDox) GetStat() (Stat, error) {
	var index []string
	var funcs []Content
	var types []Content
	var vars []Section

	for _, decl := range dox.Decls {
		if decl.Func != nil {
			args, err := decl.Func.Params.Text()
			if err != nil {
				return Stat{}, err
			}

			results, err := decl.Func.Results.Text()
			if err != nil {
				return Stat{}, err
			}

			index = append(index, decl.Func.Name)

			if decl.Func != nil {
				funcs = append(funcs, Content{
					Name: decl.Func.Name,
					Type: fmt.Sprintf("func %s(%s) (%s)", decl.Func.Name, strings.Join(args, ", "), strings.Join(results, ", ")),
					Doc:  decl.Func.Doc,
				})
			}
		}

		if decl.Struct != nil {
			types = append(types, Content{
				Name: decl.Struct.Name,
				Type: "",
				Doc:  decl.Struct.Doc,
			})
		}

		if decl.Var != nil {
			var cs []NamesContent
			for _, v := range decl.Var.Vars {
				cs = append(cs, NamesContent{
					Names: v.Names,
					Doc:   v.Doc,
				})
			}

			vars = append(vars, Section{
				Doc:      decl.Var.Doc,
				Contents: cs,
			})
		}
	}

	return Stat{
		Package: dox.Package,
		Index:   index,
		Funcs:   funcs,
		Types:   types,
		Vars:    vars,
	}, nil
}

// Shows in text
func (dox *FileDox) Text() string {
	stat, err := dox.GetStat()
	if err != nil {
		panic(err)
	}

	return fmt.Sprintf(`==========
= file: %s

= Index
%v

= Content
%v
`, dox.Name, strings.Join(stat.Index, "\n"), stat.Funcs)
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
			"packages": pkgs,
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
