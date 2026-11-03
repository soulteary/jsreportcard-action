package report

import (
	"encoding/xml"
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

func TestBadgeLabelVariants(t *testing.T) {
	for _, tc := range []struct{ value, label string }{
		{"", "JS/TS Report"}, {"js-ts", "JS/TS Report"}, {"js", "JS Report"}, {"ts", "TS Report"},
	} {
		for _, complete := range []bool{true, false} {
			svg := SVG(model.Result{BadgeLabel: tc.value, Mode: "typed", Complete: complete, Grade: "A+"})
			var parsed struct {
				Width int    `xml:"width,attr"`
				Title string `xml:"title"`
			}
			if err := xml.Unmarshal([]byte(svg), &parsed); err != nil {
				t.Fatal(err)
			}
			if !strings.HasPrefix(parsed.Title, tc.label+" (Typed): ") {
				t.Errorf("wrong label: %s", parsed.Title)
			}
			message := "A+"
			if !complete {
				message = "Incomplete"
			}
			if !strings.Contains(svg, ">"+message+"</text>") {
				t.Errorf("wrong message: %s", svg)
			}
			wantWidth := badgeTextWidth(tc.label+" (Typed)") + badgeTextWidth(message) + 40
			if parsed.Width != wantWidth {
				t.Errorf("width %d, want %d", parsed.Width, wantWidth)
			}
		}
	}
}
