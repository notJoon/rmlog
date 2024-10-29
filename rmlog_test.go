package rmlog

import (
	"go/ast"
	"go/parser"
	"os"
	"path/filepath"
	"strings"
	"testing"
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
			if err != nil {
				t.Fatalf("failed to parse expression: %v", err)
			}

			result := isPrintln(expr)
			if result != tt.expected {
				t.Errorf("isPrintln(%s) = %v, want %v", tt.code, result, tt.expected)
			}
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

			if result != nil && len(result.List) != tt.expected {
				t.Errorf("removeCommentedPrintln() returned %d comments, expected %d",
					len(result.List), tt.expected)
			}
		})
	}
}

func TestProcessFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "println-remover-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "remove println statements",
			input: `package main

import "fmt"

func main() {
	println("debug 1")
	fmt.Println("debug 2")
	fmt.Printf("keep this")
	// fmt.Println("debug 3")
}`,
			expected: `package main

import "fmt"

func main() {
	fmt.Printf("keep this")
}`,
		},
		{
			name: "keep non-println statements",
			input: `package main

func main() {
	doSomething()
	// normal comment
	anotherThing()
}`,
			expected: `package main

func main() {
	doSomething()
	// normal comment
	anotherThing()
}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testFile := filepath.Join(tmpDir, "test.go")
			err := os.WriteFile(testFile, []byte(tt.input), 0644)
			if err != nil {
				t.Fatalf("failed to write test file: %v", err)
			}

			err = ProcessFile(testFile)
			if err != nil {
				t.Fatalf("processFile() failed: %v", err)
			}

			content, err := os.ReadFile(testFile)
			if err != nil {
				t.Fatalf("failed to read processed file: %v", err)
			}

			normalizedResult := strings.ReplaceAll(string(content), "\r\n", "\n")
			normalizedExpected := strings.ReplaceAll(tt.expected, "\r\n", "\n")

			if normalizedResult != normalizedExpected {
				t.Errorf("processFile() result differs from expected:\nwant:\n%s\n\ngot:\n%s",
					normalizedExpected, normalizedResult)
			}
		})
	}
}
