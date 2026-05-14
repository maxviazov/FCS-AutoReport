package app

import "strings"

// MohSendValidationError — готовый xlsx не прошёл проверку перед отправкой по почте.
type MohSendValidationError struct {
	Lines []string
}

func (e *MohSendValidationError) Error() string {
	if e == nil || len(e.Lines) == 0 {
		return "moh_send_blocked"
	}
	return "moh_send_blocked: " + strings.Join(e.Lines, " | ")
}
