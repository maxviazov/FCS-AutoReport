package app

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"fcs-autoreport/internal/db"
	"fcs-autoreport/internal/domain"
	"fcs-autoreport/internal/dict"
	"fcs-autoreport/internal/mail"
	"fcs-autoreport/internal/store"
)

// ReportService объединяет работу с БД и in-memory кэшем для генерации отчётов.
// Кэш живёт в store.Store — отдельные карты в сервисе не дублируем.
type ReportService struct {
	db             *db.DB
	store          *store.Store
	mu             sync.Mutex
	lastUnresolved []string // уникальные названия городов/адреса, не распознанные при последней агрегации
	bgCancel       context.CancelFunc
	lastResetDate  string
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
			cityStr = domain.NormalizeText(inv.Address)
		}
		// Если город не удалось извлечь, всё равно показываем пользователю проблемную строку:
		// номер счёта + клиент + адрес, чтобы он понимал, что именно править.
		if cityStr == "" {
			cityStr = fmt.Sprintf("סחורה %s | לקוח %s | כתובת %s",
				domain.NormalizeText(inv.InvoiceNum),
				domain.NormalizeText(inv.ClientName),
				domain.NormalizeText(inv.Address),
			)
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

func DedupKey(invoiceNum, clientName string) string {
	return domain.NormalizeText(invoiceNum) + "|" + domain.NormalizeText(clientName)
}

func (s *ReportService) FilterInvoicesForToday(invoices []*domain.AggregatedInvoice) ([]*domain.AggregatedInvoice, error) {
	today := businessDateNow()
	out := make([]*domain.AggregatedInvoice, 0, len(invoices))
	for _, inv := range invoices {
		ok, err := s.db.ExistsSentLine(today, DedupKey(inv.InvoiceNum, inv.ClientName))
		if err != nil {
			return nil, err
		}
		if ok {
			continue
		}
		out = append(out, inv)
	}
	return out, nil
}

func (s *ReportService) MarkInvoicesSent(invoices []*domain.AggregatedInvoice, reportPath string) error {
	today := businessDateNow()
	for _, inv := range invoices {
		if err := s.db.MarkSentLine(today, DedupKey(inv.InvoiceNum, inv.ClientName), inv.InvoiceNum, inv.ClientName, inv.ClientHP, reportPath); err != nil {
			return err
		}
	}
	return nil
}

func (s *ReportService) EnsureDailyReset() error {
	today := businessDateNow()
	s.mu.Lock()
	if s.lastResetDate == today {
		s.mu.Unlock()
		return nil
	}
	s.mu.Unlock()
	if err := s.db.ResetDailySentForDate(today); err != nil {
		return err
	}
	s.mu.Lock()
	s.lastResetDate = today
	s.mu.Unlock()
	return nil
}

func (s *ReportService) QueueReportForAutoSend(reportPath string) (int, error) {
	return s.db.UpsertOutboxJob(reportPath, "queued", filepath.Base(reportPath), "")
}

func (s *ReportService) SendQueuedReport(reportPath string) error {
	settings, err := s.db.GetSettings()
	if err != nil {
		return err
	}
	jobID, err := s.db.UpsertOutboxJob(reportPath, "queued", filepath.Base(reportPath), "")
	if err != nil {
		return err
	}
	subject, err := mail.SendReport(*settings, reportPath)
	if err != nil {
		_ = s.db.UpdateOutboxJobStatus(jobID, "error", err.Error())
		return err
	}
	if _, err := s.db.UpsertOutboxJob(reportPath, "sent", subject, ""); err != nil {
		return err
	}
	return s.db.UpdateOutboxJobStatus(jobID, "sent", "")
}

func (s *ReportService) GenerateAndQueueFromRaw(rawFilePath, templatePath, outputDir string) (string, error) {
	invoices, err := s.ProcessRawReport(rawFilePath)
	if err != nil {
		return "", fmt.Errorf("обработка сырого файла: %w", err)
	}
	// Частные покупатели без ח.פ не участвуют в отчётности.
	reportable := make([]*domain.AggregatedInvoice, 0, len(invoices))
	for _, inv := range invoices {
		hp := domain.NormalizeText(inv.ClientHP)
		if hp == "" || hp == "0" {
			continue
		}
		reportable = append(reportable, inv)
	}
	if len(reportable) == 0 {
		return "", fmt.Errorf("нет строк для отчётности (все записи без ח.פ)")
	}
	s.SetLastUnresolvedCities(reportable)
	for _, inv := range reportable {
		if strings.TrimSpace(inv.CityCode) == "" {
			return "", fmt.Errorf("unresolved_cities: добавьте алиасы для городов без кода; отправка невозможна")
		}
	}
	if err := s.EnsureDailyReset(); err != nil {
		return "", fmt.Errorf("суточный сброс: %w", err)
	}
	filtered, err := s.FilterInvoicesForToday(reportable)
	if err != nil {
		return "", fmt.Errorf("дедупликация: %w", err)
	}
	if len(filtered) == 0 {
		return "", fmt.Errorf("на сегодня нет новых строк для отправки")
	}
	savedPath, err := ExportToExcel(filtered, templatePath, outputDir)
	if err != nil {
		return "", err
	}
	if err := s.MarkInvoicesSent(filtered, savedPath); err != nil {
		slog.Warn("mark sent failed", "err", err)
	}
	if _, err := s.QueueReportForAutoSend(savedPath); err != nil {
		slog.Warn("queue failed", "err", err)
	}
	settings, _ := s.db.GetSettings()
	if settings != nil && settings.AutoSend {
		if err := s.SendQueuedReport(savedPath); err != nil {
			slog.Warn("auto send failed", "report", savedPath, "err", err)
		}
	}
	return savedPath, nil
}

func (s *ReportService) ApplyReplyForJob(jobID int, replyText string, replyAttachmentPath string) error {
	rows := mail.ParseReplyText(replyText)
	if replyAttachmentPath != "" {
		if parsed, err := mail.ParseReplyExcel(replyAttachmentPath); err == nil && len(parsed) > 0 {
			rows = parsed
		}
	}
	byInvoice := make(map[string]domain.ApprovalResult, len(rows))
	for _, r := range rows {
		byInvoice[domain.NormalizeText(r.InvoiceNum)] = r
	}
	jobs, err := s.db.ListOutboxJobs(200)
	if err != nil {
		return err
	}
	var reportPath string
	for _, j := range jobs {
		if j.ID == jobID {
			reportPath = j.ReportPath
			break
		}
	}
	if reportPath == "" {
		return fmt.Errorf("job не найден")
	}
	reportRows, err := readGeneratedReportRows(reportPath)
	if err != nil {
		return err
	}
	out := make([]domain.ApprovalResult, 0, len(reportRows))
	for _, rr := range reportRows {
		key := domain.NormalizeText(rr.InvoiceNum)
		if key == "" {
			continue
		}
		item := domain.ApprovalResult{
			JobID:      jobID,
			InvoiceNum: rr.InvoiceNum,
			ClientName: rr.ClientName,
			ClientHP:   rr.ClientHP,
			Status:     "approved",
		}
		if r, ok := byInvoice[key]; ok {
			item.Status = r.Status
			item.ApprovalNum = r.ApprovalNum
			item.RejectReason = r.RejectReason
			if item.Status == "" {
				item.Status = "approved"
			}
		} else {
			item.Status = "approved"
		}
		out = append(out, item)
	}
	if err := s.db.ReplaceApprovalResults(jobID, out); err != nil {
		return err
	}
	return s.db.UpdateOutboxJobStatus(jobID, "replied", "")
}

func (s *ReportService) ListOutboxJobs(limit int) ([]domain.OutboxJob, error) {
	return s.db.ListOutboxJobs(limit)
}

func (s *ReportService) ListApprovalResults(jobID int) ([]domain.ApprovalResult, error) {
	return s.db.ListApprovalResults(jobID)
}

func (s *ReportService) pollMohRepliesOnce() (int, error) {
	settings, err := s.db.GetSettings()
	if err != nil || settings == nil {
		return 0, err
	}
	replies, err := mail.FetchMohReplies(*settings)
	if err != nil {
		return 0, err
	}
	if len(replies) == 0 {
		return 0, nil
	}
	sentJobs, err := s.db.ListOutboxJobsByStatus("sent", 300)
	if err != nil {
		return 0, err
	}
	appliedCount := 0
	for _, rep := range replies {
		msgKey := "mail:" + rep.MessageID
		processed, err := s.db.IsFileProcessed(msgKey)
		if err == nil && processed {
			if rep.AttachmentPath != "" {
				_ = os.Remove(rep.AttachmentPath)
			}
			continue
		}
		jobID := s.matchReplyToJob(rep.Subject, sentJobs)
		if jobID == 0 {
			_ = s.db.MarkFileProcessed(msgKey, "mail", "unmatched")
			if rep.AttachmentPath != "" {
				_ = os.Remove(rep.AttachmentPath)
			}
			continue
		}
		if err := s.ApplyReplyForJob(jobID, rep.TextBody, rep.AttachmentPath); err != nil {
			_ = s.db.MarkFileProcessed(msgKey, "mail", "error:"+err.Error())
		} else {
			_ = s.db.MarkFileProcessed(msgKey, "mail", "applied")
			appliedCount++
		}
		if rep.AttachmentPath != "" {
			_ = os.Remove(rep.AttachmentPath)
		}
	}
	return appliedCount, nil
}

func (s *ReportService) matchReplyToJob(subject string, jobs []domain.OutboxJob) int {
	needle := strings.ToLower(subject)
	for _, j := range jobs {
		base := strings.ToLower(filepath.Base(j.ReportPath))
		if base != "" && strings.Contains(needle, base) {
			return j.ID
		}
	}
	if len(jobs) == 1 {
		return jobs[0].ID
	}
	return 0
}

func (s *ReportService) StartBackground(ctx context.Context) {
	s.mu.Lock()
	if s.bgCancel != nil {
		s.mu.Unlock()
		return
	}
	bgCtx, cancel := context.WithCancel(ctx)
	s.bgCancel = cancel
	s.mu.Unlock()
	go s.watchLoop(bgCtx)
}

func (s *ReportService) StopBackground() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.bgCancel != nil {
		s.bgCancel()
		s.bgCancel = nil
	}
}

func (s *ReportService) watchLoop(ctx context.Context) {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_ = s.EnsureDailyReset()
			_ = s.processWatchFolderOnce()
			_, _ = s.pollMohRepliesOnce()
		}
	}
}

func (s *ReportService) processWatchFolderOnce() error {
	settings, err := s.db.GetSettings()
	if err != nil {
		return err
	}
	if settings == nil || !settings.WatchEnabled || settings.WatchFolder == "" {
		return nil
	}
	files, err := os.ReadDir(settings.WatchFolder)
	if err != nil {
		return err
	}
	for _, f := range files {
		if f.IsDir() || !strings.HasSuffix(strings.ToLower(f.Name()), ".xlsx") {
			continue
		}
		full := filepath.Join(settings.WatchFolder, f.Name())
		processed, err := s.db.IsFileProcessed(full)
		if err == nil && processed {
			continue
		}
		lowerName := strings.ToLower(f.Name())
		// 1) Если это уже готовый FCS отчёт — только отправляем.
		if strings.Contains(lowerName, "fcs_report_") {
		jobID, err := s.db.UpsertOutboxJob(full, "queued", f.Name(), "")
		if err != nil {
			continue
		}
		// Если уже отправлен/обработан — пропускаем.
		jobs, _ := s.db.ListOutboxJobs(200)
		var status string
		for _, j := range jobs {
			if j.ID == jobID {
				status = j.Status
				break
			}
		}
		if status == "sent" || status == "replied" {
			continue
		}
		if err := s.SendQueuedReport(full); err != nil {
			slog.Warn("auto send failed", "file", full, "err", err)
				continue
			}
			_ = s.db.MarkFileProcessed(full, "report", "sent")
			continue
		}

		// 2) Иначе считаем, что это сырой SAP-файл: генерируем новый отчёт и отправляем.
		if settings.TemplatePath == "" || settings.OutputFolder == "" {
			slog.Warn("watcher skipped raw file: template/output not configured", "file", full)
			continue
		}
		if _, err := s.GenerateAndQueueFromRaw(full, settings.TemplatePath, settings.OutputFolder); err != nil {
			slog.Warn("watcher raw processing failed", "file", full, "err", err)
			continue
		}
		_ = s.db.MarkFileProcessed(full, "raw", "processed")
	}
	return nil
}

func (s *ReportService) ProcessWatchFolderNow() error {
	_ = s.EnsureDailyReset()
	return s.processWatchFolderOnce()
}

func (s *ReportService) PollRepliesNow() error {
	_, err := s.pollMohRepliesOnce()
	return err
}

func (s *ReportService) PollRepliesNowWithCount() (int, error) {
	return s.pollMohRepliesOnce()
}

func (s *ReportService) ResetSentRowsCounter() error {
	if err := s.db.ResetAllSentLines(); err != nil {
		return err
	}
	s.mu.Lock()
	s.lastResetDate = ""
	s.mu.Unlock()
	return nil
}
