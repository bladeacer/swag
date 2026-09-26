// SPDX-License-Identifier: Apache-2.0

package tui

import "testing"

// BenchmarkNewKeymap measures the build of the keymap, which runs once per
// interactive session.
func BenchmarkNewKeymap(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := NewKeymap(nil); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkKeymapMatch measures a whole-line match, the path of the line
// prompt.
func BenchmarkKeymapMatch(b *testing.B) {
	keymap, err := NewKeymap(nil)
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		keymap.Match("\x18a")
	}
}

// BenchmarkKeymapStep measures one step of the incremental matcher, the path
// of the terminal reader.
func BenchmarkKeymapStep(b *testing.B) {
	keymap, err := NewKeymap(nil)
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		keymap.Step("\x18")
	}
}
