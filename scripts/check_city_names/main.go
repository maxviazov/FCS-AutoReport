package main

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

func main() {
	dbPath := filepath.Join(os.Getenv("APPDATA"), "FCS-AutoReport", "fcs_autoreport.db")
	if len(os.Args) > 1 {
		dbPath = os.Args[1]
	}
	dsn := "file:" + filepath.ToSlash(dbPath) + "?mode=ro"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer db.Close()

	cities := []string{
		"אילת", "אשדוד", "אשקלון", "באר שבע", "בת ים", "דימונה", "הרצליה", "חדרה", "חיפה",
		"ירוחם", "כפר סבא", "מגדל העמק", "מעלה אדומים", "נצרת", "נתניה", "ערד", "קרית טבעון",
		`ראשל"צ`, "ראשל״צ", "תל אביב", "תל אביב יפו",
	}
	fmt.Println("DB:", dbPath)
	for _, c := range cities {
		var code, name string
		err := db.QueryRow(`SELECT code, name FROM cities WHERE TRIM(name) = ? LIMIT 1`, c).Scan(&code, &name)
		if err == nil {
			fmt.Printf("OK %q -> %s (%s)\n", c, code, name)
			continue
		}
		// LIKE for ראשל
		err = db.QueryRow(`SELECT code, name FROM cities WHERE name LIKE ? LIMIT 1`, "%"+c+"%").Scan(&code, &name)
		if err == nil {
			fmt.Printf("LIKE %q -> %s (%s)\n", c, code, name)
			continue
		}
		fmt.Printf("NOT FOUND %q\n", c)
	}

	rows, err := db.Query(`SELECT code, name FROM cities WHERE name LIKE '%טבעון%' ORDER BY name LIMIT 20`)
	if err == nil {
		fmt.Println("\nDB rows mentioning טבעון:")
		for rows.Next() {
			var code, name string
			rows.Scan(&code, &name)
			fmt.Println(" ", code, name)
		}
		rows.Close()
	}
}
