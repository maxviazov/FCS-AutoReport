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

// MoHStreetWithoutLeadingCity — для колонки «כתובת»: только улица и дом, без названия города в начале.
// В сыром SAP часто «עיר, רחוב מספר»; в реестре נקודות שיווק город уже в קוד עיר.
func MoHStreetWithoutLeadingCity(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	_, after, ok := strings.Cut(s, ",")
	if !ok {
		return s
	}
	return strings.TrimSpace(after)
}

// MoHStreetLineForMoH — одна строка כתובת: при «עיר, רחוב» (cityAfterComma=false) оставляем правую часть; при «רחוב, עיר» — левую.
func MoHStreetLineForMoH(normalizedAddr string, cityAfterComma bool) string {
	s := strings.TrimSpace(normalizedAddr)
	if s == "" {
		return ""
	}
	if !strings.Contains(s, ",") {
		return s
	}
	if cityAfterComma {
		before, _, ok := strings.Cut(s, ",")
		if ok {
			return strings.TrimSpace(before)
		}
		return s
	}
	return MoHStreetWithoutLeadingCity(s)
}

// InferCityPlacedAfterComma — эвристика без справочника: «улица с номером, город» (для post-export чужих файлов).
func InferCityPlacedAfterComma(normalizedAddr string) bool {
	s := strings.TrimSpace(normalizedAddr)
	if s == "" || !strings.Contains(s, ",") {
		return false
	}
	before, after, ok := strings.Cut(s, ",")
	if !ok {
		return false
	}
	b, a := strings.TrimSpace(before), strings.TrimSpace(after)
	if b == "" || a == "" {
		return false
	}
	digitLeft := strings.ContainsAny(b, "0123456789")
	digitRight := strings.ContainsAny(a, "0123456789")
	if digitLeft && !digitRight && len([]rune(a)) <= 24 {
		return true
	}
	return false
}

// stripPathSuffix убирает \ или / и всё после (артефакт SAP/Excel, напр. «העצמאות 23\87», «שבי ציון 2/124»).
func stripPathSuffix(s string) string {
	cut := -1
	if i := strings.IndexByte(s, '\\'); i >= 0 {
		cut = i
	}
	if i := strings.IndexByte(s, '/'); i >= 0 && (cut < 0 || i < cut) {
		cut = i
	}
	if cut >= 0 {
		s = strings.TrimSpace(s[:cut])
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
	s = stripPathSuffix(s)
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
