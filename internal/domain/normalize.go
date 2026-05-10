package domain

import (
	"strings"
	"unicode"
)

// Нормализуемые пробельные символы (часто приходят из Excel) — приводим к обычному пробелу.
var spaceLikeRunes = map[rune]bool{
	'\u00A0': true, // no-break space
	'\u202F': true, // narrow no-break space
	'\u2007': true, // figure space
	'\u200B': true, // zero-width space
	'\u200C': true, // zero-width non-joiner
	'\u200D': true, // zero-width joiner
	'\uFEFF': true, // BOM / zero-width no-break space
}

// NormalizeText очищает строку для сопоставления: обрезка пробелов, схлопывание повторяющихся пробелов.
// Убирает BOM и подменяет «пробелоподобные» символы из Excel на обычный пробел, чтобы алиас совпадал с полем из файла.
func NormalizeText(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return s
	}
	var b strings.Builder
	b.Grow(len(s))
	prevSpace := false
	for _, r := range s {
		if r == ' ' || spaceLikeRunes[r] || unicode.IsSpace(r) {
			if !prevSpace {
				b.WriteRune(' ')
				prevSpace = true
			}
			continue
		}
		prevSpace = false
		b.WriteRune(r)
	}
	return strings.TrimSpace(b.String())
}

// NormalizeCityLookupKey приводит фрагмент адреса к виду, совпадающему со справочником МОЗ:
// в сырых файлах город часто пишут с дефисом или maqaf между словами (מגדל-העמק), в справочнике — с пробелом (מגדל העמק).
func NormalizeCityLookupKey(s string) string {
	s = strings.ReplaceAll(s, "\u05be", " ") // maqaf
	s = strings.ReplaceAll(s, "-", " ")
	s = strings.ReplaceAll(s, "–", " ") // en dash
	s = strings.ReplaceAll(s, "—", " ") // em dash
	return NormalizeText(s)
}
