package main

import (
	"testing"
)

func TestParseConfigureArgsTemperature(t *testing.T) {
	opts, err := parseConfigureArgs([]string{"--temperature", "80"})
	if err != nil {
		t.Fatal(err)
	}
	if opts.temperature == nil || *opts.temperature != 80 {
		t.Fatalf("temperature=%v", opts.temperature)
	}
	if opts.mode != "full" {
		t.Fatalf("mode=%s", opts.mode)
	}
}

func TestParseConfigureArgsTemperatureMissingValue(t *testing.T) {
	if _, err := parseConfigureArgs([]string{"--temperature"}); err == nil {
		t.Fatal("expected error")
	}
}

func TestParseConfigureArgsTemperatureCombined(t *testing.T) {
	if _, err := parseConfigureArgs([]string{"--sites", "--temperature", "70"}); err == nil {
		t.Fatal("expected combine error")
	}
}

func TestParseConfigureArgsSites(t *testing.T) {
	opts, err := parseConfigureArgs([]string{"--sites"})
	if err != nil {
		t.Fatal(err)
	}
	if opts.mode != "sites" || opts.temperature != nil {
		t.Fatalf("opts=%+v", opts)
	}
}

func TestParseSitesDeleteArgs(t *testing.T) {
	siteID, yes, err := parseSitesDeleteArgs([]string{"district", "--yes"})
	if err != nil {
		t.Fatal(err)
	}
	if siteID != "district" || !yes {
		t.Fatalf("siteID=%q yes=%v", siteID, yes)
	}
}

func TestParseSitesDeleteArgsMissing(t *testing.T) {
	if _, _, err := parseSitesDeleteArgs([]string{"--yes"}); err == nil {
		t.Fatal("expected error")
	}
}
