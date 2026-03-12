package db

import (
	"database/sql"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"fcs-autoreport/internal/domain"

	_ "modernc.org/sqlite"
)

//go:embed schema.sql
var schemaSQL string

const defaultDBFileName = "fcs_autoreport.db"

// DB обёртка над SQLite с методами для настроек и справочников.
type DB struct {
	conn *sql.DB
	mu   sync.RWMutex
}

// New открывает или создаёт базу по пути dataDir (например, рядом с exe или в AppData).
// Если dataDir пустой, используется текущая директория.
func New(dataDir string) (*DB, error) {
	if dataDir == "" {
		dataDir = "."
	}
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}
	path := filepath.Join(dataDir, defaultDBFileName)
	conn, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite %s: %w", path, err)
	}
	if err := conn.Ping(); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("ping sqlite: %w", err)
	}
	db := &DB{conn: conn}
	if err := db.migrate(); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return db, nil
}

func (db *DB) migrate() error {
	for _, stmt := range splitSQL(schemaSQL) {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}
		if _, err := db.conn.Exec(stmt); err != nil {
			preview := stmt
			if len(preview) > 50 {
				preview = preview[:50] + "..."
			}
			return fmt.Errorf("exec %s: %w", preview, err)
		}
	}
	// Миграция: добавить city_codes водителям (для существующих БД)
	_, _ = db.conn.Exec(`ALTER TABLE drivers ADD COLUMN city_codes TEXT NOT NULL DEFAULT ''`)
	return nil
}

// EnsureTelAvivAlias добавляет алиас "תל אביב" к городу "תל אביב יפו" (если запись есть — дополняет алиасы; если нет — создаёт город).
func (db *DB) EnsureTelAvivAlias() error {
	const cityName = "תל אביב יפו"
	const alias = "תל אביב"
	const defaultCode = "N126"

	city, err := db.GetCityByName(cityName)
	if err != nil {
		return err
	}
	db.mu.Lock()
	defer db.mu.Unlock()
	if city != nil {
		hasAlias := false
		for _, a := range city.Aliases {
			if strings.TrimSpace(a) == alias {
				hasAlias = true
				break
			}
		}
		if hasAlias {
			return nil
		}
		city.Aliases = append(city.Aliases, alias)
		aliasesJSON := "[]"
		if len(city.Aliases) > 0 {
			b, _ := json.Marshal(city.Aliases)
			aliasesJSON = string(b)
		}
		_, err = db.conn.Exec(`UPDATE cities SET aliases = ? WHERE id = ?`, aliasesJSON, city.ID)
		return err
	}
	// Города нет — создаём с алиасом
	aliasesJSONBytes, _ := json.Marshal([]string{alias})
	_, err = db.conn.Exec(
		`INSERT OR IGNORE INTO cities (name, code, aliases) VALUES (?, ?, ?)`,
		cityName, defaultCode, string(aliasesJSONBytes),
	)
	return err
}

func splitSQL(s string) []string {
	var out []string
	for _, part := range strings.Split(s, ";") {
		if t := strings.TrimSpace(part); t != "" {
			out = append(out, part)
		}
	}
	return out
}

// Close закрывает соединение с БД.
func (db *DB) Close() error {
	db.mu.Lock()
	defer db.mu.Unlock()
	if db.conn == nil {
		return nil
	}
	err := db.conn.Close()
	db.conn = nil
	return err
}

// GetSettings возвращает текущие настройки (пути).
func (db *DB) GetSettings() (*domain.Settings, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()
	var s domain.Settings
	err := db.conn.QueryRow(
		`SELECT raw_reports_path, output_path, template_path FROM settings WHERE id = 1`,
	).Scan(&s.InputFolder, &s.OutputFolder, &s.TemplatePath)
	if err == sql.ErrNoRows {
		return &domain.Settings{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get settings: %w", err)
	}
	return &s, nil
}

// SaveSettings сохраняет настройки.
func (db *DB) SaveSettings(s *domain.Settings) error {
	db.mu.Lock()
	defer db.mu.Unlock()
	_, err := db.conn.Exec(
		`UPDATE settings SET raw_reports_path = ?, output_path = ?, template_path = ? WHERE id = 1`,
		s.InputFolder, s.OutputFolder, s.TemplatePath,
	)
	return err
}

// ListClients возвращает всех клиентов.
func (db *DB) ListClients() ([]domain.Client, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()
	rows, err := db.conn.Query(`SELECT hett_pay_id, name, city_code, client_type FROM clients ORDER BY hett_pay_id`)
	if err != nil {
		return nil, fmt.Errorf("list clients: %w", err)
	}
	defer rows.Close()
	var list []domain.Client
	for rows.Next() {
		var c domain.Client
		if err := rows.Scan(&c.HP, &c.Name, &c.CityCode, &c.Type); err != nil {
			return nil, err
		}
		list = append(list, c)
	}
	return list, rows.Err()
}

// GetClient возвращает клиента по Хет-Пей ID.
func (db *DB) GetClient(hettPayID string) (*domain.Client, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()
	var c domain.Client
	err := db.conn.QueryRow(
		`SELECT hett_pay_id, name, city_code, client_type FROM clients WHERE hett_pay_id = ?`,
		hettPayID,
	).Scan(&c.HP, &c.Name, &c.CityCode, &c.Type)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get client: %w", err)
	}
	return &c, nil
}

// UpsertClient вставляет или обновляет клиента.
func (db *DB) UpsertClient(c *domain.Client) error {
	db.mu.Lock()
	defer db.mu.Unlock()
	_, err := db.conn.Exec(
		`INSERT INTO clients (hett_pay_id, name, city_code, client_type) VALUES (?, ?, ?, ?)
		 ON CONFLICT(hett_pay_id) DO UPDATE SET name = excluded.name, city_code = excluded.city_code, client_type = excluded.client_type`,
		c.HP, c.Name, c.CityCode, c.Type,
	)
	return err
}

// DeleteClient удаляет клиента по Хет-Пей ID.
func (db *DB) DeleteClient(hettPayID string) error {
	db.mu.Lock()
	defer db.mu.Unlock()
	_, err := db.conn.Exec(`DELETE FROM clients WHERE hett_pay_id = ?`, hettPayID)
	return err
}

// ListDrivers возвращает всех водителей.
func (db *DB) ListDrivers() ([]domain.Driver, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()
	rows, err := db.conn.Query(`SELECT agent_name, driver_name, car_number, phone, COALESCE(city_codes, '') FROM drivers ORDER BY agent_name`)
	if err != nil {
		return nil, fmt.Errorf("list drivers: %w", err)
	}
	defer rows.Close()
	var list []domain.Driver
	for rows.Next() {
		var d domain.Driver
		if err := rows.Scan(&d.AgentName, &d.DriverName, &d.CarNumber, &d.Phone, &d.CityCodes); err != nil {
			return nil, err
		}
		list = append(list, d)
	}
	return list, rows.Err()
}

// GetDriver возвращает водителя по имени агента.
func (db *DB) GetDriver(agentName string) (*domain.Driver, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()
	var d domain.Driver
	err := db.conn.QueryRow(
		`SELECT agent_name, driver_name, car_number, phone, COALESCE(city_codes, '') FROM drivers WHERE agent_name = ?`,
		agentName,
	).Scan(&d.AgentName, &d.DriverName, &d.CarNumber, &d.Phone, &d.CityCodes)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get driver: %w", err)
	}
	return &d, nil
}

// UpsertDriver вставляет или обновляет водителя.
func (db *DB) UpsertDriver(d *domain.Driver) error {
	db.mu.Lock()
	defer db.mu.Unlock()
	_, err := db.conn.Exec(
		`INSERT INTO drivers (agent_name, driver_name, car_number, phone, city_codes) VALUES (?, ?, ?, ?, ?)
		 ON CONFLICT(agent_name) DO UPDATE SET driver_name = excluded.driver_name, car_number = excluded.car_number, phone = excluded.phone, city_codes = excluded.city_codes`,
		d.AgentName, d.DriverName, d.CarNumber, d.Phone, d.CityCodes,
	)
	return err
}

// DeleteDriver удаляет водителя по имени агента.
func (db *DB) DeleteDriver(agentName string) error {
	db.mu.Lock()
	defer db.mu.Unlock()
	_, err := db.conn.Exec(`DELETE FROM drivers WHERE agent_name = ?`, agentName)
	return err
}

// ListItems возвращает все товары.
func (db *DB) ListItems() ([]domain.Item, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()
	rows, err := db.conn.Query(`SELECT article, category FROM items ORDER BY article`)
	if err != nil {
		return nil, fmt.Errorf("list items: %w", err)
	}
	defer rows.Close()
	var list []domain.Item
	for rows.Next() {
		var it domain.Item
		if err := rows.Scan(&it.ItemCode, &it.Category); err != nil {
			return nil, err
		}
		list = append(list, it)
	}
	return list, rows.Err()
}

// GetItem возвращает товар по артикулу.
func (db *DB) GetItem(article string) (*domain.Item, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()
	var it domain.Item
	err := db.conn.QueryRow(`SELECT article, category FROM items WHERE article = ?`, article).Scan(&it.ItemCode, &it.Category)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get item: %w", err)
	}
	return &it, nil
}

// UpsertItem вставляет или обновляет товар.
func (db *DB) UpsertItem(it *domain.Item) error {
	db.mu.Lock()
	defer db.mu.Unlock()
	_, err := db.conn.Exec(
		`INSERT INTO items (article, category) VALUES (?, ?) ON CONFLICT(article) DO UPDATE SET category = excluded.category`,
		it.ItemCode, it.Category,
	)
	return err
}

// DeleteItem удаляет товар по артикулу.
func (db *DB) DeleteItem(article string) error {
	db.mu.Lock()
	defer db.mu.Unlock()
	_, err := db.conn.Exec(`DELETE FROM items WHERE article = ?`, article)
	return err
}

// BulkUpsertClients вставляет или обновляет несколько клиентов в одной транзакции.
func (db *DB) BulkUpsertClients(clients []domain.Client) error {
	db.mu.Lock()
	defer db.mu.Unlock()
	tx, err := db.conn.Begin()
	if err != nil {
		return err
	}
	stmt, err := tx.Prepare(
		`INSERT INTO clients (hett_pay_id, name, city_code, client_type) VALUES (?, ?, ?, ?)
		 ON CONFLICT(hett_pay_id) DO UPDATE SET name = excluded.name, city_code = excluded.city_code, client_type = excluded.client_type`,
	)
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	defer stmt.Close()
	for _, c := range clients {
		if _, err := stmt.Exec(c.HP, c.Name, c.CityCode, c.Type); err != nil {
			_ = tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}

// BulkUpsertDrivers вставляет или обновляет несколько водителей в одной транзакции.
func (db *DB) BulkUpsertDrivers(drivers []domain.Driver) error {
	db.mu.Lock()
	defer db.mu.Unlock()
	tx, err := db.conn.Begin()
	if err != nil {
		return err
	}
	stmt, err := tx.Prepare(
		`INSERT INTO drivers (agent_name, driver_name, car_number, phone, city_codes) VALUES (?, ?, ?, ?, ?)
		 ON CONFLICT(agent_name) DO UPDATE SET driver_name = excluded.driver_name, car_number = excluded.car_number, phone = excluded.phone, city_codes = excluded.city_codes`,
	)
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	defer stmt.Close()
	for _, d := range drivers {
		if _, err := stmt.Exec(d.AgentName, d.DriverName, d.CarNumber, d.Phone, d.CityCodes); err != nil {
			_ = tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}

// BulkUpsertItems вставляет или обновляет несколько товаров в одной транзакции.
func (db *DB) BulkUpsertItems(items []domain.Item) error {
	db.mu.Lock()
	defer db.mu.Unlock()
	tx, err := db.conn.Begin()
	if err != nil {
		return err
	}
	stmt, err := tx.Prepare(
		`INSERT INTO items (article, category) VALUES (?, ?) ON CONFLICT(article) DO UPDATE SET category = excluded.category`,
	)
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	defer stmt.Close()
	for _, it := range items {
		if _, err := stmt.Exec(it.ItemCode, it.Category); err != nil {
			_ = tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}

// ListCities возвращает все города.
func (db *DB) ListCities() ([]domain.City, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()
	rows, err := db.conn.Query(`SELECT id, name, code, aliases FROM cities ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("list cities: %w", err)
	}
	defer rows.Close()
	var list []domain.City
	for rows.Next() {
		var c domain.City
		var aliasesJSON string
		if err := rows.Scan(&c.ID, &c.Name, &c.Code, &aliasesJSON); err != nil {
			return nil, err
		}
		if aliasesJSON != "" && aliasesJSON != "[]" {
			_ = json.Unmarshal([]byte(aliasesJSON), &c.Aliases)
		}
		list = append(list, c)
	}
	return list, rows.Err()
}

// GetCityByName возвращает город по основному названию.
func (db *DB) GetCityByName(name string) (*domain.City, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()
	var c domain.City
	var aliasesJSON string
	err := db.conn.QueryRow(
		`SELECT id, name, code, aliases FROM cities WHERE name = ?`,
		name,
	).Scan(&c.ID, &c.Name, &c.Code, &aliasesJSON)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get city: %w", err)
	}
	if aliasesJSON != "" && aliasesJSON != "[]" {
		_ = json.Unmarshal([]byte(aliasesJSON), &c.Aliases)
	}
	return &c, nil
}

// GetCityByID возвращает город по ID.
func (db *DB) GetCityByID(id int) (*domain.City, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()
	var c domain.City
	var aliasesJSON string
	err := db.conn.QueryRow(
		`SELECT id, name, code, aliases FROM cities WHERE id = ?`,
		id,
	).Scan(&c.ID, &c.Name, &c.Code, &aliasesJSON)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get city by id: %w", err)
	}
	if aliasesJSON != "" && aliasesJSON != "[]" {
		_ = json.Unmarshal([]byte(aliasesJSON), &c.Aliases)
	}
	return &c, nil
}

// UpsertCity вставляет город или обновляет код (алиасы при обновлении не трогаем — последняя загрузка SoT по коду).
func (db *DB) UpsertCity(c *domain.City) error {
	db.mu.Lock()
	defer db.mu.Unlock()
	aliasesJSON := "[]"
	if len(c.Aliases) > 0 {
		b, _ := json.Marshal(c.Aliases)
		aliasesJSON = string(b)
	}
	_, err := db.conn.Exec(
		`INSERT INTO cities (name, code, aliases) VALUES (?, ?, ?)
		 ON CONFLICT(name) DO UPDATE SET code = excluded.code`,
		c.Name, c.Code, aliasesJSON,
	)
	return err
}

// UpdateCity полностью обновляет запись города (в т.ч. алиасы) по id.
func (db *DB) UpdateCity(c *domain.City) error {
	db.mu.Lock()
	defer db.mu.Unlock()
	aliasesJSON := "[]"
	if len(c.Aliases) > 0 {
		b, _ := json.Marshal(c.Aliases)
		aliasesJSON = string(b)
	}
	_, err := db.conn.Exec(
		`UPDATE cities SET name = ?, code = ?, aliases = ? WHERE id = ?`,
		c.Name, c.Code, aliasesJSON, c.ID,
	)
	return err
}

// DeleteCity удаляет город по id.
func (db *DB) DeleteCity(id int) error {
	db.mu.Lock()
	defer db.mu.Unlock()
	_, err := db.conn.Exec(`DELETE FROM cities WHERE id = ?`, id)
	return err
}

// UpsertCitiesFromPairs загружает пары (название, код) с логикой Upsert: при совпадении name обновляется только code, алиасы сохраняются.
// Дубликаты по name в слайсе обрабатываются как «последняя запись побеждает».
func (db *DB) UpsertCitiesFromPairs(pairs []domain.CityNameCode) error {
	if len(pairs) == 0 {
		return nil
	}
	db.mu.Lock()
	defer db.mu.Unlock()
	lastByName := make(map[string]string)
	for _, p := range pairs {
		lastByName[p.Name] = p.Code
	}
	tx, err := db.conn.Begin()
	if err != nil {
		return err
	}
	stmt, err := tx.Prepare(
		`INSERT INTO cities (name, code, aliases) VALUES (?, ?, '[]')
		 ON CONFLICT(name) DO UPDATE SET code = excluded.code`,
	)
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	defer stmt.Close()
	for name, code := range lastByName {
		if _, err := stmt.Exec(name, code); err != nil {
			_ = tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}
