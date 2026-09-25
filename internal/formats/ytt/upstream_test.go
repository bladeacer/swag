// SPDX-License-Identifier: Apache-2.0

package ytt

import (
	"os"
	"strings"
	"testing"

	"github.com/bladeacer/swag/internal/model"
)

// upstreamSample is the fixture that scripts/fetch-samples.sh downloads
// from YTSubConverter. The file is test data only and stays out of the
// repository, so the test skips when it is absent.
const upstreamSample = "../../../testdata/upstream/ytt.ytt"

func TestUpstreamSampleRoundTrip(t *testing.T) {
	data, err := os.ReadFile(upstreamSample)
	if err != nil {
		t.Skipf("upstream sample not present; run scripts/fetch-samples.sh: %v", err)
	}

	doc, err := NewReader().Parse(strings.NewReader(string(data)))
	if err != nil {
		t.Fatalf("Parse upstream sample: %v", err)
	}
	if len(doc.Cues) < 20 {
		t.Fatalf("got %d cues, want at least 20", len(doc.Cues))
	}

	// The sample exercises karaoke, ruby, vertical text, and positioning.
	var karaoke, ruby, vertical, positioned int
	for _, cue := range doc.Cues {
		if cue.Karaoke() {
			karaoke++
		}
		if cue.Layout != nil {
			positioned++
		}
		for _, span := range cue.Spans {
			if span.Ruby != nil {
				ruby++
			}
			if span.Vertical != nil && span.Vertical.Mode != 0 {
				vertical++
			}
		}
	}
	if karaoke == 0 || ruby == 0 || vertical == 0 || positioned == 0 {
		t.Fatalf("sample features missing: karaoke=%d ruby=%d vertical=%d positioned=%d",
			karaoke, ruby, vertical, positioned)
	}

	// The sample carries lines with several shadow kinds. The writer layers
	// those into one line per kind, so the first render grows the cue count.
	// The result must then settle: a second round changes nothing.
	doc2 := renderAndParse(t, doc)
	doc3 := renderAndParse(t, doc2)
	if len(doc2.Cues) != len(doc3.Cues) {
		t.Fatalf("layering did not settle: %d cues then %d", len(doc2.Cues), len(doc3.Cues))
	}
	for i := range doc2.Cues {
		if doc2.Cues[i].Text() != doc3.Cues[i].Text() {
			t.Errorf("cue %d text changed: %q to %q", i, doc2.Cues[i].Text(), doc3.Cues[i].Text())
		}
	}
}

// renderAndParse renders doc and reads the result back.
func renderAndParse(t *testing.T, doc *model.Document) *model.Document {
	t.Helper()
	var out strings.Builder
	if _, err := NewWriter().Render(doc, &out); err != nil {
		t.Fatalf("Render: %v", err)
	}
	again, err := NewReader().Parse(strings.NewReader(out.String()))
	if err != nil {
		t.Fatalf("re-parse: %v", err)
	}
	return again
}
