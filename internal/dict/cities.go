package dict

import (
	"fmt"

	"fcs-autoreport/internal/domain"
	"fcs-autoreport/internal/db"

	"github.com/xuri/excelize/v2"
)

// CitiesImportConfig задаёт колонки Excel для импорта справочника городов.
// Индексы колонок — 0-based. HeaderRows — сколько первых строк пропустить (заголовок).
type CitiesImportConfig struct {
	NameCol    int    // колонка с названием города
	CodeCol    int    // колонка с кодом Минздрава
	HeaderRows int    // число строк заголовка (пропускаем)
	Sheet      string // имя листа, пусто — первый лист
}

// DefaultCitiesImportConfig возвращает конфиг по умолчанию: название в 0, код в 1, одна строка заголовка.
func DefaultCitiesImportConfig() CitiesImportConfig {
	return CitiesImportConfig{
		NameCol:    0,
		CodeCol:    1,
		HeaderRows: 1,
		Sheet:      "",
	}
}

// ImportCitiesFromExcel читает Excel, нормализует названия и коды, дедуплицирует (последняя запись побеждает)
// и выполняет Upsert в БД: при совпадении name обновляется только code, алиасы не трогаем.
func ImportCitiesFromExcel(database *db.DB, path string, cfg CitiesImportConfig) (imported int, err error) {
	f, err := excelize.OpenFile(path)
	if err != nil {
		return 0, fmt.Errorf("открытие файла: %w", err)
	}
	defer f.Close()

	sheet := cfg.Sheet
	if sheet == "" {
		sheet = f.GetSheetName(0)
		if sheet == "" {
			return 0, fmt.Errorf("в файле нет листов")
		}
	}

	rows, err := f.GetRows(sheet)
	if err != nil {
		return 0, fmt.Errorf("чтение листа %q: %w", sheet, err)
	}

	if len(rows) <= cfg.HeaderRows {
		return 0, nil
	}

	// Собираем пары (name, code); дубликаты по name — последняя запись побеждает
	type pair struct{ name, code string }
	lastByName := make(map[string]string)

	for i := cfg.HeaderRows; i < len(rows); i++ {
		row := rows[i]
		name := cell(row, cfg.NameCol)
		code := cell(row, cfg.CodeCol)
		name = domain.NormalizeText(name)
		code = domain.NormalizeText(code)
		if name == "" {
			continue
		}
		lastByName[name] = code
	}

	pairs := make([]domain.CityNameCode, 0, len(lastByName))
	for n, c := range lastByName {
		pairs = append(pairs, domain.CityNameCode{Name: n, Code: c})
	}

	if err := database.UpsertCitiesFromPairs(pairs); err != nil {
		return 0, fmt.Errorf("запись в БД: %w", err)
	}
	return len(pairs), nil
}

func cell(row []string, col int) string {
	if col < 0 || col >= len(row) {
		return ""
	}
	return row[col]
}
