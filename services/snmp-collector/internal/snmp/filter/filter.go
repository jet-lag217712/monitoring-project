// Package filter applies interface selection without mutating core readings.
package filter

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/equate/ogsd/services/snmp-collector/internal/config"
	"github.com/equate/ogsd/services/snmp-collector/internal/snmp/core"
	"github.com/equate/ogsd/services/snmp-collector/internal/snmp/readings"
)

var defaultExcludedTypes = map[string]struct{}{
	"softwareloopback": {},
	"l2vlan":           {},
	"propvirtual":      {},
	"tunnel":           {},
	"bridge":           {},
}

type compiledRule struct {
	action      string
	id          string
	ifIndex     *int
	name        *regexp.Regexp
	alias       *regexp.Regexp
	ifType      string
	adminStatus string
	operStatus  string
}

// Filter is an immutable compiled interface policy.
type Filter struct {
	includes []compiledRule
	excludes []compiledRule
}

// New validates and compiles a configured interface policy.
func New(cfg config.InterfaceFilterConfig) (*Filter, error) {
	filter := &Filter{}
	add := func(rule compiledRule) {
		if rule.action == "include" {
			filter.includes = append(filter.includes, rule)
		} else {
			filter.excludes = append(filter.excludes, rule)
		}
	}

	for i, index := range cfg.IncludeIfIndexes {
		value := index
		add(compiledRule{action: "include", id: fmt.Sprintf("include_if_indexes[%d]", i), ifIndex: &value})
	}
	for i, pattern := range cfg.IncludeNameRegex {
		rule, err := regexRule("include", fmt.Sprintf("include_name_regex[%d]", i), pattern, false)
		if err != nil {
			return nil, err
		}
		add(rule)
	}
	for i, pattern := range cfg.IncludeAliasRegex {
		rule, err := regexRule("include", fmt.Sprintf("include_alias_regex[%d]", i), pattern, true)
		if err != nil {
			return nil, err
		}
		add(rule)
	}
	for i, value := range cfg.IncludeTypes {
		add(compiledRule{action: "include", id: fmt.Sprintf("include_types[%d]", i), ifType: normalizeType(value)})
	}
	for i, value := range cfg.IncludeAdminStatuses {
		add(compiledRule{action: "include", id: fmt.Sprintf("include_admin_statuses[%d]", i), adminStatus: strings.ToLower(value)})
	}
	for i, value := range cfg.IncludeOperStatuses {
		add(compiledRule{action: "include", id: fmt.Sprintf("include_oper_statuses[%d]", i), operStatus: strings.ToLower(value)})
	}

	for i, rule := range cfg.Rules {
		compiled, err := compileRule(rule, fmt.Sprintf("rules[%d]", i))
		if err != nil {
			return nil, err
		}
		add(compiled)
	}

	for i, index := range cfg.ExcludeIfIndexes {
		value := index
		add(compiledRule{action: "exclude", id: fmt.Sprintf("exclude_if_indexes[%d]", i), ifIndex: &value})
	}
	for i, pattern := range cfg.ExcludeNameRegex {
		rule, err := regexRule("exclude", fmt.Sprintf("exclude_name_regex[%d]", i), pattern, false)
		if err != nil {
			return nil, err
		}
		add(rule)
	}
	for i, pattern := range cfg.ExcludeAliasRegex {
		rule, err := regexRule("exclude", fmt.Sprintf("exclude_alias_regex[%d]", i), pattern, true)
		if err != nil {
			return nil, err
		}
		add(rule)
	}
	for i, value := range cfg.ExcludeTypes {
		add(compiledRule{action: "exclude", id: fmt.Sprintf("exclude_types[%d]", i), ifType: normalizeType(value)})
	}
	for i, value := range cfg.ExcludeAdminStatuses {
		add(compiledRule{action: "exclude", id: fmt.Sprintf("exclude_admin_statuses[%d]", i), adminStatus: strings.ToLower(value)})
	}
	for i, value := range cfg.ExcludeOperStatuses {
		add(compiledRule{action: "exclude", id: fmt.Sprintf("exclude_oper_statuses[%d]", i), operStatus: strings.ToLower(value)})
	}
	return filter, nil
}

// Apply annotates every interface and updates only the filter-owned summary.
func (f *Filter) Apply(result *readings.DevicePollResult) {
	result.Filter = readings.FilterSummary{}
	for i := range result.Interfaces {
		iface := &result.Interfaces[i]
		iface.Selection = readings.Selected
		iface.FilterReason = ""
		iface.RuleID = ""

		if _, excluded := defaultExcludedTypes[normalizeType(iface.Reading.IfTypeName)]; excluded {
			iface.Selection = readings.ExcludedDefault
			iface.FilterReason = "default virtual interface type"
		}
		for _, rule := range f.includes {
			if rule.matches(iface.Reading) {
				iface.Selection = readings.Selected
				iface.FilterReason = "explicit include"
				iface.RuleID = rule.id
			}
		}
		for _, rule := range f.excludes {
			if rule.matches(iface.Reading) {
				iface.Selection = readings.ExcludedRule
				iface.FilterReason = "explicit exclude"
				iface.RuleID = rule.id
			}
		}

		switch iface.Selection {
		case readings.Selected:
			result.Filter.Selected++
		case readings.ExcludedDefault:
			result.Filter.ExcludedDefault++
		case readings.ExcludedRule:
			result.Filter.ExcludedRule++
		}
	}
}

func regexRule(action, id, pattern string, alias bool) (compiledRule, error) {
	compiled, err := regexp.Compile(pattern)
	if err != nil {
		return compiledRule{}, fmt.Errorf("%s: %w", id, err)
	}
	rule := compiledRule{action: action, id: id}
	if alias {
		rule.alias = compiled
	} else {
		rule.name = compiled
	}
	return rule, nil
}

func compileRule(rule config.InterfaceFilterRule, id string) (compiledRule, error) {
	compiled := compiledRule{
		action:      rule.Action,
		id:          id,
		ifIndex:     rule.IfIndex,
		ifType:      normalizeType(rule.IfType),
		adminStatus: strings.ToLower(rule.AdminStatus),
		operStatus:  strings.ToLower(rule.OperStatus),
	}
	var err error
	if rule.NameRegex != "" {
		compiled.name, err = regexp.Compile(rule.NameRegex)
		if err != nil {
			return compiledRule{}, fmt.Errorf("%s.name_regex: %w", id, err)
		}
	}
	if rule.AliasRegex != "" {
		compiled.alias, err = regexp.Compile(rule.AliasRegex)
		if err != nil {
			return compiledRule{}, fmt.Errorf("%s.alias_regex: %w", id, err)
		}
	}
	return compiled, nil
}

func (r compiledRule) matches(iface core.InterfaceReading) bool {
	if r.ifIndex != nil && iface.IfIndex != *r.ifIndex {
		return false
	}
	if r.name != nil && !r.name.MatchString(iface.IfName) && !r.name.MatchString(iface.IfDescr) {
		return false
	}
	if r.alias != nil && !r.alias.MatchString(iface.IfAlias) {
		return false
	}
	if r.ifType != "" && normalizeType(iface.IfTypeName) != r.ifType && strconv.Itoa(iface.IfType) != r.ifType {
		return false
	}
	if r.adminStatus != "" && strings.ToLower(iface.AdminStatus) != r.adminStatus {
		return false
	}
	if r.operStatus != "" && strings.ToLower(iface.OperStatus) != r.operStatus {
		return false
	}
	return true
}

func normalizeType(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}
