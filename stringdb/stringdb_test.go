package stringdb

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func testServer(t *testing.T, mux *http.ServeMux) (*httptest.Server, *Client) {
	t.Helper()
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	cfg := DefaultConfig()
	cfg.BaseURL = srv.URL
	cfg.Rate = 0
	cfg.Retries = 0
	return srv, NewClient(cfg)
}

func TestResolveProteins(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/get_string_ids", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]wireProtein{
			{
				StringID:    "9606.ENSP00000269305",
				NcbiTaxonID: "9606",
				TaxonName:   "Homo sapiens",
				PrefName:    "TP53",
				Annotation:  "Tumor suppressor p53",
			},
		})
	})
	_, client := testServer(t, mux)
	proteins, err := client.ResolveProteins(context.Background(), []string{"TP53"}, 9606, 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(proteins) != 1 {
		t.Fatalf("len = %d, want 1", len(proteins))
	}
	p := proteins[0]
	if p.StringID != "9606.ENSP00000269305" {
		t.Errorf("StringID = %q", p.StringID)
	}
	if p.Name != "TP53" {
		t.Errorf("Name = %q, want TP53", p.Name)
	}
}

func TestGetInteractions(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/network", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]wireInteraction{
			{
				StringIDA: "9606.ENSP00000269305",
				StringIDB: "9606.ENSP00000254719",
				PrefNameA: "TP53",
				PrefNameB: "RPA1",
				Score:     0.9,
				EScore:    0.8,
			},
			{
				StringIDA: "9606.ENSP00000269305",
				StringIDB: "9606.ENSP00000350432",
				PrefNameA: "TP53",
				PrefNameB: "MDM2",
				Score:     0.999,
				EScore:    0.9,
			},
		})
	})
	_, client := testServer(t, mux)
	interactions, err := client.GetInteractions(context.Background(), []string{"TP53"}, 9606, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(interactions) != 2 {
		t.Fatalf("len = %d, want 2", len(interactions))
	}
	i := interactions[0]
	if i.NameA != "TP53" {
		t.Errorf("NameA = %q, want TP53", i.NameA)
	}
	if i.Score != 0.9 {
		t.Errorf("Score = %v, want 0.9", i.Score)
	}
}

func TestGetEnrichment(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/enrichment", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]wireEnrichment{
			{
				Category:       "Process",
				Term:           "GO:0000077",
				Description:    "DNA damage checkpoint signaling",
				NumberOfGenes:  2,
				FDR:            "4.5e-03",
				PreferredNames: "BRCA1,TP53",
			},
		})
	})
	_, client := testServer(t, mux)
	enrichments, err := client.GetEnrichment(context.Background(), []string{"TP53", "BRCA1"}, 9606)
	if err != nil {
		t.Fatal(err)
	}
	if len(enrichments) != 1 {
		t.Fatalf("len = %d, want 1", len(enrichments))
	}
	e := enrichments[0]
	if e.Term != "GO:0000077" {
		t.Errorf("Term = %q", e.Term)
	}
	if e.Description != "DNA damage checkpoint signaling" {
		t.Errorf("Description = %q", e.Description)
	}
}

func TestGetInteractionsEmpty(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/network", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]wireInteraction{})
	})
	_, client := testServer(t, mux)
	interactions, err := client.GetInteractions(context.Background(), []string{"UNKNOWN999"}, 9606, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(interactions) != 0 {
		t.Errorf("expected 0 interactions, got %d", len(interactions))
	}
}
