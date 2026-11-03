package project

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDiscovery(t *testing.T) {
	root := t.TempDir()
	for _, file := range []string{"src/a.ts", "src/b.tsx", "src/c.mjs", "src/types.d.ts", "dist/a.js", "node_modules/pkg/a.js"} {
		p := filepath.Join(root, file)
		os.MkdirAll(filepath.Dir(p), 0755)
		os.WriteFile(p, nil, 0644)
	}
	files, e := Discover(root, []string{"dist", "node_modules"})
	if e != nil || len(files) != 3 {
		t.Fatalf("%v %v", files, e)
	}
}
