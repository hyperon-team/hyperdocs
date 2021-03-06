package sources

import (
	"github.com/bwmarrin/discordgo"
)

// SymbolType represents the type of a symbol.
type SymbolType uint

const (
	// SymbolClass indicates that the symbol is a class or structure
	SymbolClass SymbolType = iota
	// SymbolInterface indicates that the symbol is an interface or trait (Rust).
	SymbolInterface
	// SymbolFunction indicates that the symbol is a function/method.
	SymbolFunction
	// SymbolAPIMethod indicates that the symbol is API endpoint.
	SymbolAPIMethod
	// SymbolParagraph indicates that the symbol is a simple text paragraph (in markdown for example).
	SymbolParagraph
)

// Symbol is the base representation of a symbol in the documentation.
type Symbol interface {
	GetName() string
	GetLink() string
	Type() SymbolType

	Render() (desc string, fields []*discordgo.MessageEmbedField)
}

type Type struct {
	Target string
	Link   bool
}

type FunctionParameter struct {
	Name string
	Type Type
}

type Function struct {
	Name       string
	Signature  string
	Parameters map[string]FunctionParameter
	Childs     []Symbol
}

type APIMethod struct{}

type Paragraph struct {
	Title        string
	Content      string
	Nesting      int
	Source       string
	ChildSymbols []Symbol
}

func (p Paragraph) GetName() string {
	return p.Title
}
func (p Paragraph) GetLink() string {
	return p.Source
}

func (Paragraph) Type() SymbolType {
	return SymbolParagraph
}

func (p Paragraph) Render() (desc string, fields []*discordgo.MessageEmbedField) {
	return p.Content, nil
}
