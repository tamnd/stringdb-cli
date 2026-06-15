package stringdb

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const Host = "string-db.org"
const baseURL = "https://string-db.org/api/json"

type Config struct {
	BaseURL   string
	CallerID  string
	Rate      time.Duration
	Retries   int
	Timeout   time.Duration
	UserAgent string
}

func DefaultConfig() Config {
	return Config{
		BaseURL:   baseURL,
		CallerID:  "stringdb-cli",
		Rate:      500 * time.Millisecond,
		Retries:   3,
		Timeout:   30 * time.Second,
		UserAgent: "stringdb-cli/0.1.0 (github.com/tamnd/stringdb-cli)",
	}
}

type Client struct {
	cfg  Config
	http *http.Client
	last time.Time
}

func NewClient(cfg Config) *Client {
	return &Client{
		cfg:  cfg,
		http: &http.Client{Timeout: cfg.Timeout},
	}
}

func (c *Client) wait() {
	if c.cfg.Rate > 0 {
		if since := time.Since(c.last); since < c.cfg.Rate {
			time.Sleep(c.cfg.Rate - since)
		}
	}
	c.last = time.Now()
}

func (c *Client) get(ctx context.Context, rawURL string, out any) error {
	for attempt := 0; attempt <= c.cfg.Retries; attempt++ {
		if attempt > 0 {
			d := time.Duration(attempt) * 500 * time.Millisecond
			if d > 5*time.Second {
				d = 5 * time.Second
			}
			time.Sleep(d)
		}
		c.wait()
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
		if err != nil {
			return err
		}
		req.Header.Set("User-Agent", c.cfg.UserAgent)
		resp, err := c.http.Do(req)
		if err != nil {
			if attempt < c.cfg.Retries {
				continue
			}
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode == http.StatusNotFound {
			return fmt.Errorf("not found")
		}
		if resp.StatusCode != http.StatusOK {
			if attempt < c.cfg.Retries {
				continue
			}
			return fmt.Errorf("HTTP %d", resp.StatusCode)
		}
		return json.NewDecoder(resp.Body).Decode(out)
	}
	return fmt.Errorf("all retries exhausted")
}

func (c *Client) buildURL(endpoint string, params map[string]string) string {
	q := url.Values{}
	for k, v := range params {
		q.Set(k, v)
	}
	q.Set("caller_identity", c.cfg.CallerID)
	return c.cfg.BaseURL + "/" + endpoint + "?" + q.Encode()
}

// --- wire/output types ---

// Protein represents a resolved STRING protein.
type Protein struct {
	StringID   string `json:"string_id"          kit:"id"`
	Name       string `json:"name"`
	TaxonName  string `json:"taxon_name"`
	TaxonID    string `json:"taxon_id"`
	Annotation string `json:"annotation,omitempty"`
}

type wireProtein struct {
	StringID    string `json:"stringId"`
	NcbiTaxonID string `json:"ncbiTaxonId"`
	TaxonName   string `json:"taxonName"`
	PrefName    string `json:"preferredName"`
	Annotation  string `json:"annotation"`
}

func toProtein(w wireProtein) *Protein {
	return &Protein{
		StringID:   w.StringID,
		Name:       w.PrefName,
		TaxonName:  w.TaxonName,
		TaxonID:    w.NcbiTaxonID,
		Annotation: w.Annotation,
	}
}

// Interaction represents a protein-protein interaction edge.
type Interaction struct {
	ProteinA   string  `json:"protein_a"           kit:"id"`
	ProteinB   string  `json:"protein_b"`
	NameA      string  `json:"name_a"`
	NameB      string  `json:"name_b"`
	Score      float64 `json:"score"`
	TextScore  float64 `json:"text_score,omitempty"`
	ExpScore   float64 `json:"exp_score,omitempty"`
	DbScore    float64 `json:"db_score,omitempty"`
	CoexpScore float64 `json:"coexp_score,omitempty"`
}

type wireInteraction struct {
	StringIDA   string  `json:"stringId_A"`
	StringIDB   string  `json:"stringId_B"`
	PrefNameA   string  `json:"preferredName_A"`
	PrefNameB   string  `json:"preferredName_B"`
	NcbiTaxonID string  `json:"ncbiTaxonId"`
	Score       float64 `json:"score"`
	NScore      float64 `json:"nscore"`
	FScore      float64 `json:"fscore"`
	PScore      float64 `json:"pscore"`
	AScore      float64 `json:"ascore"`
	EScore      float64 `json:"escore"`
	DScore      float64 `json:"dscore"`
	TScore      float64 `json:"tscore"`
}

func toInteraction(w wireInteraction) *Interaction {
	return &Interaction{
		ProteinA:   w.PrefNameA,
		ProteinB:   w.PrefNameB,
		NameA:      w.PrefNameA,
		NameB:      w.PrefNameB,
		Score:      w.Score,
		TextScore:  w.TScore,
		ExpScore:   w.EScore,
		DbScore:    w.DScore,
		CoexpScore: w.AScore,
	}
}

// Enrichment represents a functional enrichment term.
type Enrichment struct {
	Category    string `json:"category"              kit:"id"`
	Term        string `json:"term"`
	Description string `json:"description"`
	Genes       int    `json:"genes"`
	FDR         string `json:"fdr"`
	InputGenes  string `json:"input_genes,omitempty"`
}

type wireEnrichment struct {
	Category            string `json:"category"`
	Term                string `json:"term"`
	NumberOfGenes       int    `json:"number_of_genes"`
	NumberOfGenesInBg   int    `json:"number_of_genes_in_background"`
	NcbiTaxonID         string `json:"ncbiTaxonId"`
	InputGenes          string `json:"inputGenes"`
	PreferredNames      string `json:"preferredNames"`
	FDR                 string `json:"fdr"`
	Description         string `json:"description"`
}

func toEnrichment(w wireEnrichment) *Enrichment {
	return &Enrichment{
		Category:    w.Category,
		Term:        w.Term,
		Description: w.Description,
		Genes:       w.NumberOfGenes,
		FDR:         w.FDR,
		InputGenes:  w.PreferredNames,
	}
}

// ResolveProteins looks up STRING IDs for the given protein names.
func (c *Client) ResolveProteins(ctx context.Context, names []string, species int, limit int) ([]*Protein, error) {
	params := map[string]string{
		"identifiers": strings.Join(names, "\n"),
		"species":     fmt.Sprintf("%d", species),
		"limit":       fmt.Sprintf("%d", limit),
	}
	u := c.buildURL("get_string_ids", params)
	var wire []wireProtein
	if err := c.get(ctx, u, &wire); err != nil {
		return nil, err
	}
	out := make([]*Protein, len(wire))
	for i, w := range wire {
		out[i] = toProtein(w)
	}
	return out, nil
}

// GetInteractions returns interactions for the given protein names.
func (c *Client) GetInteractions(ctx context.Context, names []string, species int, limit int) ([]*Interaction, error) {
	params := map[string]string{
		"identifiers": strings.Join(names, "\n"),
		"species":     fmt.Sprintf("%d", species),
		"limit":       fmt.Sprintf("%d", limit),
	}
	u := c.buildURL("network", params)
	var wire []wireInteraction
	if err := c.get(ctx, u, &wire); err != nil {
		return nil, err
	}
	out := make([]*Interaction, len(wire))
	for i, w := range wire {
		out[i] = toInteraction(w)
	}
	return out, nil
}

// GetEnrichment returns functional enrichment for a set of proteins.
func (c *Client) GetEnrichment(ctx context.Context, names []string, species int) ([]*Enrichment, error) {
	params := map[string]string{
		"identifiers": strings.Join(names, "\n"),
		"species":     fmt.Sprintf("%d", species),
	}
	u := c.buildURL("enrichment", params)
	var wire []wireEnrichment
	if err := c.get(ctx, u, &wire); err != nil {
		return nil, err
	}
	out := make([]*Enrichment, len(wire))
	for i, w := range wire {
		out[i] = toEnrichment(w)
	}
	return out, nil
}
