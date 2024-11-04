package rmlog

import (
	"go/ast"
	"go/parser"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsPrintln(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		expected bool
	}{
		{
			name:     "basic println",
			code:     "println(\"debug\")",
			expected: true,
		},
		{
			name:     "fmt.Println",
			code:     "fmt.Println(\"debug\")",
			expected: true,
		},
		{
			name:     "log.Println",
			code:     "log.Println(\"debug\")",
			expected: true,
		},
		{
			name:     "regular function call",
			code:     "someFunc(\"not println\")",
			expected: false,
		},
		{
			name:     "println with multiple arguments",
			code:     "fmt.Println(\"debug\", x, y)",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expr, err := parser.ParseExpr(tt.code)
			assert.NoError(t, err)

			result := isPrintln(expr)
			assert.Equal(t, result, tt.expected)
		})
	}
}

func TestRemoveCommentedPrintln(t *testing.T) {
	tests := []struct {
		name     string
		comments []*ast.Comment
		expected int
	}{
		{
			name: "remove println comment",
			comments: []*ast.Comment{
				{Text: "// println(\"debug\")"},
				{Text: "// normal comment"},
			},
			expected: 1,
		},
		{
			name: "remove fmt.Println comment",
			comments: []*ast.Comment{
				{Text: "// fmt.Println(\"debug\")"},
				{Text: "// another comment"},
			},
			expected: 1,
		},
		{
			name: "keep normal comments",
			comments: []*ast.Comment{
				{Text: "// normal comment 1"},
				{Text: "// normal comment 2"},
			},
			expected: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cg := &ast.CommentGroup{List: tt.comments}
			result := removeCommentedPrintln(cg)

			if result == nil && tt.expected > 0 {
				t.Errorf("removeCommentedPrintln() returned nil, expected %d comments", tt.expected)
			}

			assert.Equal(t, len(result.List), tt.expected)
		})
	}
}

func TestProcessFile(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected string
	}{
		{
			name: "Remove println statements",
			content: `package main
func main() {
    println("debug message")
    fmt.Println("another debug")
    log.Println("log message")
    // println("commented debug")
    /* fmt.Println("block comment") */
    actualCode()
}`,
			expected: "package main\n\nfunc main() {\n\n\tactualCode()\n}",
		},
		{
			name: "Keep non-println statements",
			content: `package main

func main() {
    print("keep this")
    fmt.Printf("keep this too")
    actualCode()
}`,
			expected: "package main\n\nfunc main() {\n    print(\"keep this\")\n    fmt.Printf(\"keep this too\")\n    actualCode()\n}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary test file
			tempDir := t.TempDir()
			testFile := filepath.Join(tempDir, "test.noext")

			err := os.WriteFile(testFile, []byte(tt.content), 0o644)
			assert.NoError(t, err)

			// Process the file
			err = ProcessFile(testFile)
			assert.NoError(t, err)

			// Read the processed content
			processed, err := os.ReadFile(testFile)
			assert.NoError(t, err)

			// Compare the results (ignoring whitespace differences)
			processedStr := strings.TrimSpace(string(processed))
			expectedStr := strings.TrimSpace(tt.expected)

			assert.Equal(t, processedStr, expectedStr)
		})
	}
}

func TestCreateTempGoFile(t *testing.T) {
	content := "package main\nfunc main() {}\n"

	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "source.noext")

	err := os.WriteFile(testFile, []byte(content), 0o644)
	assert.NoError(t, err)

	tempGoFile, err := createTempGoFile(testFile)
	assert.NoError(t, err)

	defer os.Remove(tempGoFile)

	assert.True(t, strings.HasSuffix(tempGoFile, ".go"))

	tempContent, err := os.ReadFile(tempGoFile)
	assert.NoError(t, err)

	assert.Equal(t, string(tempContent), content)
}
