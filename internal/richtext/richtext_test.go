// SPDX-License-Identifier: Apache-2.0

package richtext

import (
	"reflect"
	"strings"
	"testing"
)

func TestParseSplitsRuns(t *testing.T) {
	runs := Parse(`{\k42}Ny{\k38}an`)
	if len(runs) != 2 {
		t.Fatalf("got %d runs, want 2: %+v", len(runs), runs)
	}
	if runs[0].Text != "Ny" || runs[1].Text != "an" {
		t.Fatalf("text split wrong: %+v", runs)
	}
	if len(runs[0].Tags) != 1 || runs[0].Tags[0].Name != "k" || runs[0].Tags[0].Value != "42" {
		t.Fatalf("first run tags = %+v", runs[0].Tags)
	}
}

func TestParseResolvesEscapes(t *testing.T) {
	runs := Parse(`one\Ntwo\nthree\htail\\end`)
	if len(runs) != 1 {
		t.Fatalf("got %d runs, want 1", len(runs))
	}
	want := "one\ntwo\nthree\u00a0tail\\end"
	if runs[0].Text != want {
		t.Fatalf("escapes = %q, want %q", runs[0].Text, want)
	}
}

func TestParseUnterminatedBlock(t *testing.T) {
	runs := Parse(`before{\k10 after`)
	if len(runs) != 1 || runs[0].Text != `before{\k10 after` {
		t.Fatalf("unterminated block mishandled: %+v", runs)
	}
}

func TestParseAccumulatesTags(t *testing.T) {
	runs := Parse(`{\b1}{\i1}text`)
	if len(runs) != 1 {
		t.Fatalf("got %d runs, want 1", len(runs))
	}
	if len(runs[0].Tags) != 2 {
		t.Fatalf("consecutive blocks must accumulate: %+v", runs[0].Tags)
	}
}

func TestParseTagsForms(t *testing.T) {
	tags := ParseTags(`\fs30\fnComic Sans MS\b\pos(100,200)\rDefault`)
	if len(tags) != 5 {
		t.Fatalf("got %d tags, want 5: %+v", len(tags), tags)
	}
	if tags[0].Name != "fs" || tags[0].Value != "30" || tags[0].Parens {
		t.Errorf("fs tag = %+v", tags[0])
	}
	if tags[1].Name != "fn" || tags[1].Value != "Comic Sans MS" {
		t.Errorf("fn tag = %+v", tags[1])
	}
	if tags[2].Name != "b" || tags[2].Value != "" {
		t.Errorf("b tag = %+v", tags[2])
	}
	if !tags[3].Parens || !reflect.DeepEqual(tags[3].Args, []string{"100", "200"}) {
		t.Errorf("pos tag = %+v", tags[3])
	}
	if tags[4].Name != "r" || tags[4].Value != "Default" {
		t.Errorf("r tag = %+v", tags[4])
	}
}

func TestParseTagsLeadingDigits(t *testing.T) {
	tags := ParseTags(`\1c&H0000FF&\3a&H40&\alpha&H80&`)
	if tags[0].Name != "1c" || tags[0].Value != "&H0000FF&" {
		t.Fatalf("1c tag = %+v", tags[0])
	}
	if tags[1].Name != "3a" || tags[1].Value != "&H40&" {
		t.Fatalf("3a tag = %+v", tags[1])
	}
	if tags[2].Name != "alpha" {
		t.Fatalf("alpha tag = %+v", tags[2])
	}
}

func TestParseTagsNestedParens(t *testing.T) {
	tags := ParseTags(`\t(0,1000,\c&H0000FF&)\move(1,2,3,4)`)
	if len(tags) != 2 {
		t.Fatalf("got %d tags, want 2: %+v", len(tags), tags)
	}
	if !reflect.DeepEqual(tags[0].Args, []string{"0", "1000", `\c&H0000FF&`}) {
		t.Fatalf("t args = %+v", tags[0].Args)
	}
	if !reflect.DeepEqual(tags[1].Args, []string{"1", "2", "3", "4"}) {
		t.Fatalf("move args = %+v", tags[1].Args)
	}
}

func TestParseTagsYtNames(t *testing.T) {
	tags := ParseTags(`\ytruby8\ytvert9\ytpack1\ytdir4\ytktFade`)
	names := make([]string, len(tags))
	for i, tag := range tags {
		names[i] = tag.Name
	}
	want := []string{"ytruby", "ytvert", "ytpack", "ytdir", "ytktFade"}
	if !reflect.DeepEqual(names, want) {
		t.Fatalf("names = %v, want %v", names, want)
	}
	if tags[0].Value != "8" || tags[3].Value != "4" {
		t.Fatalf("values = %q %q", tags[0].Value, tags[3].Value)
	}
}

func TestParseTagsIgnoresStrayText(t *testing.T) {
	tags := ParseTags(`junk\b1 more`)
	if len(tags) != 1 || tags[0].Name != "b" {
		t.Fatalf("stray text not skipped: %+v", tags)
	}
	// The value of a bare tag runs to the next backslash, spaces included.
	if tags[0].Value != "1 more" {
		t.Fatalf("bare value = %q", tags[0].Value)
	}
}

func TestSplitArgs(t *testing.T) {
	if got := splitArgs(""); got != nil {
		t.Fatalf("empty args = %v", got)
	}
	got := splitArgs("a, (b,c), d")
	if !reflect.DeepEqual(got, []string{"a", "(b,c)", "d"}) {
		t.Fatalf("splitArgs = %v", got)
	}
}

func TestParseEmpty(t *testing.T) {
	if runs := Parse(""); len(runs) != 0 {
		t.Fatalf("empty text gave %d runs", len(runs))
	}
	if !strings.Contains(Parse(`{}{}x`)[0].Text, "x") {
		t.Fatal("empty blocks must not swallow text")
	}
}
