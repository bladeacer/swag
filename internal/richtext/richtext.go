// SPDX-License-Identifier: Apache-2.0

// Package richtext parses the override tag syntax that Advanced SubStation
// Alpha (ASS) uses for styled subtitles. The parser is shared so that the
// ASS reader and the ASS writer read the same grammar.
//
// An event line is a mix of literal text and braces blocks. A braces block
// holds a run of override tags, each of which starts with a backslash. A
// tag takes either a bare value (\fs30), a parenthesised argument list
// (\pos(100,200)), or nothing (\i). Text outside a braces block resolves
// the escapes \n and \N to a line break, \h to a non-breaking space, and
// \\ to a literal backslash.
package richtext

import "strings"

// knownTags lists the tag names that the parser can tell apart from a bare
// value. A tag like \fn takes the rest of the block, so the parser reads
// the longest known name and leaves the remainder as the value. A name
// outside the list keeps the whole letter run, which is the right guess
// for the tags that the reader ignores.
var knownTags = []string{
	"ytshake", "ytchroma", "ytktFade", "ytktGlitch", "ytruby", "ytvert",
	"ytpack", "ytdir", "ytsup", "ytsub", "ytsur", "alpha", "fscx",
	"fscy", "iclip", "ybord", "xshad", "yshad", "clip", "bord", "blur",
	"shad", "move", "fade", "fad", "frx", "fry", "frz", "pos", "org",
	"fn", "fs", "an", "kf", "ko", "fr", "fe", "be", "sp", "k", "K",
	"b", "i", "u", "c", "r", "p", "s", "t", "a", "q",
}

// Tag is one override tag inside a braces block.
type Tag struct {
	// Name is the tag name without the backslash, for example "fs".
	Name string
	// Value is the bare value of the tag, for example "30" in \fs30 or
	// "Default" in \rDefault. It is empty when the tag carries no value.
	Value string
	// Args holds the arguments of a tag with parentheses, for example
	// ["100", "200"] for \pos(100,200).
	Args []string
	// Parens reports whether the tag carried a parenthesised argument
	// list.
	Parens bool
}

// Run is a stretch of event text together with the tags that precede it.
type Run struct {
	Tags []Tag
	Text string
}

// Parse splits the text of an ASS event into runs. A run carries the tags
// of every braces block that precedes its text. Escapes resolve in the
// text.
func Parse(text string) []Run {
	var runs []Run
	var pending []Tag
	var lit strings.Builder

	flush := func() {
		if lit.Len() == 0 {
			return
		}
		runs = append(runs, Run{Tags: pending, Text: lit.String()})
		pending = nil
		lit.Reset()
	}

	for i := 0; i < len(text); {
		switch text[i] {
		case '{':
			end := strings.IndexByte(text[i:], '}')
			if end < 0 {
				// An unterminated block is literal text.
				lit.WriteString(text[i:])
				i = len(text)
				continue
			}
			flush()
			pending = append(pending, ParseTags(text[i+1:i+end])...)
			i += end + 1
		case '\\':
			if i+1 < len(text) {
				switch text[i+1] {
				case 'n', 'N':
					lit.WriteByte('\n')
					i += 2
					continue
				case 'h':
					lit.WriteString("\u00a0")
					i += 2
					continue
				case '\\':
					lit.WriteByte('\\')
					i += 2
					continue
				}
			}
			lit.WriteByte(text[i])
			i++
		default:
			lit.WriteByte(text[i])
			i++
		}
	}
	flush()
	return runs
}

// ParseTags reads the content of one braces block into tags.
func ParseTags(block string) []Tag {
	var tags []Tag
	for i := 0; i < len(block); {
		if block[i] != '\\' {
			i++
			continue
		}
		i++
		name := readName(block, &i)
		if name == "" {
			continue
		}
		tag := Tag{Name: name}
		if i < len(block) && block[i] == '(' {
			inner, next := readParens(block, i)
			tag.Parens = true
			tag.Args = splitArgs(inner)
			i = next
		} else {
			start := i
			for i < len(block) && block[i] != '\\' {
				i++
			}
			tag.Value = block[start:i]
		}
		tags = append(tags, tag)
	}
	return tags
}

// readName reads a tag name. The name is a run of letters, with the two
// special forms "1c" to "4c" and "1a" to "4a" that start with a digit.
func readName(block string, i *int) string {
	if *i >= len(block) {
		return ""
	}
	c := block[*i]
	if c >= '1' && c <= '4' && *i+1 < len(block) && (block[*i+1] == 'c' || block[*i+1] == 'a') {
		name := block[*i : *i+2]
		*i += 2
		return name
	}
	start := *i
	for *i < len(block) && isLetter(block[*i]) {
		*i++
	}
	run := block[start:*i]
	if run == "" {
		return ""
	}
	// Prefer the longest known name that starts the letter run, so a tag
	// like \fnComic Sans MS reads as \fn plus a value.
	best := ""
	for _, name := range knownTags {
		if len(name) > len(best) && strings.HasPrefix(run, name) {
			best = name
		}
	}
	if best == "" {
		return run
	}
	*i = start + len(best)
	return best
}

// readParens reads a parenthesised argument list that starts at open. It
// returns the inner text and the index after the closing parenthesis.
func readParens(block string, open int) (string, int) {
	depth := 0
	for i := open; i < len(block); i++ {
		switch block[i] {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return block[open+1 : i], i + 1
			}
		}
	}
	// No closing parenthesis: the rest of the block is the argument list.
	return block[open+1:], len(block)
}

// splitArgs splits a parenthesised argument list on commas that sit at the
// top level. Commas inside a nested parentheses group stay in their
// argument.
func splitArgs(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	var args []string
	depth := 0
	start := 0
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '(':
			depth++
		case ')':
			if depth > 0 {
				depth--
			}
		case ',':
			if depth == 0 {
				args = append(args, strings.TrimSpace(s[start:i]))
				start = i + 1
			}
		}
	}
	args = append(args, strings.TrimSpace(s[start:]))
	return args
}

func isLetter(c byte) bool {
	return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z'
}
