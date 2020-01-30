package astwrapper

import (
	"go/ast"
)

type ValueGroup struct {
	Doc   *ast.CommentGroup
	Specs []ast.ValueSpec
}
