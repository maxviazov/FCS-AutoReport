// +build ignore

package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"fcs-autoreport/internal/app"
)

func main() {
	rawPath := flag.String("raw", "", "путь к сырому отчёту xlsx")
	templatePath := flag.String("template", "", "путь к шаблону xlsx (опционально)")
	outDir := flag.String("out", "", "папка сохранения отчёта (опционально)")
	flag.Parse()
	if *rawPath == "" {
		*rawPath = `c:\Users\office3\Desktop\FishKA\source\משקל.xlsx`
	}

	// Логи в stdout, пошагово
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})))

	fmt.Println("========== ШАГ 0: Инициализация ==========")
	dataDir := "."
	if d, err := os.UserConfigDir(); err == nil {
		dataDir = filepath.Join(d, "FCS-AutoReport")
	}
	fmt.Printf("dataDir = %s\n", dataDir)

	db, store, err := app.Bootstrap(dataDir)
	if err != nil {
		fmt.Println("Bootstrap err:", err)
		os.Exit(1)
	}
	defer db.Close()
	if err := app.NewReportService(db, store).LoadDictionariesToMemory(); err != nil {
		fmt.Println("LoadDictionaries err:", err)
	}
	fmt.Println("Справочники загружены.")

	fmt.Println("\n========== ШАГ 1: Агрегация сырого отчёта ==========")
	svc := app.NewReportService(db, store)
	invoices, err := svc.ProcessRawReport(*rawPath)
	if err != nil {
		fmt.Println("ProcessRawReport err:", err)
		os.Exit(1)
	}

	fmt.Println("\n========== ШАГ 2: Результат — имя клиента по накладным ==========")
	for _, inv := range invoices {
		fmt.Printf("  Накладная %s → ClientName = %q\n", inv.InvoiceNum, inv.ClientName)
		if len(inv.Errors) > 0 {
			for _, e := range inv.Errors {
				fmt.Printf("    Ошибка: %s\n", e)
			}
		}
	}

	fmt.Println("\n========== ШАГ 3: Проверка типа ==========")
	if len(invoices) > 0 {
		fmt.Printf("  inv.ClientName тип: %T, значение: %q\n", invoices[0].ClientName, invoices[0].ClientName)
	}

	if *templatePath != "" && *outDir != "" {
		fmt.Println("\n========== ШАГ 4: Экспорт в итоговый отчёт ==========")
		savedPath, err := app.ExportToExcel(invoices, *templatePath, *outDir)
		if err != nil {
			fmt.Println("ExportToExcel err:", err)
			os.Exit(1)
		}
		fmt.Printf("  Отчёт сохранён: %s\n", savedPath)
	} else {
		fmt.Println("\n========== ШАГ 4: пропущен (задайте -template и -out для экспорта) ==========")
	}

	fmt.Println("\n========== Готово ==========")
}
