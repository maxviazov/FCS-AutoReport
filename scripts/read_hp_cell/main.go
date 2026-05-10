package main

import (
	"fmt"
	"os"

	"github.com/xuri/excelize/v2"
)

func main() {
	path := `c:\Users\office3\Downloads\FCS_Report_2026-04-12_12-02-35.xlsx`
	if len(os.Args) > 1 {
		path = os.Args[1]
	}
	f, err := excelize.OpenFile(path)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer f.Close()
	s := f.GetSheetName(0)
	for row := 2; row <= 20; row++ {
		invCell, _ := excelize.CoordinatesToCellName(14, row)
		hpCell, _ := excelize.CoordinatesToCellName(12, row)
		inv, _ := f.GetCellValue(s, invCell)
		hp, _ := f.GetCellValue(s, hpCell)
		t, _ := f.GetCellType(s, hpCell)
		if inv != "" || hp != "" {
			fmt.Printf("row %d: invoice=%q hp=%q cellType=%d\n", row, inv, hp, t)
		}
	}
}
