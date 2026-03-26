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
	invoiceCol, approvalCol := -1, -1
	for i, h := range rows[0] {
		n := domain.NormalizeText(h)
		if strings.Contains(n, "משלוח") || strings.Contains(n, "חשבונית") {
			invoiceCol = i
		}
		if strings.Contains(n, "אישור") {
			approvalCol = i
		}
	}
	if invoiceCol < 0 {
		invoiceCol = 0
	}
	out := make([]domain.ApprovalResult, 0, len(rows))
	for i := 1; i < len(rows); i++ {
		r := rows[i]
		var invoice, approval string
		if invoiceCol < len(r) {
			invoice = strings.TrimSpace(r[invoiceCol])
		}
		if approvalCol >= 0 && approvalCol < len(r) {
			approval = strings.TrimSpace(r[approvalCol])
		}
		if invoice == "" {
			continue
		}
		status := "rejected"
		if approval != "" {
			status = "approved"
		}
		out = append(out, domain.ApprovalResult{
			InvoiceNum:  invoice,
			ApprovalNum: approval,
			Status:      status,
		})
	}
	return out, nil
}
