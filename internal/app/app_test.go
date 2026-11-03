package app

import (
	"bytes"
	"encoding/json"
	"github.com/soulteary/jsreportcard-action/internal/model"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestMissingNodeStillReports(t *testing.T) {
	d := t.TempDir()
	os.WriteFile(filepath.Join(d, "x.js"), []byte("const x=1;\n"), 0644)
	var out, err bytes.Buffer
	code := Run([]string{"--directory", d, "--node", "/missing/node", "--json", "--fail-on", "none"}, &out, &err, "test")
	if code != 2 {
		t.Fatalf("%d: %s", code, err.String())
	}
	var r model.Result
	if e := json.Unmarshal(out.Bytes(), &r); e != nil {
		t.Fatal(e)
	}
	if r.Complete || r.Score != nil {
		t.Fatal(r)
	}
}

func TestBuiltinDoesNotAutoloadProjectToolkit(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("Node unavailable")
	}
	root := t.TempDir()
	os.WriteFile(filepath.Join(root, "x.js"), []byte("export const x = 1;\n"), 0644)
	pkg := filepath.Join(root, "tools", "node_modules", "eslint")
	os.MkdirAll(pkg, 0755)
	os.WriteFile(filepath.Join(root, "tools", "package.json"), []byte(`{"private":true}`), 0644)
	marker := filepath.Join(root, "executed")
	encoded, _ := json.Marshal(marker)
	os.WriteFile(filepath.Join(pkg, "index.js"), []byte("require('node:fs').writeFileSync("+string(encoded)+",'bad');module.exports={};"), 0644)
	var out, errs bytes.Buffer
	Run([]string{"--directory", root, "--node", node}, &out, &errs, "test")
	if _, e := os.Stat(marker); e == nil {
		t.Fatal("Automatically executed project toolkit")
	}
}

func TestBadgeLabelOption(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "x.js"), []byte("const x = 1;\n"), 0644); err != nil {
		t.Fatal(err)
	}
	var profile string
	for _, label := range []string{"js-ts", "js", "ts"} {
		var out, errs bytes.Buffer
		code := Run([]string{"--directory", root, "--node", "/missing/node", "--json", "--badge-label", label}, &out, &errs, "test")
		if code != 2 {
			t.Fatalf("expected incomplete analysis: %d %s", code, errs.String())
		}
		var result model.Result
		if err := json.Unmarshal(out.Bytes(), &result); err != nil {
			t.Fatal(err)
		}
		if result.BadgeLabel != label {
			t.Fatalf("badge_label = %q", result.BadgeLabel)
		}
		if profile != "" && profile != result.Profile {
			t.Fatal("display label changed analysis profile")
		}
		profile = result.Profile
		svg, err := os.ReadFile(filepath.Join(root, ".jsreportcard", "badge.svg"))
		if err != nil {
			t.Fatal(err)
		}
		want := map[string]string{"js-ts": "JS/TS Report", "js": "JS Report", "ts": "TS Report"}[label]
		if !strings.Contains(string(svg), want+": Incomplete") {
			t.Fatalf("unexpected badge: %s", svg)
		}
	}
	var out, errs bytes.Buffer
	if code := Run([]string{"--badge-label", "invalid", "--directory", root}, &out, &errs, "test"); code != 2 || !strings.Contains(errs.String(), "badge-label must be") {
		t.Fatalf("invalid label accepted: %d %s", code, errs.String())
	}
}
