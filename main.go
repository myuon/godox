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

type FuncDecl struct {
	Name string
	Type ast.FuncType
	Doc  string
}

func NewFuncDecl(ast ast.FuncDecl) FuncDecl {
	return FuncDecl{
		Name: ast.Name.Name,
		Type: *ast.Type,
		Doc:  ast.Doc.Text(),
	}
}

type Decl struct {
	Func *FuncDecl
}

func NewDeclFromFunc(val FuncDecl) Decl {
	return Decl{
		Func: &val,
	}
}

type FileDox struct {
	Name    string
	FileDoc string
	Decls   []Decl
}

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

type Stat struct {
	Index   []string
	Content []string
}

func (dox *FileDox) GetStat() Stat {
	var index []string
	var content []string

	for _, decl := range dox.Decls {
		if decl.Func != nil {
			var args []string
			for _, param := range decl.Func.Type.Params.List {
				args = append(args, fmt.Sprintf("%v %v", param.Names[0].Name, param.Type))
			}

			var results []string
			if decl.Func.Type.Results != nil {
				for _, param := range decl.Func.Type.Results.List {
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

	packages, err := parser.ParseDir(fset, args[0], nil, parser.AllErrors)
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
