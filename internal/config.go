package internal

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
)

const configPath = "replacements.json"

type Rule struct {
	Replace string `json:"replace"`
	Enabled bool   `json:"enabled"`
	Regex   bool   `json:"regex,omitempty"`
}

type Config struct {
	Words       map[string]string            `json:"words,omitempty"`
	Rules       map[string]Rule              `json:"rules,omitempty"`
	Profile     string                       `json:"profile"`
	Profiles    map[string]map[string]Rule   `json:"profiles,omitempty"`
	Detect      DetectConfig                 `json:"detect"`

	mu          sync.RWMutex
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
		Rules:       make(map[string]Rule),
		Profile:     "default",
		Profiles:    make(map[string]map[string]Rule),

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

func ListRules() {
	c := GetConfig()
	c.mu.RLock()
	defer c.mu.RUnlock()

	fmt.Println(BoxTop)
	fmt.Printf("  %s📋 Reglas activas:%s\n", ColorPrimary, ColorReset)
	fmt.Println("")
	for k, v := range c.Rules {
		status := "🟢"
		if !v.Enabled {
			status = "🔴"
		}
		kind := ""
		if v.Regex {
			kind = " [regex]"
		}
		fmt.Printf("  %s%s %-22s > %s%s%s\n", ColorGreen, status, k, v.Replace, kind, ColorReset)
	}
	fmt.Println(BoxBottom)
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

func (m *Metrics) PrintStats() {
	fmt.Println(BoxTop)
	fmt.Printf("  %s📊 Estadísticas de protección%s\n", ColorPrimary, ColorReset)
	fmt.Printf("\n  %sTotal de reemplazos: %-15d%s", ColorGreen, m.TotalHits, ColorReset)
	fmt.Println("")

	if len(m.RuleHits) == 0 {
		fmt.Printf("  %sNo hay reglas activadas aún%s\n", ColorYellow, ColorReset)
		fmt.Println(BoxBottom)
		return
	}

	fmt.Println("  Detalle por regla:")
	for k, v := range m.RuleHits {
		fmt.Printf("  %s• %-20s: %d veces%s\n", ColorGreen, k, v, ColorReset)
	}
	fmt.Println(BoxBottom)
}

func SwitchProfile(name string) bool {
	c := GetConfig()
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, ok := c.Profiles[name]; !ok {
		return false
	}

	c.Profile = name
	c.Rules = make(map[string]Rule)
	for k, v := range c.Profiles[name] {
		c.Rules[k] = v
	}
	saveConfig(c)
	return true
}

func CreateProfile(name string, rules map[string]Rule) {
	c := GetConfig()
	c.mu.Lock()
	defer c.mu.Unlock()

	c.Profiles[name] = rules
	saveConfig(c)
}

func ListProfiles() []string {
	c := GetConfig()
	c.mu.RLock()
	defer c.mu.RUnlock()

	profiles := make([]string, 0, len(c.Profiles))
	for k := range c.Profiles {
		profiles = append(profiles, k)
	}
	return profiles
}
