package app

import "strings"

// MohExportValidationError — накладные не прошли проверку обязательных полей МОЗ перед экспортом.
// Используется с errors.As в сервисе, чтобы отдать UI список строк (GetLastMohValidationFailures).
type MohExportValidationError struct {
	Lines []string
}

func (e *MohExportValidationError) Error() string {
	if e == nil {
		return "moh_export_blocked"
	}
	if len(e.Lines) == 0 {
		return "moh_export_blocked"
	}
	return "moh_export_blocked: " + strings.Join(e.Lines, " | ")
}
