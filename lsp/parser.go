package lsp

import (
	"context"
	sitter "github.com/smacker/go-tree-sitter"

	// get binding from here
	// https://github.com/smacker/go-tree-sitter
	// or roll your own like here:
	// https://github.com/aaron-loa/custom_scss_lsp
	example_binding "github.com/smacker/go-tree-sitter/golang"
)

type Parser struct {
	Parser *sitter.Parser
}

func NewParser() *Parser {
	parser := sitter.NewParser()
	parser.SetLanguage(example_binding.GetLanguage())
	return &Parser{
		Parser: parser,
	}
}

func (p *Parser) ParseBytes(text *[]byte, tree *sitter.Tree) (*sitter.Tree, error) {
	tree, err := p.Parser.ParseCtx(context.Background(), tree, *text)
	return tree, err
}
