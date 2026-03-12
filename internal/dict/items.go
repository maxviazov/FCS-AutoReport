package dict

import (
	"fmt"

	"fcs-autoreport/internal/domain"
	"fcs-autoreport/internal/db"

	"github.com/xuri/excelize/v2"
)

// ItemsImportConfig задаёт колонки Excel для импорта справочника товаров (0-based).
type ItemsImportConfig struct {
	ItemCodeCol int
	CategoryCol int
	HeaderRows  int
	Sheet       string
}

// DefaultItemsImportConfig — артикул 0, категория 1, одна строка заголовка.
func DefaultItemsImportConfig() ItemsImportConfig {
	return ItemsImportConfig{
		ItemCodeCol: 0,
		CategoryCol: 1,
		HeaderRows:  1,
		Sheet:       "",
	}
}

// ImportItemsFromExcel читает Excel и выполняет Upsert товаров в БД (последняя запись по артикулу побеждает).
func ImportItemsFromExcel(database *db.DB, path string, cfg ItemsImportConfig) (imported int, err error) {
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

	lastByCode := make(map[string]domain.Item)
	for i := cfg.HeaderRows; i < len(rows); i++ {
		row := rows[i]
		it := domain.Item{
			ItemCode: domain.NormalizeText(cell(row, cfg.ItemCodeCol)),
			Category: domain.NormalizeText(cell(row, cfg.CategoryCol)),
		}
		if it.ItemCode == "" {
			continue
		}
		lastByCode[it.ItemCode] = it
	}

	list := make([]domain.Item, 0, len(lastByCode))
	for _, it := range lastByCode {
		list = append(list, it)
	}
	if len(list) == 0 {
		return 0, nil
	}
	if err := database.BulkUpsertItems(list); err != nil {
		return 0, fmt.Errorf("запись в БД: %w", err)
	}
	return len(list), nil
}
