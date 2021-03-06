package main

import (
	"flag"
	"fmt"
	"github.com/myuon/godox/dox"
	"html/template"
	"net/http"
)

// Define configurations
var (
	// Path to template file
	TemplatePath = "./template/index.html"
)

func serves(pkgs dox.PackagesDox) error {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		tpl := template.Must(template.ParseFiles(TemplatePath))

		err := tpl.Execute(w, pkgs)
		if err != nil {
			panic(err)
		}
	})
	println("Listening on http://localhost:8080...")

	return http.ListenAndServe(":8080", nil)
}

func run(path string, serveFlag bool) error {
	dx, err := dox.LoadPackages(path)
	if err != nil {
		return err
	}

	if serveFlag {
		if err := serves(dx); err != nil {
			return err
		}
	} else {
		jn, err := dx.Json()
		if err != nil {
			return err
		}

		fmt.Println(jn)
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
