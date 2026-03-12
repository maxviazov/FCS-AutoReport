// +build ignore

package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/xuri/excelize/v2"
)

func main() {
	rawPath := os.Args[1]
	reportPath := os.Args[2]
	if rawPath == "" || reportPath == "" {
		fmt.Println("usage: go run check_weights.go <raw.xlsx> <report.xlsx>")
		return
	}
	raw, err := excelize.OpenFile(rawPath)
	if err != nil {
		fmt.Println("raw open:", err)
		return
	}
	defer raw.Close()
	report, err := excelize.OpenFile(reportPath)
	if err != nil {
		fmt.Println("report open:", err)
		return
	}
	defer report.Close()

	sh := raw.GetSheetName(0)
	rows, _ := raw.GetRows(sh)
	if len(rows) < 2 {
		fmt.Println("raw: too few rows")
		return
	}
	// Find weight and boxes columns by header
	weightCol := -1
	boxesCol := -1
	for j, cell := range rows[0] {
		n := strings.TrimSpace(cell)
		if strings.Contains(n, "משקל") {
			weightCol = j
		}
		if strings.Contains(n, "קרטון") {
			boxesCol = j
		}
	}
	fmt.Printf("Raw file: weight col index %d (header %q), boxes col %d\n", weightCol, safeHeader(rows[0], weightCol), boxesCol)
	if weightCol < 0 {
		fmt.Println("Weight column not found in raw, checking col 24 (0-based)")
		weightCol = 24
	}
	// Sample first 15 data rows: raw weight value and expected kg
	fmt.Println("\n--- Raw file: first 15 data rows (weight column value) ---")
	for i := 1; i <= 15 && i < len(rows); i++ {
		row := rows[i]
		wStr := ""
		if weightCol < len(row) {
			wStr = strings.TrimSpace(row[weightCol])
		}
		bStr := ""
		if boxesCol >= 0 && boxesCol < len(row) {
			bStr = strings.TrimSpace(row[boxesCol])
		}
		wVal, _ := strconv.ParseFloat(wStr, 64)
		// Expected kg: >=100 -> /1000, [1,100) -> /100, <1 as is
		var expectedKg float64
		if wVal >= 100 {
			expectedKg = wVal / 1000
		} else if wVal >= 1 {
			expectedKg = wVal / 100
		} else {
			expectedKg = wVal
		}
		fmt.Printf("  row %2d: raw weight=%q (%.2f) -> expected kg=%.3f  boxes=%q\n", i+1, wStr, wVal, expectedKg, bStr)
	}
	// Report: columns 15-27 (category weights, boxes, total weight)
	fmt.Println("\n--- Report: first 5 data rows, cols 23,26,27 (category sample, boxes, total weight) ---")
	rsh := report.GetSheetName(0)
	rrows, _ := report.GetRows(rsh)
	for i := 1; i <= 5 && i < len(rrows); i++ {
		row := rrows[i]
		v23 := ""
		v26 := ""
		v27 := ""
		if len(row) > 22 {
			v23 = row[22]
		}
		if len(row) > 25 {
			v26 = row[25]
		}
		if len(row) > 26 {
			v27 = row[26]
		}
		fmt.Printf("  row %2d: col23=%q col26(boxes)=%q col27(total kg)=%q\n", i+1, v23, v26, v27)
	}
}

func safeHeader(row []string, j int) string {
	if j < 0 || j >= len(row) {
		return ""
	}
	return row[j]
}
