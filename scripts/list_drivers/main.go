package main

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

func main() {
	p := filepath.Join(os.Getenv("APPDATA"), "FCS-AutoReport", "fcs_autoreport.db")
	db, _ := sql.Open("sqlite", "file:"+filepath.ToSlash(p)+"?mode=ro")
	rows, _ := db.Query(`SELECT driver_name, car_number, city_codes FROM drivers`)
	for rows.Next() {
		var n, c, cc string
		rows.Scan(&n, &c, &cc)
		fmt.Printf("%-20s %-15s %s\n", n, c, cc)
	}
}
