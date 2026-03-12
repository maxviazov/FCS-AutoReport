// +build ignore

package main

import (
	"fmt"
	"os"

	"github.com/xuri/excelize/v2"
)

func main() {
	path := os.Args[1]
	if path == "" {
		path = `c:\Users\office3\Desktop\FishKA\final\אביגו 2 ירושלים.xlsx`
	}
	f, err := excelize.OpenFile(path)
	if err != nil {
		fmt.Println("open err:", err)
		return
	}
	defer f.Close()
	for i, n := range f.GetSheetList() {
		fmt.Printf("sheet %d: %s\n", i, n)
	}
	sh := f.GetSheetName(0)
	rows, err := f.GetRows(sh)
	if err != nil {
		fmt.Println("rows err:", err)
		return
	}
	fmt.Printf("rows count: %d\n", len(rows))
	if len(rows) > 0 {
		fmt.Printf("row0 (headers) cells: %d\n", len(rows[0]))
		for j, c := range rows[0] {
			fmt.Printf("  col %2d: %q\n", j+1, c)
		}
	}
	if len(rows) > 1 {
		fmt.Printf("row1 (first data) cells: %d\n", len(rows[1]))
		for j, c := range rows[1] {
			if c != "" {
				fmt.Printf("  col %2d: %q\n", j+1, c)
			}
		}
	}
}
