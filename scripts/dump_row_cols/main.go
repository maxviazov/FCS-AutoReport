package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/xuri/excelize/v2"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Println("usage: dump_row_cols <xlsx> <row1>...")
		os.Exit(1)
	}
	f, err := excelize.OpenFile(os.Args[1])
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer f.Close()
	sh := f.GetSheetName(0)
	for _, a := range os.Args[2:] {
		row, _ := strconv.Atoi(a)
		if row < 1 {
			continue
		}
		fmt.Printf("=== row %d ===\n", row)
		for col := 1; col <= 12; col++ {
			c, _ := excelize.CoordinatesToCellName(col, row)
			v, _ := f.GetCellValue(sh, c)
			if v != "" {
				fmt.Printf("%2d %q\n", col, v)
			}
		}
	}
}
