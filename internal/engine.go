package internal

import (
	"regexp"
	"strings"
)

type Engine struct {
	Rules map[string]Rule
}

type Result struct {
	OriginalText   string
	ModifiedText   string
	TriggeredRules []string
}

type AutoDetectPattern struct {
	Name    string
	Pattern *regexp.Regexp
	Replace string
}

var autoDetectPatterns = []AutoDetectPattern{
	{Name: "api_key", Pattern: regexp.MustCompile(`(?i)(api[_-]?key|apikey)[=:]\s*['"]?[A-Za-z0-9_\-]{16,}['"]?`), Replace: "API_KEY_REDACTED"},
	{Name: "token", Pattern: regexp.MustCompile(`(?i)(token|bearer|jwt|auth)[=:]\s*['"]?[A-Za-z0-9_\-\.]{20,}['"]?`), Replace: "TOKEN_REDACTED"},
	{Name: "password", Pattern: regexp.MustCompile(`(?i)(password|passwd|pwd|secret)[=:]\s*['"]?\S+['"]?`), Replace: "PASSWORD_REDACTED"},
	{Name: "email", Pattern: regexp.MustCompile(`[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}`), Replace: "EMAIL_REDACTED"},
	{Name: "ip_address", Pattern: regexp.MustCompile(`\b(?:[0-9]{1,3}\.){3}[0-9]{1,3}\b`), Replace: "IP_REDACTED"},
	{Name: "private_key", Pattern: regexp.MustCompile(`-----BEGIN[ A-Z]*PRIVATE KEY-----`), Replace: "PRIVATE_KEY_REDACTED"},
}

func NewEngine(rules map[string]Rule) *Engine {
	return &Engine{Rules: rules}
}

func (e *Engine) UpdateRules(rules map[string]Rule) {
	e.Rules = rules
}

func (e *Engine) Process(input string) Result {
	modified := input
	var triggered []string

	for k, v := range e.Rules {
		if !v.Enabled {
			continue
		}

		if v.Regex {
			re, err := regexp.Compile(k)
			if err != nil {
				continue
			}
			if re.MatchString(modified) {
				modified = re.ReplaceAllString(modified, v.Replace)
				triggered = append(triggered, k)
			}
		} else {
			if strings.Contains(modified, k) {
				modified = strings.ReplaceAll(modified, k, v.Replace)
				triggered = append(triggered, k)
			}
		}
	}

	return Result{
		OriginalText:   input,
		ModifiedText:   modified,
		TriggeredRules: triggered,
	}
}

func (e *Engine) ProcessWithAutoDetect(input string, patterns []string) Result {
	result := e.Process(input)
	modified := result.ModifiedText

	for _, ap := range autoDetectPatterns {
		enabled := false
		for _, p := range patterns {
			if p == ap.Name {
				enabled = true
				break
			}
		}
		if !enabled {
			continue
		}

		matches := ap.Pattern.FindAllString(modified, -1)
		if len(matches) > 0 {
			modified = ap.Pattern.ReplaceAllString(modified, ap.Replace)
			for range matches {
				result.TriggeredRules = append(result.TriggeredRules, "[auto] "+ap.Name)
			}
		}
	}

	result.ModifiedText = modified
	return result
}

func GetAutoDetectPatterns() []AutoDetectPattern {
	return autoDetectPatterns
}
