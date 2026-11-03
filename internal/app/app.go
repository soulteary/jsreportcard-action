package app

import (
	"crypto/sha256"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/soulteary/jsreportcard-action/internal/checker"
	"github.com/soulteary/jsreportcard-action/internal/config"
	"github.com/soulteary/jsreportcard-action/internal/model"
	"github.com/soulteary/jsreportcard-action/internal/project"
	"github.com/soulteary/jsreportcard-action/internal/report"
	"io"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func Run(args []string, out, errout io.Writer, version string) int {
	fs := flag.NewFlagSet("jsreportcard-cli", flag.ContinueOnError)
	fs.SetOutput(errout)
	root := fs.String("directory", ".", "project root")
	cfg := fs.String("config", "", "JSON config relative to project root")
	mode := fs.String("mode", "", "standard or typed")
	configMode := fs.String("config-mode", "", "builtin or project (project executes trusted JS configuration)")
	node := fs.String("node", "", "Node executable")
	toolDir := fs.String("tool-dir", "", "installed toolkit directory")
	includes := fs.String("include", "", "comma-separated relative source files/directories")
	configs := fs.String("tsconfigs", "", "comma-separated tsconfig paths")
	output := fs.String("output-dir", ".jsreportcard", "report directory relative to project root")
	baseline := fs.String("baseline", "", "previous complete compatible JSON report")
	policy := fs.String("fail-on", "new-errors", "none, errors, new-errors or findings")
	threshold := fs.Float64("threshold", 0, "minimum score, 0 disables")
	asJSON := fs.Bool("json", false, "emit JSON, preserving exit codes")
	ver := fs.Bool("version", false, "print version")
	if e := fs.Parse(args); e != nil {
		if e == flag.ErrHelp {
			return 0
		}
		return 2
	}
	if *ver {
		fmt.Fprintln(out, version)
		return 0
	}
	fail := func(e error) int { fmt.Fprintln(errout, e); return 2 }
	if fs.NArg() != 0 {
		return fail(fmt.Errorf("unexpected positional arguments"))
	}
	if math.IsNaN(*threshold) || math.IsInf(*threshold, 0) || *threshold < 0 || *threshold > 100 {
		return fail(fmt.Errorf("threshold must be 0..100"))
	}
	switch *policy {
	case "none", "errors", "new-errors", "findings":
	default:
		return fail(fmt.Errorf("invalid fail-on policy"))
	}
	abs, e := filepath.Abs(*root)
	if e != nil {
		return fail(e)
	}
	abs, e = filepath.EvalSymlinks(abs)
	if e != nil {
		return fail(e)
	}
	resolve := func(p string) string {
		if p == "" || filepath.IsAbs(p) {
			return p
		}
		return filepath.Join(abs, p)
	}
	c, e := config.Load(resolve(*cfg))
	if e != nil {
		return fail(e)
	}
	if *mode != "" {
		c.Mode = *mode
	}
	if *configMode != "" {
		c.ConfigMode = *configMode
	}
	if *node != "" {
		c.Node = *node
	}
	if *toolDir != "" {
		c.ToolDir = *toolDir
	}
	if *includes != "" {
		c.Include = strings.Split(*includes, ",")
	}
	if *configs != "" {
		c.TSConfigs = strings.Split(*configs, ",")
	}
	if e = c.Validate(); e != nil {
		return fail(e)
	}
	if strings.ContainsAny(c.Node, "/\\") {
		c.Node = resolve(c.Node)
	}
	if c.ToolDir == "" {
		executable, _ := os.Executable()
		c.ToolDir = filepath.Join(filepath.Dir(executable), "tools")
	} else {
		c.ToolDir = resolve(c.ToolDir)
	}

	for i, include := range c.Include {
		c.Include[i] = strings.TrimSpace(include)
		p := resolve(c.Include[i])
		if _, err := project.Relative(abs, p); err != nil {
			return fail(err)
		}
		if _, err := os.Stat(p); err != nil {
			return fail(fmt.Errorf("invalid include %q: %w", include, err))
		}
	}
	for i, value := range c.TSConfigs {
		c.TSConfigs[i] = strings.TrimSpace(value)
	}
	files, e := project.Discover(abs, c.Exclude)
	if e != nil {
		return fail(e)
	}
	selected := []string{}
	for _, file := range files {
		for _, include := range c.Include {
			path := filepath.Clean(resolve(include))
			rel, e := project.Relative(abs, path)
			if e != nil {
				return fail(e)
			}
			if rel == "." || file == rel || strings.HasPrefix(file, rel+"/") {
				selected = append(selected, file)
				break
			}
		}
	}
	if len(selected) == 0 {
		return fail(fmt.Errorf("no JS/TS source files selected"))
	}
	r := model.Result{Configuration: c, SchemaVersion: 1, ScoringVersion: 1, Mode: c.Mode, Files: len(selected), Checks: []model.Check{}}
	for _, tool := range []string{"eslint", "prettier"} {
		r.Checks = append(r.Checks, checker.Run(abs, selected, c, tool))
	}
	if c.Mode == "typed" {
		t := checker.Run(abs, selected, c, "typescript")
		r.Checks = append(r.Checks, t)
		if t.Status != "error" {
			r.Checks = append(r.Checks, checker.Run(abs, selected, c, "typed-eslint"))
		} else {
			r.Checks = append(r.Checks, model.Check{Tool: "typed-eslint", Status: "skipped", Error: "TypeScript context incomplete", Diagnostics: []model.Diagnostic{}})
		}
	} else {
		for _, tool := range []string{"typescript", "typed-eslint"} {
			r.Checks = append(r.Checks, model.Check{Tool: tool, Status: "skipped", Error: "requires typed mode", Diagnostics: []model.Diagnostic{}})
		}
	}
	signature, e := profileID(abs, c, r.Checks)
	if e != nil {
		return fail(e)
	}
	r.Profile = signature
	prior := map[string]int{}
	currentComplete := true
	for _, check := range r.Checks {
		if check.Status == "error" {
			currentComplete = false
		}
	}
	if *baseline != "" && currentComplete {
		data, e := os.ReadFile(resolve(*baseline))
		if e != nil {
			return fail(e)
		}
		var previous model.Result
		if e = json.Unmarshal(data, &previous); e != nil {
			return fail(e)
		}
		if !previous.Complete || previous.Profile != r.Profile || previous.SchemaVersion != 1 || previous.ScoringVersion != 1 {
			return fail(fmt.Errorf("baseline must be complete and match effective configuration, runtimes and tool versions"))
		}
		for _, check := range previous.Checks {
			for _, d := range check.Diagnostics {
				prior[d.Fingerprint]++
			}
		}
	}
	report.Finalize(abs, &r, prior)
	if e = report.Write(resolve(*output), r); e != nil {
		return fail(e)
	}
	if *asJSON {
		encoder := json.NewEncoder(out)
		encoder.SetIndent("", "  ")
		if e = encoder.Encode(r); e != nil {
			return fail(e)
		}
	} else {
		fmt.Fprintf(out, "JS/TS Report Card: %s (%s, %s)\nReports: %s\n", r.Grade, r.Mode, r.Status, resolve(*output))
	}
	if !r.Complete {
		return 2
	}
	violated := *threshold > 0 && r.Score != nil && *r.Score < *threshold
	for _, check := range r.Checks {
		for _, d := range check.Diagnostics {
			if *policy == "findings" || *policy == "errors" && d.Severity == "error" || *policy == "new-errors" && d.Severity == "error" && d.New {
				violated = true
			}
		}
	}
	if violated {
		return 1
	}
	return 0
}
func profileID(root string, c config.Config, checks []model.Check) (string, error) {
	hash := sha256.New()
	paths := []string{}
	seen := map[string]bool{}
	e := filepath.WalkDir(root, func(p string, d os.DirEntry, e error) error {
		if e != nil {
			return e
		}
		rel, _ := project.Relative(root, p)
		if rel != "." && project.Excluded(rel, c.Exclude) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			return nil
		}
		switch d.Name() {
		case "package.json", "package-lock.json", "npm-shrinkwrap.json", "pnpm-lock.yaml", "yarn.lock":
			paths = append(paths, p)
		}
		return nil
	})
	if e != nil {
		return "", e
	}
	for _, check := range checks {
		paths = append(paths, check.ConfigurationFiles...)
	}
	toolLock := filepath.Join(c.ToolDir, "package-lock.json")
	if _, e := os.Stat(toolLock); e == nil {
		data, e := os.ReadFile(toolLock)
		if e != nil {
			return "", e
		}
		fmt.Fprintf(hash, "tool-lock\x00%s\x00", data)
	}
	sort.Strings(paths)
	for _, p := range paths {
		if seen[p] {
			continue
		}
		seen[p] = true
		data, e := os.ReadFile(p)
		if e != nil {
			continue
		}
		rel, e := project.Relative(root, p)
		if e != nil {
			rel = "external/" + filepath.Base(p)
		}
		fmt.Fprintf(hash, "%s\x00%s\x00", rel, data)
	}
	c.Node = ""
	c.ToolDir = ""
	data, _ := json.Marshal(c)
	hash.Write(data)
	for _, check := range checks {
		fmt.Fprintf(hash, "%s\x00%s\x00%s\x00", check.Tool, check.Version, check.NodeVersion)
	}
	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}
