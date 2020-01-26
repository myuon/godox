package main

import (
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"html/template"
	"log"
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
	default:
		return "", fmt.Errorf("Not yet implemented: %+v", t)
	}
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
	Params  []*ast.Field
	Results []*ast.Field
	Doc     string
}

// Make a new FuncDecl from ast.FuncDecl
func NewFuncDecl(val ast.FuncDecl) FuncDecl {
	var results []*ast.Field
	if val.Type.Results != nil {
		results = val.Type.Results.List
	}

	return FuncDecl{
		Name:    val.Name.Name,
		Params:  val.Type.Params.List,
		Results: results,
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
	Name    string
	FileDoc string
	Decls   []Decl
}

// Create an array of FileDox
func Run(files []*ast.File) ([]FileDox, error) {
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
	Index []string
	Funcs []Content
	Types []Content
	Vars  []Section
}

// Calculate Stat from FileDox
func (dox *FileDox) GetStat() Stat {
	var index []string
	var funcs []Content
	var types []Content
	var vars []Section

	for _, decl := range dox.Decls {
		if decl.Func != nil {
			var args []string
			for _, param := range decl.Func.Params {
				args = append(args, fmt.Sprintf("%v %v", param.Names[0].Name, param.Type))
			}

			var results []string
			if decl.Func.Results != nil {
				for _, param := range decl.Func.Results {
					t := &TypeWrapper{Expr: param.Type}
					r, err := t.Text()
					if err != nil {
						panic(err)
					}

					results = append(results, r)
				}
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
		Index: index,
		Funcs: funcs,
		Types: types,
		Vars:  vars,
	}
}

// Shows in text
func (dox *FileDox) Text() string {
	stat := dox.GetStat()

	return fmt.Sprintf(`==========
= file: %s

= Index
%v

= Content
%v
`, dox.Name, strings.Join(stat.Index, "\n"), stat.Funcs)
}

func main() {
	fset := token.NewFileSet()
	serveFlag := flag.Bool("s", false, "serve a web server")
	flag.Parse()
	args := flag.Args()

	packages, err := parser.ParseDir(fset, args[0], nil, parser.ParseComments)
	if err != nil {
		panic(err)
	}

	if *serveFlag {
		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			tpl := template.Must(template.ParseFiles(TemplatePath))

			for _, pkg := range packages {
				var files []*ast.File
				for _, file := range pkg.Files {
					files = append(files, file)
				}

				doxs, err := Run(files)
				if err != nil {
					panic(err)
				}

				for _, dox := range doxs {
					tpl.Execute(w, dox.GetStat())
				}
			}
		})
		log.Fatal(http.ListenAndServe(":8080", nil))
	} else {
		for _, pkg := range packages {
			var files []*ast.File
			for _, file := range pkg.Files {
				files = append(files, file)
			}

			doxs, err := Run(files)
			if err != nil {
				panic(err)
			}

			for _, dox := range doxs {
				println(dox.Text())
			}
		}
	}
}
