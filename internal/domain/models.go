package domain

// City описывает город, его код Минздрава и алиасы (варианты написания для сопоставления).
type City struct {
	ID      int      `json:"id"`
	Name    string   `json:"name"`    // Основное правильное название (напр. "תל אביב")
	Code    string   `json:"code"`    // Код Минздрава (напр. "N126")
	Aliases []string `json:"aliases"` // Синонимы (напр. ["תל-אביב", "תל אביב יפו", "ת''א"])
}

// CityNameCode — пара название/код для импорта (Excel → БД).
type CityNameCode struct {
	Name string `json:"name"`
	Code string `json:"code"`
}

// Settings хранит пути к папкам и шаблону (и последний выбранный сырой файл).
type Settings struct {
	InputFolder  string `json:"inputFolder"`  // Путь к сырому отчёту или папка
	OutputFolder string `json:"outputFolder"` // Папка сохранения отчётов
	TemplatePath string `json:"templatePath"` // Путь к шаблону Минздрава
}

// Client описывает клиента для подстановки кода города
type Client struct {
	HP       string `json:"hp"`        // ח"פ (ID клиента) - это будет наш уникальный ключ
	Name     string `json:"name"`      // Название клиента
	CityCode string `json:"city_code"` // קוד עיר (Код города для Минздрава, например M1187)
	Type     string `json:"type"`      // סוג לקוח (Тип: קמעונאי и т.д.)
}

// Driver описывает данные логистики. Водитель подставляется в отчёт по городу доставки (city_codes).
type Driver struct {
	AgentName  string `json:"agent_name"`  // Уникальный идентификатор (ключ)
	DriverName string `json:"driver_name"` // Имя водителя для отчёта
	CarNumber  string `json:"car_number"`  // Номер машины (מס.רכב)
	Phone      string `json:"phone"`       // Телефон водителя
	CityCodes  string `json:"city_codes"`  // Коды городов доставки через запятую (N126,J112,I400)
}

// Item описывает товар и его категорию для колонок Минздрава
type Item struct {
	ItemCode string `json:"item_code"` // קוד פריט (Артикул)
	Category string `json:"category"`  // Категория (например: "דגים מעובדים")
}

// AggregatedInvoice — готовая к экспорту накладная после ETL (обогащение + агрегация по номеру).
type AggregatedInvoice struct {
	InvoiceNum string
	Date       string
	ClientName string  // Имя клиента (из сырого отчёта или справочника)
	ClientHP   string  // ח"פ
	Address    string  // Исходный адрес (для информации)
	CityCode   string  // Код города из справочника (N126 и т.д.)

	// Логистика
	DriverName string
	CarNumber  string
	Phone      string

	// Агрегаты
	TotalBoxes float64
	Weights    map[string]float64 // Категория Минздрава → вес в кг

	// Валидация: список ошибок (не найден город, водитель, товар и т.д.)
	Errors []string
}

// HasErrors возвращает true, если при сборке накладной были проблемы.
func (a *AggregatedInvoice) HasErrors() bool {
	return len(a.Errors) > 0
}
