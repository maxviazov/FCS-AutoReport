// +build ignore

package main

import (
	"fmt"
	"os"

	"github.com/xuri/excelize/v2"
)

func main() {
	samplePath := os.Args[1]
	ourPath := os.Args[2]
	if samplePath == "" || ourPath == "" {
		fmt.Println("usage: go run compare_xlsx.go <sample.xlsx> <our_report.xlsx>")
		return
	}
	sample, err := excelize.OpenFile(samplePath)
	if err != nil {
		fmt.Println("sample open:", err)
		return
	}
	defer sample.Close()
	our, err := excelize.OpenFile(ourPath)
	if err != nil {
		fmt.Println("our open:", err)
		return
	}
	defer our.Close()

	sh := sample.GetSheetName(0)
	// row1 = first data row (index 1)
	for col := 1; col <= 28; col++ {
		cell, _ := excelize.CoordinatesToCellName(col, 2)
		vSample, _ := sample.GetCellValue(sh, cell)
		vOur, _ := our.GetCellValue(sh, cell)
		if vSample != vOur {
			fmt.Printf("col %2d: sample %q  vs  our %q\n", col, vSample, vOur)
		} else {
			fmt.Printf("col %2d: %q\n", col, vSample)
		}
	}
}
