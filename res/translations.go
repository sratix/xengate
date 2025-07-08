package res

import "embed"

//go:embed translations
var Translations embed.FS

type TranslationInfo struct {
	Name                string
	DisplayName         string
	TranslationFileName string
}

var TranslationsInfo = []TranslationInfo{
	{Name: "en", DisplayName: "English", TranslationFileName: "en.json"},
	{Name: "fa", DisplayName: "پارسی", TranslationFileName: "fa.json"},
}
