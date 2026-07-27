package setup

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/equate/ogsd/services/snmp-collector/internal/tui"
)

type envFieldDef struct {
	key         string
	placeholder string
	mask        bool
}

func envFieldsForProfile(profile Profile) []envFieldDef {
	if profile == ProfileAppliance {
		return []envFieldDef{
			{key: "SNMP_COMMUNITY", placeholder: "SNMP community", mask: true},
			{key: "SNMP_DISCOVERY_COMMUNITY", placeholder: "Discovery community (often same)", mask: true},
		}
	}
	return []envFieldDef{
		{key: "MQTT_BROKER", placeholder: "MQTT broker (tls://host:8883)", mask: false},
		{key: "MQTT_PASSWORD", placeholder: "MQTT password", mask: true},
		{key: "SNMP_COMMUNITY", placeholder: "SNMP community", mask: true},
		{key: "SNMP_DISCOVERY_COMMUNITY", placeholder: "Discovery community (often same)", mask: true},
	}
}

func newEnvInputs(theme tui.Theme, profile Profile) []textinput.Model {
	fields := envFieldsForProfile(profile)
	inputs := make([]textinput.Model, len(fields))
	for i, f := range fields {
		inputs[i] = styleTextInput(textinput.New(), theme)
		inputs[i].Placeholder = f.placeholder
		inputs[i].CharLimit = 256
		inputs[i].Width = 50
		if f.mask {
			inputs[i].EchoMode = textinput.EchoPassword
		}
	}
	if len(inputs) > 0 {
		inputs[0].Focus()
	}
	if profile == ProfileAppliance {
		applyApplianceSNMPDefaults(inputs, fields)
	}
	return inputs
}

func applyApplianceSNMPDefaults(inputs []textinput.Model, fields []envFieldDef) {
	for i, field := range fields {
		value, err := envFileValue(applianceComposeEnv, field.key)
		if err != nil || strings.TrimSpace(value) == "" || value == "CHANGE_ME" {
			continue
		}
		inputs[i].SetValue(value)
	}
}

func (m model) sharedEnvValues() (map[string]string, error) {
	fields := envFieldsForProfile(m.profile)
	if len(fields) != len(m.envInputs) {
		return nil, fmt.Errorf("env input count mismatch")
	}
	values := make(map[string]string, len(fields)+2)
	for i, field := range fields {
		values[field.key] = strings.TrimSpace(m.envInputs[i].Value())
	}
	if values["SNMP_DISCOVERY_COMMUNITY"] == "" {
		values["SNMP_DISCOVERY_COMMUNITY"] = values["SNMP_COMMUNITY"]
	}
	if m.profile == ProfileAppliance {
		mqtt, err := applianceMQTTEnv()
		if err != nil {
			return nil, err
		}
		for k, v := range mqtt {
			values[k] = v
		}
	}
	return values, nil
}

func applianceMQTTEnv() (map[string]string, error) {
	broker, err := envFileValue(applianceComposeEnv, "MQTT_BROKER")
	if err != nil || strings.TrimSpace(broker) == "" {
		return nil, fmt.Errorf("internal MQTT broker is not configured (missing %s)", applianceComposeEnv)
	}
	password, err := envFileValue(applianceComposeEnv, "MQTT_PASSWORD")
	if err != nil || strings.TrimSpace(password) == "" {
		password, err = envFileValue(applianceComposeEnv, "MQTT_COLLECTOR_PASSWORD")
		if err != nil || strings.TrimSpace(password) == "" {
			return nil, fmt.Errorf("internal MQTT collector password is not configured (missing %s)", applianceComposeEnv)
		}
	}
	return map[string]string{
		"MQTT_BROKER":   strings.TrimSpace(broker),
		"MQTT_PASSWORD": strings.TrimSpace(password),
	}, nil
}
