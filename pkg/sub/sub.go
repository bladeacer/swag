// Package sub is the public entry point of the swag library.
//
// A caller names a file or a stream, and the package detects the format,
// parses it into the intermediate representation, and renders it back out
// in any registered format. Readers and writers report the features they
// drop, so a conversion is never silently lossy.
package sub

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/bladeacer/swag/internal/model"
)

// Document is the intermediate representation. It is an alias so callers
// of the public API need no internal import.
type Document = model.Document

// Format is one subtitle format. Register a Format to make it visible to
// Parse, Render, and Convert.
type Format interface {
	// Name returns the registry name, for example "srt".
	Name() string
	// Extensions returns the file name extensions without the dot, the
	// first one being canonical, for example ["srt"].
	Extensions() []string
}

// ReaderFormat is a Format that can parse files.
type ReaderFormat interface {
	Format
	Parse(source io.Reader) (*Document, error)
}

// WriterFormat is a Format that can render files.
type WriterFormat interface {
	Format
	// Render writes doc to sink. The returned slice carries one entry per
	// feature the format cannot express.
	Render(doc *Document, sink io.Writer) ([]string, error)
}

var (
	readers = map[string]ReaderFormat{}
	writers = map[string]WriterFormat{}
	byExt   = map[string]string{}
)

func register(f Format) {
	if r, ok := f.(ReaderFormat); ok {
		readers[f.Name()] = r
	}
	if w, ok := f.(WriterFormat); ok {
		writers[f.Name()] = w
	}
	for _, ext := range f.Extensions() {
		if _, exists := byExt[ext]; !exists {
			byExt[ext] = f.Name()
		}
	}
}

func init() {
	register(srtFormat{})
	register(sbvFormat{})
}

// Registered returns the names of all formats that can read, in
// alphabetical order.
func Registered() []string {
	names := make([]string, 0, len(readers))
	for name := range readers {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// DetectFormat returns the format name for a file name, or "" when no
// registered format claims the extension.
func DetectFormat(fileName string) string {
	dot := strings.LastIndex(fileName, ".")
	if dot < 0 {
		return ""
	}
	return byExt[strings.ToLower(fileName[dot+1:])]
}

// Identify sniffs the content of source and returns the format name.
// It falls back to the file name extension when the content carries no
// signature. It returns "" when the format stays unknown.
func Identify(fileName string, source io.Reader) (string, error) {
	if DetectFormat(fileName) != "" {
		return DetectFormat(fileName), nil
	}
	data, err := io.ReadAll(source)
	if err != nil {
		return "", fmt.Errorf("read %s: %w", fileName, err)
	}
	if looksLikeSRT(data) {
		return "srt", nil
	}
	if looksLikeSBV(data) {
		return "sbv", nil
	}
	return "", nil
}

// looksLikeSRT reports whether the content carries a SubRip timing line.
func looksLikeSRT(data []byte) bool {
	for _, line := range strings.Split(string(data), "\n") {
		if strings.Contains(line, "-->") {
			return true
		}
	}
	return false
}

// looksLikeSBV reports whether the first non-blank line looks like SBV
// timing (h:mm:ss.mmm,h:mm:ss.mmm).
func looksLikeSBV(data []byte) bool {
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		comma := strings.Index(line, ",")
		if comma < 0 {
			return false
		}
		head, tail := line[:comma], line[comma+1:]
		return strings.Count(head, ":") == 2 && strings.Contains(head, ".") &&
			strings.Count(tail, ":") == 2 && strings.Contains(tail, ".")
	}
	return false
}

// Parse reads source as the named format. An empty name means
// auto-detection. Callers that hold a file name should pass it so the
// extension can help detection.
func Parse(fileName string, source io.Reader, formatName string) (*Document, error) {
	name := formatName
	if name == "" {
		detected, err := Identify(fileName, source)
		if err != nil {
			return nil, err
		}
		if detected == "" {
			return nil, fmt.Errorf("identify %s: format is not known; name it with -f", fileName)
		}
		name = detected
	}
	reader, ok := readers[name]
	if !ok {
		return nil, fmt.Errorf("format %q cannot be read", name)
	}
	doc, err := reader.Parse(source)
	if err != nil {
		return nil, fmt.Errorf("parse %s as %s: %w", fileName, name, err)
	}
	return doc, nil
}

// Render writes doc to sink in the named format and returns the loss
// report: one entry per feature the format cannot express.
func Render(doc *Document, formatName string, sink io.Writer) ([]string, error) {
	writer, ok := writers[formatName]
	if !ok {
		return nil, fmt.Errorf("format %q cannot be written", formatName)
	}
	losses, err := writer.Render(doc, sink)
	if err != nil {
		return nil, fmt.Errorf("render %s: %w", formatName, err)
	}
	return losses, nil
}

// Convert parses source and renders it in targetFormat in one step. It
// returns the loss report of the write.
func Convert(fileName string, source io.Reader, targetFormat string, sink io.Writer) ([]string, error) {
	doc, err := Parse(fileName, source, "")
	if err != nil {
		return nil, err
	}
	return Render(doc, targetFormat, sink)
}
