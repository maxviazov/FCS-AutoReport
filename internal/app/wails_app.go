package app

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"fcs-autoreport/internal/domain"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// WailsApp — мост между фронтендом (JS/HTML) и ядром на Go.
// Публичные методы доступны в JavaScript через window.go.main.WailsApp.*
type WailsApp struct {
	ctx     context.Context
	service *ReportService
}

// NewWailsApp создаёт экземпляр моста для Wails.
func NewWailsApp(svc *ReportService) *WailsApp {
	return &WailsApp{service: svc}
}

// Startup вызывается фреймворком Wails при запуске окна.
func (a *WailsApp) Startup(ctx context.Context) {
	a.ctx = ctx
	_ = a.service.EnsureDailyReset()
	a.service.StartBackground(ctx)
	slog.Info("Wails UI запущен")
}

// GetSettings возвращает сохранённые пути (сырой файл, шаблон, папка сохранения) для подстановки в форму при старте.
func (a *WailsApp) GetSettings() (domain.Settings, error) {
	s, err := a.service.DB().GetSettings()
	if err != nil {
		return domain.Settings{}, err
	}
	if s == nil {
		return domain.Settings{}, nil
	}
	return *s, nil
}

// SaveSettings сохраняет пути в БД и обновляет кэш. Вызывать после выбора файлов или перед генерацией.
func (a *WailsApp) SaveSettings(inputFolder, outputFolder, templatePath string) error {
	s := &domain.Settings{
		InputFolder:  inputFolder,
		OutputFolder: outputFolder,
		TemplatePath: templatePath,
	}
	if err := a.service.DB().SavePaths(inputFolder, outputFolder, templatePath); err != nil {
		return fmt.Errorf("сохранение настроек: %w", err)
	}
	a.service.Store().SetSettings(s)
	return nil
}

func (a *WailsApp) SaveAutomationSettings(s domain.Settings) error {
	if s.SMTPPort <= 0 {
		s.SMTPPort = 587
	}
	if s.IMAPPort <= 0 {
		s.IMAPPort = 993
	}
	if err := a.service.DB().SaveSettings(&s); err != nil {
		return fmt.Errorf("сохранение automation settings: %w", err)
	}
	a.service.Store().SetSettings(&s)
	return nil
}

// SaveExportPerClient сохраняет режим экспорта (отдельные файлы по клиентам или один общий).
func (a *WailsApp) SaveExportPerClient(exportPerClient bool) error {
	s, err := a.service.DB().GetSettings()
	if err != nil {
		return fmt.Errorf("настройки: %w", err)
	}
	s.ExportPerClient = exportPerClient
	if err := a.service.DB().SaveSettings(s); err != nil {
		return fmt.Errorf("сохранение режима экспорта: %w", err)
	}
	a.service.Store().SetSettings(s)
	return nil
}

// GenerateReport вызывается из JavaScript: принимает пути к сырому файлу, шаблону и папке сохранения,
// сохраняет их в настройки, выполняет агрегацию и экспорт. Возвращает путь к папке fish_reports_* или к одному FCS_Report_*.xlsx.
func (a *WailsApp) GenerateReport(rawFilePath, templatePath, outputDir string) (string, error) {
	slog.Info("GenerateReport из UI", "raw", rawFilePath)
	if rawFilePath == "" || templatePath == "" || outputDir == "" {
		return "", fmt.Errorf("укажите сырой отчёт, шаблон и папку сохранения")
	}
	if err := a.SaveSettings(rawFilePath, outputDir, templatePath); err != nil {
		slog.Warn("не удалось сохранить пути", "err", err)
	}

	savedPath, err := a.service.GenerateAndQueueFromRaw(rawFilePath, templatePath, outputDir)
	if err != nil {
		return "", err
	}
	manual := a.service.GetLastManualReviewFiles()
	if len(manual) > 0 {
		msg := "Часть файлов требует ручной правки и сохранена отдельно в папке manual_review.\n\n"
		msg += "Эти файлы не учтены в счётчике обработанных и не отправлены.\n"
		if len(manual) <= 8 {
			msg += "\n" + strings.Join(manual, "\n")
		}
		_, _ = runtime.MessageDialog(a.ctx, runtime.MessageDialogOptions{
			Type:    runtime.WarningDialog,
			Title:   "Требуется ручная доработка",
			Message: msg,
		})
	}

	return savedPath, nil
}

func (a *WailsApp) GetLastManualReviewFiles() ([]string, error) {
	return a.service.GetLastManualReviewFiles(), nil
}

// OpenFileLocation открывает в проводнике папку, в которой лежит файл отчёта.
// filePath — полный путь к созданному отчёту.
func (a *WailsApp) OpenFileLocation(filePath string) error {
	if filePath == "" {
		return fmt.Errorf("путь к файлу не указан")
	}
	abs, err := filepath.Abs(filePath)
	if err != nil {
		return fmt.Errorf("получение абсолютного пути: %w", err)
	}
	info, err := os.Stat(abs)
	if err != nil {
		return fmt.Errorf("проверка пути: %w", err)
	}
	openTarget := abs
	if !info.IsDir() {
		openTarget = filepath.Dir(abs)
	}
	cmd := exec.Command("explorer", openTarget)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("запуск проводника: %w", err)
	}
	slog.Info("Открыта папка отчёта", "dir", openTarget)
	return nil
}

// SelectRawReport открывает системный диалог выбора сырого отчёта (xlsx/csv).
// При отмене возвращает пустую строку.
func (a *WailsApp) SelectRawReport() (string, error) {
	slog.Info("Диалог выбора сырого отчёта")
	return runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Выберите сырой отчёт",
		Filters: []runtime.FileFilter{
			{DisplayName: "Excel (*.xlsx)", Pattern: "*.xlsx"},
			{DisplayName: "CSV (*.csv)", Pattern: "*.csv"},
		},
	})
}

// SelectTemplate открывает системный диалог выбора шаблона Минздрава.
func (a *WailsApp) SelectTemplate() (string, error) {
	slog.Info("Диалог выбора шаблона")
	return runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Выберите шаблон Минздрава",
		Filters: []runtime.FileFilter{
			{DisplayName: "Excel (*.xlsx)", Pattern: "*.xlsx"},
		},
	})
}

// SelectOutputDir открывает системный диалог выбора папки для сохранения отчёта.
func (a *WailsApp) SelectOutputDir() (string, error) {
	slog.Info("Диалог выбора папки сохранения")
	return runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Где сохранить готовый отчёт?",
	})
}

// ImportCitiesDict открывает диалог выбора Excel и импортирует справочник городов (Upsert + обновление кэша).
func (a *WailsApp) ImportCitiesDict() (string, error) {
	path, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Выберите Excel со справочником ГОРОДОВ",
		Filters: []runtime.FileFilter{{DisplayName: "Excel (*.xlsx)", Pattern: "*.xlsx"}},
	})
	if err != nil || path == "" {
		return "", err
	}
	if err := a.service.ImportCities(path); err != nil {
		return "", fmt.Errorf("ошибка импорта городов: %w", err)
	}
	return "Справочник городов обновлён: " + path, nil
}

// ImportDriversDict открывает диалог и импортирует справочник водителей.
func (a *WailsApp) ImportDriversDict() (string, error) {
	path, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Выберите Excel со справочником ВОДИТЕЛЕЙ",
		Filters: []runtime.FileFilter{{DisplayName: "Excel (*.xlsx)", Pattern: "*.xlsx"}},
	})
	if err != nil || path == "" {
		return "", err
	}
	if err := a.service.ImportDrivers(path); err != nil {
		return "", fmt.Errorf("ошибка импорта водителей: %w", err)
	}
	return "Справочник водителей обновлён: " + path, nil
}

// ImportItemsDict открывает диалог и импортирует справочник товаров.
func (a *WailsApp) ImportItemsDict() (string, error) {
	path, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Выберите Excel со справочником ТОВАРОВ",
		Filters: []runtime.FileFilter{{DisplayName: "Excel (*.xlsx)", Pattern: "*.xlsx"}},
	})
	if err != nil || path == "" {
		return "", err
	}
	if err := a.service.ImportItems(path); err != nil {
		return "", fmt.Errorf("ошибка импорта товаров: %w", err)
	}
	return "Справочник товаров обновлён: " + path, nil
}

// --- CRUD для городов ---

// GetCities возвращает полный список городов из БД для отображения в таблице.
func (a *WailsApp) GetCities() ([]domain.City, error) {
	slog.Info("Запрос списка городов из UI")
	cities, err := a.service.DB().ListCities()
	if err != nil {
		return nil, fmt.Errorf("получение городов: %w", err)
	}
	return cities, nil
}

// SaveCity добавляет новый город (ID == 0) или обновляет существующий по ID. После изменения перезагружает кэш.
func (a *WailsApp) SaveCity(city domain.City) error {
	slog.Info("Сохранение города из UI", "name", city.Name)
	if city.ID == 0 {
		if err := a.service.DB().UpsertCity(&city); err != nil {
			return fmt.Errorf("добавление города: %w", err)
		}
	} else {
		if err := a.service.DB().UpdateCity(&city); err != nil {
			return fmt.Errorf("обновление города: %w", err)
		}
	}
	if err := a.service.LoadDictionariesToMemory(); err != nil {
		return fmt.Errorf("город сохранён, но кэш не обновлён: %w", err)
	}
	return nil
}

// DeleteCity удаляет город по ID и перезагружает кэш.
func (a *WailsApp) DeleteCity(id int) error {
	slog.Info("Удаление города из UI", "id", id)
	if err := a.service.DB().DeleteCity(id); err != nil {
		return fmt.Errorf("удаление города: %w", err)
	}
	if err := a.service.LoadDictionariesToMemory(); err != nil {
		return fmt.Errorf("город удалён, но кэш не обновлён: %w", err)
	}
	return nil
}

// GetLastUnresolvedCities возвращает список нераспознанных городов/адресов с последней генерации отчёта.
// Используется для запроса у пользователя добавления алиасов.
func (a *WailsApp) GetLastUnresolvedCities() ([]string, error) {
	return a.service.GetLastUnresolvedCities(), nil
}

// AddCityAlias добавляет алиас к существующему городу по ID и перезагружает кэш.
// alias — строка из отчёта (например "תל אביב"), которая будет подставляться к городу с данным id.
func (a *WailsApp) AddCityAlias(cityID int, alias string) error {
	if alias == "" {
		return fmt.Errorf("алиас не может быть пустым")
	}
	city, err := a.service.DB().GetCityByID(cityID)
	if err != nil {
		return fmt.Errorf("город с ID %d: %w", cityID, err)
	}
	if city == nil {
		return fmt.Errorf("город с ID %d не найден", cityID)
	}
	aliasNorm := domain.NormalizeText(alias)
	if aliasNorm == "" {
		return fmt.Errorf("алиас после нормализации пустой")
	}
	for _, existing := range city.Aliases {
		if domain.NormalizeText(existing) == aliasNorm {
			// уже есть — всё равно перезагружаем кэш
			if err := a.service.LoadDictionariesToMemory(); err != nil {
				return err
			}
			return nil
		}
	}
	// сохраняем нормализованную форму, чтобы поиск из сырого файла (тоже нормализованный) совпадал
	city.Aliases = append(city.Aliases, aliasNorm)
	if err := a.service.DB().UpdateCity(city); err != nil {
		return fmt.Errorf("обновление города: %w", err)
	}
	if err := a.service.LoadDictionariesToMemory(); err != nil {
		return fmt.Errorf("алиас добавлен, но кэш не обновлён: %w", err)
	}
	slog.Info("Добавлен алиас к городу", "city_id", cityID, "alias", alias)
	return nil
}

// --- CRUD для водителей ---

// GetDrivers возвращает список всех водителей.
func (a *WailsApp) GetDrivers() ([]domain.Driver, error) {
	slog.Info("Запрос списка водителей из UI")
	return a.service.DB().ListDrivers()
}

// SaveDriver добавляет или обновляет водителя (ключ — AgentName). После изменения перезагружает кэш.
func (a *WailsApp) SaveDriver(driver domain.Driver) error {
	slog.Info("Сохранение водителя из UI", "agent", driver.AgentName)
	if err := a.service.DB().UpsertDriver(&driver); err != nil {
		return fmt.Errorf("сохранение водителя: %w", err)
	}
	if err := a.service.LoadDictionariesToMemory(); err != nil {
		return fmt.Errorf("водитель сохранён, но кэш не обновлён: %w", err)
	}
	return nil
}

// DeleteDriver удаляет водителя по имени агента и перезагружает кэш.
func (a *WailsApp) DeleteDriver(agentName string) error {
	slog.Info("Удаление водителя из UI", "agent", agentName)
	if err := a.service.DB().DeleteDriver(agentName); err != nil {
		return fmt.Errorf("удаление водителя: %w", err)
	}
	if err := a.service.LoadDictionariesToMemory(); err != nil {
		return fmt.Errorf("водитель удалён, но кэш не обновлён: %w", err)
	}
	return nil
}

// --- CRUD для товаров ---

// GetItems возвращает список всех товаров.
func (a *WailsApp) GetItems() ([]domain.Item, error) {
	slog.Info("Запрос списка товаров из UI")
	return a.service.DB().ListItems()
}

// SaveItem добавляет или обновляет товар (ключ — ItemCode). После изменения перезагружает кэш.
func (a *WailsApp) SaveItem(item domain.Item) error {
	slog.Info("Сохранение товара из UI", "code", item.ItemCode)
	if err := a.service.DB().UpsertItem(&item); err != nil {
		return fmt.Errorf("сохранение товара: %w", err)
	}
	if err := a.service.LoadDictionariesToMemory(); err != nil {
		return fmt.Errorf("товар сохранён, но кэш не обновлён: %w", err)
	}
	return nil
}

// DeleteItem удаляет товар по артикулу и перезагружает кэш.
func (a *WailsApp) DeleteItem(itemCode string) error {
	slog.Info("Удаление товара из UI", "code", itemCode)
	if err := a.service.DB().DeleteItem(itemCode); err != nil {
		return fmt.Errorf("удаление товара: %w", err)
	}
	if err := a.service.LoadDictionariesToMemory(); err != nil {
		return fmt.Errorf("товар удалён, но кэш не обновлён: %w", err)
	}
	return nil
}

func (a *WailsApp) GetOutboxJobs(limit int) ([]domain.OutboxJob, error) {
	return a.service.ListOutboxJobs(limit)
}

func (a *WailsApp) GetApprovalResults(jobID int) ([]domain.ApprovalResult, error) {
	return a.service.ListApprovalResults(jobID)
}

func (a *WailsApp) SendReportNow(reportPath string) error {
	slog.Info("SendReportNow вызван", "report", reportPath)
	return a.service.SendQueuedReport(reportPath)
}

func (a *WailsApp) ApplyReplyForJob(jobID int, replyText, replyAttachmentPath string) error {
	return a.service.ApplyReplyForJob(jobID, replyText, replyAttachmentPath)
}

func (a *WailsApp) ProcessWatchFolderNow() error {
	return a.service.ProcessWatchFolderNow()
}

func (a *WailsApp) PollRepliesNow() error {
	slog.Info("PollRepliesNow вызван")
	return a.service.PollRepliesNow()
}

func (a *WailsApp) PollRepliesNowWithCount() (int, error) {
	slog.Info("PollRepliesNowWithCount вызван")
	return a.service.PollRepliesNowWithCount()
}

func (a *WailsApp) ResetSentRowsCounter() error {
	return a.service.ResetSentRowsCounter()
}

func (a *WailsApp) TestMailConnections() error {
	slog.Info("TestMailConnections вызван")
	return a.service.TestMailConnections()
}
