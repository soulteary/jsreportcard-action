package config

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"
)

type Config struct {
	Mode            string   `json:"mode"`
	ConfigMode      string   `json:"config_mode"`
	Node            string   `json:"node"`
	ToolDir         string   `json:"tool_dir"`
	Include         []string `json:"include"`
	Exclude         []string `json:"exclude"`
	TSConfigs       []string `json:"tsconfigs"`
	Environment     string   `json:"environment"`
	ComplexityLimit int      `json:"complexity_limit"`
	TimeoutSeconds  int      `json:"timeout_seconds"`
}

func Default() Config {
	return Config{Mode: "standard", ConfigMode: "builtin", Node: "node", Include: []string{"."}, Exclude: []string{".git", "node_modules", "dist", "build", "coverage", "vendor", "generated"}, TSConfigs: []string{"tsconfig.json"}, Environment: "universal", ComplexityLimit: 10, TimeoutSeconds: 120}
}
func (c Config) Validate() error {
	if c.Mode != "standard" && c.Mode != "typed" {
		return fmt.Errorf("mode must be standard or typed")
	}
	if c.ConfigMode != "builtin" && c.ConfigMode != "project" {
		return fmt.Errorf("config_mode must be builtin or project")
	}
	if c.Environment != "node" && c.Environment != "browser" && c.Environment != "universal" {
		return fmt.Errorf("environment must be node, browser or universal")
	}
	if c.Node == "" || len(c.Include) == 0 || c.ComplexityLimit < 1 || c.TimeoutSeconds < 1 {
		return fmt.Errorf("node, include and positive limits are required")
	}
	if c.Mode == "typed" && len(c.TSConfigs) == 0 {
		return fmt.Errorf("typed mode requires tsconfigs")
	}
	return nil
}
func Load(path string) (Config, error) {
	c := Default()
	if path != "" {
		f, e := os.Open(path)
		if e != nil {
			return c, e
		}
		defer f.Close()
		d := json.NewDecoder(f)
		d.DisallowUnknownFields()
		if e = d.Decode(&c); e != nil {
			return c, e
		}
		if e = d.Decode(new(any)); e != io.EOF {
			return c, fmt.Errorf("expected exactly one configuration object")
		}
	}
	return c, c.Validate()
}
func (c Config) Timeout() time.Duration { return time.Duration(c.TimeoutSeconds) * time.Second }
