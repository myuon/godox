package main

import (
	"flag"
	"fmt"
	"godox/astwrapper"
	"html/template"
	"net/http"
)

// Define configurations
var (
	// Path to template file
	TemplatePath = "./template/index.html"
)

func prints(pkgs astwrapper.Packages) error {
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

func serves(pkgs astwrapper.Packages) error {
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
	pkgs, err := astwrapper.LoadPackages(path)
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
