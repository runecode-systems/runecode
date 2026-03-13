package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"

	"github.com/uudashr/gocognit"
)

func checkGoFunctionLengths(file fileInfo, content string, cfg runtimeConfig) ([]violation, error) {
	fset := token.NewFileSet()
	parsed, err := parser.ParseFile(fset, file.absPath, content, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", file.relPath, err)
	}

	limit := defaultFunctionLengthLimit(file)
	if entry, ok := cfg.baseline[file.relPath]; ok && entry.MaxFunctionLength > limit {
		limit = entry.MaxFunctionLength
	}

	violations := make([]violation, 0)
	for _, decl := range parsed.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}

		violations = append(violations, checkGoFunctionComplexity(file, fn, cfg)...)

		start := fset.Position(fn.Body.Lbrace).Line
		end := fset.Position(fn.Body.Rbrace).Line
		length := end - start + 1
		if length <= limit {
			continue
		}

		violations = append(violations, violation{
			rule:        ruleFunctionLength,
			path:        file.relPath,
			context:     fn.Name.Name,
			observed:    fmt.Sprintf("%d lines", length),
			expected:    fmt.Sprintf("<= %d lines", limit),
			remediation: "split the function or add a reviewed baseline entry",
		})
	}

	return violations, nil
}

func checkGoFunctionComplexity(file fileInfo, fn *ast.FuncDecl, cfg runtimeConfig) []violation {
	limit := defaultCognitiveComplexityLimit(file)
	if entry, ok := cfg.baseline[file.relPath]; ok && entry.MaxCognitiveComplexity > limit {
		limit = entry.MaxCognitiveComplexity
	}

	complexity := gocognit.Complexity(fn)
	if complexity <= limit {
		return nil
	}

	return []violation{{
		rule:        ruleFunctionCognitiveComplexity,
		path:        file.relPath,
		context:     fn.Name.Name,
		observed:    fmt.Sprintf("%d", complexity),
		expected:    fmt.Sprintf("<= %d", limit),
		remediation: "split the function or simplify control flow",
	}}
}

func defaultFunctionLengthLimit(file fileInfo) int {
	if file.tier == tierOne {
		return 40
	}
	return 60
}

func defaultCognitiveComplexityLimit(file fileInfo) int {
	if file.tier == tierOne {
		return 10
	}
	return 15
}
