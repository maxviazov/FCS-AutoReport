package domain

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// CitySubstringMatchesToken — вхождение названия города как отдельного токена (иврит):
// иначе «לאילת» в адресе даёт ложное «אילת» (N61 в портале МОЗ — Эйлат).
func CitySubstringMatchesToken(haystack, name string) bool {
	start := 0
	for {
		i := strings.Index(haystack[start:], name)
		if i < 0 {
			return false
		}
		i += start
		if hebrewCitySubstringAtOKBoundaries(haystack, name, i) {
			return true
		}
		start = i + 1
	}
}

func hebrewCitySubstringAtOKBoundaries(haystack, name string, iByte int) bool {
	if iByte > 0 {
		prev, _ := utf8.DecodeLastRuneInString(haystack[:iByte])
		if unicode.Is(unicode.Hebrew, prev) && !hebrewPrevOKBeforeCityToken(prev, name) {
			return false
		}
	}
	past := iByte + len(name)
	if past < len(haystack) {
		next, _ := utf8.DecodeRuneInString(haystack[past:])
		if unicode.Is(unicode.Hebrew, next) {
			return false
		}
	}
	return true
}

func hebrewPrevOKBeforeCityToken(prev rune, cityName string) bool {
	if prev == 'ל' && cityName == "אילת" {
		return false
	}
	switch prev {
	case 'ב', 'ל', 'כ', 'מ', 'ה', 'ו', 'ש':
		return true
	default:
		return false
	}
}

// AllowMoHN61CityCode — N61 в реестре МОЗ = אילת (Эйлат). Используется при подборе кода города из сырого отчёта.
// В SAP часто «אילת, רחוב …» — это достаточное основание для N61 (город явно в префиксе).
// В колонке כתובת файла МОЗ город при «עיר, רחוב» убирается — самопроверка экспорта не должна опираться только на эту колонку для N61.
// מחוז не учитываем: в FishKA часто «Эйлат»/«נפת אילת» при доставке не в Эйлате.
func AllowMoHN61CityCode(addr, clientName, district, rawCityCol string) bool {
	_ = clientName
	_ = rawCityCol
	addrTrim := strings.TrimSpace(addr)
	addrNorm := NormalizeText(addrTrim)

	before, after, ok := strings.Cut(addrNorm, ",")
	afterTrim := strings.TrimSpace(after)
	beforeTrim := strings.TrimSpace(before)
	// Явный префикс «אילת,» в сыром адресе — реальный Эйлат (напр. אילת, החורש 9).
	prefixEilat := ok && beforeTrim == "אילת" && afterTrim != ""
	if prefixEilat && !CitySubstringMatchesToken(NormalizeCityLookupKey(afterTrim), "אילת") {
		return true
	}

	addrKey := NormalizeCityLookupKey(addrTrim)
	if CitySubstringMatchesToken(addrKey, "אילת") {
		return true
	}
	_ = district
	return false
}
