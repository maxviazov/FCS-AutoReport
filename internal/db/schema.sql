-- Настройки приложения (одна строка, id = 1)
CREATE TABLE IF NOT EXISTS settings (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    raw_reports_path TEXT NOT NULL DEFAULT '',
    output_path TEXT NOT NULL DEFAULT '',
    template_path TEXT NOT NULL DEFAULT '',
    smtp_host TEXT NOT NULL DEFAULT '',
    smtp_port INTEGER NOT NULL DEFAULT 587,
    smtp_user TEXT NOT NULL DEFAULT '',
    smtp_password TEXT NOT NULL DEFAULT '',
    imap_host TEXT NOT NULL DEFAULT '',
    imap_port INTEGER NOT NULL DEFAULT 993,
    imap_user TEXT NOT NULL DEFAULT '',
    imap_password TEXT NOT NULL DEFAULT '',
    auto_send INTEGER NOT NULL DEFAULT 0,
    watch_enabled INTEGER NOT NULL DEFAULT 0,
    watch_folder TEXT NOT NULL DEFAULT ''
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

-- Суточный реестр отправленных строк (для дедупликации в течение дня)
CREATE TABLE IF NOT EXISTS daily_sent_lines (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    business_date TEXT NOT NULL,
    dedup_key TEXT NOT NULL,
    invoice_num TEXT NOT NULL,
    client_name TEXT NOT NULL DEFAULT '',
    client_hp TEXT NOT NULL DEFAULT '',
    report_path TEXT NOT NULL DEFAULT '',
    sent_at TEXT NOT NULL DEFAULT ''
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_daily_sent_unique ON daily_sent_lines(business_date, dedup_key);

-- Очередь отправок и их состояние
CREATE TABLE IF NOT EXISTS outbox_jobs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    report_path TEXT NOT NULL UNIQUE,
    status TEXT NOT NULL DEFAULT 'queued',
    error TEXT NOT NULL DEFAULT '',
    subject_hint TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL DEFAULT '',
    sent_at TEXT NOT NULL DEFAULT '',
    reply_at TEXT NOT NULL DEFAULT ''
);

-- Результаты обработки ответов (approved/rejected)
CREATE TABLE IF NOT EXISTS approval_results (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    job_id INTEGER NOT NULL,
    invoice_num TEXT NOT NULL DEFAULT '',
    client_name TEXT NOT NULL DEFAULT '',
    client_hp TEXT NOT NULL DEFAULT '',
    approval_num TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT '',
    reject_reason TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL DEFAULT '',
    FOREIGN KEY(job_id) REFERENCES outbox_jobs(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_approval_results_job ON approval_results(job_id);

-- Файлы, уже обработанные фоновым watcher (чтобы не крутить один и тот же файл бесконечно)
CREATE TABLE IF NOT EXISTS processed_files (
    file_path TEXT PRIMARY KEY,
    kind TEXT NOT NULL DEFAULT '',
    processed_at TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT ''
);
