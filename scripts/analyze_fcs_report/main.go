package main

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"unicode"

	"github.com/xuri/excelize/v2"
)

func norm(s string) string {
	var b strings.Builder
	for _, r := range strings.TrimSpace(s) {
		if unicode.IsSpace(r) {
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

// colByHeader finds 0-based column index: first header cell whose norm contains all non-empty keywords.
func colByHeader(header []string, keywords ...string) int {
outer:
	for j, cell := range header {
		n := norm(cell)
		if n == "" {
			continue
		}
		for _, kw := range keywords {
			if kw == "" {
				continue
			}
			if !strings.Contains(n, norm(kw)) {
				continue outer
			}
		}
		return j
	}
	return -1
}

func digitsOnly(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("usage: go run main.go <path.xlsx>")
		os.Exit(1)
	}
	path := os.Args[1]
	f, err := excelize.OpenFile(path)
	if err != nil {
		fmt.Println("open:", err)
		os.Exit(1)
	}
	defer f.Close()
	sheet := f.GetSheetName(0)
	rows, _ := f.GetRows(sheet)
	if len(rows) < 2 {
		fmt.Println("no data rows")
		return
	}
	h := rows[0]
	fmt.Println("file:", path)
	fmt.Println("sheet:", sheet, "data rows:", len(rows)-1)

	// Map columns by Hebrew headers (template order may vary slightly).
	clientCol := colByHeader(h, "לקוח")
	if clientCol >= 0 && strings.Contains(norm(h[clientCol]), "סוג") {
		clientCol = -1
	}
	if clientCol < 0 {
		for j, cell := range h {
			n := norm(cell)
			if strings.Contains(n, "לקוח") && !strings.Contains(n, "סוג") {
				clientCol = j
				break
			}
		}
	}
	cityCol := colByHeader(h, "קוד", "עיר")
	streetCol := colByHeader(h, "כתובת")
	// ח"פ לקוח — do not match סוג לקוח (מחסן/מפעל contain ח/פ as letters in other words)
	hpCol := -1
	for j, cell := range h {
		n := strings.TrimSpace(cell)
		if strings.Contains(n, "ספק") && !strings.Contains(n, "לקוח") {
			continue
		}
		if !strings.Contains(n, "לקוח") {
			continue
		}
		if strings.Contains(n, "סוג") {
			continue
		}
		hasChp := strings.Contains(n, "ח\"") || strings.Contains(n, "ח״") || strings.Contains(n, "ח''")
		if hasChp {
			hpCol = j
			break
		}
	}
	invCol := colByHeader(h, "תעודת", "משלוח")
	if invCol < 0 {
		invCol = colByHeader(h, "מספר", "תעודת")
	}
	vehCol := colByHeader(h, "רכב")
	drvCol := colByHeader(h, "נהג")
	if drvCol >= 0 && strings.Contains(norm(h[drvCol]), "טלפון") {
		drvCol = -1
	}
	if drvCol < 0 {
		drvCol = colByHeader(h, "שם", "נהג")
	}
	phoneCol := colByHeader(h, "טלפון", "נהג")
	if phoneCol < 0 {
		phoneCol = colByHeader(h, "טלפון")
	}

	fmt.Println("column map (0-based):",
		"לקוח", clientCol, "קוד עיר", cityCol, "כתובת", streetCol, "ח.פ לקוח", hpCol, "תעודה", invCol,
		"רכב", vehCol, "נהג", drvCol, "טלפון", phoneCol)

	get := func(r []string, j int) string {
		if j >= 0 && j < len(r) {
			return strings.TrimSpace(r[j])
		}
		return ""
	}

	invRe := regexp.MustCompile(`^\d+$`)
	cityRe := regexp.MustCompile(`^\d{1,5}$`)
	seenInv := make(map[string]int)
	var issues []string

	for i := 1; i < len(rows); i++ {
		r := rows[i]
		rowNum := i + 1
		inv := get(r, invCol)
		if inv == "" || invCol < 0 {
			continue
		}
		if prev, ok := seenInv[inv]; ok {
			issues = append(issues, fmt.Sprintf("duplicate invoice %s at rows %d and %d", inv, prev, rowNum))
		}
		seenInv[inv] = rowNum

		client := get(r, clientCol)
		city := get(r, cityCol)
		street := get(r, streetCol)
		hp := get(r, hpCol)
		hpDigits := digitsOnly(hp)
		car := get(r, vehCol)
		driver := get(r, drvCol)
		phone := get(r, phoneCol)

		if hpCol < 0 || cityCol < 0 || streetCol < 0 || invCol < 0 {
			issues = append(issues, fmt.Sprintf("row %d: could not map all required columns from header", rowNum))
			continue
		}

		if hp == "" || hp == "0" {
			issues = append(issues, fmt.Sprintf("row %d invoice %s: empty or zero ח.פ (נקודת שיווק)", rowNum, inv))
		} else if len(hpDigits) != 9 || !invRe.MatchString(hpDigits) {
			issues = append(issues, fmt.Sprintf("row %d invoice %s: ח.פ not 9 digits (raw %q → %q)", rowNum, inv, hp, hpDigits))
		}
		if city == "" {
			issues = append(issues, fmt.Sprintf("row %d invoice %s: empty קוד עיר", rowNum, inv))
		} else if !cityRe.MatchString(city) {
			issues = append(issues, fmt.Sprintf("row %d invoice %s: קוד עיר looks unusual %q", rowNum, inv, city))
		}
		if street == "" {
			issues = append(issues, fmt.Sprintf("row %d invoice %s: empty כתובת", rowNum, inv))
		}
		if client == "" {
			issues = append(issues, fmt.Sprintf("row %d invoice %s: empty לקוח", rowNum, inv))
		}
		if vehCol >= 0 && drvCol >= 0 && phoneCol >= 0 {
			if car == "" || driver == "" || phone == "" {
				issues = append(issues, fmt.Sprintf("row %d invoice %s: incomplete logistics (car=%q driver=%q phone=%q)", rowNum, inv, car, driver, phone))
			}
		}

		// weight columns: find first header containing "בשר" for category block start, else fixed 14..24
		wStart := 14
		for j, cell := range h {
			if strings.Contains(norm(cell), "בשר") && strings.Contains(norm(cell), "בהמות") {
				wStart = j
				break
			}
		}
		var wsum float64
		for c := wStart; c < len(h) && c <= wStart+10 && c < len(r); c++ {
			var v float64
			_, _ = fmt.Sscanf(strings.TrimSpace(r[c]), "%f", &v)
			wsum += v
		}
		if wsum <= 0 {
			issues = append(issues, fmt.Sprintf("row %d invoice %s: zero category weights (from col index %d)", rowNum, inv, wStart))
		}
	}

	if len(issues) == 0 {
		fmt.Println("\nNo obvious structural issues (mapped columns, 9-digit ח.פ, city code, address, logistics, weights).")
		fmt.Println("MoH message «פרטי … מנקודות השיווק לא תקינים» often means the ח.פ is wrong or that establishment is not registered as an approved marketing point in their system — the file can look valid locally.")
	} else {
		fmt.Println("\nPotential issues:")
		for _, s := range issues {
			fmt.Println("-", s)
		}
	}
}
