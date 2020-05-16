package main

import (
	"flag"
	"fmt"
	"github.com/myuon/godox/dox"
	"html/template"
	"net/http"

	"github.com/myuon/godox/astwrapper"
)

// Define configurations
var (
	// Path to template file
	TemplatePath = "./template/index.html"
)

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
	dx, err := dox.LoadPackages(path)
	if err != nil {
		return err
	}

	if serveFlag {
		panic("not implemented yet")
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
