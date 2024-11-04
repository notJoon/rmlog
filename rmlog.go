package rmlog

import (
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"io"
	"os"
	"regexp"
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
		(ident.Name == "ufmt" && selExpr.Sel.Name == "Println") ||
		(ident.Name == "ufmt" && selExpr.Sel.Name == "Sprintf") ||
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
			strings.Contains(text, "ufmt.Println") ||
			strings.Contains(text, "ufmt.Sprintf") ||
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

type PrintlnRemover struct {
	fileSet *token.FileSet
	changes bool
}

// NewPrintlnRemover creates a new PrintlnRemover instance
func NewPrintlnRemover(fset *token.FileSet) *PrintlnRemover {
	return &PrintlnRemover{
		fileSet: fset,
		changes: false,
	}
}

// Visit implements the ast.Visitor interface
func (r *PrintlnRemover) Visit(node ast.Node) ast.Visitor {
	if node == nil {
		return nil
	}

	switch n := node.(type) {
	case *ast.BlockStmt:
		// Filter out println statements from the block's statement list
		newList := make([]ast.Stmt, 0, len(n.List))
		for _, stmt := range n.List {
			if exprStmt, ok := stmt.(*ast.ExprStmt); ok {
				if isPrintln(exprStmt.X) {
					r.changes = true
					continue
				}
			}
			newList = append(newList, stmt)
		}
		n.List = newList
		return r

	case *ast.File:
		// Handle file-level comments
		n.Comments = filterComments(n.Comments)
		return r
	}

	return r
}

func filterComments(cs []*ast.CommentGroup) []*ast.CommentGroup {
	if cs == nil {
		return nil
	}

	filtered := make([]*ast.CommentGroup, 0, len(cs))
	for _, c := range cs {
		if newCg := removeCommentedPrintln(c); newCg != nil {
			filtered = append(filtered, newCg)
		}
	}
	return filtered
}

// createTempGoFile creates a temporary .go file and copies the content from the source file
func createTempGoFile(sourcePath string) (string, error) {
	content, err := os.ReadFile(sourcePath)
	if err != nil {
		return "", fmt.Errorf("failed to read source file: %w", err)
	}

	tempDir := os.TempDir()
	tempFile, err := os.CreateTemp(tempDir, "*.go")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}

	_, err = tempFile.Write(content)
	if err != nil {
		tempFile.Close()
		os.Remove(tempFile.Name())
		return "", fmt.Errorf("failed to write to temp file: %w", err)
	}

	tempFile.Close()
	return tempFile.Name(), nil
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer source.Close()

	destination, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destination.Close()

	_, err = io.Copy(destination, source)
	return err
}

// ProcessFile removes println statements from a Go file and writes the formatted result.
func ProcessFile(path string) error {
	tempGoFile, err := createTempGoFile(path)
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tempGoFile)

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, tempGoFile, nil, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("failed to parse file: %w", err)
	}

	remover := NewPrintlnRemover(fset)
	ast.Walk(remover, file)

	// if there were changes, write the modified and formatted content back to the file
	if remover.changes {
		buffer := new(strings.Builder)

		if err := format.Node(buffer, fset, file); err != nil {
			return fmt.Errorf("failed to format file: %w", err)
		}

		formattedString := buffer.String()
		re := regexp.MustCompile(`\n{2,}`)
		formattedString = re.ReplaceAllString(formattedString, "\n\n")

		processedTempFile := tempGoFile + ".processed"
		if err := os.WriteFile(processedTempFile, []byte(formattedString), 0o644); err != nil {
			return fmt.Errorf("failed to write processed file: %w", err)
		}
		defer os.Remove(processedTempFile)

		if err := copyFile(processedTempFile, path); err != nil {
			return fmt.Errorf("failed to copy processed content back to original file: %w", err)
		}
	}

	return nil
}
