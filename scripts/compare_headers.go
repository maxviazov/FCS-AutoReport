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
		fmt.Println("usage: go run compare_headers.go <sample.xlsx> <our_report.xlsx>")
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

	shSample := sample.GetSheetName(0)
	shOur := our.GetSheetName(0)

	fmt.Println("=== ROW 1 (headers) - SAMPLE (Ministry) ===")
	for col := 1; col <= 30; col++ {
		cell, _ := excelize.CoordinatesToCellName(col, 1)
		v, _ := sample.GetCellValue(shSample, cell)
		if v != "" {
			fmt.Printf("  col %2d: %q\n", col, v)
		}
	}
	fmt.Println("\n=== ROW 2 (first data) - SAMPLE (Ministry) ===")
	for col := 1; col <= 30; col++ {
		cell, _ := excelize.CoordinatesToCellName(col, 2)
		v, _ := sample.GetCellValue(shSample, cell)
		fmt.Printf("  col %2d: %q\n", col, v)
	}

	fmt.Println("\n=== ROW 1 (headers) - OUR REPORT ===")
	for col := 1; col <= 30; col++ {
		cell, _ := excelize.CoordinatesToCellName(col, 1)
		v, _ := our.GetCellValue(shOur, cell)
		if v != "" {
			fmt.Printf("  col %2d: %q\n", col, v)
		}
	}
	fmt.Println("\n=== ROW 2 (first data) - OUR REPORT ===")
	for col := 1; col <= 30; col++ {
		cell, _ := excelize.CoordinatesToCellName(col, 2)
		v, _ := our.GetCellValue(shOur, cell)
		fmt.Printf("  col %2d: %q\n", col, v)
	}
}
