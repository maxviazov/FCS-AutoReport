-- Настройки приложения (одна строка, id = 1)
CREATE TABLE IF NOT EXISTS settings (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    raw_reports_path TEXT NOT NULL DEFAULT '',
    output_path TEXT NOT NULL DEFAULT '',
    template_path TEXT NOT NULL DEFAULT ''
);

-- Ограничение: только одна строка в settings
INSERT OR IGNORE INTO settings (id, raw_reports_path, output_path, template_path) VALUES (1, '', '', '');

-- Справочник городов: название, код Минздрава, алиасы (JSON-массив строк)
CREATE TABLE IF NOT EXISTS cities (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE,
    code TEXT NOT NULL DEFAULT '',
    aliases TEXT NOT NULL DEFAULT '[]'
);
CREATE INDEX IF NOT EXISTS idx_cities_name ON cities(name);

-- Справочник клиентов: Хет-Пей ID -> название, код города, тип
CREATE TABLE IF NOT EXISTS clients (
    hett_pay_id TEXT PRIMARY KEY,
    name TEXT NOT NULL DEFAULT '',
    city_code TEXT NOT NULL DEFAULT '',
    client_type TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_clients_hett_pay_id ON clients(hett_pay_id);

-- Справочник водителей: имя агента (или идентификатор) -> водитель, авто, телефон, города доставки
CREATE TABLE IF NOT EXISTS drivers (
    agent_name TEXT PRIMARY KEY,
    driver_name TEXT NOT NULL DEFAULT '',
    car_number TEXT NOT NULL DEFAULT '',
    phone TEXT NOT NULL DEFAULT '',
    city_codes TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_drivers_agent_name ON drivers(agent_name);

-- Справочник товаров: артикул -> категория для отчета
CREATE TABLE IF NOT EXISTS items (
    article TEXT PRIMARY KEY,
    category TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_items_article ON items(article);
