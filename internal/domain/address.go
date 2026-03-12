package domain

import "strings"

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
