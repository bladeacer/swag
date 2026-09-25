// SPDX-License-Identifier: Apache-2.0

package ass

import (
	"os"
	"strings"
	"testing"
)

// The upstream ASS samples, downloaded by scripts/fetch-samples.sh. The
// files are test data only and stay out of the repository, so the test
// skips when they are absent.
var upstreamSamples = []string{
	"../../../testdata/upstream/sample1.ass",
	"../../../testdata/upstream/sample2.ass",
}

func TestUpstreamSamples(t *testing.T) {
	any := false
	for _, name := range upstreamSamples {
		data, err := os.ReadFile(name)
		if err != nil {
			continue
		}
		any = true
		doc, err := NewReader().Parse(strings.NewReader(string(data)))
		if err != nil {
			t.Fatalf("Parse %s: %v", name, err)
		}
		if len(doc.Cues) == 0 {
			t.Errorf("%s read to no cues", name)
		}
		if len(doc.Styles) == 0 {
			t.Errorf("%s read to no styles", name)
		}
		for i, cue := range doc.Cues {
			if cue.End <= cue.Start {
				t.Errorf("%s cue %d has no duration: %v..%v", name, i, cue.Start, cue.End)
			}
		}
	}
	if !any {
		t.Skip("upstream samples not present; run scripts/fetch-samples.sh")
	}
}
