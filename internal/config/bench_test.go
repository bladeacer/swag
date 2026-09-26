// SPDX-License-Identifier: Apache-2.0

package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// BenchmarkDecodeDefaultFile measures the decode of the shipped default
// file, which runs on every startup that reads a configuration.
func BenchmarkDecodeDefaultFile(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := Decode(strings.NewReader(DefaultFile)); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkLoadCached measures the cached lookup of an unchanged file, which
// the second and later loads of a path take.
func BenchmarkLoadCached(b *testing.B) {
	path := filepath.Join(b.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte(DefaultFile), 0o644); err != nil {
		b.Fatal(err)
	}
	if _, err := Load(path); err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := Load(path); err != nil {
			b.Fatal(err)
		}
	}
}
