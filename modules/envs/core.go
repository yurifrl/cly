// Package envs loads 1Password environment items without depending on CLY internals.
package envs

import (
	"encoding/json"
	"fmt"
	"strings"
)

const (
	defaultAccount = "my.1password.com"
	defaultVault   = "Private"
	defaultProfile = "all"
)

type Config struct {
	DefaultAccount string   `json:"default_account"`
	DefaultVault   string   `json:"default_vault"`
	DefaultProfile string   `json:"default_profile"`
	Secrets        []Secret `json:"secrets"`
}

type Secret struct {
	Name    string `json:"name"`
	Vault   string `json:"vault"`
	Account string `json:"account"`
}

type Field struct {
	Label   string
	Value   string
	Type    string
	Section string
}

type item struct {
	Fields []struct {
		Label   string `json:"label"`
		Value   string `json:"value"`
		Type    string `json:"type"`
		Section *struct {
			Label string `json:"label"`
		} `json:"section"`
	} `json:"fields"`
	Files []Attachment `json:"files"`
}

type Attachment struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func ParseConfig(data []byte) (Config, error) {
	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return Config{}, fmt.Errorf("decode config: %w", err)
	}
	if config.DefaultAccount == "" {
		config.DefaultAccount = defaultAccount
	}
	if config.DefaultVault == "" {
		config.DefaultVault = defaultVault
	}
	if config.DefaultProfile == "" {
		config.DefaultProfile = defaultProfile
	}
	if len(config.Secrets) == 0 {
		return Config{}, fmt.Errorf("config has no secrets")
	}
	for index := range config.Secrets {
		if config.Secrets[index].Name == "" {
			return Config{}, fmt.Errorf("secret %d has no name", index+1)
		}
		if config.Secrets[index].Vault == "" {
			config.Secrets[index].Vault = config.DefaultVault
		}
		if config.Secrets[index].Account == "" {
			config.Secrets[index].Account = config.DefaultAccount
		}
	}
	return config, nil
}

func decodeItem(data []byte) ([]Field, error) {
	value, err := parseItem(data)
	if err != nil {
		return nil, err
	}
	return fieldsFromItem(value), nil
}

func parseItem(data []byte) (item, error) {
	var value item
	if err := json.Unmarshal(data, &value); err != nil {
		return item{}, fmt.Errorf("decode item: %w", err)
	}
	return value, nil
}

func fieldsFromItem(value item) []Field {
	fields := make([]Field, 0, len(value.Fields))
	for _, raw := range value.Fields {
		section := ""
		if raw.Section != nil {
			section = strings.ToLower(raw.Section.Label)
		}
		fields = append(fields, Field{Label: raw.Label, Value: raw.Value, Type: raw.Type, Section: section})
	}
	return fields
}

func SelectFields(fields []Field, profile string) []Field {
	selected := make([]Field, 0, len(fields))
	for _, field := range fields {
		if !validEnvironmentLabel(field.Label) || field.Value == "" || field.Value == "null" {
			continue
		}
		if (field.Section == "work" || field.Section == "personal") && profile != "all" && field.Section != profile {
			continue
		}
		selected = append(selected, field)
	}
	return selected
}

func validEnvironmentLabel(label string) bool {
	if label == "" || label == "username" || label == "password" || label == "notesPlain" || label == "server" || label == "port" || label == "database" {
		return false
	}
	for index, char := range label {
		if !(char == '_' || char >= 'a' && char <= 'z' || char >= 'A' && char <= 'Z' || (index > 0 && char >= '0' && char <= '9')) {
			return false
		}
	}
	return true
}
