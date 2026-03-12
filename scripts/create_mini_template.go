// +build ignore

package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/xuri/excelize/v2"
)

func main() {
	dir := "."
	if len(os.Args) > 1 {
		dir = os.Args[1]
	}
	outPath := filepath.Join(dir, "template_mini.xlsx")
	f := excelize.NewFile()
	sheet := "Sheet1"
	// Row 1: headers as in Ministry template. Column 8 = לקוח (client name).
	headers := []string{"שם הספק", "ח\"פ ספק ", "מספר משרד הבריאות", "תאריך", "מס.רכב", "שם הנהג", "טלפון נהג", "לקוח", "סוג לקוח (קמעונאי,מפעל/מחסן)", "קוד עיר", "כתובת", "ח\"פ לקוח", "מספר סניף הרשת", "מספר תעודת משלוח"}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		_ = f.SetCellValue(sheet, cell, h)
	}
	// Empty row 2 for style copy
	if err := f.SaveAs(outPath); err != nil {
		fmt.Println("err:", err)
		os.Exit(1)
	}
	fmt.Println("Created:", outPath)
}
