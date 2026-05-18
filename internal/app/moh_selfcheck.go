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

// mohSelfCheckDataRow возвращает (критичные для блокировки отправки, подсказки в sidecar).
func mohSelfCheckDataRow(f *excelize.File, sheet string, row, clientCol int, dc mohDriverCols) (hard, soft []string) {
	supplier := mohSelfCheckCell(f, sheet, 1, row)
	hp := mohSelfCheckCell(f, sheet, 12, row)
	if supplier == "" && hp == "" {
		return nil, nil
	}
	addHard := func(msg string) { hard = append(hard, msg) }
	addSoft := func(msg string) { soft = append(soft, msg) }

	city := mohSelfCheckCell(f, sheet, 10, row)
	addr := mohSelfCheckCell(f, sheet, 11, row)
	client := mohSelfCheckCell(f, sheet, clientCol, row)
	car := mohSelfCheckCell(f, sheet, dc.Vehicle, row)
	driver := mohSelfCheckCell(f, sheet, dc.DriverName, row)
	phone := mohSelfCheckCell(f, sheet, dc.Phone, row)
	inv := mohSelfCheckCell(f, sheet, 14, row)
	date := mohSelfCheckCell(f, sheet, 4, row)
	if date == "" {
		addHard("תאריך пуст (колонка D)")
	}

	if city == "" {
		addHard("קוד עיר пуст — ветслужба часто отклоняет")
	} else if !domain.IsMoHCityCodeFormat(city) {
		addHard(fmt.Sprintf("קוד עיר %q не похож на формат МОЗ (буква + 2–4 цифры)", city))
	}
	if addr == "" {
		addHard("כתובת пустая — נקודת שיווק без адреса")
	}
	if strings.Contains(addr, "רח'") || strings.Contains(addr, "רח׳") || strings.Contains(addr, "רח\u2018") || strings.Contains(addr, "רח\u2019") {
		addSoft("В адресе осталось «רח'» — лучше «רחוב» (как в реестре)")
	}
	if strings.Contains(addr, `\`) || strings.Contains(addr, "/") {
		addSoft("В адресе есть «\\» или «/» — при экспорте обрезается хвост; проверьте номер дома")
	}
	if hp == "" {
		addHard("ח\"פ לקוח пуст")
	} else {
		d := domain.ClientHPDigits(hp)
		if len(d) < 8 || len(d) > 9 {
			addHard(fmt.Sprintf("ח\"פ: ожидают 8–9 цифр, сейчас %d (%q)", len(d), hp))
		}
	}
	if client == "" {
		addHard("שם לקוח (לקוח) пуст")
	}
	if domain.ContainsCyrillic(client) || domain.ContainsCyrillic(addr) {
		addHard("В שם לקוח или כתובת есть кириллица — в реестре נקודות שיווק обычно иврит")
	}
	if car == "" {
		addSoft("מס.רכב пуст — укажите водителя для קוד עיר в справочнике")
	}
	if driver == "" {
		addSoft("שם הנהג пуст — укажите водителя для קוד עיר в справочнике")
	}
	if phone == "" {
		addSoft("טלפון נהג пуст")
	}
	if inv == "" {
		addHard("מספר תעודת משלוח пуст")
	}

	if mohWeightOnlyInOtherColumn(f, sheet, row) {
		addSoft("Весь вес в колонке «אחר» (24) — заполните справочник товаров или проверьте «שם קבוצה» в FishKA")
	}

	for col := 15; col <= 27; col++ {
		s := mohSelfCheckCell(f, sheet, col, row)
		if s == "" {
			continue
		}
		if n, err := strconv.ParseFloat(strings.ReplaceAll(s, ",", "."), 64); err == nil && n < 0 {
			addHard(fmt.Sprintf("отрицательное значение в колонке %d: %s", col, s))
		}
	}

	sum := calcWeightsSumForRow(f, sheet, row)
	if roundWeight(sum) <= 0 {
		addHard("нулевой суммарный вес по категориям (столбцы 15–25)")
	}
	boxStr := mohSelfCheckCell(f, sheet, 26, row)
	totalStr := mohSelfCheckCell(f, sheet, 27, row)
	if tw, errTw := strconv.ParseFloat(strings.ReplaceAll(totalStr, ",", "."), 64); errTw == nil && tw > 0 {
		if b, errB := strconv.ParseFloat(strings.ReplaceAll(boxStr, ",", "."), 64); errB == nil && b > 0 && b < domain.MoHMinBoxesLightFraction {
			addSoft(fmt.Sprintf("סה\"כ קרטונים %.2f < порога МОЗ %.2f при положительном весе", b, domain.MoHMinBoxesLightFraction))
		}
	}
	if totalStr != "" {
		if total, err := strconv.ParseFloat(strings.ReplaceAll(totalStr, ",", "."), 64); err == nil {
			if roundWeight(sum) != roundWeight(total) {
				addHard(fmt.Sprintf("סה\"כ משקל (%.2f) ≠ сумма категорий (%.2f)", total, sum))
			}
		}
	}

	return hard, soft
}

func mohWeightOnlyInOtherColumn(f *excelize.File, sheet string, row int) bool {
	var fishSum, other float64
	for col := 15; col <= 23; col++ {
		v := mohSelfCheckCell(f, sheet, col, row)
		if v == "" {
			continue
		}
		n, err := strconv.ParseFloat(strings.ReplaceAll(v, ",", "."), 64)
		if err != nil || n <= 0 {
			continue
		}
		fishSum += n
	}
	v24 := mohSelfCheckCell(f, sheet, 24, row)
	if v24 != "" {
		n, err := strconv.ParseFloat(strings.ReplaceAll(v24, ",", "."), 64)
		if err == nil {
			other = n
		}
	}
	return fishSum == 0 && other > 0
}

// ValidateMoHReportFile блокирует отправку только по критичным полям; подсказки — в sidecar.
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
		hard, soft := mohSelfCheckDataRow(f, sh, row, clientCol, dc)
		for _, m := range hard {
			all = append(all, fmt.Sprintf("строка %d: %s", row, m))
		}
		_ = soft
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
		hard, soft := mohSelfCheckDataRow(f, sheet, row, clientCol, dc)
		for _, msg := range append(hard, soft...) {
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
