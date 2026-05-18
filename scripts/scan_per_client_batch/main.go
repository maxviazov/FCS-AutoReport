package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/xuri/excelize/v2"
)

func cell(f *excelize.File, sh string, c, r int) string {
	n, _ := excelize.CoordinatesToCellName(c, r)
	v, _ := f.GetCellValue(sh, n)
	return strings.TrimSpace(v)
}

func main() {
	dir := `c:\Users\office3\Downloads\fish_reports_ready`
	if len(os.Args) > 1 {
		dir = os.Args[1]
	}
	entries, _ := os.ReadDir(dir)
	type row struct {
		file, inv, city, addr, hp, client, car, driver string
		w24, w22, w20                             string
	}
	var rows []row
	hpCount := make(map[string][]string)
	invSet := make(map[string]string)

	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(strings.ToLower(e.Name()), ".xlsx") {
			continue
		}
		path := filepath.Join(dir, e.Name())
		f, err := excelize.OpenFile(path)
		if err != nil {
			fmt.Println("ERR", e.Name(), err)
			continue
		}
		sh := f.GetSheetName(0)
		inv := cell(f, sh, 14, 2)
		if inv == "" {
			f.Close()
			continue
		}
		r := row{
			file:   e.Name(),
			inv:    inv,
			city:   cell(f, sh, 10, 2),
			addr:   cell(f, sh, 11, 2),
			hp:     cell(f, sh, 12, 2),
			client: cell(f, sh, 8, 2),
			car:    cell(f, sh, 5, 2),
			driver: cell(f, sh, 6, 2),
			w24:    cell(f, sh, 24, 2),
			w22:    cell(f, sh, 22, 2),
			w20:    cell(f, sh, 20, 2),
		}
		rows = append(rows, r)
		hpCount[r.hp] = append(hpCount[r.hp], inv+" "+filepath.Base(e.Name()))
		if prev, ok := invSet[inv]; ok && prev != e.Name() {
			fmt.Printf("DUP INV %s in %s and %s\n", inv, prev, e.Name())
		}
		invSet[inv] = e.Name()
		f.Close()
	}

	fmt.Printf("per-client files: %d\n\n", len(rows))
	fmt.Printf("%-12s %-6s %-28s %-12s %-12s %-14s\n", "invoice", "city", "address", "hp", "car", "driver")
	for _, r := range rows {
		addr := r.addr
		if len([]rune(addr)) > 26 {
			addr = string([]rune(addr)[:24]) + "…"
		}
		fmt.Printf("%-12s %-6s %-28s %-12s %-12s %-14s\n", r.inv, r.city, addr, r.hp, r.car, r.driver)
	}

	fmt.Println("\n--- HP used in multiple files ---")
	for hp, invs := range hpCount {
		if len(invs) > 1 {
			fmt.Printf("HP %s (%d): %v\n", hp, len(invs), invs)
		}
	}

	combined := `c:\Users\office3\Downloads\FCS_Report_2026-05-17_14-37-11.xlsx`
	if _, err := os.Stat(combined); err == nil {
		fmt.Println("\n--- combined 14-37-11 (failed batch) ---")
		f, _ := excelize.OpenFile(combined)
		sh := f.GetSheetName(0)
		for row := 2; row <= 20; row++ {
			inv := cell(f, sh, 14, row)
			if inv == "" {
				break
			}
			fmt.Printf("row %d inv=%s city=%s hp=%s car=%s driver=%s addr=%q\n",
				row, inv, cell(f, sh, 10, row), cell(f, sh, 12, row),
				cell(f, sh, 5, row), cell(f, sh, 6, row), cell(f, sh, 11, row))
		}
		f.Close()
	}

	// rows in combined but not in per-client folder
	fmt.Println("\n--- in combined batch but NOT in fish_reports_ready ---")
	pcInv := make(map[string]bool)
	for _, r := range rows {
		pcInv[r.inv] = true
	}
	f, err := excelize.OpenFile(combined)
	if err == nil {
		sh := f.GetSheetName(0)
		for row := 2; row <= 20; row++ {
			inv := cell(f, sh, 14, row)
			if inv == "" {
				break
			}
			if !pcInv[inv] {
				fmt.Printf("  MISSING per-client: inv=%s hp=%s addr=%q\n", inv, cell(f, sh, 12, row), cell(f, sh, 11, row))
			}
		}
		f.Close()
	}
	fmt.Println("\n--- in per-client but NOT in combined 14-37-11 ---")
	if err == nil {
		f, _ := excelize.OpenFile(combined)
		sh := f.GetSheetName(0)
		cb := make(map[string]bool)
		for row := 2; row <= 20; row++ {
			inv := cell(f, sh, 14, row)
			if inv != "" {
				cb[inv] = true
			}
		}
		f.Close()
		for _, r := range rows {
			if !cb[r.inv] {
				fmt.Printf("  only per-client: inv=%s %s\n", r.inv, r.file)
			}
		}
	}
}
