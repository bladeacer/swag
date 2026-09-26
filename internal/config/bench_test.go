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

// BenchmarkLoadMergedCached measures the kept merge of two unchanged files,
// which the second and later resolution of the configuration takes.
func BenchmarkLoadMergedCached(b *testing.B) {
	dir := b.TempDir()
	base := filepath.Join(dir, "global.toml")
	override := filepath.Join(dir, "local.toml")
	if err := os.WriteFile(base, []byte("font = \"Verdana\"\n"), 0o644); err != nil {
		b.Fatal(err)
	}
	if err := os.WriteFile(override, []byte("font = \"Courier New\"\n"), 0o644); err != nil {
		b.Fatal(err)
	}
	if _, err := LoadMerged(base, override); err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := LoadMerged(base, override); err != nil {
			b.Fatal(err)
		}
	}
}
