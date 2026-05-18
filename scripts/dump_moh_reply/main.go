package main

import (
	"fmt"
	"os"

	"github.com/xuri/excelize/v2"
)

func main() {
	for _, path := range os.Args[1:] {
		f, err := excelize.OpenFile(path)
		if err != nil {
			fmt.Println(path, err)
			continue
		}
		sh := f.GetSheetName(0)
		rows, _ := f.GetRows(sh)
		fmt.Printf("\n=== %s (%d rows) ===\n", path, len(rows))
		for i, r := range rows {
			if i > 25 {
				break
			}
			fmt.Printf("row %d (%d cols):\n", i+1, len(r))
			for j, v := range r {
				if v != "" {
					fmt.Printf("  %d %q\n", j, v)
				}
			}
		}
		f.Close()
	}
}
