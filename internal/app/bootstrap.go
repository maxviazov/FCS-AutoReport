package app

import (
	"fmt"

	"fcs-autoreport/internal/db"
	"fcs-autoreport/internal/store"
)

// Bootstrap подключается к локальной БД, выполняет миграции и загружает данные в in-memory Store.
// dataDir — путь к папке с файлом БД (например, рядом с exe или в AppData).
func Bootstrap(dataDir string) (*db.DB, *store.Store, error) {
	database, err := db.New(dataDir)
	if err != nil {
		return nil, nil, fmt.Errorf("init db: %w", err)
	}
	if err := database.EnsureTelAvivAlias(); err != nil {
		// не прерываем старт при ошибке добавления алиаса
		_ = err
	}

	s := store.New()
	settings, err := database.GetSettings()
	if err != nil {
		_ = database.Close()
		return nil, nil, fmt.Errorf("load settings: %w", err)
	}
	clients, err := database.ListClients()
	if err != nil {
		_ = database.Close()
		return nil, nil, fmt.Errorf("load clients: %w", err)
	}
	drivers, err := database.ListDrivers()
	if err != nil {
		_ = database.Close()
		return nil, nil, fmt.Errorf("load drivers: %w", err)
	}
	items, err := database.ListItems()
	if err != nil {
		_ = database.Close()
		return nil, nil, fmt.Errorf("load items: %w", err)
	}

	cities, err := database.ListCities()
	if err != nil {
		_ = database.Close()
		return nil, nil, fmt.Errorf("load cities: %w", err)
	}

	s.LoadFrom(settings, clients, drivers, items, cities)
	return database, s, nil
}
