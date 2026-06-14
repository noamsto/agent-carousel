package main

import (
	"os"
	"testing"
)

func keysOf(m map[string]region) []string {
	var k []string
	for s := range m {
		k = append(k, s)
	}
	return k
}

func TestParseRegionsSketch(t *testing.T) {
	data, err := os.ReadFile("tests/fixtures/regions-sketch.svg")
	if err != nil {
		t.Fatal(err)
	}
	rs := parseRegions(data)
	paths := map[string]region{}
	for _, r := range rs {
		paths[r.path] = r
	}
	for _, want := range []string{"ingest", "store", "ingest.read", "ingest.parse", "store.t"} {
		if _, ok := paths[want]; !ok {
			t.Errorf("missing region %q (got %v)", want, keysOf(paths))
		}
	}
	for got := range paths {
		if got == "(ingest -> store)[0]" || got == "(ingest -&gt; store)[0]" {
			t.Errorf("connection group must be skipped: %q", got)
		}
	}
	ing, ok := paths["ingest"]
	if !ok {
		t.Fatal("no ingest region")
	}
	if !(ing.x0 >= 0 && ing.x1 <= 1.0001 && ing.x0 < ing.x1 && ing.y0 < ing.y1) {
		t.Errorf("ingest bbox not a normalized rect: %+v", ing)
	}
	if rd, ok := paths["ingest.read"]; ok {
		if !(ing.x0 <= rd.x0+1e-6 && ing.x1 >= rd.x1-1e-6 && ing.y0 <= rd.y0+1e-6 && ing.y1 >= rd.y1-1e-6) {
			t.Errorf("ingest must contain ingest.read: %+v vs %+v", ing, rd)
		}
	}
}
