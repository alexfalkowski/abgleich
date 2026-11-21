package workflow

import (
	"github.com/alexfalkowski/go-service/v3/strings"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/text"
)

const requiredCheckPrefix = "Required check:"

var descriptionParser = goldmark.New(goldmark.WithExtensions(extension.Table)).Parser()

func requiredChecks(description string) []string {
	if !strings.Contains(description, requiredCheckPrefix) {
		return nil
	}

	source := strings.Bytes(description)
	document := descriptionParser.Parse(text.NewReader(source))
	var checks []string
	for block := document.FirstChild(); block != nil; block = block.NextSibling() {
		if block.Kind() != ast.KindParagraph {
			continue
		}
		checks = append(checks, paragraphChecks(block, source, description)...)
	}
	return checks
}

func paragraphChecks(paragraph ast.Node, source []byte, description string) []string {
	var checks []string
	for inline := paragraph.FirstChild(); inline != nil; inline = inline.NextSibling() {
		if inline.Kind() == ast.KindRawHTML {
			// Raw HTML can hide text in the surrounding paragraph.
			return nil
		}
		node, ok := inline.(*ast.Text)
		if !ok || (node.Segment.Start > 0 && source[node.Segment.Start-1] != '\n') {
			continue
		}
		value := strings.TrimSpace(string(node.Value(source)))
		if value == requiredCheckPrefix || !strings.HasPrefix(value, requiredCheckPrefix) {
			continue
		}
		line, _, _ := strings.Cut(description[node.Segment.Start:], "\n")
		if value == strings.TrimSpace(line) {
			checks = append(checks, value)
		}
	}
	return checks
}
