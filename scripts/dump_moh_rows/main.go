package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/xuri/excelize/v2"
)

func cell(f *excelize.File, sh string, c, r int) string {
	n, _ := excelize.CoordinatesToCellName(c, r)
	v, _ := f.GetCellValue(sh, n)
	return strings.TrimSpace(v)
}

func main() {
	path := `c:\Users\office3\Downloads\FCS_Report_2026-05-17_12-13-45.xlsx`
	if len(os.Args) > 1 {
		path = os.Args[1]
	}
	f, err := excelize.OpenFile(path)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer f.Close()
	sh := f.GetSheetName(0)
	for row := 2; row <= 20; row++ {
		hp := cell(f, sh, 12, row)
		if hp == "" && cell(f, sh, 1, row) == "" {
			break
		}
		fmt.Printf("row %d: inv=%s city=%s addr=%q hp=%s client=%q branch=%s date=%s\n",
			row, cell(f, sh, 14, row), cell(f, sh, 10, row), cell(f, sh, 11, row),
			hp, cell(f, sh, 8, row), cell(f, sh, 13, row), cell(f, sh, 4, row))
	}
}
