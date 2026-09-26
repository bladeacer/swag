// SPDX-License-Identifier: Apache-2.0

package config

import (
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
