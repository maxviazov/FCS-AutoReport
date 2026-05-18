package mail

import (
	"strings"

	"fcs-autoreport/internal/domain"

	"github.com/xuri/excelize/v2"
)

func ParseReplyExcel(path string) ([]domain.ApprovalResult, error) {
	f, err := excelize.OpenFile(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	sheet := f.GetSheetName(0)
	if sheet == "" {
		return nil, nil
	}
	rows, err := f.GetRows(sheet)
	if err != nil {
		return nil, err
	}
	invoiceCol, approvalCol, statusCol := -1, -1, -1
	headerRow := 0
	for ri, row := range rows {
		if len(row) == 0 {
			continue
		}
		for i, h := range row {
			n := domain.NormalizeText(h)
			if strings.Contains(n, "משלוח") || strings.Contains(n, "חשבונית") {
				invoiceCol = i
				headerRow = ri
			}
			if strings.Contains(n, "קוד") && strings.Contains(n, "אישור") {
				approvalCol = i
				headerRow = ri
			}
			if n == "אישור שיווק" || (strings.Contains(n, "אישור") && strings.Contains(n, "שיווק") && !strings.Contains(n, "קוד")) {
				statusCol = i
				headerRow = ri
			}
		}
		if invoiceCol >= 0 && (approvalCol >= 0 || statusCol >= 0) {
			break
		}
	}
	if invoiceCol < 0 {
		invoiceCol = 0
	}
	out := make([]domain.ApprovalResult, 0, len(rows))
	for i := headerRow + 1; i < len(rows); i++ {
		r := rows[i]
		var invoice, approval, statusText string
		if invoiceCol >= 0 && invoiceCol < len(r) {
			invoice = strings.TrimSpace(r[invoiceCol])
		}
		if approvalCol >= 0 && approvalCol < len(r) {
			approval = strings.TrimSpace(r[approvalCol])
		}
		if statusCol >= 0 && statusCol < len(r) {
			statusText = strings.TrimSpace(r[statusCol])
		}
		if invoice == "" {
			continue
		}
		status := "rejected"
		rejectReason := ""
		if approval != "" && !strings.Contains(strings.ToLower(approval), "נחסמה") {
			status = "approved"
		} else if strings.Contains(statusText, "כן") || strings.EqualFold(statusText, "yes") {
			status = "approved"
		} else if statusText != "" {
			rejectReason = statusText
		}
		out = append(out, domain.ApprovalResult{
			InvoiceNum:   invoice,
			ApprovalNum:  approval,
			Status:       status,
			RejectReason: rejectReason,
		})
	}
	return out, nil
}
