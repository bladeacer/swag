// SPDX-License-Identifier: Apache-2.0

package tui

import (
	"errors"
	"strings"
	"testing"
)

func TestFrameOrder(t *testing.T) {
	frame := Layout{Heading: "head", Rows: []string{"a", "b"}, Status: "status"}.frame()
	want := []string{"head", "a", "b", "status"}
	if len(frame) != len(want) {
		t.Fatalf("frame = %q, want %q", frame, want)
	}
	for i := range want {
		if frame[i] != want[i] {
			t.Fatalf("frame = %q, want %q", frame, want)
		}
	}
	// An empty status occupies no row.
	if got := (Layout{Heading: "head"}).frame(); len(got) != 1 {
		t.Fatalf("frame = %q, want only the heading", got)
	}
}

func TestChanged(t *testing.T) {
	base := Layout{Heading: "head", Rows: []string{"a", "b"}, Status: "end"}
	tests := []struct {
		name string
		next Layout
		want []int
	}{
		{
			// The heading and the rows match, the status differs.
			name: "status",
			next: Layout{Heading: "head", Rows: []string{"a", "b"}, Status: "done"},
			want: []int{3},
		},
		{
			// Only the second row differs.
			name: "row",
			next: Layout{Heading: "head", Rows: []string{"a", "z"}, Status: "end"},
			want: []int{2},
		},
		{
			// The new frame loses its status, so the row disappears.
			name: "shorter",
			next: Layout{Heading: "head", Rows: []string{"a", "b"}},
			want: []int{3},
		},
		{
			// A new row pushes the status down, so both rows differ.
			name: "longer",
			next: Layout{Heading: "head", Rows: []string{"a", "b", "c"}, Status: "end"},
			want: []int{3, 4},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Changed(base, tt.next)
			if len(got) != len(tt.want) {
				t.Fatalf("Changed = %v, want %v", got, tt.want)
			}
			for i := range tt.want {
				if got[i] != tt.want[i] {
					t.Fatalf("Changed = %v, want %v", got, tt.want)
				}
			}
		})
	}
	if got := Changed(base, base); len(got) != 0 {
		t.Fatalf("Changed of an identical frame = %v, want none", got)
	}
}

// TestDrawSkipsUnchangedFrames proves the optimisation: a frame equal to the
// previous one writes no bytes at all.
func TestDrawSkipsUnchangedFrames(t *testing.T) {
	var out strings.Builder
	r := NewRenderer(&out)
	if r.Frames() != 0 {
		t.Fatalf("a fresh renderer reported %d frames", r.Frames())
	}
	first := Layout{Heading: "swag", Rows: []string{"input", "target"}}
	redrew, err := r.Draw(first)
	if err != nil {
		t.Fatalf("Draw: %v", err)
	}
	if !redrew {
		t.Fatal("the first frame must paint")
	}
	painted := out.String()
	if !strings.Contains(painted, "input") || !strings.Contains(painted, "\x1b[1;1H") {
		t.Fatalf("the first frame is incomplete: %q", painted)
	}

	out.Reset()
	redrew, err = r.Draw(first)
	if err != nil {
		t.Fatalf("Draw: %v", err)
	}
	if redrew {
		t.Fatal("an identical frame must not paint")
	}
	if out.Len() != 0 {
		t.Fatalf("an identical frame wrote %q, want no bytes", out.String())
	}
	if r.Frames() != 1 {
		t.Fatalf("frames = %d, want 1", r.Frames())
	}
}

// TestDrawPaintsOnlyChangedRows proves the second half of the optimisation:
// a frame that changes one row writes that row and leaves the others alone.
func TestDrawPaintsOnlyChangedRows(t *testing.T) {
	var out strings.Builder
	r := NewRenderer(&out)
	first := Layout{Heading: "swag", Rows: []string{"input", "target"}, Status: "ready"}
	if _, err := r.Draw(first); err != nil {
		t.Fatalf("Draw: %v", err)
	}

	out.Reset()
	second := Layout{Heading: "swag", Rows: []string{"input", "target"}, Status: "written"}
	redrew, err := r.Draw(second)
	if err != nil {
		t.Fatalf("Draw: %v", err)
	}
	if !redrew {
		t.Fatal("a changed frame must paint")
	}
	painted := out.String()
	if !strings.Contains(painted, "written") {
		t.Fatalf("the changed row is missing: %q", painted)
	}
	if strings.Contains(painted, "input") || strings.Contains(painted, "target") {
		t.Fatalf("an unchanged row was repainted: %q", painted)
	}
	// The status sits on row 4 of the frame.
	if !strings.Contains(painted, "\x1b[4;1H") {
		t.Fatalf("the changed row is at the wrong position: %q", painted)
	}
}

// TestDrawClearsLeftoverRows covers a shorter frame, whose surplus rows are
// blanked so stale text does not stay on screen.
func TestDrawClearsLeftoverRows(t *testing.T) {
	var out strings.Builder
	r := NewRenderer(&out)
	long := Layout{Heading: "head", Rows: []string{"a", "b", "c"}, Status: "end"}
	if _, err := r.Draw(long); err != nil {
		t.Fatalf("Draw: %v", err)
	}
	out.Reset()
	short := Layout{Heading: "head", Rows: []string{"a"}}
	if _, err := r.Draw(short); err != nil {
		t.Fatalf("Draw: %v", err)
	}
	painted := out.String()
	for _, row := range []string{"\x1b[3;1H", "\x1b[4;1H"} {
		if !strings.Contains(painted, row) {
			t.Fatalf("row %s was not cleared: %q", row, painted)
		}
	}
	if strings.Contains(painted, "\x1b[1;1H") || strings.Contains(painted, "\x1b[2;1H") {
		t.Fatalf("an unchanged row was repainted: %q", painted)
	}
}

func TestDrawScrollingRows(t *testing.T) {
	// A frame whose rows grow paints the new rows and keeps the old ones.
	var out strings.Builder
	r := NewRenderer(&out)
	first := Layout{Heading: "head", Rows: []string{"a"}, Status: "one"}
	if _, err := r.Draw(first); err != nil {
		t.Fatalf("Draw: %v", err)
	}
	out.Reset()
	grown := Layout{Heading: "head", Rows: []string{"a", "b"}, Status: "two"}
	if _, err := r.Draw(grown); err != nil {
		t.Fatalf("Draw: %v", err)
	}
	painted := out.String()
	if !strings.Contains(painted, "\x1b[3;1H") || !strings.Contains(painted, "\x1b[4;1H") {
		t.Fatalf("the new rows were not painted: %q", painted)
	}
	if strings.Contains(painted, "\x1b[1;1H") || strings.Contains(painted, "\x1b[2;1H") {
		t.Fatalf("an unchanged row was repainted: %q", painted)
	}
}

func TestResetRepaints(t *testing.T) {
	var out strings.Builder
	r := NewRenderer(&out)
	frame := Layout{Heading: "head"}
	if _, err := r.Draw(frame); err != nil {
		t.Fatalf("Draw: %v", err)
	}
	out.Reset()
	r.Reset()
	redrew, err := r.Draw(frame)
	if err != nil {
		t.Fatalf("Draw: %v", err)
	}
	if !redrew {
		t.Fatal("Draw after Reset must paint again")
	}
	if out.Len() == 0 {
		t.Fatal("Draw after Reset wrote no bytes")
	}
	if r.Frames() != 2 {
		t.Fatalf("frames = %d, want 2", r.Frames())
	}
}

func TestDrawFirstFramePaintsAnEmptyRow(t *testing.T) {
	// The very first frame has no previous row, so even an empty row is
	// painted and the sink sees the cursor move.
	var out strings.Builder
	r := NewRenderer(&out)
	redrew, err := r.Draw(Layout{})
	if err != nil {
		t.Fatalf("Draw: %v", err)
	}
	if !redrew {
		t.Fatal("the first frame must paint")
	}
	if !strings.Contains(out.String(), "\x1b[1;1H") {
		t.Fatalf("the first row was not painted: %q", out.String())
	}
}

// failWriter fails every write.
type failWriter struct{}

func (failWriter) Write([]byte) (int, error) { return 0, errors.New("sink gone") }

func TestDrawWriteError(t *testing.T) {
	r := NewRenderer(failWriter{})
	if _, err := r.Draw(Layout{Heading: "head"}); err == nil {
		t.Fatal("a failing sink must return an error")
	}
	if r.Frames() != 0 {
		t.Fatalf("frames = %d, want 0 after a failed draw", r.Frames())
	}
	// The failure must not be remembered as the last frame, so the next
	// draw tries again.
	if _, err := r.Draw(Layout{Heading: "head"}); err == nil {
		t.Fatal("a failing sink must return an error on every draw")
	}
}
