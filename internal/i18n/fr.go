// SPDX-License-Identifier: Apache-2.0

package i18n

// frenchLocale is the second shipped locale. It proves that a new language
// lands as one catalogue file, with no change to the CLI. The values carry
// the same placeholders as en-GB, so a lookup formats identically.
const frenchLocale Locale = "fr-FR"

// frFR is the French message set. Missing keys fall back to en-GB.
var frFR = map[Key]string{
	MsgBannerTitle:       "swag (Subtitles With A Gopher)",
	MsgBannerTagline:     "Lisez, écrivez et convertissez des sous-titres.",
	MsgCliDescription:    "Subtitles With A Gopher : lire, écrire et convertir des sous-titres.",
	MsgTargetPlaceholder: "?",
	MsgOutputStdout:      "la sortie standard",
	MsgConvertStart:      "Conversion de %s vers le format %s.",
	MsgConvertSuccess:    "%s écrit.",
	MsgConvertFailed:     "La conversion a échoué. %s",
	MsgConvertLosses:     "Fonctionnalités absentes du format cible (%d) :",
	MsgUsageFailed:       "La ligne de commande n'a pas pu être lue. %s",
	MsgInputMissing:      "Le fichier d'entrée n'existe pas. Indiquez le chemin d'un fichier de sous-titres avec -i.",
	MsgInputUnreadable:   "Le fichier d'entrée n'a pas pu être lu. %s",
	MsgInputIsDirectory:  "Le chemin d'entrée %s est un dossier. Indiquez le chemin d'un fichier de sous-titres.",
	MsgFormatUnknown:     "Le format de %s est inconnu. Nommez le format avec -f.",
	MsgFormatUnsupported: "Le format %s n'est pas pris en charge pour cette opération.",
	MsgOutputCreate:      "Le fichier de sortie %s n'a pas pu être créé.",
}

func init() {
	Register(frenchLocale, frFR)
}
