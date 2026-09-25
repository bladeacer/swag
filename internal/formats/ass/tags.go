// SPDX-License-Identifier: Apache-2.0

package ass

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/bladeacer/swag/internal/model"
	"github.com/bladeacer/swag/internal/richtext"
)

// eventState carries the mutable state of one event line. The effective
// style starts at the style of the cue and is changed by the override
// tags.
type eventState struct {
	base     model.Style
	cueStyle model.Style
	styles   map[string]model.Style
	eff      model.Style

	pos     *model.Point
	anchor  *model.Anchor
	move    *model.Move
	fade    *model.Fade
	shake   *model.Shake
	chroma  *model.Chroma
	karaoke *model.Karaoke
	keys    []model.Keyframe

	vertical  *model.Vertical
	script    *model.Script
	direction *model.Direction
	packed    *bool
	ruby      *model.RubyPosition

	// strikeout, scaleX, and scaleY are span overrides that have no
	// style equivalent in the IR.
	strikeout *bool
	scaleX    *float64
	scaleY    *float64

	// outlineSet and shadowSet report that an inline tag changed the
	// outline or shadow colour, so the span needs an explicit shadow even
	// when the style thickness is unchanged.
	outlineSet bool
	shadowSet  bool

	kTime  time.Duration
	kStart time.Duration
	kEnd   time.Duration
	kOn    bool
}

// parseEvent turns one Dialogue line into a cue.
func parseEvent(ev eventLine, format []string, styles map[string]model.Style, base model.Style) (model.Cue, error) {
	start, err := parseASSTime(fieldAt(ev.fields, format, "start"))
	if err != nil {
		return model.Cue{}, err
	}
	end, err := parseASSTime(fieldAt(ev.fields, format, "end"))
	if err != nil {
		return model.Cue{}, err
	}

	cueStyle := base
	if name := fieldAt(ev.fields, format, "style"); name != "" {
		if s, ok := styles[strings.ToLower(name)]; ok {
			cueStyle = s
		}
	}
	st := newEventState(cueStyle, base, styles)
	text := fieldAt(ev.fields, format, "text")
	spans := st.run(text)

	return model.Cue{
		Start:      start,
		End:        end,
		Spans:      spans,
		Layout:     st.layout(),
		Animations: st.animations(),
	}, nil
}

func newEventState(cueStyle, base model.Style, styles map[string]model.Style) *eventState {
	st := &eventState{base: base, cueStyle: cueStyle, styles: styles, eff: cueStyle}
	if cueStyle.Alignment != base.Alignment {
		a := cueStyle.Alignment
		st.anchor = &a
	}
	return st
}

// run walks the runs of an event line and returns the spans.
func (st *eventState) run(text string) []model.TextSpan {
	var spans []model.TextSpan
	for _, run := range richtext.Parse(text) {
		st.applyTags(run.Tags)
		spans = append(spans, st.spansFor(run.Text)...)
	}
	return spans
}

// spansFor turns a text stretch into spans, expanding ruby patterns when
// ruby is active.
func (st *eventState) spansFor(text string) []model.TextSpan {
	if st.ruby == nil {
		return []model.TextSpan{st.span(text)}
	}
	return st.rubySpans(text)
}

// span builds one span from the current state.
func (st *eventState) span(text string) model.TextSpan {
	sp := model.TextSpan{Text: text}
	st.applyDiff(&sp)
	if st.kOn {
		sp.Start = st.kStart
		sp.End = st.kEnd
	}
	return sp
}

// rubySpans expands the [base/reading] patterns of an ASS ruby line into
// base spans and annotation spans.
func (st *eventState) rubySpans(text string) []model.TextSpan {
	var spans []model.TextSpan
	for {
		open := strings.IndexByte(text, '[')
		if open < 0 {
			if text != "" {
				spans = append(spans, st.span(text))
			}
			return spans
		}
		if open > 0 {
			spans = append(spans, st.span(text[:open]))
		}
		rest := text[open:]
		close := strings.IndexByte(rest, ']')
		if close < 0 {
			spans = append(spans, st.span(rest))
			return spans
		}
		inner := rest[1:close]
		base, reading, ok := strings.Cut(inner, "/")
		if !ok {
			spans = append(spans, st.span(inner))
		} else {
			baseSpan := st.span(base)
			spans = append(spans, baseSpan)
			if reading != "" {
				ann := st.span(reading)
				pos := *st.ruby
				ann.Ruby = &model.Ruby{Position: pos}
				ann.Start = baseSpan.Start
				ann.End = baseSpan.End
				spans = append(spans, ann)
			}
		}
		text = rest[close+1:]
	}
}

// applyDiff writes the difference between the effective style and the base
// style onto a span. The base style is the document style, so a span that
// matches it needs no overrides.
func (st *eventState) applyDiff(sp *model.TextSpan) {
	eff, base := st.eff, st.base
	if eff.Font != base.Font {
		v := eff.Font
		sp.Font = &v
	}
	if eff.Size != base.Size {
		v := eff.Size
		sp.Size = &v
	}
	if eff.Bold != base.Bold {
		v := eff.Bold
		sp.Bold = &v
	}
	if eff.Italic != base.Italic {
		v := eff.Italic
		sp.Italic = &v
	}
	if eff.Underline != base.Underline {
		v := eff.Underline
		sp.Underline = &v
	}
	if eff.Primary != base.Primary {
		v := eff.Primary
		sp.Fore = &v
	}
	if eff.Secondary != base.Secondary {
		v := eff.Secondary
		sp.Secondary = &v
	}
	if eff.OutlineWidth != base.OutlineWidth {
		v := eff.OutlineWidth
		sp.OutlineWidth = &v
	}
	if eff.ShadowDepth != base.ShadowDepth {
		v := eff.ShadowDepth
		sp.ShadowDepth = &v
	}
	// The ASS outline colour doubles as the box colour of a BorderStyle 3
	// style, and the IR carries it on the span as the background colour.
	switch {
	case eff.Box && !base.Box:
		v := eff.Outline
		sp.Back = &v
	case st.outlineSet:
		v := eff.Outline
		sp.Back = &v
	case eff.OutlineWidth > 0 && !eff.Box && eff.Outline != base.Outline:
		v := eff.Outline
		sp.Back = &v
	}
	if st.shadowSet || (eff.ShadowDepth != base.ShadowDepth && eff.ShadowDepth > 0) {
		sp.Shadows = []model.Shadow{{Kind: model.ShadowHard, Colour: eff.Shadow}}
	}
	if st.vertical != nil {
		v := *st.vertical
		sp.Vertical = &v
	}
	if st.script != nil {
		v := *st.script
		sp.Script = &v
	}
	if st.direction != nil {
		v := *st.direction
		sp.Direction = &v
	}
	if st.packed != nil {
		v := *st.packed
		sp.Packed = &v
	}
	if st.strikeout != nil {
		v := *st.strikeout
		sp.Strikeout = &v
	}
	if st.scaleX != nil {
		v := *st.scaleX
		sp.ScaleX = &v
	}
	if st.scaleY != nil {
		v := *st.scaleY
		sp.ScaleY = &v
	}
}

// layout builds the cue-level placement from the tags that were seen.
func (st *eventState) layout() *model.Layout {
	if st.pos == nil && st.anchor == nil && st.move == nil && st.fade == nil {
		return nil
	}
	return &model.Layout{Position: st.pos, Anchor: st.anchor, Move: st.move, Fade: st.fade}
}

// animations builds the cue-level effects from the tags that were seen.
func (st *eventState) animations() []model.Animation {
	var out []model.Animation
	if len(st.keys) > 0 {
		out = append(out, model.Animation{Keyframes: st.keys})
	}
	if st.shake != nil {
		out = append(out, model.Animation{Shake: st.shake})
	}
	if st.chroma != nil {
		out = append(out, model.Animation{Chroma: st.chroma})
	}
	if st.karaoke != nil {
		out = append(out, model.Animation{Karaoke: st.karaoke})
	}
	return out
}

// applyTags runs a braces block over the state.
func (st *eventState) applyTags(tags []richtext.Tag) {
	for _, tag := range tags {
		st.applyTag(tag)
	}
}

// applyTag applies one override tag. A tag outside the supported tiers is
// ignored, which is the Tier 4 behaviour of the format matrix.
func (st *eventState) applyTag(tag richtext.Tag) {
	switch tag.Name {
	case "b":
		st.eff.Bold = flagValue(tag.Value, st.cueStyle.Bold)
	case "i":
		st.eff.Italic = flagValue(tag.Value, st.cueStyle.Italic)
	case "u":
		st.eff.Underline = flagValue(tag.Value, st.cueStyle.Underline)
	case "fn":
		if tag.Value == "" {
			st.eff.Font = st.cueStyle.Font
		} else {
			st.eff.Font = tag.Value
		}
	case "fs":
		if tag.Value == "" {
			st.eff.Size = st.cueStyle.Size
		} else if v, err := strconv.ParseFloat(strings.TrimSpace(tag.Value), 64); err == nil {
			st.eff.Size = v
		}
	case "c", "1c":
		st.setColour(&st.eff.Primary, tag.Value, st.cueStyle.Primary)
	case "2c":
		st.setColour(&st.eff.Secondary, tag.Value, st.cueStyle.Secondary)
	case "3c":
		st.setColour(&st.eff.Outline, tag.Value, st.cueStyle.Outline)
		st.outlineSet = tag.Value != ""
	case "4c":
		st.setColour(&st.eff.Shadow, tag.Value, st.cueStyle.Shadow)
		st.shadowSet = tag.Value != ""
	case "3a":
		st.setAlpha("3a", tag.Value)
		st.outlineSet = tag.Value != ""
	case "4a":
		st.setAlpha("4a", tag.Value)
		st.shadowSet = tag.Value != ""
	case "bord":
		if tag.Value == "" {
			st.eff.OutlineWidth = st.cueStyle.OutlineWidth
		} else if v, err := strconv.ParseFloat(strings.TrimSpace(tag.Value), 64); err == nil {
			st.eff.OutlineWidth = v
		}
	case "shad", "xshad", "yshad":
		st.setShadowDepth(tag.Value)
	case "s":
		v := flagValue(tag.Value, false)
		st.strikeout = &v
	case "fscx":
		st.scaleX = scaleValue(tag.Value, st.scaleX)
	case "fscy":
		st.scaleY = scaleValue(tag.Value, st.scaleY)
	case "1a", "2a":
		st.setAlpha(tag.Name, tag.Value)
	case "alpha":
		st.setAllAlpha(tag.Value)
	case "k", "K", "kf", "ko":
		st.karaokeTag(tag.Value)
	case "r":
		st.reset(tag.Value)
	case "an":
		if v, err := strconv.Atoi(strings.TrimSpace(tag.Value)); err == nil {
			a := anchor(v)
			st.anchor = &a
		}
	case "a":
		if v, err := strconv.Atoi(strings.TrimSpace(tag.Value)); err == nil {
			if a, ok := legacyAnchor(v); ok {
				st.anchor = &a
			}
		}
	case "pos":
		st.pos = pointFromArgs(tag.Args)
	case "move":
		st.move = moveFromArgs(tag.Args)
	case "fad":
		st.fade = fadeFromArgs(tag.Args)
	case "fade":
		st.fade = complexFadeFromArgs(tag.Args)
	case "t":
		st.addKeyframe(tag.Args)
	case "ytshake":
		st.shake = shakeFromArgs(tag.Args)
	case "ytchroma":
		st.chroma = chromaFromArgs(tag.Args)
	case "ytktFade":
		st.karaoke = &model.Karaoke{Kind: model.KaraokeFade}
	case "ytktGlitch":
		st.karaoke = &model.Karaoke{Kind: model.KaraokeGlitch}
	case "ytkt":
		st.karaoke = karaokeFromArgs(tag.Args)
	case "ytsup":
		st.setScript(model.ScriptSuperscript)
	case "ytsub":
		st.setScript(model.ScriptSubscript)
	case "ytsur":
		st.setScript(model.ScriptRegular)
	case "ytruby":
		pos := model.RubyOver
		if strings.TrimSpace(tag.Value) == "2" {
			pos = model.RubyUnder
		}
		st.ruby = &pos
	case "ytvert":
		st.setVertical(strings.TrimSpace(tag.Value))
	case "ytpack":
		v := strings.TrimSpace(tag.Value) != "0"
		st.packed = &v
	case "ytdir":
		st.setDirection(strings.TrimSpace(tag.Value))
	default:
		// Tier 4: accepted and ignored.
	}
}

func (st *eventState) setColour(dst *model.Colour, value string, reset model.Colour) {
	if strings.TrimSpace(value) == "" {
		*dst = reset
		return
	}
	c, err := parseColour(value, dst.A)
	if err != nil {
		return
	}
	*dst = c
}

func (st *eventState) setAlpha(name, value string) {
	if strings.TrimSpace(value) == "" {
		switch name {
		case "1a":
			st.eff.Primary = st.cueStyle.Primary
		case "2a":
			st.eff.Secondary = st.cueStyle.Secondary
		case "3a":
			st.eff.Outline = st.cueStyle.Outline
		case "4a":
			st.eff.Shadow = st.cueStyle.Shadow
		}
		return
	}
	a, err := parseTransparency(value)
	if err != nil {
		return
	}
	switch name {
	case "1a":
		st.eff.Primary.A = a
	case "2a":
		st.eff.Secondary.A = a
	case "3a":
		st.eff.Outline.A = a
	case "4a":
		st.eff.Shadow.A = a
	}
}

func (st *eventState) setAllAlpha(value string) {
	if strings.TrimSpace(value) == "" {
		st.eff.Primary.A = st.cueStyle.Primary.A
		st.eff.Secondary.A = st.cueStyle.Secondary.A
		st.eff.Outline.A = st.cueStyle.Outline.A
		st.eff.Shadow.A = st.cueStyle.Shadow.A
		return
	}
	a, err := parseTransparency(value)
	if err != nil {
		return
	}
	st.eff.Primary.A = a
	st.eff.Secondary.A = a
	st.eff.Outline.A = a
	st.eff.Shadow.A = a
}

// karaokeTag starts a new karaoke segment. The value is a duration in
// centiseconds.
func (st *eventState) karaokeTag(value string) {
	cs, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return
	}
	st.kStart = st.kTime
	st.kEnd = st.kTime + time.Duration(cs)*10*time.Millisecond
	st.kTime = st.kEnd
	st.kOn = true
}

// reset returns the effective style to the style of the cue, or to a named
// style when the tag names one.
func (st *eventState) reset(value string) {
	name := strings.TrimSpace(value)
	if name == "" {
		st.eff = st.cueStyle
	} else if s, ok := st.styles[strings.ToLower(name)]; ok {
		st.eff = s
	}
	st.outlineSet = false
	st.shadowSet = false
	st.strikeout = nil
	st.scaleX = nil
	st.scaleY = nil
}

// setShadowDepth applies a shadow distance override. A blank value returns
// to the style distance.
func (st *eventState) setShadowDepth(value string) {
	if strings.TrimSpace(value) == "" {
		st.eff.ShadowDepth = st.cueStyle.ShadowDepth
		return
	}
	v, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
	if err != nil {
		return
	}
	st.eff.ShadowDepth = v
}

// legacyAnchors maps an SSA \a alignment onto a numpad anchor. The legacy
// values cluster by row: 1 to 3 bottom, 5 to 7 top, 9 to 11 middle. The
// values 4 and 8 stay out because the legacy numbering does not use them.
var legacyAnchors = map[int]model.Anchor{
	1:  model.AnchorBottomLeft,
	2:  model.AnchorBottomCentre,
	3:  model.AnchorBottomRight,
	5:  model.AnchorTopLeft,
	6:  model.AnchorTopCentre,
	7:  model.AnchorTopRight,
	9:  model.AnchorMiddleLeft,
	10: model.AnchorCentre,
	11: model.AnchorMiddleRight,
}

// legacyAnchor reads an SSA \a alignment. It reports false for a value
// outside the legacy table.
func legacyAnchor(v int) (model.Anchor, bool) {
	a, ok := legacyAnchors[v]
	return a, ok
}

// scaleValue reads a glyph scale override. A blank value clears the
// override, and an unreadable one keeps the current value.
func scaleValue(value string, current *float64) *float64 {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	v, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
	if err != nil {
		return current
	}
	return &v
}

func (st *eventState) setScript(kind model.ScriptKind) {
	v := model.Script{Kind: kind}
	st.script = &v
}

// setVertical applies a \ytvert mode. A blank value returns the run to
// horizontal text, which lets a writer leave vertical text without a loss.
func (st *eventState) setVertical(value string) {
	var mode model.VerticalMode
	switch value {
	case "":
		mode = model.VerticalNone
	case "9":
		mode = model.VerticalColumnsRTL
	case "7":
		mode = model.VerticalColumnsLTR
	case "1":
		mode = model.VerticalRotated
	case "3":
		mode = model.VerticalRotatedReversed
	default:
		return
	}
	st.vertical = &model.Vertical{Mode: mode}
}

func (st *eventState) setDirection(value string) {
	switch value {
	case "4":
		d := model.DirRightToLeft
		st.direction = &d
	case "6":
		d := model.DirLeftToRight
		st.direction = &d
	}
}

// flagValue reads a boolean override value. An empty value resets to the
// style value.
func flagValue(value string, reset bool) bool {
	value = strings.TrimSpace(value)
	switch strings.ToLower(value) {
	case "":
		return reset
	case "0", "false", "no":
		return false
	case "1", "true", "yes", "-1":
		return true
	}
	if v, err := strconv.Atoi(value); err == nil {
		return v != 0
	}
	return reset
}

func pointFromArgs(args []string) *model.Point {
	if len(args) < 2 {
		return nil
	}
	x, errX := strconv.ParseFloat(args[0], 64)
	y, errY := strconv.ParseFloat(args[1], 64)
	if errX != nil || errY != nil {
		return nil
	}
	return &model.Point{X: x, Y: y}
}

func moveFromArgs(args []string) *model.Move {
	if len(args) < 4 {
		return nil
	}
	nums := make([]float64, 4)
	for i := 0; i < 4; i++ {
		v, err := strconv.ParseFloat(args[i], 64)
		if err != nil {
			return nil
		}
		nums[i] = v
	}
	m := &model.Move{From: model.Point{X: nums[0], Y: nums[1]}, To: model.Point{X: nums[2], Y: nums[3]}}
	if len(args) >= 6 {
		if t1, err := strconv.Atoi(args[4]); err == nil {
			m.Start = time.Duration(t1) * time.Millisecond
		}
		if t2, err := strconv.Atoi(args[5]); err == nil {
			m.End = time.Duration(t2) * time.Millisecond
		}
	}
	return m
}

func fadeFromArgs(args []string) *model.Fade {
	if len(args) < 2 {
		return nil
	}
	in, errIn := strconv.Atoi(args[0])
	out, errOut := strconv.Atoi(args[1])
	if errIn != nil || errOut != nil {
		return nil
	}
	return &model.Fade{
		In:  time.Duration(in) * time.Millisecond,
		Out: time.Duration(out) * time.Millisecond,
	}
}

func complexFadeFromArgs(args []string) *model.Fade {
	if len(args) < 7 {
		return nil
	}
	nums := make([]int, 7)
	for i := 0; i < 7; i++ {
		v, err := strconv.Atoi(args[i])
		if err != nil {
			return nil
		}
		nums[i] = v
	}
	return &model.Fade{
		StartAlpha: uint8(clamp255(nums[0])),
		MidAlpha:   uint8(clamp255(nums[1])),
		EndAlpha:   uint8(clamp255(nums[2])),
		StartIn:    time.Duration(nums[3]) * time.Millisecond,
		EndIn:      time.Duration(nums[4]) * time.Millisecond,
		StartOut:   time.Duration(nums[5]) * time.Millisecond,
		EndOut:     time.Duration(nums[6]) * time.Millisecond,
	}
}

func clamp255(v int) int {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return v
}

// addKeyframe records a \t animation.
func (st *eventState) addKeyframe(args []string) {
	if len(args) == 0 {
		return
	}
	kf := model.Keyframe{Easing: 1}
	tags := args
	switch len(args) {
	case 1:
		// \t(tags)
	case 3:
		// \t(t1,t2,tags)
		kf.Start = millis(args[0])
		kf.End = millis(args[1])
		tags = args[2:]
	default:
		// \t(t1,t2,accel,tags)
		kf.Start = millis(args[0])
		kf.End = millis(args[1])
		if v, err := strconv.ParseFloat(args[2], 64); err == nil {
			kf.Easing = v
		}
		tags = args[3:]
	}
	for _, step := range tagSteps(strings.Join(tags, ",")) {
		kf.Steps = append(kf.Steps, step)
	}
	if len(kf.Steps) == 0 {
		return
	}
	st.keys = append(st.keys, kf)
}

// tagSteps turns the inner tags of a \t block into keyframe steps.
func tagSteps(text string) []model.KeyframeStep {
	var steps []model.KeyframeStep
	for _, tag := range richtext.ParseTags(text) {
		steps = append(steps, model.KeyframeStep{
			Property: keyframeProperty(tag.Name),
			Value:    strings.TrimSpace(tag.Value),
		})
	}
	return steps
}

func keyframeProperty(name string) string {
	switch name {
	case "c", "1c":
		return "fore"
	case "2c":
		return "secondary"
	case "3c":
		return "outline"
	case "4c":
		return "shadow"
	case "1a":
		return "forealpha"
	case "2a":
		return "secondaryalpha"
	case "3a":
		return "outlinealpha"
	case "4a":
		return "shadowalpha"
	case "alpha":
		return "alpha"
	case "fs":
		return "fontsize"
	case "bord":
		return "outlinewidth"
	case "shad":
		return "shadowdepth"
	}
	return name
}

func millis(s string) time.Duration {
	v, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil {
		return 0
	}
	return time.Duration(v) * time.Millisecond
}

func shakeFromArgs(args []string) *model.Shake {
	sh := &model.Shake{RadiusX: 20, RadiusY: 20}
	nums := make([]float64, 0, len(args))
	for _, a := range args {
		v, err := strconv.ParseFloat(strings.TrimSpace(a), 64)
		if err != nil {
			return sh
		}
		nums = append(nums, v)
	}
	switch len(nums) {
	case 0:
	case 1:
		sh.RadiusX, sh.RadiusY = nums[0], nums[0]
	case 2:
		sh.RadiusX, sh.RadiusY = nums[0], nums[1]
	case 3:
		sh.RadiusX, sh.RadiusY = nums[0], nums[0]
		sh.Start, sh.End = millisFloat(nums[1]), millisFloat(nums[2])
	default:
		sh.RadiusX, sh.RadiusY = nums[0], nums[1]
		sh.Start, sh.End = millisFloat(nums[2]), millisFloat(nums[3])
	}
	return sh
}

func millisFloat(v float64) time.Duration {
	return time.Duration(v * float64(time.Millisecond))
}

func chromaFromArgs(args []string) *model.Chroma {
	ch := &model.Chroma{InTime: 270 * time.Millisecond, OutTime: 270 * time.Millisecond}
	offsetX, offsetY := 20.0, 0.0
	inTime, outTime := 270.0, 270.0
	copies := 3

	switch {
	case len(args) == 2:
		if v, err := strconv.ParseFloat(args[0], 64); err == nil {
			inTime = v
		}
		if v, err := strconv.ParseFloat(args[1], 64); err == nil {
			outTime = v
		}
	case len(args) == 4:
		if v, err := strconv.ParseFloat(args[0], 64); err == nil {
			offsetX = v
		}
		if v, err := strconv.ParseFloat(args[1], 64); err == nil {
			offsetY = v
		}
		if v, err := strconv.ParseFloat(args[2], 64); err == nil {
			inTime = v
		}
		if v, err := strconv.ParseFloat(args[3], 64); err == nil {
			outTime = v
		}
	case len(args) >= 5:
		// colours..., alpha, offsetX, offsetY, intime, outtime
		end := len(args) - 5
		for _, raw := range args[:end] {
			if col, err := parseColour(raw, 255); err == nil {
				ch.Colours = append(ch.Colours, col)
			}
		}
		copies = len(ch.Colours)
		if copies < 1 {
			copies = 1
		}
		if a, err := parseTransparency(args[end]); err == nil {
			ch.Alpha = a
		}
		if v, err := strconv.ParseFloat(args[len(args)-4], 64); err == nil {
			offsetX = v
		}
		if v, err := strconv.ParseFloat(args[len(args)-3], 64); err == nil {
			offsetY = v
		}
		if v, err := strconv.ParseFloat(args[len(args)-2], 64); err == nil {
			inTime = v
		}
		if v, err := strconv.ParseFloat(args[len(args)-1], 64); err == nil {
			outTime = v
		}
	}
	ch.InTime = millisFloat(inTime)
	ch.OutTime = millisFloat(outTime)
	ch.Offsets = spreadOffsets(offsetX, offsetY, copies)
	return ch
}

// spreadOffsets returns one offset per copy, spread from the first copy to
// the last around the centre.
func spreadOffsets(offsetX, offsetY float64, n int) []model.Point {
	if n < 1 {
		n = 1
	}
	offsets := make([]model.Point, n)
	if n == 1 {
		return offsets
	}
	for i := 0; i < n; i++ {
		f := 2*float64(i)/float64(n-1) - 1
		offsets[i] = model.Point{X: f * offsetX, Y: f * offsetY}
	}
	return offsets
}

// karaokeFromArgs reads a \ytkt cursor form. The first argument names the
// cursor kind. The remaining arguments are one text, a tag set and a text,
// or an interval followed by tag and text pairs for an animated cursor.
func karaokeFromArgs(args []string) *model.Karaoke {
	if len(args) == 0 {
		return nil
	}
	kind := strings.ToLower(strings.TrimSpace(args[0]))
	switch kind {
	case "cursor", "lcursor", "rcursor":
	default:
		return nil
	}
	k := &model.Karaoke{Kind: model.KaraokeCursor, CursorLeft: kind == "lcursor"}
	rest := args[1:]
	switch {
	case len(rest) == 1:
		k.Cursor = rest[0]
	case len(rest) == 2:
		k.CursorTags = rest[0]
		k.Cursor = rest[1]
	case len(rest) >= 3:
		// (kind, interval, tags1, text1, tags2, text2, ...)
		k.CursorInterval = millis(rest[0])
		for i := 1; i+1 < len(rest); i += 2 {
			k.CursorFrames = append(k.CursorFrames, model.KaraokeFrame{Tags: rest[i], Text: rest[i+1]})
		}
	}
	return k
}

// parseASSTime reads an ASS timestamp of the form H:MM:SS.cc.
func parseASSTime(s string) (time.Duration, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, nil
	}
	parts := strings.Split(s, ":")
	if len(parts) != 3 {
		return 0, fmt.Errorf("parse ass time %q: want H:MM:SS.cc", s)
	}
	h, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, fmt.Errorf("parse ass time %q: bad hours", s)
	}
	m, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, fmt.Errorf("parse ass time %q: bad minutes", s)
	}
	sec, frac, _ := strings.Cut(parts[2], ".")
	secs, err := strconv.Atoi(sec)
	if err != nil {
		return 0, fmt.Errorf("parse ass time %q: bad seconds", s)
	}
	var cs int
	if frac != "" {
		// The fraction is centiseconds, so pad or trim to two digits.
		if len(frac) > 2 {
			frac = frac[:2]
		}
		for len(frac) < 2 {
			frac += "0"
		}
		cs, err = strconv.Atoi(frac)
		if err != nil {
			return 0, fmt.Errorf("parse ass time %q: bad centiseconds", s)
		}
	}
	return time.Duration(h)*time.Hour + time.Duration(m)*time.Minute +
		time.Duration(secs)*time.Second + time.Duration(cs)*10*time.Millisecond, nil
}
