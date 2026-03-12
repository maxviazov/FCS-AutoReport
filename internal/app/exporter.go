package app

import (
	"fmt"
	"log/slog"
	"math"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"fcs-autoreport/internal/domain"

	"github.com/xuri/excelize/v2"
)

// CategoryColumnMap связывает название категории Минздрава с номером столбца (1-based) в шаблоне.
// Добавлены варианты из шаблона (מקומי/יבוא и "אחר").
var CategoryColumnMap = map[string]int{
	"בשר בהמות גולמי":          15,
	"בשר בהמות גולמי מקומי":   15,
	"בשר בהמות מיבוא קפוא":    16,
	"בשר בהמות גולמי יבוא":    16,
	"בשר בהמות מעובד":         17,
	"עוף גולמי (עוף שחוט)":    18,
	"עוף מעובד":               19,
	"דגים גולמי (מקומי)":      20,
	"דגים גולמי מקומי":        20,
	"דגים יבוא":               21,
	"דגים גולמי יבוא":        21,
	"דגים מעובדים":            22,
	"מוצרים מוכנים לאכילה":    23,
	"נוסף א":                  24,
	"אחר (דגים חיים, חלב, ביצים)": 24,
	"אחר":                     24,
	"נוסף ב":                  25,
}

// normalizedCategoryToColumn — поиск колонки по нормализованному названию категории (без опечаток/пробелов).
var normalizedCategoryToColumn map[string]int

func init() {
	normalizedCategoryToColumn = make(map[string]int, len(CategoryColumnMap))
	for name, col := range CategoryColumnMap {
		normalizedCategoryToColumn[domain.NormalizeText(name)] = col
	}
}

// getCategoryColumn возвращает номер колонки (1-based) для категории; 0 если не найдено.
func getCategoryColumn(category string) int {
	key := domain.NormalizeText(category)
	if key == "" {
		return 0
	}
	return normalizedCategoryToColumn[key]
}

// roundWeight округляет вес до 2 знаков после запятой (убирает float-хвосты).
func roundWeight(kg float64) float64 {
	return math.Round(kg*100) / 100
}

// Данные компании-поставщика (при необходимости вынести в domain.Settings).
const (
	SupplierName = "דולינה גרופ בע\"מ"
	SupplierHP   = "511777856"
	MoHNumber    = "P1908"
)

// mohDriverCols — номера колонок (1-based) в шаблоне Минздрава: машина, имя водителя, телефон.
type mohDriverCols struct {
	Vehicle, DriverName, Phone int
}

// detectClientColumn возвращает номер колонки (1-based) с заголовком "לקוח" (имя клиента). Не "סוג לקוח".
func detectClientColumn(f *excelize.File, sheet string) int {
	rows, err := f.GetRows(sheet)
	if err != nil || len(rows) < 1 {
		return 8
	}
	for j, cell := range rows[0] {
		n := domain.NormalizeText(cell)
		if n == "" {
			continue
		}
		if n == "לקוח" {
			return j + 1
		}
		if strings.Contains(n, "לקוח") && !strings.Contains(n, "סוג") {
			return j + 1
		}
	}
	return 8
}

// detectDriverColumns читает первую строку шаблона и находит колонки по заголовкам (מס.רכב, שם הנהג, טלפון נהג).
func detectDriverColumns(f *excelize.File, sheet string) mohDriverCols {
	out := mohDriverCols{Vehicle: 5, DriverName: 6, Phone: 7}
	rows, err := f.GetRows(sheet)
	if err != nil || len(rows) < 1 {
		return out
	}
	header := rows[0]
	var phoneFromDriver bool
	for j, cell := range header {
		col := j + 1
		n := domain.NormalizeText(cell)
		if n == "" {
			continue
		}
		// מס.רכב или מס רכב
		if strings.Contains(n, "רכב") && (strings.Contains(n, "מס") || strings.Contains(n, "מספר")) {
			out.Vehicle = col
		}
		if strings.Contains(n, "נהג") && strings.Contains(n, "שם") {
			out.DriverName = col
		}
		if strings.Contains(n, "טלפון") {
			if strings.Contains(n, "נהג") {
				out.Phone = col
				phoneFromDriver = true
			} else if !phoneFromDriver {
				out.Phone = col
			}
		}
	}
	return out
}

// ExportToExcel заполняет шаблон Минздрава агрегированными накладными и сохраняет в outputDir.
// Соответствует требованиям שידורים למערכת הממוחשבת (апрель 2024): колонки A–AB, формат xlsx.
// Возвращает путь к сохранённому файлу.
func ExportToExcel(invoices []*domain.AggregatedInvoice, templatePath, outputDir string) (string, error) {
	slog.Info("Экспорт в шаблон Минздрава", "invoices_count", len(invoices))

	f, err := excelize.OpenFile(templatePath)
	if err != nil {
		return "", fmt.Errorf("открытие шаблона %s: %w", templatePath, err)
	}
	defer f.Close()

	sheetName := f.GetSheetName(0)
	if sheetName == "" {
		return "", fmt.Errorf("в шаблоне нет листов")
	}

	driverCols := detectDriverColumns(f, sheetName)
	clientCol := detectClientColumn(f, sheetName)

	currentRow := 2

	for _, inv := range invoices {
		if inv.CityCode == "" {
			slog.Warn("Накладная без кода города — выводится с пустыми полями города/водителя", "invoice", inv.InvoiceNum, "client", inv.ClientName)
		}
		setCell(f, sheetName, 1, currentRow, SupplierName)
		setCell(f, sheetName, 2, currentRow, 511777856) // ח"פ ספק — число, не текст
		setCell(f, sheetName, 3, currentRow, MoHNumber)
		setCell(f, sheetName, 4, currentRow, inv.Date)
		setCell(f, sheetName, driverCols.Vehicle, currentRow, inv.CarNumber)
		setCell(f, sheetName, driverCols.DriverName, currentRow, inv.DriverName)
		setCell(f, sheetName, driverCols.Phone, currentRow, inv.Phone)
		slog.Info("Итоговый отчёт: в ячейку לקוח",
			"invoice", inv.InvoiceNum,
			"row", currentRow,
			"col", clientCol,
			"client_name", inv.ClientName)
		setCell(f, sheetName, clientCol, currentRow, inv.ClientName)
		setCell(f, sheetName, 9, currentRow, "קמעונאי")
		// Код города и водитель могут быть пустыми, если адрес не распознан — строка всё равно выводится
		setCell(f, sheetName, 10, currentRow, inv.CityCode)
		setCell(f, sheetName, 11, currentRow, streetFromAddress(inv.Address))
		setCell(f, sheetName, 12, currentRow, numericOrString(inv.ClientHP))
		setCell(f, sheetName, 13, currentRow, 0)
		setCell(f, sheetName, 14, currentRow, numericOrString(inv.InvoiceNum)) // מספר תעודת משלוח — число, не текст

		// Обязательные 0 в колонках категорий (15–25): как в образце Минздрава, пустых ячеек не оставляем
		for col := 15; col <= 25; col++ {
			setCell(f, sheetName, col, currentRow, 0)
		}
		var totalWeight float64
		colWeights := make(map[int]float64)
		for category, weight := range inv.Weights {
			if category == "UNKNOWN" {
				slog.Warn("Нераспределённый вес", "invoice", inv.InvoiceNum, "weight_kg", weight)
			}
			totalWeight += weight
			colIdx := getCategoryColumn(category)
			if colIdx <= 0 && category == "UNKNOWN" {
				colIdx = 24 // אחר (דגים חיים, חלב, ביצים)
			}
			if colIdx > 0 {
				colWeights[colIdx] += weight
			}
		}
		for colIdx, w := range colWeights {
			setCell(f, sheetName, colIdx, currentRow, roundWeight(w)) // вес уже в кг (г→кг в агрегаторе)
		}

		setCell(f, sheetName, 26, currentRow, roundWeight(inv.TotalBoxes))
		setCell(f, sheetName, 27, currentRow, roundWeight(totalWeight))
		setCell(f, sheetName, 28, currentRow, 1)

		currentRow++
	}

	// --- 100% форматирование как в образце: копируем стиль строки 2 шаблона на все строки данных ---
	lastRow := currentRow - 1
	if lastRow >= 2 {
		styleID, err := f.GetCellStyle(sheetName, "A2")
		if err != nil {
			slog.Warn("Не удалось прочитать стиль строки 2 шаблона", "err", err)
		} else {
			startCell, _ := excelize.CoordinatesToCellName(1, 2)
			endCell, _ := excelize.CoordinatesToCellName(28, lastRow)
			if err := f.SetCellStyle(sheetName, startCell, endCell, styleID); err != nil {
				slog.Warn("Применение стиля шаблона к диапазону данных", "err", err)
			}
		}
	}

	fileName := fmt.Sprintf("FCS_Report_%s.xlsx", time.Now().Format("2006-01-02_15-04-05"))
	savePath := filepath.Join(outputDir, fileName)

	if err := f.SaveAs(savePath); err != nil {
		return "", fmt.Errorf("сохранение отчёта: %w", err)
	}

	slog.Info("Отчёт сохранён", "path", savePath)
	return savePath, nil
}

// numericOrString возвращает число для ячейки, если строка — число (убирает «число как текст» в Excel).
func numericOrString(s string) interface{} {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	if i, err := strconv.ParseInt(s, 10, 64); err == nil {
		return i
	}
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return roundWeight(f)
	}
	return s
}

// streetFromAddress возвращает часть адреса после первой запятой (улица без города) или весь адрес, если запятой нет.
func streetFromAddress(addr string) string {
	_, after, ok := strings.Cut(strings.TrimSpace(addr), ",")
	if !ok {
		return strings.TrimSpace(addr)
	}
	return strings.TrimSpace(after)
}

// applyDataFormatting задаёт числовой формат для колонок с числами (с теми же бордюрами и выравниванием, чтобы не затереть стиль).
func applyDataFormatting(f *excelize.File, sheet string, firstRow, lastRow int) {
	numStyle, err := f.NewStyle(&excelize.Style{
		NumFmt: 1,
		Border: []excelize.Border{
			{Type: "left", Color: "000000", Style: 1},
			{Type: "top", Color: "000000", Style: 1},
			{Type: "bottom", Color: "000000", Style: 1},
			{Type: "right", Color: "000000", Style: 1},
		},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})
	if err != nil {
		return
	}
	numCols := []int{2, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28}
	for _, col := range numCols {
		start, _ := excelize.CoordinatesToCellName(col, firstRow)
		end, _ := excelize.CoordinatesToCellName(col, lastRow)
		_ = f.SetCellStyle(sheet, start, end, numStyle)
	}
}

// setCell записывает value в ячейку по номерам колонки и строки (1-based).
func setCell(f *excelize.File, sheet string, col, row int, value interface{}) {
	cellName, err := excelize.CoordinatesToCellName(col, row)
	if err != nil {
		slog.Error("Координаты ячейки", "col", col, "row", row, "err", err)
		return
	}
	if err := f.SetCellValue(sheet, cellName, value); err != nil {
		slog.Error("Запись в ячейку", "cell", cellName, "err", err)
	}
}
