// SPDX-License-Identifier: Apache-2.0

package srv3

import (
	"os"
	"strings"
	"testing"
)

func TestNames(t *testing.T) {
	if got := NewReader().Name(); got != FormatName {
		t.Fatalf("reader name = %q", got)
	}
	if got := NewWriter().Name(); got != FormatName {
		t.Fatalf("writer name = %q", got)
	}
}

func TestReaderParseSample(t *testing.T) {
	data, err := os.ReadFile("testdata/sample.srv3")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	doc, err := NewReader().Parse(strings.NewReader(string(data)))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(doc.Cues) != 2 {
		t.Fatalf("got %d cues, want 2", len(doc.Cues))
	}
	if doc.Cues[0].Text() != "One line" {
		t.Fatalf("first cue text = %q", doc.Cues[0].Text())
	}
}

func TestUpstreamSample(t *testing.T) {
	// The same upstream sample as the ytt test: SRV3 shares the model, so it
	// must read the document too. The test skips when the sample is absent.
	const upstreamSample = "../../../testdata/upstream/ytt.ytt"
	data, err := os.ReadFile(upstreamSample)
	if err != nil {
		t.Skipf("upstream sample not present; run scripts/fetch-samples.sh: %v", err)
	}
	doc, err := NewReader().Parse(strings.NewReader(string(data)))
	if err != nil {
		t.Fatalf("Parse upstream sample: %v", err)
	}
	if len(doc.Cues) == 0 {
		t.Fatal("upstream sample read to no cues")
	}
}

func TestWriterSharesThePenModel(t *testing.T) {
	data, err := os.ReadFile("testdata/sample.srv3")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	doc, err := NewReader().Parse(strings.NewReader(string(data)))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	var out strings.Builder
	if _, err := NewWriter().Render(doc, &out); err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !strings.Contains(out.String(), "<timedtext") {
		t.Fatalf("output must carry the root element:\n%s", out.String())
	}
	again, err := NewReader().Parse(strings.NewReader(out.String()))
	if err != nil {
		t.Fatalf("re-parse: %v", err)
	}
	if len(again.Cues) != len(doc.Cues) {
		t.Fatalf("cue count changed: %d to %d", len(doc.Cues), len(again.Cues))
	}
}
