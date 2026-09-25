// SPDX-License-Identifier: Apache-2.0

package model

import (
	"testing"
	"time"
)

func TestNormaliseKaraokeBumpsZeroDuration(t *testing.T) {
	spans := []TextSpan{
		{Text: "a", Start: time.Second, End: time.Second},
	}
	got := NormaliseKaraoke(spans)
	if got[0].End != time.Second+KaraokeBump {
		t.Fatalf("zero-duration segment not bumped: %v", got[0].End)
	}
	if spans[0].End != time.Second {
		t.Fatalf("NormaliseKaraoke mutated the input: %v", spans[0].End)
	}
}

func TestNormaliseKaraokeOrdersSegments(t *testing.T) {
	spans := []TextSpan{
		{Text: "a", Start: 0, End: time.Second},
		{Text: "b", Start: 500 * time.Millisecond, End: 1500 * time.Millisecond},
	}
	got := NormaliseKaraoke(spans)
	if got[1].Start != time.Second {
		t.Fatalf("overlapping segment did not move to the previous end: %v", got[1].Start)
	}
	if got[1].End != 1500*time.Millisecond {
		t.Fatalf("later segment was changed: %v", got[1].End)
	}
}

func TestNormaliseKaraokeBumpsAfterMove(t *testing.T) {
	// A segment that moves onto the previous end also becomes zero length
	// and must gain the bump.
	spans := []TextSpan{
		{Text: "a", Start: 0, End: time.Second},
		{Text: "b", Start: 200 * time.Millisecond, End: 800 * time.Millisecond},
	}
	got := NormaliseKaraoke(spans)
	if got[1].Start != time.Second || got[1].End != time.Second+KaraokeBump {
		t.Fatalf("moved segment not bumped: %v..%v", got[1].Start, got[1].End)
	}
}

func TestNormaliseKaraokeLeavesUntimedSpans(t *testing.T) {
	spans := []TextSpan{{Text: "plain"}, {Text: "words"}}
	got := NormaliseKaraoke(spans)
	for i := range got {
		if got[i].Start != 0 || got[i].End != 0 {
			t.Fatalf("untimed span %d was timed: %v..%v", i, got[i].Start, got[i].End)
		}
	}
}
