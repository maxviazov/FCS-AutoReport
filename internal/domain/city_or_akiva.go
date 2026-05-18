package domain

import "strings"

// AdjustCityFromFishKA — правки типичных ошибок SAP/FishKA перед поиском קוד עיר.
// «אורות, רחוב…» при מחוז חדרה+ часто означает אור עקיבא (F1373), а не יישוב אורות (M37).
func AdjustCityFromFishKA(cityFromAddr, district string) string {
	cityFromAddr = strings.TrimSpace(cityFromAddr)
	if cityFromAddr != "אורות" {
		return cityFromAddr
	}
	if districtSuggestsOrAkiva(district) {
		return "אור עקיבא"
	}
	return cityFromAddr
}

func districtSuggestsOrAkiva(district string) bool {
	district = strings.TrimSpace(district)
	if district == "" {
		return false
	}
	if heb, ok := HebrewCityHintFromDistrictLabel(district); ok {
		switch NormalizeText(heb) {
		case "חדרה", "אור עקיבא", "קיסריה", "זיכרון יעקב":
			return true
		}
	}
	n := strings.ToLower(NormalizeText(district))
	return strings.Contains(n, "хадер") || strings.Contains(n, "hadera") ||
		strings.Contains(n, "חדרה") || strings.Contains(n, "עקיבא")
}
