package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/xuri/excelize/v2"
)

func cell(f *excelize.File, sh string, c, r int) string {
	n, _ := excelize.CoordinatesToCellName(c, r)
	v, _ := f.GetCellValue(sh, n)
	return strings.TrimSpace(v)
}

func main() {
	files := []string{
		`c:\Users\office3\Downloads\fish_reports_ready\מעדנייתאלמוגמשטבעמאשדוד_515020121.xlsx`,
		`c:\Users\office3\Downloads\fish_reports_ready\מנמאירשיווקומסחרבעמסניףאשדודדליתשח_513752089.xlsx`,
		`c:\Users\office3\Downloads\fish_reports_ready\נטליאשדוד_306131368.xlsx`,
	}
	if len(os.Args) > 1 {
		files = os.Args[1:]
	}
	for _, path := range files {
		f, err := excelize.OpenFile(path)
		if err != nil {
			fmt.Println(path, err)
			continue
		}
		sh := f.GetSheetName(0)
		fmt.Printf("\n=== %s ===\n", path[strings.LastIndex(path, `\`)+1:])
		for col := 1; col <= 14; col++ {
			v := cell(f, sh, col, 2)
			if v != "" {
				h := cell(f, sh, col, 1)
				fmt.Printf("  %2d %-8s %q\n", col, truncate(h, 20), v)
			}
		}
		// raw bytes in address
		addr := cell(f, sh, 11, 2)
		fmt.Printf("  addr runes: ")
		for _, r := range addr {
			if r == '\\' || r == '/' || r == '"' {
				fmt.Printf("[%U]", r)
			}
		}
		fmt.Println()
		f.Close()
	}
}

func truncate(s string, n int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) <= n {
		return s
	}
	return s[:n]
}
