package main

import (
	"fmt"
	"go/ast"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/singlechecker"
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

// This is run function.
func run(pass *analysis.Pass) (interface{}, error) {
	var doxs []FileDox

	for _, file := range pass.Files {
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

		fmt.Println(dox.Text())
	}
	fmt.Printf("%+v", doxs)

	return nil, nil
}

func (dox *FileDox) Text() string {
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

	return fmt.Sprintf(`==========
= file: %s

= Index
%v

= Content
%v
`, dox.Name, strings.Join(index, "\n"), strings.Join(content, "\n"))
}

func main() {
	singlechecker.Main(&analysis.Analyzer{
		Name: "godox",
		Doc:  "document tool for golang",
		Run:  run,
	})
}
