package astwrapper

import (
	"fmt"
	"go/ast"
)

type Type struct {
	ast.Expr
}

func (t *Type) Text() (string, error) {
	switch t := t.Expr.(type) {
	case *ast.ArrayType:
		elt := &Type{Expr: t.Elt}
		r, err := elt.Text()
		if err != nil {
			return "", err
		}

		return fmt.Sprintf("[]%s", r), nil
	case *ast.Ident:
		return t.Name, nil
	case *ast.StarExpr:
		r, err := (&Type{Expr: t.X}).Text()
		if err != nil {
			return "", err
		}

		return fmt.Sprintf("*%s", r), nil
	case *ast.SelectorExpr:
		r, err := (&Type{Expr: t.X}).Text()
		if err != nil {
			return "", err
		}

		return fmt.Sprintf("%s.%s", r, t.Sel.Name), nil
	default:
		return "", fmt.Errorf("Not yet implemented: %+v", t)
	}
}
