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
	for _, c := range []string{"F1012", "N526", "J112"} {
		var n string
		_ = db.QueryRow(`SELECT name FROM cities WHERE code = ?`, c).Scan(&n)
		fmt.Printf("code %s -> %s\n", c, n)
	}
	for _, q := range []string{"%עקיבא%", "%אורות%", "%אור %"} {
		rows, _ := db.Query(`SELECT code, name FROM cities WHERE name LIKE ? ORDER BY name`, q)
		fmt.Println("LIKE", q)
		for rows.Next() {
			var c, n string
			rows.Scan(&c, &n)
			fmt.Printf("  %s  %s\n", c, n)
		}
		rows.Close()
	}
}
