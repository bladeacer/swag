// SPDX-License-Identifier: Apache-2.0

package richtext

import "testing"

// TestParseLiteralBackslash covers a backslash that starts no escape. It
// stays in the text as a literal character.
func TestParseLiteralBackslash(t *testing.T) {
	runs := Parse(`a\z`)
	if len(runs) != 1 || runs[0].Text != `a\z` {
		t.Fatalf("Parse kept %+v, want one run with the literal text", runs)
	}
	trailing := Parse("tail\\")
	if len(trailing) != 1 || trailing[0].Text != "tail\\" {
		t.Fatalf("Parse of a trailing backslash kept %+v", trailing)
	}
}

// TestParseTagsUnreadableName covers a backslash that carries no tag name,
// and a name that is not a known tag.
func TestParseTagsUnreadableName(t *testing.T) {
	if tags := ParseTags("\\"); len(tags) != 0 {
		t.Fatalf("a lone backslash must give no tag: %+v", tags)
	}
	if tags := ParseTags("\\1"); len(tags) != 0 {
		t.Fatalf("a bare digit run must give no tag: %+v", tags)
	}
}

// TestParseTagsMissingParen covers a parenthesised tag with no closing
// parenthesis. The rest of the block becomes the argument list.
func TestParseTagsMissingParen(t *testing.T) {
	tags := ParseTags("\\pos(1,2")
	if len(tags) != 1 {
		t.Fatalf("got %d tags, want 1: %+v", len(tags), tags)
	}
	if !tags[0].Parens || len(tags[0].Args) != 2 || tags[0].Args[1] != "2" {
		t.Fatalf("arguments not read: %+v", tags[0])
	}
}
