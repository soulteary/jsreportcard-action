package report

import (
	"github.com/soulteary/jsreportcard-action/internal/model"
	"strings"
	"testing"
)

func TestBadgeDisplayCapitalization(t *testing.T) {
	for _, tc := range []struct {
		mode     string
		complete bool
		want     string
	}{
		{"standard", true, "JS/TS Report"},
		{"typed", true, "JS/TS Report (Typed)"},
		{"standard", false, ">Incomplete<"},
	} {
		svg := SVG(model.Result{Mode: tc.mode, Complete: tc.complete, Grade: "A+"})
		if !strings.Contains(svg, tc.want) {
			t.Errorf("badge missing %q: %s", tc.want, svg)
		}
	}
}
