package domain

import "strings"

// cyrillicDistrictToHebrew — колонка «מחוז» в FishKA часто на русском; сопоставление со справочником городов (иврит).
var cyrillicDistrictToHebrew = map[string]string{
	"холон":     "חולון",
	"хайфа":     "חיפה",
	"тель-авив": "תל אביב",
	"тель авив": "תל אביב",
	"иерусалим": "ירושלים",
	"бат-ям":    "בת ים",
	"бат ям":    "בת ים",
	"рамат-ган": "רמת גן",
	"рамат ган": "רמת גן",
	"бней брак": "בני ברק",
	"бней-брак": "בני ברק",
	"ашдод":     "אשדוד",
	"ашкелон":   "אשקלון",
	"беэр шева": "באר שבע",
	"беэр-שева": "באר שבע",
	"нетания":   "נתניה",
	"герцлия":   "הרצליה",
	"рамат ха-шарон": "רמת השרון",
	"кфар саба": "כפר סבא",
	"петах тиква": "פתח תקווה",
	"рейховот":  "רחובות",
	"ришон":    "ראשון לציון",
	"модиин":   "מודיעין",
	"эйлат":    "אילת",
	"ейлат":    "אילת",
	"эилат":    "אילת",
	"хадера":   "חדרה",
	"хадера+":  "חדרה",
}

// HebrewCityHintFromDistrictLabel возвращает ивритское название города для поиска в справочнике, если в «מחуз» указан русский ярлык.
func HebrewCityHintFromDistrictLabel(s string) (hebrew string, ok bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", false
	}
	if !ContainsCyrillic(s) {
		return "", false
	}
	key := NormalizeText(strings.ToLower(s))
	if key == "" {
		return "", false
	}
	if h, found := cyrillicDistrictToHebrew[key]; found && h != "" {
		return h, true
	}
	return "", false
}
