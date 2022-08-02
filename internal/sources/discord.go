package sources

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/go-redis/redis/v8"
	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/ast"
	"github.com/gomarkdown/markdown/parser"
	log "github.com/sirupsen/logrus"

	"hyperdocs/config"
	discordgoutil "hyperdocs/pkg/discordgo"
	mdutil "hyperdocs/pkg/markdown"
)

var (
	discordDocsRepo    = "discord/discord-api-docs"
	discordDocsRepoURL = "https://raw.githubusercontent.com/discord/discord-api-docs"
	// discordDocsBranch   = "master"
	discordDocsBranch   = "main"
	discordDocsFilesURL = discordDocsRepoURL + "/" + discordDocsBranch + "/docs"
	discordDocsFileURL  = func(filepath string) string { return discordDocsFilesURL + "/" + filepath }

	discordDocsURL        = "https://discord.dev"
	discordDocsHeadingURL = func(topic, page, heading string) string {
		return discordDocsURL + "/" + topic + "/" + page + "#" + heading
	}

	discordDocsCacheKey = "discord.sources"
)

type Discord struct {
	Config config.Config
	Cache  *redis.Client
}

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
	LiteralPrefix string
	LiteralSuffix string

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
	var literal string

	if leaf := node.AsLeaf(); leaf != nil {
		literal = string(leaf.Literal)
	} else if container := node.AsContainer(); container != nil {
		literal = string(container.Literal)
	}

	if (f.LiteralTarget != "" && literal == f.LiteralTarget) ||
		(f.LiteralPrefix != "" && strings.HasPrefix(literal, f.LiteralPrefix)) ||
		(f.LiteralSuffix != "" && strings.HasSuffix(literal, f.LiteralSuffix)) {
		f.ResultNode = node
		return ast.SkipChildren
	}

	return ast.GoToNext
}

var referenceLinkRegex = regexp.MustCompile(`(?:([\w.]+)(#[\w\-\/:]+))`)

func (discord *Discord) resolveDiscordMarkdownReferences(ref string) string {
	if strings.HasPrefix(ref, "#DOCS") {
		// TODO: fallback situation (hitting ratelimits when cache provider is down)
		if v, err := discord.Cache.Exists(context.TODO(), discordDocsCacheKey).Result(); err != nil || v == 0 {
			resp, err := http.Get("https://api.github.com/repos/" + discordDocsRepo + "/git/trees/" + discordDocsBranch + "?recursive=1")
			if err != nil { // TODO: proper error handling
				log.Println(err)
				return "https://discord.com/developers/docs/intro"
			}
			var payload struct {
				Tree []struct {
					Path string `json:"path"`
				} `json:"tree"`
			}
			err = json.NewDecoder(resp.Body).Decode(&payload)
			if err != nil { // TODO: proper error handling
				log.Println(err)
				return "https://discord.com/developers/docs/intro"
			}

			for _, item := range payload.Tree {
				item.Path = strings.TrimSuffix(item.Path, ".md")
				res := discord.Cache.HSet(
					context.TODO(),
					discordDocsCacheKey,
					strings.ReplaceAll(strings.ToUpper(item.Path), "/", "_"),
					item.Path,
				)
				if res.Err() != nil {
					log.Println(fmt.Errorf("cannot cache %s: %w", item, res.Err()))
				}
			}

			discord.Cache.Expire(
				context.TODO(),
				discordDocsCacheKey,
				time.Duration(time.Duration(discord.Config.Sources.Discord.RedisTTL)*time.Second),
			)
		} else if err != nil {
			log.Println(err)
			return "https://discord.com/developers/docs/intro"
		}

		ref = strings.TrimPrefix(ref, "#")
		pathSegments := strings.Split(ref, "/")
		path, err := discord.Cache.HGet(context.TODO(), discordDocsCacheKey, pathSegments[0]).Result()
		if err != nil { // TODO: proper error handling
			log.Println(err)
			return "https://discord.com/developers/docs/intro"
		}
		return strings.ToLower("https://discord.com/developers/" + path + "#" + pathSegments[1])
		// pathSegments[2] = strings.SplitAfter strings.ReplaceAll(strings.Title(strings.ToLower(pathSegments[2]), "And", "and"),
	}
	return ref
}

func (discord *Discord) encodeMDReferenceLinks(src string) string {
	// split := strings.SplitN(referenceLinkRegex.FindStringSubmatch(src)[2], "_", 2)[1:]
	// split[0]
	res := referenceLinkRegex.ReplaceAllStringFunc(src, func(s string) string {
		submatches := referenceLinkRegex.FindStringSubmatch(s)

		return fmt.Sprintf("[%s](%s)", submatches[1], discord.resolveDiscordMarkdownReferences(submatches[2]))
	})
	// res := referenceLinkRegex.ReplaceAllString(src, "[${1}](${2})")
	return res
}

func normalizeDiscordUrlParameter(param string) string {
	return strings.ReplaceAll(strings.ToLower(param), " ", "-")
}

func parseEndpointParams(table *ast.Table) (params []APIEndpointParameter) {
	header := []string{}
	for _, v := range ast.GetFirstChild(ast.GetFirstChild(table)).GetChildren() {
		header = append(header, string(ast.GetFirstChild(v).AsLeaf().Literal))
	}
	rows := table.GetChildren()[1].GetChildren()
	for _, row := range rows {
		param := APIEndpointParameter{
			Additional: make(map[string]string),
		}
		children := row.GetChildren()
		for idx, col := range header {
			switch strings.ToLower(col) {
			case "field":
				param.Name = mdutil.RenderTextNode(children[idx])
			case "description":
				param.Description = mdutil.RenderTextNode(children[idx])
			case "type":
				param.Type = mdutil.RenderTextNode(children[idx])
			default:
				param.Additional[col] = mdutil.RenderTextNode(children[idx])
			}
		}
		params = append(params, param)
	}
	return
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

	visitor := &Visitor{LiteralPrefix: options["paragraph-name"].StringValue()}
	ast.Walk(parsed, visitor)
	top := visitor.ResultNode

	if top == nil {
		return nil, ErrSymbolNotFound
	}

	if top.GetParent() != nil {
		top = top.GetParent()
	}

	if split := strings.Split(string(visitor.ResultNode.AsLeaf().Literal), "%"); len(split) > 1 {
		split[0] = strings.TrimSpace(split[0])
		split[1] = strings.TrimSpace(split[1])
		endpointSignature := strings.Split(split[1], " ")
		var params, query []APIEndpointParameter
		_ = query
		var rendered string
		current := ast.GetNextNode(visitor.ResultNode.GetParent())
	stopCollectingAPIMethod:
		for current != nil {
			var result string
			switch v := current.(type) {
			case *ast.BlockQuote:
				result = mdutil.RenderHintNode(v, mdutil.HintKindMapping{
					"info":   ":information_source:",
					"warn":   ":warning:",
					"danger": ":octagonal_sign:",
				})
			case *ast.Heading:
				if v.Level <= 6 {
					switch strings.ToLower(string(ast.GetFirstChild(current).AsLeaf().Literal)) {
					case "json params":
						current = ast.GetNextNode(current)
						table, ok := current.(*ast.Table)
						if !ok {
							continue stopCollectingAPIMethod
						}
						params = parseEndpointParams(table)
						continue stopCollectingAPIMethod
					}
				} else {
					break stopCollectingAPIMethod
				}
			default:
				result = mdutil.RenderStringNode(current)
				if result == "" {
					break stopCollectingAPIMethod
				}
			}
			rendered += result + "\n\n"
			current = ast.GetNextNode(current)
		}
		// fmt.Println(rendered)
		return APIEndpoint{
			Name:        strings.TrimSpace(split[0]),
			Description: rendered,
			Method:      strings.ToUpper(endpointSignature[0]),
			Endpoint:    d.encodeMDReferenceLinks(endpointSignature[1]),
			Parameters:  params,
			Link: discordDocsHeadingURL(
				normalizeDiscordUrlParameter(options["topic"].StringValue()),
				normalizeDiscordUrlParameter(options["page"].StringValue()),
				normalizeDiscordUrlParameter(strings.ToLower(strings.ReplaceAll(split[0], " ", "-"))),
			),
		}, nil
	}

	current := ast.GetNextNode(top)

	var rendered string
stopCollecting:
	for current != nil {
		var result string
		switch v := current.(type) {
		case *ast.BlockQuote:
			result = mdutil.RenderHintNode(v, mdutil.HintKindMapping{
				"info":   ":information_source:",
				"warn":   ":warning:",
				"danger": ":octagonal_sign:",
			})
		default:
			result = mdutil.RenderStringNode(current)
			if result == "" {
				break stopCollecting
			}
		}
		rendered += result + "\n\n"
		current = ast.GetNextNode(current)
	}

	return Paragraph{
		Content: rendered,
		Title:   string(ast.GetFirstChild(visitor.ResultNode.GetParent()).AsLeaf().Literal),
		Source: discordDocsHeadingURL(
			normalizeDiscordUrlParameter(options["topic"].StringValue()),
			normalizeDiscordUrlParameter(options["page"].StringValue()),
			normalizeDiscordUrlParameter((visitor.ResultNode.GetParent().(*ast.Heading)).HeadingID),
		),
	}, nil
}

func NewDiscord(cfg config.Config, redisClient *redis.Client) Source {
	return &Discord{
		Config: cfg,
		Cache:  redisClient,
	}
}
