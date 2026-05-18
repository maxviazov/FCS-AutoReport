package main

import (
	"fmt"
	"os"
	"path/filepath"

	"fcs-autoreport/internal/app"
	"fcs-autoreport/internal/domain"

	"github.com/xuri/excelize/v2"
)

func main() {
	raw := `c:\Users\office3\Desktop\FishKA\source\משקל.xlsx`
	tpl := filepath.Join("samples", "approved_template.xlsx")
	if len(os.Args) > 1 {
		raw = os.Args[1]
	}

	dataDir := filepath.Join(os.Getenv("APPDATA"), "FCS-AutoReport")
	db, store, err := app.Bootstrap(dataDir)
	if err != nil {
		fmt.Println("bootstrap:", err)
		os.Exit(1)
	}
	defer db.Close()
	_ = app.NewReportService(db, store).LoadDictionariesToMemory()
	svc := app.NewReportService(db, store)
	all, err := svc.ProcessRawReport(raw)
	if err != nil {
		fmt.Println("aggregate:", err)
		os.Exit(1)
	}
	var one []*domain.AggregatedInvoice
	for _, inv := range all {
		if inv.InvoiceNum == "253887" {
			one = append(one, inv)
		}
	}
	if len(one) == 0 {
		fmt.Println("invoice 253887 not found")
		os.Exit(1)
	}
	fmt.Printf("aggregated: inv=%s city=%s hp=%s client=%q addr=%q driver=%s car=%s\n",
		one[0].InvoiceNum, one[0].CityCode, one[0].ClientHP, one[0].ClientName, one[0].Address,
		one[0].DriverName, one[0].CarNumber)

	saved, err := app.ExportToExcel(one, tpl, os.TempDir())
	if err != nil {
		fmt.Println("export:", err)
		os.Exit(1)
	}
	fmt.Println("written:", saved)
	dumpCells(saved)
}

func dumpCells(path string) {
	f, _ := excelize.OpenFile(path)
	defer f.Close()
	sh := f.GetSheetName(0)
	dim, _ := f.GetSheetDimension(sh)
	fmt.Println("dimension:", dim)
	for col := 1; col <= 30; col++ {
		cell, _ := excelize.CoordinatesToCellName(col, 2)
		v, _ := f.GetCellValue(sh, cell)
		fm, _ := f.GetCellFormula(sh, cell)
		t, _ := f.GetCellType(sh, cell)
		if v != "" || fm != "" {
			fmt.Printf("col %2d type=%v val=%q formula=%q\n", col, t, v, fm)
		}
	}
}
