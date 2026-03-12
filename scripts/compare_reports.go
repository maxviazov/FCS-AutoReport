// +build ignore

// Сравнивает сырой отчёт (משקל.xlsx) и сгенерированный FCS отчёт: имена клиентов и количество строк.
// Запуск: go run scripts/compare_reports.go <путь_к_משקל.xlsx> <путь_к_FCS_Report.xlsx>
package main

import (
	"fmt"
	"os"
	"strings"

	"fcs-autoreport/internal/domain"

	"github.com/xuri/excelize/v2"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Использование: go run scripts/compare_reports.go <сырой_משקל.xlsx> <FCS_Report.xlsx>")
		os.Exit(1)
	}
	rawPath := os.Args[1]
	genPath := os.Args[2]

	// Сырой: колонка "שם לועזי" (клиент) — ищем по заголовку "כתובת", берём предыдущую колонку
	rawClients, err := extractRawClientNames(rawPath)
	if err != nil {
		fmt.Printf("Сырой отчёт: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Сырой отчёт: %d записей (уникальных клиентов по строкам)\n", len(rawClients))
	for i, c := range rawClients {
		if i >= 10 {
			fmt.Printf("  ... и ещё %d\n", len(rawClients)-10)
			break
		}
		fmt.Printf("  %q\n", c)
	}

	// Итоговый: колонка "לקוח"
	genClients, err := extractGeneratedClientNames(genPath)
	if err != nil {
		fmt.Printf("Итоговый отчёт: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("\nИтоговый отчёт: %d записей (колонка לקוח)\n", len(genClients))
	for i, c := range genClients {
		if i >= 10 {
			fmt.Printf("  ... и ещё %d\n", len(genClients)-10)
			break
		}
		fmt.Printf("  %q\n", c)
	}

	// Краткое сравнение
	fmt.Println()
	if len(rawClients) != len(genClients) {
		fmt.Printf("Различие: в сыром %d строк с клиентом, в итоговом %d.\n", len(rawClients), len(genClients))
	}
	min := len(rawClients)
	if len(genClients) < min {
		min = len(genClients)
	}
	mismatch := 0
	for i := 0; i < min; i++ {
		if domain.NormalizeText(rawClients[i]) != domain.NormalizeText(genClients[i]) {
			mismatch++
			if mismatch <= 5 {
				fmt.Printf("Расхождение строка %d: сырой %q → итог %q\n", i+1, rawClients[i], genClients[i])
			}
		}
	}
	if mismatch > 5 {
		fmt.Printf("... всего расхождений: %d\n", mismatch)
	} else if mismatch == 0 && len(rawClients) == len(genClients) {
		fmt.Println("Клиенты совпадают.")
	}
}

func extractRawClientNames(path string) ([]string, error) {
	f, err := excelize.OpenFile(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	sheet := f.GetSheetName(0)
	rows, err := f.GetRows(sheet)
	if err != nil || len(rows) == 0 {
		return nil, fmt.Errorf("нет строк")
	}
	var clientNameCol int = -1
	var headerRow int
	for hi := 0; hi < len(rows) && hi < 5; hi++ {
		for j, c := range rows[hi] {
			if domain.NormalizeText(c) == "כתובת" {
				headerRow = hi
				if j > 0 {
					clientNameCol = j - 1
				}
				break
			}
		}
		if clientNameCol >= 0 {
			break
		}
	}
	if clientNameCol < 0 && len(rows[headerRow]) > 11 {
		clientNameCol = 11
	}
	if clientNameCol < 0 {
		return nil, fmt.Errorf("колонка имени клиента не найдена")
	}
	var out []string
	for i := headerRow + 1; i < len(rows); i++ {
		row := rows[i]
		if clientNameCol < len(row) {
			v := strings.TrimSpace(row[clientNameCol])
			out = append(out, v)
		}
	}
	return out, nil
}

func extractGeneratedClientNames(path string) ([]string, error) {
	f, err := excelize.OpenFile(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	sheet := f.GetSheetName(0)
	rows, err := f.GetRows(sheet)
	if err != nil || len(rows) == 0 {
		return nil, fmt.Errorf("нет строк")
	}
	clientCol := -1
	for j, c := range rows[0] {
		n := domain.NormalizeText(c)
		if n == "לקוח" || (strings.Contains(n, "לקוח") && !strings.Contains(n, "סוג")) {
			clientCol = j
			break
		}
	}
	if clientCol < 0 {
		clientCol = 7
	}
	var out []string
	for i := 1; i < len(rows); i++ {
		row := rows[i]
		if clientCol < len(row) {
			out = append(out, strings.TrimSpace(row[clientCol]))
		}
	}
	return out, nil
}
