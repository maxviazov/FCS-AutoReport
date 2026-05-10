package main

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/xuri/excelize/v2"
	_ "modernc.org/sqlite"
)

func main() {
	reportPath := `c:\Users\office3\Downloads\FCS_Report_2026-03-29_18-09-33.xlsx`
	if len(os.Args) >= 2 {
		reportPath = os.Args[1]
	}
	dbPath := filepath.Join(os.Getenv("APPDATA"), "FCS-AutoReport", "fcs_autoreport.db")
	if len(os.Args) >= 3 {
		dbPath = os.Args[2]
	}

	codesFromReport := cityCodesFromReport(reportPath)
	if len(codesFromReport) == 0 {
		fmt.Println("no city codes in report")
		os.Exit(1)
	}

	dbCodes, err := loadCodesFromDB(dbPath)
	if err != nil {
		fmt.Println("db:", err)
		os.Exit(1)
	}

	fmt.Println("Report file:", reportPath)
	fmt.Println("Database:", dbPath)
	fmt.Println("Unique קוד עיר in report:", len(codesFromReport))
	fmt.Println("Unique codes in cities table:", len(dbCodes))
	fmt.Println()

	var missing []string
	for c := range codesFromReport {
		if !dbCodes[c] {
			missing = append(missing, c)
		}
	}
	sort.Strings(missing)
	if len(missing) == 0 {
		fmt.Println("All report city codes exist in the imported cities table (SQLite).")
		names, err := namesByCode(dbPath, codesFromReport)
		if err == nil && len(names) > 0 {
			fmt.Println("\nקוד עיר → שם בעיריות (מהטבלה שיובאה ל־DB):")
			var keys []string
			for c := range codesFromReport {
				keys = append(keys, c)
			}
			sort.Strings(keys)
			for _, c := range keys {
				fmt.Printf("  %s → %s\n", c, strings.Join(names[c], "; "))
			}
		}
	} else {
		fmt.Println("Report codes NOT found in cities.code:")
		for _, c := range missing {
			fmt.Println(" ", c, "(rows:", strings.Join(codesFromReport[c], ", "), ")")
		}
	}

	// Optional: codes in DB never used in this report (informational)
	inReport := make(map[string]bool, len(codesFromReport))
	for c := range codesFromReport {
		inReport[c] = true
	}
	var unused []string
	for c := range dbCodes {
		if !inReport[c] {
			unused = append(unused, c)
		}
	}
	if len(unused) > 200 {
		fmt.Printf("\n(Info: %d DB codes not appearing in this report — list omitted)\n", len(unused))
	}
}

func cityCodesFromReport(path string) map[string][]string {
	f, err := excelize.OpenFile(path)
	if err != nil {
		fmt.Println("open report:", err)
		os.Exit(1)
	}
	defer f.Close()
	sheet := f.GetSheetName(0)
	rows, _ := f.GetRows(sheet)
	if len(rows) < 2 {
		return nil
	}
	h := rows[0]
	// First match only: last header cell often mentions "תעודת משלוח" in cancellation text.
	cityCol := -1
	invCol := -1
	for j, cell := range h {
		n := strings.TrimSpace(cell)
		if cityCol < 0 && strings.Contains(n, "קוד") && strings.Contains(n, "עיר") {
			cityCol = j
		}
		if invCol < 0 && strings.Contains(n, "תעודת") && strings.Contains(n, "משלוח") {
			invCol = j
		}
	}
	if cityCol < 0 || invCol < 0 {
		fmt.Println("could not find city or invoice column")
		os.Exit(1)
	}
	out := make(map[string][]string)
	for i := 1; i < len(rows); i++ {
		r := rows[i]
		inv := cell(r, invCol)
		if inv == "" {
			continue
		}
		code := strings.TrimSpace(cell(r, cityCol))
		if code == "" {
			continue
		}
		out[code] = append(out[code], inv)
	}
	return out
}

func cell(row []string, col int) string {
	if col < 0 || col >= len(row) {
		return ""
	}
	return strings.TrimSpace(row[col])
}

func namesByCode(dbPath string, codes map[string][]string) (map[string][]string, error) {
	dsn := "file:" + filepath.ToSlash(dbPath) + "?mode=ro"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	out := make(map[string][]string)
	for c := range codes {
		rows, err := db.Query(`SELECT DISTINCT TRIM(name) FROM cities WHERE TRIM(code) = ? ORDER BY name`, c)
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			var n string
			if rows.Scan(&n) != nil {
				continue
			}
			n = strings.TrimSpace(n)
			if n != "" {
				out[c] = append(out[c], n)
			}
		}
		rows.Close()
	}
	return out, nil
}

func loadCodesFromDB(path string) (map[string]bool, error) {
	dsn := path
	if !strings.HasPrefix(dsn, "file:") {
		dsn = "file:" + filepath.ToSlash(path) + "?mode=ro"
	}
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	rows, err := db.Query(`SELECT DISTINCT TRIM(code) FROM cities WHERE TRIM(code) != ''`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	m := make(map[string]bool)
	for rows.Next() {
		var c string
		if rows.Scan(&c) != nil {
			continue
		}
		m[strings.TrimSpace(c)] = true
	}
	return m, rows.Err()
}
