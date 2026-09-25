// SPDX-License-Identifier: Apache-2.0

package i18n

// enGB holds the messages whose wording differs from the default locale.
// The project ships two English catalogues rather than a translated one,
// because a new language arrives through machine assistance and a human
// audit, and an English variant keeps that audit small. Every other key
// falls back to the default catalogue, so an auditor reviews the differences
// and not the whole message set.
//
// The two differences that matter are spelling and vocabulary. An American
// "license" is a British "licence", an American "color" is a British
// "colour", and an American "catalog" is a British "catalogue".
var enGB = map[Key]string{
	MsgVersionLicence: "Apache-2.0 licence",
}

func init() {
	Register(BritishLocale, enGB)
}
