package domain

import (
	"regexp"
	"strings"
)

// ExtractCityFromAddress вычленяет название города из сырого адреса.
// Берётся часть до первой запятой, затем нормализация пробелов.
// Пример: "תל-אביב, אלנבי 15" → "תל-אביב".
func ExtractCityFromAddress(addr string) string {
	before, _, _ := strings.Cut(strings.TrimSpace(addr), ",")
	return NormalizeText(before)
}

// CityPrefixes — префиксы, которые часто идут перед названием города (напр. "איזור תעשייה כנות" → город "כנות").
// При поиске кода города их можно отбросить и искать по оставшейся части.
var CityPrefixes = []string{
	"איזור תעשייה ",  // промышленная зона
	"אזור תעשייה ",
	"מתחם ",          // комплекс
}

// StripCityPrefix убирает первый подходящий префикс из строки и возвращает нормализованный остаток.
// Если префикс не найден, возвращает исходную строку (нормализованную).
func StripCityPrefix(s string) string {
	s = NormalizeText(s)
	for _, p := range CityPrefixes {
		if strings.HasPrefix(s, p) {
			return NormalizeText(strings.TrimPrefix(s, p))
		}
	}
	return s
}

// NormalizeMinistryAddress приводит כתובת к виду, который чаще совпадает с реестром נקודות שיווק:
// «רח'»/типографская кавычка после «רח» → «רחוב», пробел перед запятой.
func NormalizeMinistryAddress(addr string) string {
	s := NormalizeText(addr)
	if s == "" {
		return ""
	}
	s = strings.ReplaceAll(s, "רח'", "רחוב")
	s = strings.ReplaceAll(s, "רח׳", "רחוב")
	s = strings.ReplaceAll(s, "רח\u2018", "רחוב")
	s = strings.ReplaceAll(s, "רח\u2019", "רחוב")
	s = strings.ReplaceAll(s, "רח`", "רחוב")
	s = strings.ReplaceAll(s, " ,", ",")
	// «2,עד» после удаления пробела перед запятой — добавляем пробел после запятой.
	s = commaNeedSpaceAfter.ReplaceAllString(s, ", $1")
	return NormalizeText(s)
}

// Запятая, за которой сразу идёт непробельный символ (кроме цифры) — вставить пробел.
var commaNeedSpaceAfter = regexp.MustCompile(`,([^\s\d])`)
