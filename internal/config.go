package internal

import (
	"encoding/json"
	"os"
	"sync"
)

const configPath = "config/replacements.json"

type Rule struct {
	Replace string `json:"replace"`
	Enabled bool   `json:"enabled"`
	Regex   bool   `json:"regex,omitempty"`
}

type Config struct {
	Words    map[string]string          `json:"words,omitempty"`
	Rules    map[string]Rule            `json:"rules,omitempty"`
	Profile  string                     `json:"profile"`
	Profiles map[string]map[string]Rule `json:"profiles,omitempty"`
	Detect   DetectConfig               `json:"detect"`

	mu sync.RWMutex
}

type DetectConfig struct {
	AutoEnabled bool     `json:"auto_enabled"`
	Patterns    []string `json:"patterns"`
}

var globalConfig *Config

func init() {
	globalConfig = LoadConfig()
}

func GetConfig() *Config {
	return globalConfig
}

func LoadConfig() *Config {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return defaultConfig()
	}

	var c Config
	if err := json.Unmarshal(data, &c); err != nil {
		return defaultConfig()
	}

	if c.Words != nil && c.Rules == nil {
		c.Rules = make(map[string]Rule)
		for k, v := range c.Words {
			c.Rules[k] = Rule{Replace: v, Enabled: true}
		}
		c.Words = nil
	}

	if c.Rules == nil {
		c.Rules = make(map[string]Rule)
	}
	if c.Profiles == nil {
		c.Profiles = make(map[string]map[string]Rule)
	}
	if c.Profile == "" {
		c.Profile = "default"
	}
	return &c
}

func defaultConfig() *Config {
	return &Config{
		Rules:    make(map[string]Rule),
		Profile:  "default",
		Profiles: make(map[string]map[string]Rule),

		Detect: DetectConfig{
			AutoEnabled: false,
			Patterns: []string{
				"email",
				"api_key",
				"token",
				"password",
				"ip_address",
			},
		},
	}
}

func AddRule(search, replace string) {
	c := GetConfig()
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.Profiles[c.Profile] == nil {
		c.Profiles[c.Profile] = make(map[string]Rule)
	}

	c.Profiles[c.Profile][search] = Rule{Replace: replace, Enabled: true}
	c.Rules[search] = Rule{Replace: replace, Enabled: true}
	saveConfig(c)
}

func AddRuleEx(search, replace string, enabled, regex bool) {
	c := GetConfig()
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.Profiles[c.Profile] == nil {
		c.Profiles[c.Profile] = make(map[string]Rule)
	}

	r := Rule{Replace: replace, Enabled: enabled, Regex: regex}
	c.Profiles[c.Profile][search] = r
	c.Rules[search] = r
	saveConfig(c)
}

func RemoveRule(search string) {
	c := GetConfig()
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.Rules, search)
	if c.Profiles[c.Profile] != nil {
		delete(c.Profiles[c.Profile], search)
	}
	saveConfig(c)
}

func ToggleRule(search string) bool {
	c := GetConfig()
	c.mu.Lock()
	defer c.mu.Unlock()

	r, ok := c.Rules[search]
	if !ok {
		return false
	}
	r.Enabled = !r.Enabled
	c.Rules[search] = r
	if c.Profiles[c.Profile] != nil {
		c.Profiles[c.Profile][search] = r
	}
	saveConfig(c)
	return r.Enabled
}

func SetRuleEnabled(search string, enabled bool) bool {
	c := GetConfig()
	c.mu.Lock()
	defer c.mu.Unlock()

	r, ok := c.Rules[search]
	if !ok {
		return false
	}
	r.Enabled = enabled
	c.Rules[search] = r
	if c.Profiles[c.Profile] != nil {
		c.Profiles[c.Profile][search] = r
	}
	saveConfig(c)
	return true
}

func ListRulesMap() map[string]Rule {
	c := GetConfig()
	c.mu.RLock()
	defer c.mu.RUnlock()

	rules := make(map[string]Rule, len(c.Rules))
	for k, v := range c.Rules {
		rules[k] = v
	}
	return rules
}

func GetEnabledRules() map[string]Rule {
	c := GetConfig()
	c.mu.RLock()
	defer c.mu.RUnlock()

	rules := make(map[string]Rule)
	for k, v := range c.Rules {
		if v.Enabled {
			rules[k] = v
		}
	}
	return rules
}

func SaveConfig(c *Config) {
	c.mu.Lock()
	defer c.mu.Unlock()
	saveConfig(c)
}

func saveConfig(c *Config) {
	data, _ := json.MarshalIndent(c, "", "  ")
	os.WriteFile(configPath, data, 0644)
}
