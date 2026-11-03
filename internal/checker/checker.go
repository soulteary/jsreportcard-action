package checker

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"github.com/soulteary/jsreportcard-action/internal/config"
	"github.com/soulteary/jsreportcard-action/internal/model"
	"os"
	"os/exec"
	"strings"
)

//go:embed worker.cjs
var worker string

func Run(root string, files []string, c config.Config, name string) model.Check {
	result := model.Check{Tool: name, Status: "error", Total: len(files), Diagnostics: []model.Diagnostic{}}
	request := struct {
		Root   string        `json:"root"`
		Files  []string      `json:"files"`
		Config config.Config `json:"config"`
		Check  string        `json:"check"`
	}{root, files, c, name}
	input, e := json.Marshal(request)
	if e != nil {
		result.Error = e.Error()
		return result
	}
	ctx, cancel := context.WithTimeout(context.Background(), c.Timeout())
	defer cancel()
	cmd := exec.CommandContext(ctx, c.Node, "-e", worker)
	cmd.Dir = root
	cmd.Stdin = bytes.NewReader(input)
	for _, value := range os.Environ() {
		if !strings.HasPrefix(value, "NODE_OPTIONS=") && !strings.HasPrefix(value, "NODE_PATH=") {
			cmd.Env = append(cmd.Env, value)
		}
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	output, e := cmd.Output()
	if e != nil {
		result.Error = fmt.Sprintf("%s: %v: %s", name, e, stderr.String())
		return result
	}
	if e = json.Unmarshal(output, &result); e != nil {
		result.Status = "error"
		result.Error = "Invalid worker JSON: " + e.Error()
		return result
	}
	if result.Tool != name || result.Total != len(files) || result.Analyzed > result.Total {
		result.Status = "error"
		result.Error = "Inconsistent worker response"
	}
	return result
}
