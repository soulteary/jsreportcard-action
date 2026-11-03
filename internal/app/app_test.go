package app

import (
	"bytes"
	"encoding/json"
	"github.com/soulteary/jsreportcard-action/internal/model"
	"os"
	"path/filepath"
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
