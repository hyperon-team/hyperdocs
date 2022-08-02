package mdutil

import (
	"fmt"
	"strings"

	"github.com/gomarkdown/markdown/ast"
)

// RenderCode renders one-line code into a string.
func RenderCode(code string) string {
	return fmt.Sprintf("`%s`", code)
}

// RenderCodeNode wraps RenderCode for *ast.Code node.
func RenderCodeNode(node *ast.Code) string {
	return RenderCode(string(node.Literal))
}

// RenderCodeBlock renders multiline code blocks into a string.
func RenderCodeBlock(language, block string) string {
	return fmt.Sprintf("```%s\n%s\n```", language, block)
}

// RenderCodeBlockNode wraps RenderCodeBlock for *ast.CodeBlock node.
func RenderCodeBlockNode(node *ast.CodeBlock) string {
	return RenderCodeBlock(string(node.Info), string(node.Literal))
}

// RenderTextNode renders a text node into a string.
func RenderTextNode(node ast.Node) (result string) {
	if node == nil {
		return ""
	}

	ast.Walk(node, ast.NodeVisitorFunc(func(node ast.Node, entering bool) ast.WalkStatus {
		if !entering {
			return ast.GoToNext
		}
		switch v := node.(type) {
		case *ast.Text:
			result += string(v.Literal)
		case *ast.Emph:
			result += "_" + string(ast.GetFirstChild(v).AsLeaf().Literal) + "_"
		case *ast.Strong:
			result += "**" + string(ast.GetFirstChild(v).AsLeaf().Literal) + "**"
		case *ast.Code:
			result += RenderCodeNode(v)
		default:
			return ast.GoToNext
		}
		return ast.SkipChildren
	}))

	return result
}

// RenderBlockQuote renders markdown block quote into a string.
// If multiline is true it uses multiline block quote syntax.
func RenderBlockQuote(content string, multiline bool) string {
	if multiline {
		return ">>> " + content
	}

	return "> " + content
}

// RenderBlockQuoteNode wraps RenderBlockQuote for *ast.BlockQuote node.
func RenderBlockQuoteNode(node *ast.BlockQuote) (res string) {
	switch v := ast.GetFirstChild(node).(type) {
	case *ast.Paragraph:
		res = RenderTextNode(v)
	case *ast.CodeBlock:
		res = RenderCodeBlockNode(v)
	}
	return RenderBlockQuote(res, false)
}

// HintKindMapping is a mapping for hint types.
type HintKindMapping map[string]string

// RenderHintNode renders *ast.BlockQuote node as a hint into a string
func RenderHintNode(node *ast.BlockQuote, kindMappings HintKindMapping) (res string) {
	if p, ok := ast.GetFirstChild(node).(*ast.Paragraph); ok {
		prefixed := false
		for k := range kindMappings {
			if strings.HasPrefix(string(ast.GetFirstChild(p).AsLeaf().Literal), k) {
				prefixed = true
			}
		}

		if prefixed {
			kind := string(ast.GetFirstChild(p).AsLeaf().Literal)
			split := strings.Split(kind, "\n")
			kind = split[0]
			literal := kindMappings[kind] + ": "
			if len(split) > 1 {
				literal += split[1]
			}

			ast.GetFirstChild(p).AsLeaf().Literal = []byte(literal)
		}
	}

	return RenderBlockQuoteNode(node)
}

// RenderStringNode renders node into a string.
func RenderStringNode(node ast.Node) string {
	switch v := node.(type) {
	case *ast.Paragraph:
		return RenderTextNode(v)
	case *ast.CodeBlock:
		return RenderCodeBlockNode(v)
	case *ast.BlockQuote:
		return RenderHintNode(v, HintKindMapping{
			"info":   ":information_source:",
			"warn":   ":warning:",
			"danger": ":octagonal_sign:",
		})
	}
	return ""
}
