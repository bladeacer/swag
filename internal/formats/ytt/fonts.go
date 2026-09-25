// SPDX-License-Identifier: Apache-2.0

package ytt

import (
	"math"
	"strings"
)

// FontStyle is the YouTube font family, stored in the "fs" attribute of a
// pen. The values match the YouTube player font list.
type FontStyle int

// The YouTube font families. The player offers these eight choices only.
const (
	// FontDefault is the default proportional sans family, the same as
	// FontProportionalSans.
	FontDefault FontStyle = iota
	// FontMonospaceSerif is a serif family with a fixed pitch.
	FontMonospaceSerif
	// FontProportionalSerif is a serif family with a variable pitch.
	FontProportionalSerif
	// FontMonospaceSans is a sans family with a fixed pitch.
	FontMonospaceSans
	// FontProportionalSans is a sans family with a variable pitch.
	FontProportionalSans
	// FontCasual is a relaxed hand-written family.
	FontCasual
	// FontCursive is a joined hand-written family.
	FontCursive
	// FontSmallCaps is a sans family rendered in small capitals.
	FontSmallCaps
)

// fontFamily ties one YouTube font choice to its canonical name and to the
// names that map onto it.
type fontFamily struct {
	style     FontStyle
	canonical string
	names     []string
}

// fontFamilies is the single font table of the project. The writer looks a
// font name up here and snaps an unknown name to Roboto. The reader turns
// an "fs" value into the canonical name. Every name is compared without
// case.
var fontFamilies = []fontFamily{
	{FontDefault, "Roboto", nil},
	{FontMonospaceSerif, "Courier New", []string{"Courier New"}},
	{FontProportionalSerif, "Times New Roman", []string{"Times New Roman", "Georgia"}},
	{FontMonospaceSans, "Lucida Console", []string{"Lucida Console"}},
	{FontProportionalSans, "Roboto", []string{
		"Arial", "Arial Black", "Arial Narrow", "Impact", "Roboto",
		"Tahoma", "Trebuchet MS", "Verdana",
	}},
	{FontCasual, "Comic Sans MS", []string{"Comic Sans MS"}},
	{FontCursive, "Monotype Corsiva", []string{"Monotype Corsiva"}},
	{FontSmallCaps, "Arial Small Caps", nil},
}

// Roboto is the family that every unknown font snaps to. The YouTube
// player uses it as the default.
const Roboto = "Roboto"

// LookupFont returns the YouTube font choice for a family name. It reports
// false when the name is outside the allow-list, so the caller can snap to
// Roboto and record a loss.
func LookupFont(name string) (FontStyle, bool) {
	name = strings.TrimSpace(name)
	for _, family := range fontFamilies {
		for _, known := range family.names {
			if strings.EqualFold(known, name) {
				return family.style, true
			}
		}
	}
	return FontProportionalSans, false
}

// FontName returns the canonical family name for a YouTube font choice.
func FontName(style FontStyle) string {
	for _, family := range fontFamilies {
		if family.style == style {
			return family.canonical
		}
	}
	return Roboto
}

// ScaleFromYTT converts a pen "sz" value into a real scale factor. The
// YouTube value is a virtual percentage: the player applies the factor
// 1 + (sz/100 - 1)/4, so "sz" 200 gives a 25% increase, not a doubling.
func ScaleFromYTT(sz int) float64 {
	return 1 + (float64(sz)-100)/400
}

// ScaleToYTT converts a real scale factor into a pen "sz" value, the
// inverse of ScaleFromYTT. The value can not be negative, so a factor
// below 0.75 clamps to 0, the smallest value the upload accepts.
func ScaleToYTT(scale float64) int {
	sz := int(math.Round(400*scale - 300))
	if sz < 0 {
		return 0
	}
	return sz
}
