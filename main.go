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
func NewFuncDecl(ast ast.FuncDecl) FuncDecl {
	return FuncDecl{
		Name:    ast.Name.Name,
		Params:  ast.Type.Params.List,
		Results: ast.Type.Results.List,
		Doc:     ast.Doc.Text(),
	}
}

// Decl is a large union of possible declarations.
// It has many pointers but only one could be non-nil at the same time.
type Decl struct {
	Func *FuncDecl
}

// Make a new decl from FuncDecl
func NewDeclFromFunc(val FuncDecl) Decl {
	return Decl{
		Func: &val,
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

// Stat for templates
type Stat struct {
	Index   []string
	Content []string
}

// Calculate Stat from FileDox
func (dox *FileDox) GetStat() Stat {
	var index []string
	var content []string

	for _, decl := range dox.Decls {
		if decl.Func != nil {
			var args []string
			for _, param := range decl.Func.Params {
				args = append(args, fmt.Sprintf("%v %v", param.Names[0].Name, param.Type))
			}

			var results []string
			if decl.Func.Results != nil {
				for _, param := range decl.Func.Results {
					results = append(results, fmt.Sprintf("%v", param.Type))
				}
			}

			index = append(index, decl.Func.Name)
			content = append(content, fmt.Sprintf(`== %s
func %s(%v) (%s)
%s
`, decl.Func.Name, decl.Func.Name, args, strings.Join(results, ", "), decl.Func.Doc))
		}
	}

	return Stat{
		Index:   index,
		Content: content,
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
`, dox.Name, strings.Join(stat.Index, "\n"), strings.Join(stat.Content, "\n"))
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
			tpl := template.Must(template.ParseFiles("./template/index.html"))

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
