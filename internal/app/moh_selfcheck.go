package app

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"fcs-autoreport/internal/domain"

	"github.com/xuri/excelize/v2"
)

func mohSelfCheckCell(f *excelize.File, sheet string, col, row int) string {
	cell, err := excelize.CoordinatesToCellName(col, row)
	if err != nil {
		return ""
	}
	v, err := f.GetCellValue(sheet, cell)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(v)
}

func mohSelfCheckDataRow(f *excelize.File, sheet string, row, clientCol int, dc mohDriverCols) []string {
	supplier := mohSelfCheckCell(f, sheet, 1, row)
	hp := mohSelfCheckCell(f, sheet, 12, row)
	if supplier == "" && hp == "" {
		return nil
	}
	var w []string
	add := func(msg string) { w = append(w, msg) }

	city := mohSelfCheckCell(f, sheet, 10, row)
	addr := mohSelfCheckCell(f, sheet, 11, row)
	client := mohSelfCheckCell(f, sheet, clientCol, row)
	car := mohSelfCheckCell(f, sheet, dc.Vehicle, row)
	driver := mohSelfCheckCell(f, sheet, dc.DriverName, row)
	phone := mohSelfCheckCell(f, sheet, dc.Phone, row)
	inv := mohSelfCheckCell(f, sheet, 14, row)
	date := mohSelfCheckCell(f, sheet, 4, row)
	if date == "" {
		add("תאריך пуст (колонка D)")
	}

	if city == "" {
		add("קוד עיר пуст — ветслужба часто отклоняет")
	} else if !domain.IsMoHCityCodeFormat(city) {
		add(fmt.Sprintf("קוד עיר %q не похож на формат МОЗ (буква + 3–4 цифры)", city))
	}
	if addr == "" {
		add("כתובת пустая — נקודת שיווק без адреса")
	}
	if strings.Contains(addr, "רח'") || strings.Contains(addr, "רח׳") || strings.Contains(addr, "רח\u2018") || strings.Contains(addr, "רח\u2019") {
		add("В адресе осталось «רח'» — лучше «רחוב» (как в реестре)")
	}
	if hp == "" {
		add("ח\"פ לקוח пуст")
	} else {
		d := domain.ClientHPDigits(hp)
		if len(d) < 8 || len(d) > 9 {
			add(fmt.Sprintf("ח\"פ: ожидают 8–9 цифр, сейчас %d (%q)", len(d), hp))
		}
	}
	if client == "" {
		add("שם לקוח (לקוח) пуст")
	}
	if domain.ContainsCyrillic(client) || domain.ContainsCyrillic(addr) {
		add("В שם לקוח или כתובת есть кириллица — в реестре נקודות שיווק обычно иврит")
	}
	if car == "" {
		add("מס.רכב пуст")
	}
	if driver == "" {
		add("שם הנהג пуст")
	}
	if phone == "" {
		add("טלפון נהג пуст")
	}
	if inv == "" {
		add("מספר תעודת משלוח пуст")
	}

	sum := calcWeightsSumForRow(f, sheet, row)
	if roundWeight(sum) <= 0 {
		add("нулевой суммарный вес по категориям (столбцы 15–25)")
	}
	totalStr := mohSelfCheckCell(f, sheet, 27, row)
	if totalStr != "" {
		if total, err := strconv.ParseFloat(strings.ReplaceAll(totalStr, ",", "."), 64); err == nil {
			if roundWeight(sum) != roundWeight(total) {
				add(fmt.Sprintf("סה\"כ משקל (%.2f) ≠ сумма категорий (%.2f)", total, sum))
			}
		}
	}

	return w
}

// ValidateMoHReportFile блокирует отправку, если файл отчёта МОЗ не проходит проверку полей.
func ValidateMoHReportFile(reportPath string) error {
	f, err := excelize.OpenFile(reportPath)
	if err != nil {
		return fmt.Errorf("открытие отчёта: %w", err)
	}
	defer f.Close()
	sh := f.GetSheetName(0)
	if sh == "" {
		return fmt.Errorf("в файле нет листов")
	}
	rows, err := f.GetRows(sh)
	if err != nil {
		return fmt.Errorf("чтение строк: %w", err)
	}
	lastRow := len(rows)
	if lastRow < 2 {
		return fmt.Errorf("в отчёте нет строк с данными")
	}
	clientCol := detectClientColumn(f, sh)
	dc := detectDriverColumns(f, sh)
	dataRows := 0
	var all []string
	for row := 2; row <= lastRow; row++ {
		sup := mohSelfCheckCell(f, sh, 1, row)
		hp := mohSelfCheckCell(f, sh, 12, row)
		if sup == "" && hp == "" {
			continue
		}
		dataRows++
		for _, m := range mohSelfCheckDataRow(f, sh, row, clientCol, dc) {
			all = append(all, fmt.Sprintf("строка %d: %s", row, m))
		}
	}
	if dataRows == 0 {
		return fmt.Errorf("нет заполненных строк данных (поставщик/ח\"פ)")
	}
	if len(all) > 0 {
		return &MohSendValidationError{Lines: all}
	}
	return nil
}

// mohSelfCheckAfterExport — эвристики под отказ «פרטי נקודות השיווק לא תקינים» (без пояснений от МОЗ).
func mohSelfCheckAfterExport(f *excelize.File, sheet string, lastRow int, exportPath string) {
	clientCol := detectClientColumn(f, sheet)
	dc := detectDriverColumns(f, sheet)
	var lines []string
	for row := 2; row <= lastRow; row++ {
		for _, msg := range mohSelfCheckDataRow(f, sheet, row, clientCol, dc) {
			line := fmt.Sprintf("row %d: %s", row, msg)
			lines = append(lines, line)
			slog.Warn("MoH самопроверка (נקודות שיווק)", "file", exportPath, "row", row, "hint", msg)
		}
	}
	if len(lines) == 0 {
		return
	}
	sidecar := strings.TrimSuffix(exportPath, filepath.Ext(exportPath)) + "_moh_selfcheck.txt"
	body := strings.Join(lines, "\n") + "\n"
	if err := os.WriteFile(sidecar, []byte(body), 0o644); err != nil {
		slog.Warn("Не удалось записать файл самопроверки MoH", "path", sidecar, "err", err)
		return
	}
	slog.Info("Самопроверка MoH: есть замечания — см. файл", "path", sidecar, "count", len(lines))
}
