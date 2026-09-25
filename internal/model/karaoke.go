// SPDX-License-Identifier: Apache-2.0

package model

import "time"

// KaraokeBump is the duration added to a karaoke segment that would
// otherwise have no length. YTT upload removes the timing of a span that
// repeats the time of the span before it, so a writer must bump the
// segment before it sends the document.
const KaraokeBump = time.Millisecond

// NormaliseKaraoke returns a copy of spans whose karaoke offsets obey the
// tiling rules that writers rely on. Untimed spans pass through unchanged.
//
// For the timed spans, the function makes two changes. A span whose End is
// at or before its Start gains KaraokeBump, because a format that rejects
// zero-duration spans would drop its timing. A span that starts before the
// end of the span before it moves to that end, so the segments stay in
// order and never overlap.
func NormaliseKaraoke(spans []TextSpan) []TextSpan {
	out := append([]TextSpan(nil), spans...)
	var prevEnd time.Duration
	for i := range out {
		s := &out[i]
		if s.Start == 0 && s.End == 0 {
			// Untimed: the source set no karaoke window, so leave it.
			continue
		}
		if s.Start < prevEnd {
			s.Start = prevEnd
		}
		if s.End <= s.Start {
			s.End = s.Start + KaraokeBump
		}
		prevEnd = s.End
	}
	return out
}
