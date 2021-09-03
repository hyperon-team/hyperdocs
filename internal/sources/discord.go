package sources

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"path"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/ast"
	"github.com/gomarkdown/markdown/parser"

	discordgoutil "hyperdocs/pkg/discordgo"
)

var (
	discordDocsRepoURL  = "https://raw.githubusercontent.com/discord/discord-api-docs"
	discordDocsBranch   = "master"
	discordDocsFilesURL = discordDocsRepoURL + "/" + discordDocsBranch + "/docs"
	discordDocsFileURL  = func(filepath string) string { return discordDocsFilesURL + "/" + filepath }

	discordDocsURL        = "https://discord.dev"
	discordDocsHeadingURL = func(topic, page, heading string) string {
		return discordDocsURL + "/" + topic + "/" + page + "#" + heading
	}
)

type Discord struct{}

// Name of the resource. It is being used as the source command name
func (d Discord) Name() string {
	return "discord"
}

// Source description. It is set as the source command description
func (d Discord) Description() string {
	return "Discord API documentation"
}

// Source command options
func (d Discord) Options() []*discordgo.ApplicationCommandOption {
	return []*discordgo.ApplicationCommandOption{
		{
			Name:        "topic",
			Description: "Topic name",
			Type:        discordgo.ApplicationCommandOptionString,
			Required:    true,
		},
		{
			Name:        "page",
			Description: "Page (subtopic) you're interested in",
			Type:        discordgo.ApplicationCommandOptionString,
			Required:    true,
		},
		{
			Name:        "paragraph-name",
			Description: "Paragraph name to search for",
			Type:        discordgo.ApplicationCommandOptionString,
		},
	}
}

// Process is a hook to process and prepare data for the Search function
func (d Discord) Process(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) error {
	return nil
}

type Visitor struct {
	LiteralTarget string

	ResultNode ast.Node
}

//nolint:unused // Debug functionality, might be not used
func printNode(node ast.Node, nesting int) {
	if node == nil {
		fmt.Println(node)
		return
	}

	padding := strings.Repeat("\t", nesting)
	if c := node.AsContainer(); c != nil {
		fmt.Printf("%scontainer (%T): {\n", padding, node)
		if c.Attribute != nil {
			for k, attr := range c.Attribute.Attrs {
				fmt.Printf("%s\tattr: %s %s\n", padding, k, string(attr))
			}
			for _, class := range c.Attribute.Classes {
				fmt.Printf("%s\tclass: %s\n", padding, string(class))
			}
		}

		fmt.Printf("%s\tliteral: %s\n", padding, string(c.Literal))
		fmt.Printf("%s\tcontent: %s\n", padding, string(c.Content))

		for _, node := range c.Children {
			fmt.Printf("%s\tchildren: {\n", padding)
			printNode(node, nesting+2)
			fmt.Printf("%s\t}\n", padding)
		}
		fmt.Printf("%s}\n", padding)
	} else if l := node.AsLeaf(); l != nil {
		fmt.Printf("%sleaf (%T): %s\n", padding, node, string(l.Literal))
	}
}

func (f *Visitor) Visit(node ast.Node, entering bool) ast.WalkStatus {
	var literal []byte

	if leaf := node.AsLeaf(); leaf != nil {
		literal = leaf.Literal
	} else if container := node.AsContainer(); container != nil {
		literal = container.Literal
	}

	if string(literal) == f.LiteralTarget {
		f.ResultNode = node
		return ast.SkipChildren
	}

	return ast.GoToNext
}

func renderParagraph(node *ast.Paragraph) (res string) {
	if node == nil {
		return ""
	}

	ast.Walk(node, ast.NodeVisitorFunc(func(node ast.Node, entering bool) ast.WalkStatus {
		if !entering {
			return ast.GoToNext
		}
		switch v := node.(type) {
		case *ast.Text:
			res += string(v.Literal)
		case *ast.Emph:
			res += "_" + string(ast.GetFirstChild(v).AsLeaf().Literal) + "_"
		case *ast.Strong:
			res += "**" + string(ast.GetFirstChild(v).AsLeaf().Literal) + "**"
		default:
			return ast.GoToNext
		}
		return ast.SkipChildren
	}))

	return
}

func renderCodeBlock(node *ast.CodeBlock) (res string) {
	return fmt.Sprintf("```%s\n%s\n```", node.Info, node.Literal)
}

func formatDiscordUrlParameter(param string) string {
	return strings.ReplaceAll(strings.ToLower(param), " ", "-")
}

// Search processes the input and returns the symbol by specified parameters.
func (d Discord) Search(ctx context.Context, s *discordgo.Session, i *discordgo.InteractionCreate) (Symbol, error) {
	data := i.ApplicationCommandData()
	localData := data.Options[0]

	options := discordgoutil.OptionsToMap(localData.Options)

	filePath := path.Clean(
		strings.ReplaceAll(strings.ToLower(options["topic"].StringValue()), " ", "_") +
			"/" +
			strings.ReplaceAll(
				strings.ReplaceAll(strings.Title(strings.ToLower(options["page"].StringValue())), "And", "and"),
				" ", "_",
			) + ".md", // TODO: regex out bad ones

	)

	response, err := http.Get(discordDocsFileURL(filePath))

	if err != nil {
		return nil, fmt.Errorf("http: %w", err)
	}
	defer response.Body.Close()

	body, err := ioutil.ReadAll(response.Body)

	if err != nil {
		return nil, fmt.Errorf("readall: %w", err)
	}

	parsed := markdown.Parse(body, parser.NewWithExtensions(parser.CommonExtensions|parser.AutoHeadingIDs))
	visitor := &Visitor{LiteralTarget: options["paragraph-name"].StringValue()}
	ast.Walk(parsed, visitor)

	top := visitor.ResultNode

	if top == nil {
		return nil, ErrSymbolNotFound
	}

	if top.GetParent() != nil {
		top = top.GetParent()
	}

	current := ast.GetNextNode(top)

	var rendered string
stopCollecting:
	for current != nil {
		switch v := current.(type) {
		case *ast.Paragraph:
			rendered += renderParagraph(v)
		case *ast.CodeBlock:
			rendered += renderCodeBlock(v)
		default:
			break stopCollecting
		}
		rendered += "\n\n"
		current = ast.GetNextNode(current)
	}

	return Paragraph{
		Content: rendered,
		Title:   string(ast.GetFirstChild(visitor.ResultNode.GetParent()).AsLeaf().Literal),
		Source: discordDocsHeadingURL(
			formatDiscordUrlParameter(options["topic"].StringValue()),
			formatDiscordUrlParameter(options["page"].StringValue()),
			formatDiscordUrlParameter((visitor.ResultNode.GetParent().(*ast.Heading)).HeadingID),
		),
	}, nil
}
