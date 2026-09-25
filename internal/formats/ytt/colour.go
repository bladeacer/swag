// SPDX-License-Identifier: Apache-2.0

package ytt

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/bladeacer/swag/internal/model"
)

// maxOpacity is the highest alpha value a writer may emit. YouTube drops a
// colour attribute whose opacity is 255, so the ceiling of 254 keeps the
// attribute alive and overrules the viewer's saved settings.
const maxOpacity = 254

// parseHexColour reads "#RRGGBB". The result is opaque: the caller sets the
// alpha from the matching opacity attribute.
func parseHexColour(s string) (model.Colour, error) {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "#")
	if len(s) != 6 {
		return model.Colour{}, fmt.Errorf("parse colour %q: want #RRGGBB", s)
	}
	v, err := strconv.ParseUint(s, 16, 32)
	if err != nil {
		return model.Colour{}, fmt.Errorf("parse colour %q: %w", s, err)
	}
	return model.NewColour(uint8(v>>16), uint8(v>>8), uint8(v), 255), nil
}

// formatHexColour renders a colour as "#RRGGBB" and drops the alpha
// channel, which travels in the separate opacity attribute.
func formatHexColour(c model.Colour) string {
	return fmt.Sprintf("#%02X%02X%02X", c.R, c.G, c.B)
}

// clampOpacity holds the alpha channel at or below maxOpacity.
func clampOpacity(c model.Colour) model.Colour {
	if c.A > maxOpacity {
		c.A = maxOpacity
	}
	return c
}

// shiftWhite moves a pure white foreground to #FEFEFE. The YouTube Android
// client ignores a foreground that equals #FFFFFF, so the shifted colour
// survives the upload.
func shiftWhite(c model.Colour) model.Colour {
	if c.R == 255 && c.G == 255 && c.B == 255 {
		c.R, c.G, c.B = 0xFE, 0xFE, 0xFE
	}
	return c
}

// brightenDark lifts a pure black foreground to #010101. YouTube drops a
// foreground that matches its default background, which makes dark text
// invisible.
func brightenDark(c model.Colour) model.Colour {
	if c.R == 0 && c.G == 0 && c.B == 0 {
		c.R, c.G, c.B = 1, 1, 1
	}
	return c
}

// parseOpacity reads an opacity attribute. A missing attribute means the
// default of 255, so the caller can leave the attribute out.
func parseOpacity(s string) (uint8, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 255, nil
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("parse opacity %q: %w", s, err)
	}
	if v < 0 || v > 255 {
		return 0, fmt.Errorf("parse opacity %q: want 0 to 255", s)
	}
	return uint8(v), nil
}
