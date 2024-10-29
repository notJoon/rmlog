package rmlog

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
)

type Visitor struct {
	fileSet *token.FileSet
	changes bool
}

func (v *Visitor) Visit(node ast.Node) ast.Visitor {
	switch n := node.(type) {
	case *ast.ExprStmt:
		if isPrintln(n.X) {
			v.changes = true
			return nil
		}
	case *ast.File:
		n.Comments = filterComments(n.Comments)
	}
	return v
}

func filterComments(cs []*ast.CommentGroup) []*ast.CommentGroup {
	f := make([]*ast.CommentGroup, 0, len(cs))
	for _, c := range cs {
		if newCg := removeCommentedPrintln(c); newCg != nil {
			f = append(f, newCg)
		}
	}
	return f
}

func ProcessFile(path string) error {
	fset := token.NewFileSet()

	file, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("failed to parse file %s: %w", path, err)
	}

	v := &Visitor{fileSet: fset}
	ast.Walk(v, file)

	if v.changes || len(file.Comments) != len(filterComments(file.Comments)) {
		file.Comments = filterComments(file.Comments)
		f, err := os.Create(path)
		if err != nil {
			return fmt.Errorf("failed to create file %s: %w", path, err)
		}
		defer f.Close()

		cfg := printer.Config{
			Mode:     printer.UseSpaces | printer.TabIndent,
			Tabwidth: 8,
		}
		err = cfg.Fprint(f, fset, file)
		if err != nil {
			return fmt.Errorf("failed to write file %s: %w", path, err)
		}
	}
	return nil
}
