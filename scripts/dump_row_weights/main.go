package main

import (
	"fmt"
	"os"

	"github.com/xuri/excelize/v2"
)

func main() {
	path := `c:\Users\office3\Downloads\FCS_Report_2026-05-17_14-37-11.xlsx`
	if len(os.Args) > 1 {
		path = os.Args[1]
	}
	f, err := excelize.OpenFile(path)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer f.Close()
	sh := f.GetSheetName(0)
	for row := 2; row <= 15; row++ {
		inv, _ := f.GetCellValue(sh, mustCell(14, row))
		if inv == "" {
			break
		}
		fmt.Printf("\ninv %s:\n", inv)
		for col := 15; col <= 27; col++ {
			v, _ := f.GetCellValue(sh, mustCell(col, row))
			if v != "" && v != "0" {
				h, _ := f.GetCellValue(sh, mustCell(col, 1))
				fmt.Printf("  col %d %s = %s\n", col, h, v)
			}
		}
	}
}

func mustCell(col, row int) string {
	c, _ := excelize.CoordinatesToCellName(col, row)
	return c
}
