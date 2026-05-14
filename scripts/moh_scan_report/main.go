// Печать проверок по xlsx отчёту МОЗ (как перед отправкой).
package main

import (
	"fmt"
	"os"
	"strings"

	"fcs-autoreport/internal/app"

	"github.com/xuri/excelize/v2"
)

func main() {
	path := `c:\Users\office3\Downloads\FCS_Report_2026-05-14_19-55-35.xlsx`
	if len(os.Args) > 1 {
		path = os.Args[1]
	}
	if err := app.ValidateMoHReportFile(path); err != nil {
		fmt.Println("ValidateMoHReportFile:", err)
	} else {
		fmt.Println("ValidateMoHReportFile: OK (наш контур не нашёл блокирующих замечаний)")
	}

	f, err := excelize.OpenFile(path)
	if err != nil {
		fmt.Println("open:", err)
		os.Exit(1)
	}
	defer f.Close()
	sh := f.GetSheetName(0)
	dim, _ := f.GetSheetDimension(sh)
	fmt.Println("sheet:", sh, "dimension:", dim)

	rows, _ := f.GetRows(sh)
	last := len(rows)
	if dim != "" {
		parts := strings.Split(dim, ":")
		if len(parts) == 2 {
			_, r, err := excelize.CellNameToCoordinates(parts[1])
			if err == nil && r > last {
				last = r
			}
		}
	}
	fmt.Println("data rows scan 2..", last)
	for row := 2; row <= last && row <= 80; row++ {
		cell := func(col int) string {
			c, _ := excelize.CoordinatesToCellName(col, row)
			v, _ := f.GetCellValue(sh, c)
			return strings.TrimSpace(v)
		}
		sup, hp := cell(1), cell(12)
		if sup == "" && hp == "" {
			continue
		}
		z, _ := excelize.CoordinatesToCellName(26, row)
		fm, _ := f.GetCellFormula(sh, z)
		fmt.Printf("--- row %d inv=%q city=%q client=%q Z=%q formulaZ=%q AA=%q\n",
			row, cell(14), cell(10), cell(8), cell(26), fm, cell(27))
	}
}
