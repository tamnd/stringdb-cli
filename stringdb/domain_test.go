package stringdb

import (
	"testing"
)

func TestDomainInfo(t *testing.T) {
	info := Domain{}.Info()
	if info.Scheme != "stringdb" {
		t.Errorf("Scheme = %q, want stringdb", info.Scheme)
	}
	if len(info.Hosts) == 0 || info.Hosts[0] != Host {
		t.Errorf("Hosts = %v, want [%s]", info.Hosts, Host)
	}
	if info.Identity.Binary != "stringdb" {
		t.Errorf("Identity.Binary = %q, want stringdb", info.Identity.Binary)
	}
}

func TestClassify(t *testing.T) {
	typ, id, err := Domain{}.Classify("TP53")
	if err != nil {
		t.Fatalf("Classify error: %v", err)
	}
	if typ != "protein" {
		t.Errorf("type = %q, want protein", typ)
	}
	if id != "TP53" {
		t.Errorf("id = %q, want TP53", id)
	}
}

func TestClassifyStringID(t *testing.T) {
	typ, id, err := Domain{}.Classify("9606.ENSP00000269305")
	if err != nil {
		t.Fatalf("Classify error: %v", err)
	}
	if typ != "protein" {
		t.Errorf("type = %q, want protein", typ)
	}
	_ = id
}

func TestClassifyEmpty(t *testing.T) {
	_, _, err := Domain{}.Classify("")
	if err == nil {
		t.Fatal("expected error for empty input")
	}
}

func TestLocate(t *testing.T) {
	got, err := Domain{}.Locate("protein", "TP53")
	if err != nil {
		t.Fatalf("Locate error: %v", err)
	}
	if got == "" {
		t.Error("Locate returned empty URL")
	}
}
