// SPDX-License-Identifier: Apache-2.0

package config

import (
	"strings"
	"testing"
)

// FuzzDecode checks that the TOML decoder never panics and that a document it
// accepts keeps no blank entry in the preferred list, which normalise trims.
func FuzzDecode(f *testing.F) {
	for _, seed := range []string{
		DefaultFile,
		"",
		"locale = \"en-US\"\n",
		"jobs = 4\n",
		"[keybinds]\nquit = [\"q\"]\n",
		"[keybinds\nquit = [\n",
		"bogus = 1\n",
		"verbose = \"yes\"\n",
		"preferred = [\" srt \", \"\"]\n",
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, data string) {
		settings, err := Decode(strings.NewReader(data))
		if err != nil {
			return
		}
		for _, name := range settings.Preferred {
			if strings.TrimSpace(name) == "" || name != strings.TrimSpace(name) {
				t.Fatalf("a parsed document kept a blank preferred name: %q", name)
			}
		}
	})
}
