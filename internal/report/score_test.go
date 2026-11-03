package report

import (
	"github.com/soulteary/jsreportcard-action/internal/model"
	"testing"
)

func TestIncomplete(t *testing.T) {
	r := model.Result{Checks: []model.Check{{Status: "error"}}}
	Finalize(t.TempDir(), &r, nil)
	if r.Complete || r.Score != nil {
		t.Fatal(r)
	}
}
func TestErrorsNotDiluted(t *testing.T) {
	r := model.Result{Files: 1000, Checks: []model.Check{{Tool: "typescript", Status: "findings", Analyzed: 1, Total: 1, Diagnostics: []model.Diagnostic{{Tool: "typescript", Rule: "TS2322", Severity: "error", File: "x.ts", Line: 1, Message: "wrong type"}}}}}
	Finalize(t.TempDir(), &r, nil)
	if *r.Score != 59 || r.NewErrors != 1 {
		t.Fatal(r)
	}
}
