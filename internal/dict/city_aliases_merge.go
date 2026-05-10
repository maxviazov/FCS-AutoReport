package dict

import (
	"fmt"
	"log/slog"

	"fcs-autoreport/internal/db"
	"fcs-autoreport/internal/domain"
)

// Встроенные алиасы: каноническое имя города (как в импорте МОЗ) → варианты из сырых отчётов.
var builtinCityAliases = []struct {
	CanonicalName string
	Aliases       []string
}{
	{"תל אביב יפו", []string{"תל אביב"}},
	{"ראשון לציון", []string{`ראשל"צ`, "ראשל״צ"}},
	{"קריית טבעון", []string{"קרית טבעון"}},
}

// reservedAliasOwners — алиас (после NormalizeText) может принадлежать только этому каноническому name.
// Используется, чтобы убрать ошибочные привязки (напр. קרית טבעון у נצרת).
func reservedAliasOwners() map[string]string {
	m := make(map[string]string)
	for _, rule := range builtinCityAliases {
		for _, a := range rule.Aliases {
			k := domain.NormalizeText(a)
			if k != "" {
				m[k] = rule.CanonicalName
			}
		}
	}
	return m
}

// MergeBuiltinCityAliases добавляет встроенные алиасы и снимает их с «чужих» городов (идемпотентно).
func MergeBuiltinCityAliases(database *db.DB) error {
	// 1) Добавить алиасы к каноническим городам
	for _, rule := range builtinCityAliases {
		city, err := database.GetCityByName(rule.CanonicalName)
		if err != nil {
			return fmt.Errorf("город %q: %w", rule.CanonicalName, err)
		}
		if city == nil {
			slog.Warn("Встроенный алиас: город не найден в БД (пропуск)", "name", rule.CanonicalName)
			continue
		}
		seen := make(map[string]bool)
		for _, a := range city.Aliases {
			seen[domain.NormalizeText(a)] = true
		}
		changed := false
		for _, a := range rule.Aliases {
			k := domain.NormalizeText(a)
			if k == "" || seen[k] {
				continue
			}
			city.Aliases = append(city.Aliases, k)
			seen[k] = true
			changed = true
		}
		if changed {
			if err := database.UpdateCity(city); err != nil {
				return fmt.Errorf("обновление города %q: %w", rule.CanonicalName, err)
			}
			slog.Info("Встроенные алиасы: обновлён город", "name", rule.CanonicalName)
		}
	}

	// 2) Снять зарезервированные алиасы с других городов
	owner := reservedAliasOwners()
	list, err := database.ListCities()
	if err != nil {
		return err
	}
	for i := range list {
		c := &list[i]
		filtered := c.Aliases[:0]
		removedAny := false
		for _, a := range c.Aliases {
			k := domain.NormalizeText(a)
			wantOwner, reserved := owner[k]
			if reserved && c.Name != wantOwner {
				removedAny = true
				slog.Info("Встроенные алиасы: снят алиас с чужого города",
					"alias", a, "from_city", c.Name, "belongs_to", wantOwner)
				continue
			}
			filtered = append(filtered, a)
		}
		if removedAny {
			c.Aliases = filtered
			if err := database.UpdateCity(c); err != nil {
				return fmt.Errorf("обновление города %q (очистка алиасов): %w", c.Name, err)
			}
		}
	}

	return nil
}
