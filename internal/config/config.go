package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/karpulix/ai-cli/internal/prompt"
)


type Profile struct {
	APIKey  string `json:"api_key,omitempty"`
	Model   string `json:"model,omitempty"`
	BaseURL string `json:"base_url,omitempty"`
}

type Config struct {
	PromptTemplate string             `json:"prompt_template,omitempty"`
	SystemPrompt   string             `json:"system_prompt,omitempty"` // legacy
	APIKey         string             `json:"api_key,omitempty"`
	Model          string             `json:"model,omitempty"`
	ActiveProfile  string             `json:"active_profile,omitempty"`
	Profiles       map[string]Profile `json:"profiles,omitempty"`
}

func Path() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "ai-cli", "config.json"), nil
}

func Load() (*Config, error) {
	path, err := Path()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			cfg := &Config{Profiles: map[string]Profile{}}
			cfg.migrate()
			if err := cfg.ensurePromptTemplate(); err != nil {
				return nil, err
			}
			return cfg, nil
		}
		return nil, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	cfg.migrate()
	if err := cfg.ensurePromptTemplate(); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (c *Config) ensurePromptTemplate() error {
	changed := false
	if c.PromptTemplate == "" {
		c.PromptTemplate = prompt.Default()
		changed = true
	} else if !prompt.HasPlaceholder(c.PromptTemplate) {
		c.PromptTemplate = prompt.MigrateTemplate(c.PromptTemplate)
		changed = true
	}
	if changed {
		return c.Save()
	}
	return nil
}

func (c *Config) migrate() {
	if c.Profiles == nil {
		c.Profiles = map[string]Profile{}
	}
	if c.PromptTemplate == "" && c.SystemPrompt != "" {
		c.PromptTemplate = c.SystemPrompt
		c.SystemPrompt = ""
	}
	if c.APIKey != "" {
		if _, ok := c.Profiles["default"]; !ok {
			c.Profiles["default"] = Profile{
				APIKey: c.APIKey,
				Model:  c.Model,
			}
		}
		c.APIKey = ""
		c.Model = ""
	}
	if c.ActiveProfile == "" && len(c.Profiles) > 0 {
		names := c.ProfileNames()
		c.ActiveProfile = names[0]
	}
}

func (c *Config) ResetPromptTemplate() error {
	c.PromptTemplate = prompt.Default()
	c.SystemPrompt = ""
	return c.Save()
}

func (c *Config) Save() error {
	path, err := Path()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

func (c *Config) ProfileNames() []string {
	names := make([]string, 0, len(c.Profiles))
	for n := range c.Profiles {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

func (c *Config) HasProfiles() bool {
	return len(c.Profiles) > 0
}

func (c *Config) Active() (Profile, string, error) {
	if !c.HasProfiles() {
		return Profile{}, "", fmt.Errorf("no profiles configured")
	}
	name := c.ActiveProfile
	p, ok := c.Profiles[name]
	if !ok {
		name = c.ProfileNames()[0]
		p = c.Profiles[name]
	}
	if p.APIKey == "" && p.BaseURL == "" {
		return Profile{}, "", fmt.Errorf("profile %q has no api key", name)
	}
	return p, name, nil
}

func (c *Config) SetActive(name string) error {
	if _, ok := c.Profiles[name]; !ok {
		return fmt.Errorf("profile %q not found", name)
	}
	c.ActiveProfile = name
	return c.Save()
}

func (c *Config) Upsert(name string, p Profile) error {
	if name == "" {
		return fmt.Errorf("profile name required")
	}
	c.Profiles[name] = p
	if c.ActiveProfile == "" {
		c.ActiveProfile = name
	}
	return c.Save()
}

func (c *Config) Delete(name string) error {
	if _, ok := c.Profiles[name]; !ok {
		return nil
	}
	delete(c.Profiles, name)
	if c.ActiveProfile == name {
		c.ActiveProfile = ""
		if len(c.Profiles) > 0 {
			c.ActiveProfile = c.ProfileNames()[0]
		}
	}
	return c.Save()
}
