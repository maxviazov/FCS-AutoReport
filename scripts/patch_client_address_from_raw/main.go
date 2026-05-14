// Патчит колонку «כתובת» (11) в готовом отчёте МОЗ по первой строке сырья с указанным ח"פ.
// Пример:
// go run ./scripts/patch_client_address_from_raw/main.go report.xlsx raw.xlsx 512642182
package main

import (
	"fmt"
	"os"
	"strings"

	"fcs-autoreport/internal/domain"

	"github.com/xuri/excelize/v2"
)

func main() {
	if len(os.Args) != 4 {
		fmt.Println("usage: patch_client_address_from_raw <report.xlsx> <raw.xlsx> <client_hp_digits>")
		os.Exit(1)
	}
	reportPath, rawPath, hp := os.Args[1], os.Args[2], strings.TrimSpace(os.Args[3])
	if hp == "" {
		fmt.Println("empty hp")
		os.Exit(1)
	}

	rawF, err := excelize.OpenFile(rawPath)
	if err != nil {
		fmt.Println("raw open:", err)
		os.Exit(1)
	}
	defer rawF.Close()
	rawSheet := rawF.GetSheetName(0)
	rawRows, err := rawF.GetRows(rawSheet)
	if err != nil {
		fmt.Println("raw rows:", err)
		os.Exit(1)
	}

	var addr string
	for _, row := range rawRows {
		for _, cell := range row {
			if strings.Contains(strings.TrimSpace(cell), hp) {
				if len(row) > 12 {
					addr = domain.NormalizeMinistryAddress(domain.NormalizeText(row[12]))
				}
				break
			}
		}
		if addr != "" {
			break
		}
	}
	if addr == "" {
		fmt.Println("no row with hp", hp, "in", rawPath)
		os.Exit(1)
	}

	repF, err := excelize.OpenFile(reportPath)
	if err != nil {
		fmt.Println("report open:", err)
		os.Exit(1)
	}
	defer repF.Close()
	repSheet := repF.GetSheetName(0)
	cell, err := excelize.CoordinatesToCellName(11, 2)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	if err := repF.SetCellStr(repSheet, cell, addr); err != nil {
		fmt.Println("set cell:", err)
		os.Exit(1)
	}
	if err := repF.Save(); err != nil {
		fmt.Println("save:", err)
		os.Exit(1)
	}
	fmt.Println("patched", cell, "→", addr)
}
