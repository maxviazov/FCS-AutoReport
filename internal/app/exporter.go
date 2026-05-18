package app

import (
	"fmt"
	"log/slog"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

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

// normalizeMoHBoxColumn26 — колонка «אריזות» (26): любое число строго меньше MoHMinBoxesLightFraction
// поднимается до порога (требование портала МОЗ).
func normalizeMoHBoxColumn26(boxes float64) float64 {
	b := roundWeight(boxes)
	if b < domain.MoHMinBoxesLightFraction {
		return domain.MoHMinBoxesLightFraction
	}
	return b
}

// setMoHBoxColumn26 записывает «סה"כ קרטונים» (кол. 26). Если в шаблоне в ячейке была формула,
// одного SetCellValue недостаточно — сначала сбрасываем формулу, иначе в файле остаются некорректные значения.
func setMoHBoxColumn26(f *excelize.File, sheet string, row int, value float64) {
	cell, err := excelize.CoordinatesToCellName(26, row)
	if err != nil {
		return
	}
	if fm, _ := f.GetCellFormula(sheet, cell); strings.TrimSpace(fm) != "" {
		if err := f.SetCellFormula(sheet, cell, ""); err != nil {
			slog.Warn("Сброс формулы Z", "cell", cell, "err", err)
		}
	}
	if err := f.SetCellValue(sheet, cell, value); err != nil {
		slog.Error("Запись סה\"כ קרטונים", "cell", cell, "err", err)
	}
}

// enforceMoHBoxColumn26ForDataRows — для строк с номером накладной (кол. 14) или положительным весом в кол. 27
// перечитывает Z и поднимает значение до минимума МОЗ.
func enforceMoHBoxColumn26ForDataRows(f *excelize.File, sheet string, lastRow int) {
	for row := 2; row <= lastRow; row++ {
		inv := strings.TrimSpace(mohSelfCheckCell(f, sheet, 14, row))
		twStr := strings.TrimSpace(mohSelfCheckCell(f, sheet, 27, row))
		tw, errTw := strconv.ParseFloat(strings.ReplaceAll(twStr, ",", "."), 64)
		if inv == "" && (errTw != nil || tw <= 0) {
			continue
		}
		cell, err := excelize.CoordinatesToCellName(26, row)
		if err != nil {
			continue
		}
		s, _ := f.GetCellValue(sheet, cell)
		v, err := strconv.ParseFloat(strings.ReplaceAll(strings.TrimSpace(s), ",", "."), 64)
		if err != nil {
			v = 0
		}
		setMoHBoxColumn26(f, sheet, row, normalizeMoHBoxColumn26(v))
	}
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

// ClientXLSXExport — один сгенерированный файл и накладные, попавшие в него.
type ClientXLSXExport struct {
	Path     string
	Invoices []*domain.AggregatedInvoice
}

const clientFileBaseMaxRunes = 48
const (
	readyExportsDirName  = "fish_reports_ready"
	manualReviewDirName  = "fish_reports_manual_review"
)

// sanitizeClientFileBase убирает пробелы и небуквенно-цифровые символы (имя для имени файла).
func sanitizeClientFileBase(name string) string {
	name = sanitizeClientName(name)
	var b strings.Builder
	for _, r := range name {
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			b.WriteRune(r)
		}
	}
	s := b.String()
	if s == "" {
		return ""
	}
	runes := []rune(s)
	if len(runes) > clientFileBaseMaxRunes {
		s = string(runes[:clientFileBaseMaxRunes])
	}
	return s
}

// sanitizeClientName очищает название клиента: буквы, цифры, пробел; сохраняет גרשיים в «בע"מ».
func sanitizeClientName(name string) string {
	name = domain.NormalizeText(name)
	if name == "" {
		return ""
	}
	var b strings.Builder
	b.Grow(len(name))
	prevSpace := false
	for _, r := range name {
		switch {
		case unicode.IsLetter(r), unicode.IsNumber(r):
			b.WriteRune(r)
			prevSpace = false
		case r == '"', r == '\'', r == '״', r == '׳', r == '-', r == '.':
			b.WriteRune(r)
			prevSpace = false
		case unicode.IsSpace(r):
			if !prevSpace {
				b.WriteRune(' ')
				prevSpace = true
			}
		}
	}
	return strings.TrimSpace(b.String())
}

// clientExportFileStem — короткая метка клиента + ח"פ для уникального имени файла.
func clientExportFileStem(inv *domain.AggregatedInvoice) string {
	base := sanitizeClientFileBase(inv.ClientName)
	if base == "" {
		base = "client"
	}
	d := hpDigitsOnly(strings.TrimSpace(inv.ClientHP))
	if d != "" {
		return base + "_" + d
	}
	return base
}

// validateAggregatedInvoicesForMoHExport блокирует экспорт, если любая накладная не соответствует требованиям МОЗ.
func validateAggregatedInvoicesForMoHExport(invoices []*domain.AggregatedInvoice) error {
	if len(invoices) == 0 {
		return &MohExportValidationError{Lines: []string{"нет накладных для экспорта"}}
	}
	var lines []string
	for _, inv := range invoices {
		for _, msg := range domain.ValidateInvoiceForMoHExport(inv) {
			lines = append(lines, fmt.Sprintf("%s / %s: %s", inv.InvoiceNum, inv.ClientName, msg))
		}
	}
	if len(lines) == 0 {
		return nil
	}
	return &MohExportValidationError{Lines: lines}
}

// detectLastMoHDataRow — последняя строка с номером накладной (кол. 14); хвост шаблона без תעודה не считается данными.
func detectLastMoHDataRow(f *excelize.File, sheet string, scanThrough int) int {
	last := 0
	for row := 2; row <= scanThrough; row++ {
		inv := strings.TrimSpace(mohSelfCheckCell(f, sheet, 14, row))
		if inv != "" {
			last = row
		}
	}
	return last
}

// clearMoHSheetTail очищает строки ниже данных и сжимает used range листа (остаток примера из шаблона даёт «чужие» числа и форматы в X–AB).
func clearMoHSheetTail(f *excelize.File, sheet string, dataLastRow int) error {
	if dataLastRow < 2 {
		return nil
	}
	dim, err := f.GetSheetDimension(sheet)
	if err != nil || strings.TrimSpace(dim) == "" {
		return nil
	}
	parts := strings.Split(dim, ":")
	if len(parts) != 2 {
		return nil
	}
	startCol, startRow, err1 := excelize.CellNameToCoordinates(parts[0])
	endCol, endRow, err2 := excelize.CellNameToCoordinates(parts[1])
	if err1 != nil || err2 != nil {
		if err1 != nil {
			return err1
		}
		return err2
	}
	_ = startRow // обычно 1
	newStart, err := excelize.CoordinatesToCellName(startCol, startRow)
	if err != nil {
		return err
	}
	newEnd, err := excelize.CoordinatesToCellName(endCol, dataLastRow)
	if err != nil {
		return err
	}
	if endRow <= dataLastRow {
		return f.SetSheetDimension(sheet, newStart+":"+newEnd)
	}
	for row := dataLastRow + 1; row <= endRow; row++ {
		for col := 1; col <= endCol; col++ {
			cell, err := excelize.CoordinatesToCellName(col, row)
			if err != nil {
				continue
			}
			_ = f.SetCellFormula(sheet, cell, "")
			_ = f.SetCellStr(sheet, cell, "")
		}
	}
	return f.SetSheetDimension(sheet, newStart+":"+newEnd)
}

// exportAggregatedToPath заполняет шаблон и сохраняет по savePath (полный путь к .xlsx).
func exportAggregatedToPath(invoices []*domain.AggregatedInvoice, templatePath, savePath string) error {
	slog.Info("Экспорт в шаблон Минздрава", "invoices_count", len(invoices), "save_path", savePath)

	if err := validateAggregatedInvoicesForMoHExport(invoices); err != nil {
		return err
	}

	f, err := excelize.OpenFile(templatePath)
	if err != nil {
		return fmt.Errorf("открытие шаблона %s: %w", templatePath, err)
	}
	defer f.Close()

	sheetName := f.GetSheetName(0)
	if sheetName == "" {
		return fmt.Errorf("в шаблоне нет листов")
	}

	driverCols := detectDriverColumns(f, sheetName)
	clientCol := detectClientColumn(f, sheetName)

	nineDigitFmt := "000000000"
	clientHPStyleID, err := f.NewStyle(&excelize.Style{
		CustomNumFmt: &nineDigitFmt,
		Border: []excelize.Border{
			{Type: "left", Color: "000000", Style: 1},
			{Type: "top", Color: "000000", Style: 1},
			{Type: "bottom", Color: "000000", Style: 1},
			{Type: "right", Color: "000000", Style: 1},
		},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})
	if err != nil {
		slog.Warn("Не удалось создать стиль ח\"פ (9 цифр)", "err", err)
		clientHPStyleID = 0
	}

	currentRow := 2

	for _, inv := range invoices {
		setCell(f, sheetName, 1, currentRow, SupplierName)
		setCell(f, sheetName, 2, currentRow, 511777856)
		setCell(f, sheetName, 3, currentRow, MoHNumber)
		setReportDateCell(f, sheetName, 4, currentRow, inv.Date)
		setCell(f, sheetName, driverCols.Vehicle, currentRow, inv.CarNumber)
		setCell(f, sheetName, driverCols.DriverName, currentRow, inv.DriverName)
		setCell(f, sheetName, driverCols.Phone, currentRow, inv.Phone)
		slog.Info("Итоговый отчёт: в ячейку לקוח",
			"invoice", inv.InvoiceNum,
			"row", currentRow,
			"col", clientCol,
			"client_name", inv.ClientName)
		cleanClientName := sanitizeClientName(inv.ClientName)
		if cleanClientName == "" {
			cleanClientName = domain.NormalizeText(inv.ClientName)
		}
		setCell(f, sheetName, clientCol, currentRow, cleanClientName)
		setCell(f, sheetName, 9, currentRow, "קמעונאי")
		setCell(f, sheetName, 10, currentRow, strings.TrimSpace(inv.CityCode))
		setCell(f, sheetName, 11, currentRow, domain.MoHAddressForReport(inv.Address, inv.ClientName, inv.MoHCityAfterComma))
		setCell(f, sheetName, 13, currentRow, 0)
		setCell(f, sheetName, 14, currentRow, numericOrString(inv.InvoiceNum))

		for col := 15; col <= 25; col++ {
			setCell(f, sheetName, col, currentRow, 0)
		}
		var totalWeight float64
		colWeights := make(map[int]float64)
		for category, weight := range inv.Weights {
			if weight < 0 {
				slog.Warn("Отрицательный вес по категории — в отчёт не попадёт", "invoice", inv.InvoiceNum, "category", category, "kg", weight)
				continue
			}
			if category == "UNKNOWN" {
				slog.Warn("Нераспределённый вес", "invoice", inv.InvoiceNum, "weight_kg", weight)
			}
			totalWeight += weight
			colIdx := getCategoryColumn(category)
			if colIdx <= 0 && category == "UNKNOWN" {
				colIdx = 24
			}
			if colIdx > 0 {
				colWeights[colIdx] += weight
			}
		}
		for colIdx, w := range colWeights {
			setCell(f, sheetName, colIdx, currentRow, roundWeight(w))
		}

		boxesOut := normalizeMoHBoxColumn26(inv.TotalBoxes)
		if boxesOut != roundWeight(inv.TotalBoxes) {
			slog.Info("МОЗ: нормализация אריזות для портала",
				"invoice", inv.InvoiceNum,
				"raw_boxes", inv.TotalBoxes,
				"exported_boxes", boxesOut)
		}
		setMoHBoxColumn26(f, sheetName, currentRow, boxesOut)
		setCell(f, sheetName, 27, currentRow, roundWeight(totalWeight))
		setCell(f, sheetName, 28, currentRow, 1)

		currentRow++
	}

	lastRow := currentRow - 1
	if err := clearMoHSheetTail(f, sheetName, lastRow); err != nil {
		slog.Warn("Не удалось очистить хвост листа ниже строк данных", "sheet", sheetName, "data_last_row", lastRow, "err", err)
	}
	if lastRow >= 2 {
		applyTemplateRow2Styles(f, sheetName, 2, lastRow)
	}
	// applyTemplateRow2Styles копирует стили диапазоном — в некоторых шаблонах значение колонки 26
	// может снова стать как в строке 2 шаблона; повторно выставляем אריזות и итоговый вес.
	for i, inv := range invoices {
		row := 2 + i
		var tw float64
		for _, w := range inv.Weights {
			if w < 0 {
				continue
			}
			tw += w
		}
		setMoHBoxColumn26(f, sheetName, row, normalizeMoHBoxColumn26(inv.TotalBoxes))
		setCell(f, sheetName, 27, row, roundWeight(tw))
	}

	for i, inv := range invoices {
		setClientHPCell(f, sheetName, 12, 2+i, inv.ClientHP, clientHPStyleID)
	}

	enforceMoHBoxColumn26ForDataRows(f, sheetName, lastRow)

	if err := f.SaveAs(savePath); err != nil {
		return fmt.Errorf("сохранение отчёта: %w", err)
	}
	slog.Info("Отчёт сохранён", "path", savePath)
	return nil
}

// postExportValidateAndRepair выполняет авто-проверку и авто-доработку экспортированного файла.
// mohStreetFromPostExportAddress — второй проход по кол. «כתובת» только для «רחוב, עיר».
// Не трогаем «רחוב, מספר» (напр. «קרן היסוד, 68»).
func mohStreetFromPostExportAddress(v string) (street string, ok bool) {
	norm := domain.NormalizeMinistryAddress(v)
	if norm == "" {
		return "", false
	}
	if !domain.InferCityPlacedAfterComma(norm) {
		return "", false
	}
	street = domain.MoHStreetLineForMoH(norm, true)
	if street == "" {
		return "", false
	}
	return street, true
}

// Возвращает needsManual=true, если файл невозможно автоматически привести к требованиям.
func postExportValidateAndRepair(exportPath, templatePath string) (needsManual bool, reason string) {
	template, err := excelize.OpenFile(templatePath)
	if err != nil {
		return true, fmt.Sprintf("открытие шаблона: %v", err)
	}
	defer template.Close()

	exported, err := excelize.OpenFile(exportPath)
	if err != nil {
		return true, fmt.Sprintf("открытие экспортированного файла: %v", err)
	}
	defer exported.Close()

	tSheet := template.GetSheetName(0)
	eSheet := exported.GetSheetName(0)
	if tSheet == "" || eSheet == "" {
		return true, "не найден первый лист в шаблоне или экспортированном файле"
	}

	rows, err := exported.GetRows(eSheet)
	if err != nil {
		return true, fmt.Sprintf("чтение строк экспортированного файла: %v", err)
	}
	lastRow := len(rows)
	if lastRow < 2 {
		lastRow = 2
	}
	// GetRows иногда обрезает хвост пустых строк — размер слайса может быть меньше,
	// чем реальный последний ряд с данными; берём max с dimension листа.
	if dim, err := exported.GetSheetDimension(eSheet); err == nil && dim != "" {
		parts := strings.Split(dim, ":")
		if len(parts) == 2 {
			_, endRow, err := excelize.CellNameToCoordinates(parts[1])
			if err == nil && endRow > lastRow {
				lastRow = endRow
			}
		}
	}

	dataEnd := detectLastMoHDataRow(exported, eSheet, lastRow)
	if dataEnd >= 2 {
		if err := clearMoHSheetTail(exported, eSheet, dataEnd); err != nil {
			slog.Warn("postExport: не удалось очистить хвост листа", "err", err)
		}
		lastRow = dataEnd
	}

	// 1) Приводим стили построчно по колонкам из строки 2 шаблона.
	applyTemplateRow2Styles(exported, eSheet, 2, lastRow)

	// 2) Обязательные формулы/типы по требованиям.
	for row := 2; row <= lastRow; row++ {
		normalizeDateCell(exported, eSheet, 4, row)
		sum := calcWeightsSumForRow(exported, eSheet, row)
		setCell(exported, eSheet, 27, row, roundWeight(sum))
		boxCell, err := excelize.CoordinatesToCellName(26, row)
		if err != nil {
			continue
		}
		boxStr, _ := exported.GetCellValue(eSheet, boxCell)
		boxVal, err := strconv.ParseFloat(strings.ReplaceAll(strings.TrimSpace(boxStr), ",", "."), 64)
		if err != nil {
			boxVal = 0
		}
		fixed := normalizeMoHBoxColumn26(boxVal)
		setMoHBoxColumn26(exported, eSheet, row, fixed)
	}

	// 3) Повторная очистка названия клиента (защита от спецсимволов).
	clientCol := detectClientColumn(exported, eSheet)
	for row := 2; row <= lastRow; row++ {
		cell, _ := excelize.CoordinatesToCellName(clientCol, row)
		v, _ := exported.GetCellValue(eSheet, cell)
		clean := sanitizeClientName(v)
		if clean != "" && clean != v {
			_ = exported.SetCellValue(eSheet, cell, clean)
		}
	}

	// 3b) כתובת — только «רחוב, עיר» (город в конце). «עיר, רחוב» уже нормализовано в MoHAddressForReport;
	// повторный Cut по запятой ломает «קרן היסוד, 68» → «68».
	for row := 2; row <= lastRow; row++ {
		cell11, err := excelize.CoordinatesToCellName(11, row)
		if err != nil {
			continue
		}
		v, _ := exported.GetCellValue(eSheet, cell11)
		if street, ok := mohStreetFromPostExportAddress(v); ok && street != strings.TrimSpace(v) {
			_ = exported.SetCellValue(eSheet, cell11, street)
		}
	}

	enforceMoHBoxColumn26ForDataRows(exported, eSheet, lastRow)

	mohSelfCheckAfterExport(exported, eSheet, lastRow, exportPath)

	if err := exported.Save(); err != nil {
		return true, fmt.Sprintf("сохранение после авто-доработки: %v", err)
	}
	return false, ""
}

func writeManualReviewReport(manualDir string, reviewPaths []string) {
	if len(reviewPaths) == 0 {
		return
	}
	reportPath := filepath.Join(manualDir, "manual_review_required.txt")
	var b strings.Builder
	b.WriteString("Эти файлы не удалось автоматически привести к требованиям и требуют ручной доработки:\n")
	for _, p := range reviewPaths {
		b.WriteString("- ")
		b.WriteString(p)
		b.WriteString("\n")
	}
	if err := os.WriteFile(reportPath, []byte(b.String()), 0o644); err != nil {
		slog.Warn("Не удалось записать отчёт по файлам для ручной доработки", "path", reportPath, "err", err)
		return
	}
	slog.Warn("Создан отчёт по файлам для ручной доработки", "path", reportPath, "count", len(reviewPaths))
}

func moveToManualReviewDir(manualDir, filePath string) string {
	if strings.TrimSpace(filePath) == "" {
		return filePath
	}
	if err := os.MkdirAll(manualDir, 0o755); err != nil {
		slog.Warn("Не удалось создать папку manual_review", "dir", manualDir, "err", err)
		return filePath
	}
	dst := filepath.Join(manualDir, filepath.Base(filePath))
	if err := os.Rename(filePath, dst); err != nil {
		slog.Warn("Не удалось переместить файл в manual_review", "src", filePath, "dst", dst, "err", err)
		return filePath
	}
	return dst
}

func resetManagedDir(dir string) error {
	if err := os.RemoveAll(dir); err != nil {
		return err
	}
	return os.MkdirAll(dir, 0o755)
}

// ExportPerClientToRunFolder создаёт подпапку fish_reports_ГГГГ-ММ-ДД_ЧЧ-ММ-СС в parentOutDir
// и по одному xlsx на клиента (группировка по ח"פ). Имя файла — короткое имя + _חפ.xlsx.
func ExportPerClientToRunFolder(parentOutDir, templatePath string, invoices []*domain.AggregatedInvoice) (runDir string, exports []ClientXLSXExport, manualReview []string, err error) {
	if len(invoices) == 0 {
		return "", nil, nil, fmt.Errorf("нет накладных для экспорта")
	}
	runDir = filepath.Join(parentOutDir, readyExportsDirName)
	manualDir := filepath.Join(parentOutDir, manualReviewDirName)
	if err := resetManagedDir(runDir); err != nil {
		return "", nil, nil, fmt.Errorf("создание папки отчёта: %w", err)
	}
	_ = os.RemoveAll(manualDir)

	byHP := make(map[string][]*domain.AggregatedInvoice)
	order := make([]string, 0)
	for _, inv := range invoices {
		key := hpDigitsOnly(strings.TrimSpace(inv.ClientHP))
		if key == "" {
			key = domain.NormalizeText(inv.ClientHP)
		}
		if _, ok := byHP[key]; !ok {
			order = append(order, key)
		}
		byHP[key] = append(byHP[key], inv)
	}
	sort.Strings(order)

	exports = make([]ClientXLSXExport, 0, len(order))
	manualReview = make([]string, 0)
	for _, hpKey := range order {
		group := byHP[hpKey]
		stem := clientExportFileStem(group[0])
		path := filepath.Join(runDir, stem+".xlsx")
		if err := exportAggregatedToPath(group, templatePath, path); err != nil {
			slog.Warn("Не удалось экспортировать файл клиента, требуется ручная доработка", "path", path, "err", err)
			manualReview = append(manualReview, path)
			continue
		}
		if needsManual, reason := postExportValidateAndRepair(path, templatePath); needsManual {
			manualPath := moveToManualReviewDir(manualDir, path)
			slog.Warn("Авто-доработка невозможна, требуется ручная проверка файла", "path", manualPath, "reason", reason)
			manualReview = append(manualReview, manualPath)
			continue
		}
		exports = append(exports, ClientXLSXExport{Path: path, Invoices: group})
	}
	writeManualReviewReport(manualDir, manualReview)

	slog.Info("Пакет отчётов по клиентам сохранён", "run_dir", runDir, "files", len(exports), "manual_review_count", len(manualReview))
	return runDir, exports, manualReview, nil
}

// ExportToExcel заполняет шаблон Минздрава агрегированными накладными и сохраняет в outputDir одним файлом.
// Соответствует требованиям שידורים למערכת הממוחשבת (апрель 2024): колонки A–AB, формат xlsx.
func ExportToExcel(invoices []*domain.AggregatedInvoice, templatePath, outputDir string) (string, error) {
	fileName := fmt.Sprintf("FCS_Report_%s.xlsx", time.Now().Format("2006-01-02_15-04-05"))
	savePath := filepath.Join(outputDir, fileName)
	if err := exportAggregatedToPath(invoices, templatePath, savePath); err != nil {
		return "", err
	}
	if needsManual, reason := postExportValidateAndRepair(savePath, templatePath); needsManual {
		slog.Warn("Авто-доработка файла после экспорта невозможна", "path", savePath, "reason", reason)
	}
	return savePath, nil
}

func hpDigitsOnly(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// setClientHPCell записывает ח"פ как число (тип number в xlsx), с форматом отображения 000000000.
func setClientHPCell(f *excelize.File, sheet string, col, row int, hp string, nineDigitStyleID int) {
	cellName, err := excelize.CoordinatesToCellName(col, row)
	if err != nil {
		slog.Error("Координаты ячейки ח\"פ", "col", col, "row", row, "err", err)
		return
	}
	s := strings.TrimSpace(hp)
	d := hpDigitsOnly(s)
	if d == "" {
		if s == "" {
			_ = f.SetCellValue(sheet, cellName, "")
			return
		}
		_ = f.SetCellValue(sheet, cellName, s)
		return
	}
	n, err := strconv.ParseInt(d, 10, 64)
	if err != nil {
		_ = f.SetCellValue(sheet, cellName, s)
		return
	}
	if err := f.SetCellValue(sheet, cellName, n); err != nil {
		slog.Error("Запись ח\"פ", "cell", cellName, "err", err)
		return
	}
	if len(d) <= 9 && nineDigitStyleID != 0 {
		_ = f.SetCellStyle(sheet, cellName, cellName, nineDigitStyleID)
	}
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
	// 26 — «סה"כ קרטונים»: значения задаёт только экспорт; массовый SetCellStyle по Z2:Z{n} не применять.
	numCols := []int{2, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 27, 28}
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

// setCellFormula записывает формулу в ячейку по номерам колонки и строки (1-based).
func setCellFormula(f *excelize.File, sheet string, col, row int, formula string) {
	cellName, err := excelize.CoordinatesToCellName(col, row)
	if err != nil {
		slog.Error("Координаты ячейки формулы", "col", col, "row", row, "err", err)
		return
	}
	if err := f.SetCellFormula(sheet, cellName, formula); err != nil {
		slog.Error("Запись формулы", "cell", cellName, "formula", formula, "err", err)
	}
}

func setReportDateCell(f *excelize.File, sheet string, col, row int, raw string) {
	cellName, err := excelize.CoordinatesToCellName(col, row)
	if err != nil {
		slog.Error("Координаты ячейки даты", "col", col, "row", row, "err", err)
		return
	}
	if dt, ok := parseReportDate(raw); ok {
		if err := f.SetCellValue(sheet, cellName, dt); err != nil {
			slog.Warn("Не удалось записать дату как date, используем исходную строку", "cell", cellName, "err", err)
			_ = f.SetCellValue(sheet, cellName, strings.TrimSpace(raw))
		} else {
			setMoHDateCellStyle(f, sheet, cellName)
		}
		return
	}
	_ = f.SetCellValue(sheet, cellName, strings.TrimSpace(raw))
}

func parseReportDate(raw string) (time.Time, bool) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return time.Time{}, false
	}
	layouts := []string{
		"02.01.2006",
		"2.1.2006",
		"2006-01-02",
		"02/01/2006",
		"2/1/2006",
	}
	for _, layout := range layouts {
		if dt, err := time.ParseInLocation(layout, s, time.Local); err == nil {
			return dt, true
		}
	}
	return time.Time{}, false
}

func normalizeDateCell(f *excelize.File, sheet string, col, row int) {
	cellName, err := excelize.CoordinatesToCellName(col, row)
	if err != nil {
		return
	}
	v, err := f.GetCellValue(sheet, cellName)
	if err != nil {
		return
	}
	if dt, ok := parseReportDate(v); ok {
		_ = f.SetCellValue(sheet, cellName, dt)
		setMoHDateCellStyle(f, sheet, cellName)
	}
}

var mohDateDisplayFmt = "dd.mm.yyyy"

func setMoHDateCellStyle(f *excelize.File, sheet, cellName string) {
	styleID, err := f.GetCellStyle(sheet, cellName)
	if err != nil {
		return
	}
	style, err := f.GetStyle(styleID)
	if err != nil || style == nil {
		style = &excelize.Style{}
	}
	style.CustomNumFmt = &mohDateDisplayFmt
	newID, err := f.NewStyle(style)
	if err != nil {
		return
	}
	_ = f.SetCellStyle(sheet, cellName, cellName, newID)
}

func calcWeightsSumForRow(f *excelize.File, sheet string, row int) float64 {
	var sum float64
	for col := 15; col <= 25; col++ {
		cellName, err := excelize.CoordinatesToCellName(col, row)
		if err != nil {
			continue
		}
		v, err := f.GetCellValue(sheet, cellName)
		if err != nil {
			continue
		}
		v = strings.TrimSpace(strings.ReplaceAll(v, ",", "."))
		if v == "" {
			continue
		}
		if n, err := strconv.ParseFloat(v, 64); err == nil {
			sum += n
		}
	}
	return sum
}

// applyTemplateRow2Styles копирует стиль из строки 2 шаблона на все строки данных по колонкам.
// Это сохраняет точные шаблонные различия (например формат даты в D и отдельные стили справа).
func applyTemplateRow2Styles(f *excelize.File, sheet string, firstRow, lastRow int) {
	if firstRow > lastRow {
		return
	}
	for col := 1; col <= 28; col++ {
		// Колонка 26 — «סה"כ קרטונים»: значение задаётся только кодом. SetCellStyle на диапазон Z2:Z{lastRow}
		// тиражирует образец из Z2 на все строки и перетирает числа; колонку 26 из стилей исключаем.
		if col == 26 {
			continue
		}
		srcCell, err := excelize.CoordinatesToCellName(col, 2)
		if err != nil {
			continue
		}
		styleID, err := f.GetCellStyle(sheet, srcCell)
		if err != nil {
			slog.Warn("Не удалось прочитать стиль колонки из строки 2", "col", col, "err", err)
			continue
		}
		dstStart, _ := excelize.CoordinatesToCellName(col, firstRow)
		dstEnd, _ := excelize.CoordinatesToCellName(col, lastRow)
		if err := f.SetCellStyle(sheet, dstStart, dstEnd, styleID); err != nil {
			slog.Warn("Не удалось применить стиль колонки", "col", col, "err", err)
		}
	}
}
