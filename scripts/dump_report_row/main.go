package main

import (
	"fmt"
	"os"

	"github.com/xuri/excelize/v2"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("usage: dump_report_row <xlsx>")
		os.Exit(1)
	}
	f, err := excelize.OpenFile(os.Args[1])
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer f.Close()
	sh := f.GetSheetName(0)
	rows, _ := f.GetRows(sh)
	for i := 0; i < len(rows) && i < 3; i++ {
		fmt.Printf("--- row %d (%d cols) ---\n", i+1, len(rows[i]))
		for j, v := range rows[i] {
			if v != "" {
				fmt.Printf("%2d %q\n", j+1, v)
			}
		}
	}
}
