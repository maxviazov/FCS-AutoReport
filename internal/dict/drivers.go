package dict

import (
	"fmt"
	"strings"

	"fcs-autoreport/internal/domain"
	"fcs-autoreport/internal/db"

	"github.com/xuri/excelize/v2"
)

// DriversImportConfig задаёт колонки Excel для импорта справочника водителей (0-based).
type DriversImportConfig struct {
	AgentNameCol  int
	DriverNameCol int
	CarNumberCol  int
	PhoneCol      int
	CityCodesCol  int // -1 если нет колонки с кодами городов
	HeaderRows    int
	Sheet         string
}

// DefaultDriversImportConfig — порядок колонок как в образце Минздрава и drivers_summary:
// 0 = имя водителя (агент/ключ и имя для отчёта), 1 = מס.רכב (номер машины/лицензия), 2 = טלפון נהג, 3 = коды городов.
func DefaultDriversImportConfig() DriversImportConfig {
	return DriversImportConfig{
		AgentNameCol:  0, // имя водителя как ключ
		DriverNameCol: 0, // שם הנהג — то же имя для отчёта
		CarNumberCol:  1, // מס.רכב — номер машины (лицензия)
		PhoneCol:      2, // טלפון נהג
		CityCodesCol:  3, // коды городов доставки
		HeaderRows:    1,
		Sheet:         "",
	}
}

// ImportDriversFromExcel читает Excel и выполняет Upsert водителей в БД (последняя запись по agent_name побеждает).
func ImportDriversFromExcel(database *db.DB, path string, cfg DriversImportConfig) (imported int, err error) {
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

	lastByAgent := make(map[string]domain.Driver)
	for i := cfg.HeaderRows; i < len(rows); i++ {
		row := rows[i]
		d := domain.Driver{
			AgentName:  domain.NormalizeText(cell(row, cfg.AgentNameCol)),
			DriverName: domain.NormalizeText(cell(row, cfg.DriverNameCol)),
			CarNumber:  domain.NormalizeText(cell(row, cfg.CarNumberCol)),
			Phone:      domain.NormalizeText(cell(row, cfg.PhoneCol)),
		}
		if cfg.CityCodesCol >= 0 {
			d.CityCodes = domain.NormalizeText(cell(row, cfg.CityCodesCol))
			// Убрать кавычки и скобки (формат ['F1381','F2376'] или F1381,F2376)
			if d.CityCodes != "" {
				d.CityCodes = strings.Trim(d.CityCodes, "[]'\" ")
				d.CityCodes = strings.ReplaceAll(d.CityCodes, "'", "")
			}
		}
		if d.AgentName == "" {
			continue
		}
		lastByAgent[d.AgentName] = d
	}

	list := make([]domain.Driver, 0, len(lastByAgent))
	for _, d := range lastByAgent {
		list = append(list, d)
	}
	if len(list) == 0 {
		return 0, nil
	}
	if err := database.BulkUpsertDrivers(list); err != nil {
		return 0, fmt.Errorf("запись в БД: %w", err)
	}
	return len(list), nil
}
