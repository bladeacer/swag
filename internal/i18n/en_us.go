// SPDX-License-Identifier: Apache-2.0

package i18n

// americanLocale is the second shipped English locale. The project ships two
// English catalogues rather than a translated one, because a new language
// arrives through machine assistance and a human audit, and an English
// variant keeps that audit small.
const americanLocale Locale = "en-US"

// enUS holds the entries whose wording differs from British English. Every
// other key falls back to en-GB, so an auditor reviews the differences and
// not the whole message set.
//
// The two differences that matter are spelling and vocabulary. A British
// "licence" is an American "license", a British "colour" is an American
// "color", and a British "catalogue" is an American "catalog".
var enUS = map[Key]string{
	MsgVersionLicence: "Apache-2.0 license",
}

func init() {
	Register(americanLocale, enUS)
}
