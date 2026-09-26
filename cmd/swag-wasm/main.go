// SPDX-License-Identifier: Apache-2.0

//go:build js && wasm

// Command swag-wasm exposes the core conversion of swag to a browser. The
// build runs on the WebAssembly target, and the demo page under
// [the demo directory](../../docs/demo/index.html) loads the module and
// calls swag.convert.
//
// The command holds no format logic. It wires the public library (pkg/sub)
// to the JavaScript global, exactly as cmd/swag wires it to the terminal.
package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"syscall/js"

	"github.com/bladeacer/swag/pkg/sub"
)

// result is the value that the page reads back: the rendered text, the loss
// report, and an error message.
type result struct {
	Output string   `json:"output"`
	Losses []string `json:"losses"`
	Error  string   `json:"error,omitempty"`
}

// convert parses the content and renders it in the target format. A failure
// comes back in the error field, so the page shows the message instead of a
// thrown value.
func convert(name, content, target string) result {
	var sink bytes.Buffer
	losses, err := sub.ConvertWith(name, strings.NewReader(content), sub.Options{Target: target}, &sink)
	if err != nil {
		return result{Error: err.Error()}
	}
	return result{Output: sink.String(), Losses: losses}
}

// jsConvert bridges convert to JavaScript. The arguments are the file name,
// the content, and the target format, and the value that comes back is a
// JSON string.
func jsConvert(_ js.Value, args []js.Value) any {
	if len(args) < 3 {
		return string(mustJSON(result{Error: "swag.convert needs a file name, a content, and a target format."}))
	}
	return string(mustJSON(convert(args[0].String(), args[1].String(), args[2].String())))
}

// mustJSON encodes a result. A value that cannot be encoded becomes a fixed
// error object, because the page must still read a valid answer.
func mustJSON(value any) []byte {
	data, err := json.Marshal(value)
	if err != nil {
		return []byte(`{"error":"The result could not be encoded."}`)
	}
	return data
}

// formats lists the registry names that can read a document, so the page
// can build its target list from the same registry as the command.
func formats() []any {
	names := sub.Registered()
	out := make([]any, 0, len(names))
	for _, name := range names {
		out = append(out, name)
	}
	return out
}

// main registers the global and then waits, because a WebAssembly program
// ends when main returns.
func main() {
	js.Global().Set("swag", js.ValueOf(map[string]any{
		"convert": js.FuncOf(jsConvert),
		"formats": formats(),
	}))
	<-make(chan struct{})
}
