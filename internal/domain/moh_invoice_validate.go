package domain

import (
	"fmt"
	"math"
	"regexp"
	"strings"
)

// MoH публикует коды вида N126, N61, N610: одна латинская буква в верхнем регистре + 2–4 цифры (как в выгрузке реестра, без подмены регистра).
var mohCityCodePattern = regexp.MustCompile(`^[A-Z]\d{2,4}$`)

// CanonicalMoHCityCode приводит код из Excel/БД к виду для проверки и экспорта (латинская буква в верхнем регистре).
func CanonicalMoHCityCode(code string) string {
	return strings.TrimSpace(strings.ToUpper(strings.TrimSpace(code)))
}

// IsMoHCityCodeFormat true, если строка в точности совпадает с форматом кода в реестре МОЗ.
func IsMoHCityCodeFormat(code string) bool {
	return mohCityCodePattern.MatchString(CanonicalMoHCityCode(code))
}

// ClientHPDigits возвращает только цифры из ח"פ.
func ClientHPDigits(hp string) string {
	var b strings.Builder
	for _, r := range strings.TrimSpace(hp) {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// ContainsCyrillic true, если в строке есть кириллица (в реестре МОЗ обычно иврит).
func ContainsCyrillic(s string) bool {
	for _, r := range s {
		if r >= '\u0400' && r <= '\u04FF' {
			return true
		}
	}
	return false
}

// MoHAddressForReport — значение для колонки «כתובת»: адрес из SAP или fallback из שם לקוח; город убираем по правилам עיר,רחוב / רחוב,עיר (קוד в кол. J).
func MoHAddressForReport(addr, clientName string, cityAfterComma bool) string {
	addr = NormalizeText(addr)
	var out string
	if addr != "" {
		out = NormalizeMinistryAddress(addr)
	} else {
		clientName = NormalizeText(clientName)
		if clientName == "" {
			return ""
		}
		if _, after, ok := strings.Cut(clientName, " - "); ok {
			after = strings.TrimSpace(after)
			if after != "" {
				out = NormalizeMinistryAddress(after)
			}
		}
		if out == "" {
			out = NormalizeMinistryAddress(clientName)
		}
	}
	return MoHStreetLineForMoH(out, cityAfterComma)
}

// RoundWeightKg округляет вес до 2 знаков (как в экспорте).
func RoundWeightKg(kg float64) float64 {
	return math.Round(kg*100) / 100
}

// ValidateInvoiceForMoHExport возвращает список нарушений; пустой слайс — накладная годна к экспорту.
func ValidateInvoiceForMoHExport(inv *AggregatedInvoice) []string {
	if inv == nil {
		return []string{"внутренняя ошибка: пустая накладная"}
	}
	var w []string
	add := func(msg string) { w = append(w, msg) }

	for _, e := range inv.Errors {
		add(e)
	}
	if strings.TrimSpace(inv.InvoiceNum) == "" {
		add("пустой номер накладной (מספר תעודת משלוח)")
	}
	if strings.TrimSpace(inv.Date) == "" {
		add("пустая дата (תאריך)")
	}
	if strings.TrimSpace(inv.ClientName) == "" {
		add("пустое имя клиента (לקוח)")
	}
	code := strings.TrimSpace(inv.CityCode)
	if code == "" {
		add("пустой код города (קוד עיר)")
	} else if !IsMoHCityCodeFormat(code) {
		add(fmt.Sprintf("код города %q не в формате МОЗ (латинская буква и 2–4 цифры)", code))
	}
	addr := MoHAddressForReport(inv.Address, inv.ClientName, inv.MoHCityAfterComma)
	if strings.TrimSpace(addr) == "" {
		add("пустой адрес (כתובת): заполните адрес в сыром файле или имя клиента для подстановки")
	}
	d := ClientHPDigits(inv.ClientHP)
	if len(d) < 8 || len(d) > 9 {
		add(fmt.Sprintf("ח\"פ: нужно 8–9 цифр, сейчас %d", len(d)))
	}
	if strings.TrimSpace(inv.DriverName) == "" {
		add("не назначен водитель (שם הנהג) для кода города")
	}
	if strings.TrimSpace(inv.CarNumber) == "" {
		add("пустой номер ТС (מס.רכב)")
	}
	if strings.TrimSpace(inv.Phone) == "" {
		add("пустой телефон водителя (טלפון נהג)")
	}
	var tw float64
	for cat, kg := range inv.Weights {
		if kg < 0 {
			add(fmt.Sprintf("отрицательный вес по категории %q: %.3f кг", cat, kg))
			continue
		}
		tw += kg
	}
	if inv.TotalBoxes < 0 {
		add("отрицательное количество אריזות (קרטונים)")
	}
	if RoundWeightKg(tw) <= 0 {
		add("нулевой суммарный вес по категориям МОЗ")
	}
	if ContainsCyrillic(inv.ClientName) || ContainsCyrillic(addr) {
		add("кириллица в имени клиента или адресе — в отчёте МОЗ ожидается иврит")
	}
	return w
}
