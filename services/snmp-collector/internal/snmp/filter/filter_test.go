package filter

import (
	"testing"

	"github.com/equate/ogsd/services/snmp-collector/internal/config"
	"github.com/equate/ogsd/services/snmp-collector/internal/snmp/core"
	"github.com/equate/ogsd/services/snmp-collector/internal/snmp/readings"
)

func TestApplyAnnotatesEveryInterface(t *testing.T) {
	t.Parallel()

	includeLoopback := 2
	filter, err := New(config.InterfaceFilterConfig{
		Rules: []config.InterfaceFilterRule{
			{Action: "include", IfIndex: &includeLoopback},
			{Action: "exclude", AliasRegex: "blocked"},
		},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	result := readings.DevicePollResult{Interfaces: []readings.InterfaceResult{
		{Reading: core.InterfaceReading{IfIndex: 1, IfType: 6, IfTypeName: "ethernetcsmacd"}},
		{Reading: core.InterfaceReading{IfIndex: 2, IfType: 24, IfTypeName: "softwareloopback"}},
		{Reading: core.InterfaceReading{IfIndex: 3, IfType: 6, IfTypeName: "ethernetcsmacd", IfAlias: "blocked uplink"}},
		{Reading: core.InterfaceReading{IfIndex: 4, IfType: 135, IfTypeName: "l2vlan"}},
	}}

	filter.Apply(&result)

	want := []readings.Selection{
		readings.Selected,
		readings.Selected,
		readings.ExcludedRule,
		readings.ExcludedDefault,
	}
	for i, selection := range want {
		if result.Interfaces[i].Selection != selection {
			t.Fatalf("interface %d selection=%q, want %q", i, result.Interfaces[i].Selection, selection)
		}
	}
	if result.Interfaces[1].RuleID != "rules[0]" {
		t.Fatalf("include rule ID=%q", result.Interfaces[1].RuleID)
	}
	if result.Interfaces[2].RuleID != "rules[1]" {
		t.Fatalf("exclude rule ID=%q", result.Interfaces[2].RuleID)
	}
	if result.Filter.Selected != 2 || result.Filter.ExcludedDefault != 1 || result.Filter.ExcludedRule != 1 {
		t.Fatalf("summary=%#v", result.Filter)
	}
}

func TestExplicitExcludeWinsOverInclude(t *testing.T) {
	t.Parallel()

	index := 7
	filter, err := New(config.InterfaceFilterConfig{Rules: []config.InterfaceFilterRule{
		{Action: "exclude", IfIndex: &index},
		{Action: "include", IfIndex: &index},
	}})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	result := readings.DevicePollResult{Interfaces: []readings.InterfaceResult{{
		Reading: core.InterfaceReading{IfIndex: index, IfTypeName: "ethernetcsmacd"},
	}}}
	filter.Apply(&result)
	if result.Interfaces[0].Selection != readings.ExcludedRule {
		t.Fatalf("selection=%q", result.Interfaces[0].Selection)
	}
}
