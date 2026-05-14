package app

import (
	"fmt"
	"log/slog"
	"math"
	"strconv"
	"strings"

	"fcs-autoreport/internal/domain"

	"github.com/xuri/excelize/v2"
)

// parseFloat безопасно переводит строку из Excel в float64 (пустое или нечисловое → 0).
func parseFloat(s string) float64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	val, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return val
}

// ProcessRawReport читает сырой Excel, обогащает данными из Store и агрегирует по накладным.
// Возвращает слайс накладных с заполненными Weights (кг), TotalBoxes и списком Errors при проблемах.
func (s *ReportService) ProcessRawReport(rawFilePath string) ([]*domain.AggregatedInvoice, error) {
	slog.Info("Агрегация сырого отчёта", "file", rawFilePath)

	f, err := excelize.OpenFile(rawFilePath)
	if err != nil {
		return nil, fmt.Errorf("открытие сырого отчёта: %w", err)
	}
	defer f.Close()

	sheetName := f.GetSheetName(0)
	if sheetName == "" {
		return nil, fmt.Errorf("в файле нет листов")
	}

	rows, err := f.GetRows(sheetName)
	if err != nil {
		return nil, fmt.Errorf("чтение строк листа: %w", err)
	}

	// Колонки: "שם לועזי" до "כתובת" — клиент, после "קוד פריט" — товар. Берём только до "כתובת".
	clientNameCol := -1
	addressCol := -1
	rawCityNameCol := -1 // явное название города в сыром отчёте (עיר / שם עיר / ישוב)
	itemCodeCol := -1
	districtCol := -1
	weightCol := 25   // סה"כ משקל или משקל
	boxesCol := 23
	var headerRow int
findHeader:
	for hi := 0; hi < len(rows) && hi < 5; hi++ {
		for _, c := range rows[hi] {
			if domain.NormalizeText(c) == "כתובת" {
				headerRow = hi
				break findHeader
			}
		}
	}
	if len(rows) > headerRow {
		header := rows[headerRow]
		slog.Info("Сырой отчёт: строка заголовков", "row_index", headerRow)
		for j, cell := range header {
			n := domain.NormalizeText(cell)
			if n == "כתובת" {
				addressCol = j
			}
			if n == "מחוז" || (strings.Contains(n, "מחוז") && !strings.Contains(n, "קוד")) {
				districtCol = j
			}
			if itemCodeCol < 0 && (n == "קוד פריט" || (strings.Contains(n, "קוד") && strings.Contains(n, "פריט"))) {
				itemCodeCol = j
			}
			if rawCityNameCol < 0 {
				switch n {
				case "עיר", "שם עיר", "שם העיר", "ישוב":
					rawCityNameCol = j
				default:
					if strings.Contains(n, "עיר") && !strings.Contains(n, "קוד") {
						rawCityNameCol = j
					}
				}
			}
			if strings.Contains(n, "משקל") {
				weightCol = j
			}
			if strings.Contains(n, "קרטון") || strings.Contains(n, "אריזות") {
				boxesCol = j
			}
		}
		// Оригинальное название клиента: колонка "שם לועזי" сразу перед "כתובת" и до "קוד פריט".
		if addressCol > 0 && addressCol <= len(header) {
			idxBeforeAddr := addressCol - 1
			prev := domain.NormalizeText(header[idxBeforeAddr])
			isLoazi := prev == "שם לועזי" || strings.Contains(prev, "לועזי")
			beforeItemCode := itemCodeCol < 0 || idxBeforeAddr < itemCodeCol
			if isLoazi && beforeItemCode {
				clientNameCol = idxBeforeAddr
			}
		}
		if clientNameCol < 0 && len(header) > 11 && (itemCodeCol < 0 || 11 < itemCodeCol) {
			n := domain.NormalizeText(header[11])
			if n == "שם לועזי" || strings.Contains(n, "לועזי") {
				clientNameCol = 11
			}
		}
		slog.Info("Сырой отчёт: колонка имени клиента", "header_row", headerRow, "clientNameCol", clientNameCol, "rawCityNameCol", rawCityNameCol, "addressCol", addressCol, "districtCol", districtCol, "itemCodeCol", itemCodeCol)
	}

	invoiceMap := make(map[string]*domain.AggregatedInvoice)

	for i, row := range rows {
		if i <= headerRow {
			continue // заголовки и пустые строки до них
		}

		getCol := func(idx int) string {
			if idx < len(row) {
				return row[idx]
			}
			return ""
		}

		invoiceNum := domain.NormalizeText(getCol(4)) // אסמכתת בסיס
		if invoiceNum == "" {
			continue
		}

		date := domain.NormalizeDateString(domain.NormalizeText(getCol(5)))
		hp := domain.NormalizeText(getCol(9))
		var clientNameRaw string
		if clientNameCol >= 0 {
			clientNameRaw = domain.NormalizeText(getCol(clientNameCol))
		}
		var rawAddress string
		if addressCol >= 0 {
			rawAddress = domain.NormalizeText(getCol(addressCol))
		} else {
			rawAddress = domain.NormalizeText(getCol(12))
		}
		var districtRaw string
		if districtCol >= 0 {
			districtRaw = getCol(districtCol)
		}
		rawCityNameCell := ""
		if rawCityNameCol >= 0 {
			rawCityNameCell = domain.NormalizeText(getCol(rawCityNameCol))
		}
		itemCode := domain.NormalizeText(getCol(15))
		boxesStr := getCol(boxesCol)
		weightStr := getCol(weightCol)

		// 1. Парсим сырой вес (в сыром файле может быть в граммах или в декаграммах: 1400→1.4 кг, 20→0.2 кг)
		rawWeight := parseFloat(weightStr)
		if rawWeight <= 0 {
			continue
		}
		// Граммы → кг: значение >= 100 считаем граммами (/1000). Иначе — декаграммы, 10г (/100). Уже < 1 — считаем кг.
		var weightKg float64
		if rawWeight >= 100 {
			weightKg = rawWeight / 1000.0
		} else if rawWeight >= 1 {
			weightKg = rawWeight / 100.0 // 20 декаграмм → 0.2 кг
		} else {
			weightKg = rawWeight
		}
		weightKg = math.Round(weightKg*1000) / 1000
		if weightKg < 0 {
			continue
		}

		boxes := parseFloat(boxesStr)
		if boxes < 0 {
			boxes = 0
		}

		inv, exists := invoiceMap[invoiceNum]
		if !exists {
			// Имя клиента — только из сырого отчёта (колонка שם לועזי)
			slog.Info("Сырой отчёт: название клиента",
				"invoice", invoiceNum,
				"col_index", clientNameCol,
				"client_name", clientNameRaw)
			inv = &domain.AggregatedInvoice{
				InvoiceNum: invoiceNum,
				Date:       date,
				ClientName: clientNameRaw,
				ClientHP:   hp,
				Address:    rawAddress,
				Weights:    make(map[string]float64),
				Errors:     nil,
			}

			// Код города: часть до запятой → точное совпадение; снятие префикса; поиск по подстроке; затем клиент по HP.
			// Берём только коды вида «лат. буква + 2–4 цифры» (в т.ч. N61, N610 из реестра МОЗ).
			trySetCityCode := func(code string) {
				if inv.CityCode != "" {
					return
				}
				c := domain.CanonicalMoHCityCode(code)
				if c == "" {
					return
				}
				if domain.IsMoHCityCodeFormat(c) {
					if c == "N61" && !domain.AllowMoHN61CityCode(rawAddress, clientNameRaw, districtRaw, rawCityNameCell) {
						slog.Warn("N61 без контекста אילת — пропускаем код города", "invoice", invoiceNum, "address", rawAddress, "client", clientNameRaw, "raw_city_col", rawCityNameCell)
						return
					}
					inv.CityCode = c
					return
				}
				slog.Warn("Код города не в формате МОЗ — не используем", "invoice", invoiceNum, "code", c, "address", rawAddress, "client", clientNameRaw)
			}

			// 0) Название города из колонки «עיר» / «ישוב» сырого отчёта — приоритет над כתובת и прочими эвристиками.
			if rawCityNameCell != "" {
				if code, _ := s.store.ResolveCityCode(rawCityNameCell); code != "" {
					trySetCityCode(code)
				}
				if inv.CityCode == "" {
					st := domain.StripCityPrefix(rawCityNameCell)
					if code, _ := s.store.ResolveCityCode(st); code != "" {
						trySetCityCode(code)
					}
				}
				if inv.CityCode == "" {
					key := domain.NormalizeCityLookupKey(rawCityNameCell)
					if code, _ := s.store.ResolveCityCodeBySubstring(key); code != "" {
						trySetCityCode(code)
					}
				}
			}

			cityStr := domain.ExtractCityFromAddress(rawAddress)
			if cityStr != "" {
				if code, _ := s.store.ResolveCityCode(cityStr); code != "" {
					trySetCityCode(code)
				}
				if inv.CityCode == "" {
					stripped := domain.StripCityPrefix(cityStr)
					if code, _ := s.store.ResolveCityCode(stripped); code != "" {
						trySetCityCode(code)
					}
				}
				if inv.CityCode == "" {
					if code, _ := s.store.ResolveCityCodeBySubstring(cityStr); code != "" {
						trySetCityCode(code)
					}
				}
			}
			// «רחוב, עיר» — город после первой запятой (напр. סוקולוב 63, חולון).
			if inv.CityCode == "" && rawAddress != "" {
				if _, after, ok := strings.Cut(rawAddress, ","); ok {
					afterPart := domain.NormalizeText(strings.TrimSpace(after))
					if afterPart != "" {
						suffixMatched := false
						if code, _ := s.store.ResolveCityCode(afterPart); code != "" {
							trySetCityCode(code)
							suffixMatched = true
						}
						if inv.CityCode == "" {
							st := domain.StripCityPrefix(afterPart)
							if st != afterPart {
								if code, _ := s.store.ResolveCityCode(st); code != "" {
									trySetCityCode(code)
									suffixMatched = true
								}
							}
						}
						if inv.CityCode == "" {
							if code, _ := s.store.ResolveCityCodeBySubstring(afterPart); code != "" {
								trySetCityCode(code)
								suffixMatched = true
							}
						}
						if inv.CityCode != "" && suffixMatched {
							inv.MoHCityAfterComma = true
						}
					}
				}
			}
			// Полная כתובת: подстрочное совпадение с длинными алиасами (עיר, רחוב).
			if inv.CityCode == "" && rawAddress != "" {
				fullKey := domain.NormalizeCityLookupKey(rawAddress)
				if code, _ := s.store.ResolveCityCodeBySubstring(fullKey); code != "" {
					trySetCityCode(code)
				}
			}
			// כתובת + שם לועזי: город может быть только в названии клиента, адрес — только улица.
			if inv.CityCode == "" && rawAddress != "" && clientNameRaw != "" {
				combo := domain.NormalizeCityLookupKey(strings.TrimSpace(rawAddress + " " + clientNameRaw))
				if code, _ := s.store.ResolveCityCodeBySubstring(combo); code != "" {
					trySetCityCode(code)
				}
			}
			// שם לועזי: город внутри длинного названия (טיב טעם חולון 20); при пустой כתובת — ещё и точное совпадение.
			if inv.CityCode == "" && clientNameRaw != "" {
				lookup := domain.NormalizeCityLookupKey(clientNameRaw)
				if strings.TrimSpace(rawAddress) == "" {
					if code, _ := s.store.ResolveCityCode(lookup); code != "" {
						trySetCityCode(code)
					}
				}
				if inv.CityCode == "" {
					if code, _ := s.store.ResolveCityCodeBySubstring(lookup); code != "" {
						trySetCityCode(code)
					}
				}
			}
			// מחוז: русское название округа — после כתובת и שם לועזי.
			if inv.CityCode == "" && strings.TrimSpace(districtRaw) != "" {
				dNorm := domain.NormalizeText(districtRaw)
				if code, _ := s.store.ResolveCityCode(dNorm); code != "" {
					trySetCityCode(code)
				}
				if inv.CityCode == "" {
					if code, _ := s.store.ResolveCityCodeBySubstring(domain.NormalizeCityLookupKey(dNorm)); code != "" {
						trySetCityCode(code)
					}
				}
				if inv.CityCode == "" {
					if heb, ok := domain.HebrewCityHintFromDistrictLabel(districtRaw); ok {
						if code, _ := s.store.ResolveCityCode(heb); code != "" {
							trySetCityCode(code)
						}
					}
				}
			}
			if inv.CityCode == "" && hp != "" {
				if c := s.store.GetClient(hp); c != nil && c.CityCode != "" {
					trySetCityCode(c.CityCode)
				}
			}
			if inv.CityCode == "" {
				if rawCityNameCell != "" {
					inv.Errors = append(inv.Errors, fmt.Sprintf("Город из колонки сырого отчёта не найден в справочнике: %s", rawCityNameCell))
				} else if cityStr != "" {
					inv.Errors = append(inv.Errors, fmt.Sprintf("Город не найден: %s (Адрес: %s)", cityStr, rawAddress))
				}
				inv.Errors = append(inv.Errors, "Нет кода города: укажите адрес с городом или добавьте клиента с кодом города в справочник")
			}

			if inv.CityCode != "" && rawAddress != "" {
				if domain.InferCityPlacedAfterComma(domain.NormalizeMinistryAddress(rawAddress)) {
					inv.MoHCityAfterComma = true
				}
			}

			// Водителя подставляем по городу доставки (в сыром отчёте колонки водителей нет)
			driver := s.store.GetDriverForCity(inv.CityCode)
			if driver != nil {
				inv.DriverName = driver.DriverName
				inv.CarNumber = driver.CarNumber
				inv.Phone = driver.Phone
			} else if inv.CityCode != "" {
				inv.Errors = append(inv.Errors, fmt.Sprintf("Для города %s не назначен водитель в справочнике", inv.CityCode))
			}

			invoiceMap[invoiceNum] = inv
		}

		inv.TotalBoxes += boxes

		item := s.store.GetItem(itemCode)
		if item == nil {
			if itemCode != "" {
				// Не блокируем экспорт: вес уходит в UNKNOWN (кол. «נוסף א»); в справочник товаров можно добавить позже.
				slog.Warn("Артикул не найден в справочнике — вес в отчёте как нераспределённый",
					"invoice", invoiceNum, "item_code", itemCode, "kg", weightKg)
			}
			inv.Weights["UNKNOWN"] += weightKg
		} else {
			catKey := domain.NormalizeText(item.Category)
			if catKey == "" {
				inv.Weights["UNKNOWN"] += weightKg
			} else {
				inv.Weights[catKey] += weightKg
			}
		}
	}

	result := make([]*domain.AggregatedInvoice, 0, len(invoiceMap))
	for _, inv := range invoiceMap {
		result = append(result, inv)
	}

	slog.Info("Агрегация завершена", "unique_invoices", len(result))
	return result, nil
}
