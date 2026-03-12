package store

import (
	"fmt"
	"strings"
	"sync"

	"fcs-autoreport/internal/domain"
)

// Store — in-memory кэш справочников и настроек для быстрого доступа при генерации отчётов.
// Загружается из БД при старте приложения; изменения через репозиторий синхронизируются с БД и с Store.
type Store struct {
	mu sync.RWMutex

	Settings domain.Settings

	// Клиенты: ключ — HP (ח"פ)
	Clients map[string]*domain.Client
	// Водители: ключ — AgentName
	Drivers map[string]*domain.Driver
	// Водители по коду города доставки (для подстановки в отчёт)
	DriversByCity map[string][]*domain.Driver
	// Товары: ключ — ItemCode (артикул)
	Items map[string]*domain.Item
	// Города: по основному названию и по алиасам для быстрого поиска кода
	Cities       map[string]*domain.City // ключ: Name (нормализованный)
	CitiesByAlias map[string]*domain.City // ключ: алиас (нормализованный)
}

// New создаёт пустой Store (загрузка из БД выполняется отдельно).
func New() *Store {
	return &Store{
		Clients:       make(map[string]*domain.Client),
		Drivers:       make(map[string]*domain.Driver),
		DriversByCity: make(map[string][]*domain.Driver),
		Items:         make(map[string]*domain.Item),
		Cities:        make(map[string]*domain.City),
		CitiesByAlias: make(map[string]*domain.City),
	}
}

// LoadFrom при необходимости принимает уже загруженные из БД данные и заполняет Store.
// cities может быть nil — тогда кэш городов не меняется.
func (s *Store) LoadFrom(settings *domain.Settings, clients []domain.Client, drivers []domain.Driver, items []domain.Item, cities []domain.City) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if settings != nil {
		s.Settings = *settings
	}
	s.Clients = make(map[string]*domain.Client, len(clients))
	for i := range clients {
		c := &clients[i]
		s.Clients[c.HP] = c
	}
	s.Drivers = make(map[string]*domain.Driver, len(drivers))
	s.DriversByCity = make(map[string][]*domain.Driver)
	for i := range drivers {
		d := &drivers[i]
		s.Drivers[d.AgentName] = d
		for _, code := range parseCityCodes(d.CityCodes) {
			s.DriversByCity[code] = append(s.DriversByCity[code], d)
		}
	}
	s.Items = make(map[string]*domain.Item, len(items))
	for i := range items {
		it := &items[i]
		s.Items[it.ItemCode] = it
	}
	if cities != nil {
		s.citiesMapsFromSlice(cities)
	}
}

// citiesMapsFromSlice заполняет Cities и CitiesByAlias; вызывать под захваченным mu.
func (s *Store) citiesMapsFromSlice(cities []domain.City) {
	s.Cities = make(map[string]*domain.City, len(cities))
	s.CitiesByAlias = make(map[string]*domain.City)
	for i := range cities {
		c := &cities[i]
		s.Cities[domain.NormalizeText(c.Name)] = c
		for _, a := range c.Aliases {
			key := domain.NormalizeText(a)
			if key != "" {
				s.CitiesByAlias[key] = c
			}
		}
	}
}

// GetSettings возвращает копию настроек (потокобезопасно).
func (s *Store) GetSettings() domain.Settings {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Settings
}

// SetSettings обновляет настройки в кэше.
func (s *Store) SetSettings(settings *domain.Settings) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if settings != nil {
		s.Settings = *settings
	}
}

// GetClient возвращает клиента по HP (ח"פ) (nil если не найден).
func (s *Store) GetClient(hp string) *domain.Client {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Clients[hp]
}

// SetClient добавляет или обновляет клиента в кэше.
func (s *Store) SetClient(c *domain.Client) {
	if c == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Clients[c.HP] = c
}

// DeleteClient удаляет клиента из кэша.
func (s *Store) DeleteClient(hp string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.Clients, hp)
}

// parseCityCodes разбивает строку кодов городов (N126,J112,I400) на слайс, нормализует в верхний регистр.
func parseCityCodes(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	var out []string
	for _, part := range strings.Split(s, ",") {
		code := strings.TrimSpace(strings.ToUpper(part))
		if code != "" {
			out = append(out, code)
		}
	}
	return out
}

// GetDriverForCity возвращает водителя для подстановки в запись по коду города доставки.
// Сначала ищет водителя, у которого в city_codes есть этот город; если нет — любого водителя (fallback).
func (s *Store) GetDriverForCity(cityCode string) *domain.Driver {
	s.mu.RLock()
	defer s.mu.RUnlock()
	code := strings.TrimSpace(strings.ToUpper(cityCode))
	if code != "" {
		if list := s.DriversByCity[code]; len(list) > 0 {
			return list[0]
		}
	}
	for _, d := range s.Drivers {
		return d
	}
	return nil
}

// GetDriver возвращает водителя по имени агента (nil если не найден).
func (s *Store) GetDriver(agentName string) *domain.Driver {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Drivers[agentName]
}

// GetDriverByDriverName возвращает водителя по имени водителя (DriverName).
func (s *Store) GetDriverByDriverName(driverName string) *domain.Driver {
	s.mu.RLock()
	defer s.mu.RUnlock()
	key := domain.NormalizeText(driverName)
	if key == "" {
		return nil
	}
	for _, d := range s.Drivers {
		if domain.NormalizeText(d.DriverName) == key {
			return d
		}
	}
	return nil
}

// SetDriver добавляет или обновляет водителя в кэше.
func (s *Store) SetDriver(d *domain.Driver) {
	if d == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Drivers[d.AgentName] = d
}

// DeleteDriver удаляет водителя из кэша.
func (s *Store) DeleteDriver(agentName string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.Drivers, agentName)
}

// GetItem возвращает товар по артикулу (nil если не найден).
func (s *Store) GetItem(article string) *domain.Item {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Items[article]
}

// SetItem добавляет или обновляет товар в кэше.
func (s *Store) SetItem(it *domain.Item) {
	if it == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Items[it.ItemCode] = it
}

// DeleteItem удаляет товар из кэша.
func (s *Store) DeleteItem(article string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.Items, article)
}

// ReloadClients полностью перезаписывает кэш клиентов (после массового импорта).
func (s *Store) ReloadClients(clients []domain.Client) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Clients = make(map[string]*domain.Client, len(clients))
	for i := range clients {
		c := &clients[i]
		s.Clients[c.HP] = c
	}
}

// ReloadDrivers полностью перезаписывает кэш водителей и индекс по городам.
func (s *Store) ReloadDrivers(drivers []domain.Driver) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Drivers = make(map[string]*domain.Driver, len(drivers))
	s.DriversByCity = make(map[string][]*domain.Driver)
	for i := range drivers {
		d := &drivers[i]
		s.Drivers[d.AgentName] = d
		for _, code := range parseCityCodes(d.CityCodes) {
			s.DriversByCity[code] = append(s.DriversByCity[code], d)
		}
	}
}

// ReloadItems полностью перезаписывает кэш товаров.
func (s *Store) ReloadItems(items []domain.Item) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Items = make(map[string]*domain.Item, len(items))
	for i := range items {
		it := &items[i]
		s.Items[it.ItemCode] = it
	}
}

// ReloadCities полностью перезаписывает кэш городов (по имени и алиасам).
func (s *Store) ReloadCities(cities []domain.City) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.citiesMapsFromSlice(cities)
}

// ResolveCityCode возвращает код Минздрава для строки города (или части адреса).
// Сначала поиск по основному названию, затем по алиасам. rawCity — например "תל-אביב" или "תל אביב".
// Если город не найден, возвращает пустую строку и ненулевую ошибку с понятным текстом.
func (s *Store) ResolveCityCode(rawCity string) (code string, err error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	key := domain.NormalizeText(rawCity)
	if key == "" {
		return "", fmt.Errorf("пустое значение города")
	}
	if c, ok := s.Cities[key]; ok {
		return c.Code, nil
	}
	if c, ok := s.CitiesByAlias[key]; ok {
		return c.Code, nil
	}
	return "", fmt.Errorf("не найден город %q: добавьте его в справочник или укажите как алиас", rawCity)
}

// ResolveCityCodeBySubstring ищет город по вхождению: если в строке rawCity встречается
// название города или алиас из справочника, возвращает код этого города (при нескольких совпадениях — самое длинное).
// Используется, когда точного совпадения нет (напр. "איזור תעשייה כנות" → находится "כנות").
func (s *Store) ResolveCityCodeBySubstring(rawCity string) (code string, err error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	key := domain.NormalizeText(rawCity)
	if key == "" {
		return "", fmt.Errorf("пустое значение города")
	}
	var bestLen int
	var bestCode string
	for name, c := range s.Cities {
		if name != "" && len(name) > bestLen && strings.Contains(key, name) {
			bestLen = len(name)
			bestCode = c.Code
		}
	}
	for alias, c := range s.CitiesByAlias {
		if alias != "" && len(alias) > bestLen && strings.Contains(key, alias) {
			bestLen = len(alias)
			bestCode = c.Code
		}
	}
	if bestCode == "" {
		return "", fmt.Errorf("город не найден по подстроке в %q", rawCity)
	}
	return bestCode, nil
}
