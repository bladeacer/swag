// SPDX-License-Identifier: Apache-2.0

package ass

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/bladeacer/swag/internal/model"
)

// parseColour reads an ASS colour. The stored order is &HBBGGRR& for an
// inline tag and &HAABBGGRR& for a style, where AA is the transparency and
// not the alpha. The decimal form is also accepted. alpha is the alpha to
// use when the value carries no transparency byte.
func parseColour(s string, alpha uint8) (model.Colour, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return model.Colour{}, fmt.Errorf("parse ass colour: empty value")
	}
	body := s
	if len(body) > 2 && strings.EqualFold(body[:2], "&h") {
		body = strings.TrimSuffix(body[2:], "&")
		body = strings.TrimSpace(body)
		v, err := strconv.ParseUint(body, 16, 64)
		if err != nil {
			return model.Colour{}, fmt.Errorf("parse ass colour %q: %w", s, err)
		}
		c := model.NewColour(uint8(v), uint8(v>>8), uint8(v>>16), alpha)
		if len(body) > 6 {
			// The top byte is transparency, where 0 is opaque.
			c.A = 255 - uint8(v>>24)
		}
		return c, nil
	}
	v, err := strconv.ParseUint(body, 10, 64)
	if err != nil {
		return model.Colour{}, fmt.Errorf("parse ass colour %q: %w", s, err)
	}
	return model.NewColour(uint8(v), uint8(v>>8), uint8(v>>16), alpha), nil
}

// parseTransparency reads the value of an alpha tag and returns the alpha
// channel. An ASS alpha tag holds the transparency, where 0 is opaque.
func parseTransparency(s string) (uint8, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 255, nil
	}
	body := s
	if len(body) > 2 && strings.EqualFold(body[:2], "&h") {
		body = strings.TrimSuffix(body[2:], "&")
		body = strings.TrimSpace(body)
	}
	v, err := strconv.ParseUint(body, 16, 64)
	if err != nil {
		return 0, fmt.Errorf("parse ass alpha %q: %w", s, err)
	}
	if v > 255 {
		return 0, fmt.Errorf("parse ass alpha %q: want 0 to 255", s)
	}
	return 255 - uint8(v), nil
}
