package domain

import (
	"strings"
	"unicode"
)

// NormalizeText очищает строку для сопоставления: обрезка пробелов, схлопывание повторяющихся пробелов.
// Учитывает иврит: нормализация пробелов и границы слов (RTL не меняем — сравнение побайтово после нормализации).
func NormalizeText(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return s
	}
	// Схлопываем множественные пробелы и прочие пробельные символы в один пробел
	var b strings.Builder
	b.Grow(len(s))
	prevSpace := false
	for _, r := range s {
		if unicode.IsSpace(r) {
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
