package main

import (
	"fmt"
	"os"
	"strings"

	"fcs-autoreport/internal/domain"

	"github.com/xuri/excelize/v2"
)

func main() {
	path := `c:\Users\office3\Desktop\FishKA\source\משקל.xlsx`
	if len(os.Args) > 1 {
		path = os.Args[1]
	}
	f, err := excelize.OpenFile(path)
	if err != nil {
		fmt.Println("open:", err)
		os.Exit(1)
	}
	defer f.Close()
	sheet := f.GetSheetName(0)
	rows, _ := f.GetRows(sheet)
	if len(rows) < 1 {
		fmt.Println("empty sheet")
		return
	}
	h := rows[0]
	fmt.Println("file:", path)
	fmt.Println("sheet:", sheet, "rows:", len(rows))
	// find address-like columns
	var cols []int
	for j, cell := range h {
		n := strings.TrimSpace(cell)
		low := strings.ToLower(n)
		if n == "" {
			continue
		}
		if strings.Contains(n, "כתובת") || strings.Contains(n, "адрес") ||
			strings.Contains(low, "address") || strings.Contains(n, "מען") {
			cols = append(cols, j)
		}
	}
	if len(cols) == 0 {
		fmt.Println("\nNo column matched כתובת/address — printing all headers:")
		for j, cell := range h {
			fmt.Printf("  %d\t%s\n", j, cell)
		}
		return
	}
	fmt.Println("\nAddress columns (0-based):", cols)
	for _, j := range cols {
		fmt.Printf("  [%d] %q\n", j, cellStr(h, j))
	}

	var issues []string
	for i := 1; i < len(rows); i++ {
		r := rows[i]
		rowNum := i + 1
		for _, j := range cols {
			addr := strings.TrimSpace(cellStr(r, j))
			if addr == "" {
				continue
			}
			if !strings.Contains(addr, ",") {
				issues = append(issues, fmt.Sprintf("row %d col %d: no comma (city/street split unclear): %q", rowNum, j, truncate(addr, 80)))
			}
			city := domain.ExtractCityFromAddress(addr)
			if city == "" {
				issues = append(issues, fmt.Sprintf("row %d col %d: empty city after extract: %q", rowNum, j, truncate(addr, 80)))
			}
			street := streetAfterComma(addr)
			if street == "" && strings.Contains(addr, ",") {
				issues = append(issues, fmt.Sprintf("row %d col %d: empty street part: %q", rowNum, j, truncate(addr, 80)))
			}
		}
	}

	// stats
	var emptyAddr, noComma, ok int
	for i := 1; i < len(rows); i++ {
		r := rows[i]
		for _, j := range cols {
			addr := strings.TrimSpace(cellStr(r, j))
			if addr == "" {
				emptyAddr++
				continue
			}
			if !strings.Contains(addr, ",") {
				noComma++
				continue
			}
			ok++
		}
	}
	fmt.Printf("\nStats (data rows %d): empty address cells=%d, no comma=%d, with comma=%d\n", len(rows)-1, emptyAddr, noComma, ok)

	cityFreq := make(map[string]int)
	var badMultSign []string
	for i := 1; i < len(rows); i++ {
		r := rows[i]
		for _, j := range cols {
			addr := strings.TrimSpace(cellStr(r, j))
			if addr == "" {
				continue
			}
			cityFreq[domain.ExtractCityFromAddress(addr)]++
			if strings.ContainsRune(addr, '\u00d7') {
				badMultSign = append(badMultSign, fmt.Sprintf("row %d: %q", i+1, addr))
			}
		}
	}
	fmt.Println("\nCities in file (for city-code lookup), count:")
	keys := make([]string, 0, len(cityFreq))
	for c := range cityFreq {
		keys = append(keys, c)
	}
	// simple sort
	for i := 0; i < len(keys); i++ {
		for j := i + 1; j < len(keys); j++ {
			if keys[j] < keys[i] {
				keys[i], keys[j] = keys[j], keys[i]
			}
		}
	}
	for _, c := range keys {
		fmt.Printf("  %q — %d\n", c, cityFreq[c])
	}
	if len(badMultSign) > 0 {
		fmt.Println("\nWARNING: U+00D7 (×) in address — often wrong char instead of Hebrew; rows:")
		for _, s := range badMultSign {
			fmt.Println(" -", s)
		}
	}

	fmt.Println("\n--- Sample rows (first 12 non-empty per address col) ---")
	shown := 0
	for i := 1; i < len(rows) && shown < 12; i++ {
		r := rows[i]
		parts := make([]string, 0, len(cols))
		empty := true
		for _, j := range cols {
			a := strings.TrimSpace(cellStr(r, j))
			if a != "" {
				empty = false
			}
			city := domain.ExtractCityFromAddress(a)
			st := streetAfterComma(a)
			parts = append(parts, fmt.Sprintf("[%d] city=%q street=%q raw=%q", j, city, truncate(st, 40), truncate(a, 50)))
		}
		if !empty {
			fmt.Printf("row %d: %s\n", i+1, strings.Join(parts, " | "))
			shown++
		}
	}

	if len(issues) == 0 {
		fmt.Println("\nNo structural flags: commas present, city/street parts non-empty (where address non-empty).")
	} else {
		fmt.Printf("\nPotential issues (%d), first 50:\n", len(issues))
		n := len(issues)
		if n > 50 {
			n = 50
		}
		for _, s := range issues[:n] {
			fmt.Println("-", s)
		}
	}
}

func cellStr(row []string, j int) string {
	if j < 0 || j >= len(row) {
		return ""
	}
	return row[j]
}

func streetAfterComma(addr string) string {
	_, after, ok := strings.Cut(strings.TrimSpace(addr), ",")
	if !ok {
		return ""
	}
	return strings.TrimSpace(after)
}

func truncate(s string, max int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) <= max {
		return s
	}
	return s[:max] + "…"
}
