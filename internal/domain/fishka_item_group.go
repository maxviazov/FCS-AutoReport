package domain

import "strings"

// MoHCategoryFromFishKAGroup — категория МОЗ по «שם קבוצה» / описанию из FishKA, если артикула нет в справочнике.
func MoHCategoryFromFishKAGroup(groupName, itemDesc string) string {
	g := NormalizeText(groupName)
	d := NormalizeText(itemDesc)
	combo := g + " " + d
	for _, rule := range fishkaGroupRules {
		if rule.match(g, d, combo) {
			return rule.category
		}
	}
	return ""
}

type fishkaGroupRule struct {
	category string
	match    func(group, desc, combo string) bool
}

var fishkaGroupRules = []fishkaGroupRule{
	{
		category: "דגים מעובדים",
		match: func(g, d, combo string) bool {
			return containsAny(combo, "דג", "fish", "arena", "סלמון", "טונה", "אינטיאס", "הרינג", "מקרל", "דניס")
		},
	},
	{
		category: "דגים גולמי מקומי",
		match: func(g, d, combo string) bool {
			return containsAny(combo, "גולמי", "טרי", "חי")
		},
	},
	{
		category: "עוף מעובד",
		match: func(_, _, combo string) bool {
			return containsAny(combo, "עוף", "הודו", "chicken")
		},
	},
	{
		category: "בשר בהמות מעובד",
		match: func(_, _, combo string) bool {
			return containsAny(combo, "בשר", "beef")
		},
	},
	{
		category: "מוצרים מוכנים לאכילה",
		match: func(_, _, combo string) bool {
			return containsAny(combo, "מוכן", "סלט", "מעדן")
		},
	},
}

func containsAny(s string, subs ...string) bool {
	s = strings.ToLower(s)
	for _, sub := range subs {
		if sub == "" {
			continue
		}
		if strings.Contains(s, strings.ToLower(sub)) {
			return true
		}
	}
	return false
}
