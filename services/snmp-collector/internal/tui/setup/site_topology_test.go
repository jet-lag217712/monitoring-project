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
