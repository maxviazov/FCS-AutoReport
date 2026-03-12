package app

import (
	"fmt"
	"log/slog"
	"sync"

	"fcs-autoreport/internal/db"
	"fcs-autoreport/internal/domain"
	"fcs-autoreport/internal/dict"
	"fcs-autoreport/internal/store"
)

// ReportService объединяет работу с БД и in-memory кэшем для генерации отчётов.
// Кэш живёт в store.Store — отдельные карты в сервисе не дублируем.
type ReportService struct {
	db             *db.DB
	store          *store.Store
	mu             sync.Mutex
	lastUnresolved []string // уникальные названия городов/адреса, не распознанные при последней агрегации
}

// NewReportService создаёт сервис с доступом к БД и кэшу.
func NewReportService(database *db.DB, st *store.Store) *ReportService {
	return &ReportService{
		db:    database,
		store: st,
	}
}

// LoadDictionariesToMemory загружает справочники и настройки из БД в in-memory кэш (Store).
// Вызывать при старте приложения и при необходимости обновить кэш (например после импорта).
func (s *ReportService) LoadDictionariesToMemory() error {
	slog.Info("Загружаем справочники в in-memory кэш...")

	settings, err := s.db.GetSettings()
	if err != nil {
		return fmt.Errorf("загрузка настроек: %w", err)
	}

	clients, err := s.db.ListClients()
	if err != nil {
		return fmt.Errorf("загрузка клиентов: %w", err)
	}

	drivers, err := s.db.ListDrivers()
	if err != nil {
		return fmt.Errorf("загрузка водителей: %w", err)
	}

	items, err := s.db.ListItems()
	if err != nil {
		return fmt.Errorf("загрузка товаров: %w", err)
	}

	cities, err := s.db.ListCities()
	if err != nil {
		return fmt.Errorf("загрузка городов: %w", err)
	}

	s.store.LoadFrom(settings, clients, drivers, items, cities)

	slog.Info("Справочники загружены",
		"clients", len(clients),
		"drivers", len(drivers),
		"items", len(items),
		"cities", len(cities))
	return nil
}

// Store возвращает in-memory кэш для поиска при генерации отчёта (клиенты, водители, товары, настройки).
func (s *ReportService) Store() *store.Store {
	return s.store
}

// DB возвращает слой БД для сохранения настроек и справочников (импорт, редактирование).
func (s *ReportService) DB() *db.DB {
	return s.db
}

// ImportCities читает Excel со справочником городов, обновляет БД (Upsert) и перезагружает кэш.
func (s *ReportService) ImportCities(excelPath string) error {
	slog.Info("Импорт справочника городов", "file", excelPath)
	_, err := dict.ImportCitiesFromExcel(s.db, excelPath, dict.DefaultCitiesImportConfig())
	if err != nil {
		return fmt.Errorf("импорт городов: %w", err)
	}
	if err := s.LoadDictionariesToMemory(); err != nil {
		return fmt.Errorf("города импортированы, но ошибка обновления кэша: %w", err)
	}
	return nil
}

// ImportDrivers читает Excel со справочником водителей, обновляет БД и перезагружает кэш.
func (s *ReportService) ImportDrivers(excelPath string) error {
	slog.Info("Импорт справочника водителей", "file", excelPath)
	_, err := dict.ImportDriversFromExcel(s.db, excelPath, dict.DefaultDriversImportConfig())
	if err != nil {
		return fmt.Errorf("импорт водителей: %w", err)
	}
	if err := s.LoadDictionariesToMemory(); err != nil {
		return fmt.Errorf("водители импортированы, но ошибка обновления кэша: %w", err)
	}
	return nil
}

// ImportItems читает Excel со справочником товаров, обновляет БД и перезагружает кэш.
func (s *ReportService) ImportItems(excelPath string) error {
	slog.Info("Импорт справочника товаров", "file", excelPath)
	_, err := dict.ImportItemsFromExcel(s.db, excelPath, dict.DefaultItemsImportConfig())
	if err != nil {
		return fmt.Errorf("импорт товаров: %w", err)
	}
	if err := s.LoadDictionariesToMemory(); err != nil {
		return fmt.Errorf("товары импортированы, но ошибка обновления кэша: %w", err)
	}
	return nil
}

// SetLastUnresolvedCities сохраняет список нераспознанных городов после агрегации (уникальные значения).
func (s *ReportService) SetLastUnresolvedCities(invoices []*domain.AggregatedInvoice) {
	s.mu.Lock()
	defer s.mu.Unlock()
	seen := make(map[string]bool)
	var list []string
	for _, inv := range invoices {
		if inv.CityCode != "" {
			continue
		}
		cityStr := domain.ExtractCityFromAddress(inv.Address)
		if cityStr == "" {
			cityStr = inv.Address
		}
		if cityStr == "" || seen[cityStr] {
			continue
		}
		seen[cityStr] = true
		list = append(list, cityStr)
	}
	s.lastUnresolved = list
}

// GetLastUnresolvedCities возвращает нераспознанные города с последней генерации отчёта.
func (s *ReportService) GetLastUnresolvedCities() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.lastUnresolved) == 0 {
		return nil
	}
	out := make([]string, len(s.lastUnresolved))
	copy(out, s.lastUnresolved)
	return out
}
