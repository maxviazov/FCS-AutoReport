package domain

import (
	"time"
)

// NormalizeDateString приводит дату из сырого отчёта к формату DD/MM/YYYY для шаблона Минздрава.
// Поддерживает: "10/03/26", "10/03/2026". При ошибке парсинга возвращает исходную строку.
func NormalizeDateString(s string) string {
	s = NormalizeText(s)
	if s == "" {
		return s
	}
	for _, layout := range []string{"02/01/06", "2/1/06", "02/01/2006", "2/1/2006", "02.01.2006", "2.1.2006"} {
		t, err := time.Parse(layout, s)
		if err == nil {
			return t.Format("02.01.2006") // как в образце: 08.03.2026
		}
	}
	return s
}
