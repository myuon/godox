package astwrapper

import (
	"fmt"
	"go/ast"
	"strings"
)

type FieldList struct {
	*ast.FieldList
}

func (t *FieldList) Text() ([]string, error) {
	if t == nil || t.FieldList == nil {
		return nil, nil
	}

	var rep []string
	for _, field := range t.List {
		t := Type{Expr: field.Type}
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
