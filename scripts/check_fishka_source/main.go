package main

import (
	"fmt"
	"os"
	"sort"
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
	rows, _ := f.GetRows(f.GetSheetName(0))
	if len(rows) < 2 {
		fmt.Println("no data")
		return
	}

	type inv struct {
		hp, client, addr, district, date string
		lines                            int
	}
	byInv := make(map[string]*inv)
	for i := 1; i < len(rows); i++ {
		r := rows[i]
		get := func(j int) string {
			if j < len(r) {
				return strings.TrimSpace(r[j])
			}
			return ""
		}
		num := get(4) // אסמכתת בסיס (0-based col 5)
		if num == "" {
			continue
		}
		e, ok := byInv[num]
		if !ok {
			e = &inv{
				hp:       get(9),
				client:   get(11),
				addr:     get(12),
				district: get(7),
				date:     get(5),
			}
			byInv[num] = e
		}
		e.lines++
	}

	keys := make([]string, 0, len(byInv))
	for k := range byInv {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	fmt.Println("file:", path)
	fmt.Printf("data lines: %d, unique invoices (אסמכתת בסיס): %d\n\n", len(rows)-1, len(keys))

	for _, num := range keys {
		e := byInv[num]
		city := domain.ExtractCityFromAddress(e.addr)
		_, after, _ := strings.Cut(e.addr, ",")
		fmt.Printf("invoice %s  date=%s  lines=%d\n", num, e.date, e.lines)
		fmt.Printf("  HP=%s  district=%q\n", e.hp, e.district)
		fmt.Printf("  client=%q\n", e.client)
		fmt.Printf("  addr=%q\n", e.addr)
		fmt.Printf("  → city(before comma)=%q  street/after=%q\n", city, strings.TrimSpace(after))
		fmt.Printf("  → MoH street (cityAfterComma=false)=%q\n",
			domain.MoHStreetLineForMoH(domain.NormalizeMinistryAddress(e.addr), false))
		fmt.Printf("  → AllowMoHN61=%v\n", domain.AllowMoHN61CityCode(e.addr, e.client, e.district, ""))
		var flags []string
		if strings.Contains(e.addr, `\`) {
			flags = append(flags, "backslash in address")
		}
		if strings.Contains(e.client, "דמיטרי") || strings.Contains(e.client, "לוין") {
			flags = append(flags, "client looks like personal name")
		}
		if strings.Contains(e.addr, "מרכז קניות") && !strings.ContainsAny(after, "0123456789") {
			flags = append(flags, "address is shopping-center name, no street number")
		}
		if len([]rune(domain.MoHStreetLineForMoH(domain.NormalizeMinistryAddress(e.addr), false))) < 6 {
			flags = append(flags, "very short MoH street line")
		}
		if len(flags) > 0 {
			fmt.Printf("  ⚠ %s\n", strings.Join(flags, "; "))
		}
		fmt.Println()
	}
}
