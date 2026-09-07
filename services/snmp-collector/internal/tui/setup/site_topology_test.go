package setup

import "testing"

func TestValidateSiteTopologyRejectsCycle(t *testing.T) {
	specs := []SiteSpec{
		{SiteID: "a", UpstreamSiteIDs: []string{"b"}},
		{SiteID: "b", UpstreamSiteIDs: []string{"a"}},
	}
	if err := ValidateSiteTopology(specs); err == nil {
		t.Fatal("expected cycle error")
	}
}

func TestSuggestSiteTopologyStar(t *testing.T) {
	specs := SuggestSiteTopology([]SiteSpec{
		{SiteID: "do-core"},
		{SiteID: "site-a-mdf"},
		{SiteID: "site-b-mdf"},
	})
	if len(specs[0].UpstreamSiteIDs) != 0 {
		t.Fatalf("core upstream=%v", specs[0].UpstreamSiteIDs)
	}
	if len(specs[1].UpstreamSiteIDs) != 1 || specs[1].UpstreamSiteIDs[0] != "do-core" {
		t.Fatalf("site-a upstream=%v", specs[1].UpstreamSiteIDs)
	}
}

func TestSuggestSiteTopologyNestedIDF(t *testing.T) {
	specs := SuggestSiteTopology([]SiteSpec{
		{SiteID: "do-core"},
		{SiteID: "site-a-mdf"},
		{SiteID: "site-a-idf1"},
		{SiteID: "site-a-idf2"},
		{SiteID: "site-b-mdf"},
		{SiteID: "site-c-mdf"},
		{SiteID: "site-c-idf1"},
	})
	byID := map[string][]string{}
	for _, spec := range specs {
		byID[spec.SiteID] = spec.UpstreamSiteIDs
	}
	if len(byID["do-core"]) != 0 {
		t.Fatalf("core upstream=%v", byID["do-core"])
	}
	if len(byID["site-a-mdf"]) != 1 || byID["site-a-mdf"][0] != "do-core" {
		t.Fatalf("site-a-mdf upstream=%v", byID["site-a-mdf"])
	}
	if len(byID["site-a-idf1"]) != 1 || byID["site-a-idf1"][0] != "site-a-mdf" {
		t.Fatalf("site-a-idf1 upstream=%v", byID["site-a-idf1"])
	}
	if len(byID["site-a-idf2"]) != 1 || byID["site-a-idf2"][0] != "site-a-mdf" {
		t.Fatalf("site-a-idf2 upstream=%v", byID["site-a-idf2"])
	}
	if len(byID["site-b-mdf"]) != 1 || byID["site-b-mdf"][0] != "do-core" {
		t.Fatalf("site-b-mdf upstream=%v", byID["site-b-mdf"])
	}
	if len(byID["site-c-idf1"]) != 1 || byID["site-c-idf1"][0] != "site-c-mdf" {
		t.Fatalf("site-c-idf1 upstream=%v", byID["site-c-idf1"])
	}
}

func TestApplyUpstreamSiteIDs(t *testing.T) {
	specs := []SiteSpec{{SiteID: "do-core"}, {SiteID: "site-a-mdf"}}
	out, err := ApplyUpstreamSiteIDs(specs, [][]string{nil, {"do-core"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(out[1].UpstreamSiteIDs) != 1 || out[1].UpstreamSiteIDs[0] != "do-core" {
		t.Fatalf("upstream=%v", out[1].UpstreamSiteIDs)
	}
}
