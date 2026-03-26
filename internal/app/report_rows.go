package app

import (
	"fmt"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
)

type reportRow struct {
	InvoiceNum string
	ClientName string
	ClientHP   string
	CityCode   string
}

func readGeneratedReportRows(path string) ([]reportRow, error) {
	f, err := excelize.OpenFile(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	sheet := f.GetSheetName(0)
	if sheet == "" {
		return nil, fmt.Errorf("лист не найден")
	}
	rows, err := f.GetRows(sheet)
	if err != nil {
		return nil, err
	}
	out := make([]reportRow, 0, len(rows))
	for i := 1; i < len(rows); i++ {
		row := rows[i]
		get := func(idx int) string {
			if idx >= 0 && idx < len(row) {
				return strings.TrimSpace(row[idx])
			}
			return ""
		}
		out = append(out, reportRow{
			ClientName: get(7),
			CityCode:   get(9),
			ClientHP:   get(11),
			InvoiceNum: get(13),
		})
	}
	return out, nil
}

func businessDateNow() string {
	return time.Now().Format("2006-01-02")
}
