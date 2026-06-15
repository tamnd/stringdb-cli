package stringdb

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/tamnd/any-cli/kit"
	"github.com/tamnd/any-cli/kit/errs"
)

func init() { kit.Register(Domain{}) }

type Domain struct{}

func (Domain) Info() kit.DomainInfo {
	return kit.DomainInfo{
		Scheme: "stringdb",
		Hosts:  []string{Host},
		Identity: kit.Identity{
			Binary: "stringdb",
			Short:  "A command line for the STRING protein interaction database.",
			Long: `A command line for STRING DB.

stringdb queries the STRING protein-protein interaction database,
covering 67 million proteins across 5,090 organisms. Look up protein
interactions, functional enrichment, and resolve identifiers to STRING IDs.
No API key required.`,
			Site: "https://string-db.org",
			Repo: "https://github.com/tamnd/stringdb-cli",
		},
	}
}

func (Domain) Register(app *kit.App) {
	app.SetClient(newClient)

	kit.Handle(app, kit.OpMeta{Name: "proteins", Group: "read", List: true,
		Summary: "Resolve protein names to STRING IDs (--species, --limit)",
		Args:    []kit.Arg{{Name: "name", Help: "protein name or gene symbol (e.g. TP53)"}}}, resolveProteins)

	kit.Handle(app, kit.OpMeta{Name: "interactions", Group: "read", List: true,
		Summary: "Get interactions for proteins (--species, --limit)",
		Args:    []kit.Arg{{Name: "name", Help: "protein name or gene symbol"}}}, getInteractions)

	kit.Handle(app, kit.OpMeta{Name: "network", Group: "read", List: true,
		Summary: "Get interaction network for multiple proteins (comma-separated, --species, --limit)",
		Args:    []kit.Arg{{Name: "names", Help: "comma-separated protein names (e.g. TP53,BRCA1)"}}}, getNetwork)

	kit.Handle(app, kit.OpMeta{Name: "enrich", Group: "read", List: true,
		Summary: "Get functional enrichment for a set of proteins (comma-separated, --species)",
		Args:    []kit.Arg{{Name: "names", Help: "comma-separated protein names"}}}, getEnrichment)
}

func newClient(_ context.Context, cfg kit.Config) (any, error) {
	c := DefaultConfig()
	if cfg.UserAgent != "" {
		c.UserAgent = cfg.UserAgent
	}
	if cfg.Rate > 0 {
		c.Rate = cfg.Rate
	}
	if cfg.Retries > 0 {
		c.Retries = cfg.Retries
	}
	if cfg.Timeout > 0 {
		c.Timeout = cfg.Timeout
	}
	return NewClient(c), nil
}

type proteinsInput struct {
	Name    string  `kit:"arg"          help:"protein name or gene symbol"`
	Species int     `kit:"flag"         help:"NCBI taxon ID (default 9606 = human)"`
	Limit   int     `kit:"flag,inherit" help:"max results"`
	Client  *Client `kit:"inject"`
}

type interactionsInput struct {
	Name    string  `kit:"arg"          help:"protein name or gene symbol"`
	Species int     `kit:"flag"         help:"NCBI taxon ID (default 9606 = human)"`
	Limit   int     `kit:"flag,inherit" help:"max results"`
	Client  *Client `kit:"inject"`
}

type networkInput struct {
	Names   string  `kit:"arg"          help:"comma-separated protein names"`
	Species int     `kit:"flag"         help:"NCBI taxon ID (default 9606 = human)"`
	Limit   int     `kit:"flag,inherit" help:"max results per protein"`
	Client  *Client `kit:"inject"`
}

type enrichInput struct {
	Names   string  `kit:"arg"  help:"comma-separated protein names"`
	Species int     `kit:"flag" help:"NCBI taxon ID (default 9606 = human)"`
	Client  *Client `kit:"inject"`
}

func speciesOr9606(s int) int {
	if s == 0 {
		return 9606
	}
	return s
}

func resolveProteins(ctx context.Context, in proteinsInput, emit func(*Protein) error) error {
	limit := in.Limit
	if limit <= 0 {
		limit = 10
	}
	proteins, err := in.Client.ResolveProteins(ctx, []string{in.Name}, speciesOr9606(in.Species), limit)
	if err != nil {
		return err
	}
	for _, p := range proteins {
		if err := emit(p); err != nil {
			return err
		}
	}
	return nil
}

func getInteractions(ctx context.Context, in interactionsInput, emit func(*Interaction) error) error {
	limit := in.Limit
	if limit <= 0 {
		limit = 20
	}
	interactions, err := in.Client.GetInteractions(ctx, []string{in.Name}, speciesOr9606(in.Species), limit)
	if err != nil {
		return err
	}
	for _, i := range interactions {
		if err := emit(i); err != nil {
			return err
		}
	}
	return nil
}

func getNetwork(ctx context.Context, in networkInput, emit func(*Interaction) error) error {
	limit := in.Limit
	if limit <= 0 {
		limit = 20
	}
	names := splitNames(in.Names)
	interactions, err := in.Client.GetInteractions(ctx, names, speciesOr9606(in.Species), limit)
	if err != nil {
		return err
	}
	for _, i := range interactions {
		if err := emit(i); err != nil {
			return err
		}
	}
	return nil
}

func getEnrichment(ctx context.Context, in enrichInput, emit func(*Enrichment) error) error {
	names := splitNames(in.Names)
	enrichments, err := in.Client.GetEnrichment(ctx, names, speciesOr9606(in.Species))
	if err != nil {
		return err
	}
	for _, e := range enrichments {
		if err := emit(e); err != nil {
			return err
		}
	}
	return nil
}

func splitNames(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}

func (Domain) Classify(input string) (string, string, error) {
	s := strings.TrimSpace(input)
	if s == "" {
		return "", "", errs.Usage("protein identifier required")
	}
	return "protein", s, nil
}

func (Domain) Locate(t, id string) (string, error) {
	switch t {
	case "protein":
		return fmt.Sprintf("https://string-db.org/network/%s", url.PathEscape(id)), nil
	default:
		return "", errs.Usage("stringdb has no resource type %q", t)
	}
}
