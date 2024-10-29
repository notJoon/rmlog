package rmlog

import (
	"go/ast"
	"strings"
)

// isPrintln checks if the expression is a println call.
func isPrintln(expr ast.Expr) bool {
	if callExpr, ok := expr.(*ast.CallExpr); ok {
		if ident, ok := callExpr.Fun.(*ast.Ident); ok {
			return ident.Name == "println"
		}
		if selExpr, ok := callExpr.Fun.(*ast.SelectorExpr); ok {
			if ident, ok := selExpr.X.(*ast.Ident); ok {
				return isSelectorPrintln(ident, selExpr)
			}
		}
	}
	return false
}

func isSelectorPrintln(ident *ast.Ident, selExpr *ast.SelectorExpr) bool {
	return (ident.Name == "fmt" && selExpr.Sel.Name == "Println") ||
		(ident.Name == "log" && selExpr.Sel.Name == "Println")
}

func removeCommentedPrintln(cg *ast.CommentGroup) *ast.CommentGroup {
	if cg == nil {
		return nil
	}

	newComments := make([]*ast.Comment, 0, len(cg.List))
	for _, c := range cg.List {
		text := c.Text
		if strings.Contains(text, "println") ||
			strings.Contains(text, "fmt.Println") ||
			strings.Contains(text, "log.Println") {
			continue
		}
		newComments = append(newComments, c)
	}

	if len(newComments) == 0 {
		return nil
	}
	return &ast.CommentGroup{List: newComments}
}
