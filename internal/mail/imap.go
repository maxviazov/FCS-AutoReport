package mail

import (
	"bytes"
	"bufio"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/mail"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"fcs-autoreport/internal/domain"

	"golang.org/x/text/encoding/charmap"
)

const MohReplySender = "mazon_no_relpy@moh.health.gov.il"

type ReceivedReply struct {
	MessageID      string
	From           string
	Subject        string
	TextBody       string
	AttachmentPath string
}

func FetchMohReplies(settings domain.Settings) ([]ReceivedReply, error) {
	if settings.IMAPHost == "" || settings.IMAPUser == "" || settings.IMAPPassword == "" || settings.IMAPPort <= 0 {
		return nil, nil
	}
	addr := settings.IMAPHost + ":" + strconvI(settings.IMAPPort)
	conn, err := tls.Dial("tcp", addr, &tls.Config{ServerName: settings.IMAPHost})
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	br := bufio.NewReader(conn)
	bw := bufio.NewWriter(conn)
	_, _ = br.ReadString('\n') // greeting
	if _, err := imapCmd(br, bw, `LOGIN "%s" "%s"`, escIMAP(settings.IMAPUser), escIMAP(settings.IMAPPassword)); err != nil {
		return nil, err
	}
	if _, err := imapCmd(br, bw, "SELECT INBOX"); err != nil {
		return nil, err
	}
	searchResp, err := imapCmd(br, bw, `SEARCH UNSEEN HEADER FROM "%s"`, escIMAP(MohReplySender))
	if err != nil {
		return nil, err
	}
	ids := parseSearchIDs(searchResp)
	if len(ids) == 0 {
		searchResp, err = imapCmd(br, bw, `SEARCH HEADER FROM "%s"`, escIMAP(MohReplySender))
		if err != nil {
			return nil, err
		}
		ids = parseSearchIDs(searchResp)
	}
	if len(ids) == 0 {
		return nil, nil
	}
	var out []ReceivedReply
	for _, id := range ids {
		rawMsg, err := imapFetchRFC822(br, bw, id)
		if err != nil || len(rawMsg) == 0 {
			continue
		}
		parsed, err := parseRawMail(rawMsg)
		if err != nil {
			continue
		}
		if parsed.MessageID == "" {
			parsed.MessageID = "seq-" + id
		}
		if !strings.Contains(strings.ToLower(parsed.From), strings.ToLower(MohReplySender)) {
			continue
		}
		out = append(out, parsed)
	}
	_, _ = imapCmd(br, bw, "LOGOUT")
	return out, nil
}

func parseRawMail(raw []byte) (ReceivedReply, error) {
	r, err := mail.ReadMessage(bytes.NewReader(raw))
	if err != nil {
		return ReceivedReply{}, err
	}
	res := ReceivedReply{
		MessageID: strings.TrimSpace(r.Header.Get("Message-Id")),
		From:      strings.TrimSpace(r.Header.Get("From")),
		Subject:   decodeHeader(r.Header.Get("Subject")),
	}
	ctype := r.Header.Get("Content-Type")
	mt, params, _ := mime.ParseMediaType(ctype)
	if strings.HasPrefix(mt, "multipart/") {
		mr := multipart.NewReader(r.Body, params["boundary"])
		for {
			p, err := mr.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				break
			}
			data, _ := io.ReadAll(p)
			data = decodePartByEncoding(data, p.Header.Get("Content-Transfer-Encoding"))
			pct := p.Header.Get("Content-Type")
			disp := p.Header.Get("Content-Disposition")
			filename := partFileName(p)
			if strings.Contains(strings.ToLower(filename), ".xlsx") || strings.Contains(strings.ToLower(pct), "spreadsheetml") {
				fp, _ := writeTempAttachment(filename, data)
				res.AttachmentPath = fp
				continue
			}
			if strings.Contains(strings.ToLower(pct), "text/plain") || strings.Contains(strings.ToLower(disp), "inline") {
				res.TextBody += decodeBodyText(data, pct) + "\n"
			}
		}
		return res, nil
	}
	data, _ := io.ReadAll(r.Body)
	res.TextBody = decodeBodyText(decodePartByEncoding(data, r.Header.Get("Content-Transfer-Encoding")), ctype)
	return res, nil
}

func decodeHeader(v string) string {
	dec := new(mime.WordDecoder)
	dec.CharsetReader = func(charsetName string, input io.Reader) (io.Reader, error) {
		switch strings.ToLower(charsetName) {
		case "windows-1255":
			return charmap.Windows1255.NewDecoder().Reader(input), nil
		}
		return input, nil
	}
	s, err := dec.DecodeHeader(v)
	if err != nil {
		return v
	}
	return s
}

func decodePartByEncoding(data []byte, enc string) []byte {
	switch strings.ToLower(strings.TrimSpace(enc)) {
	case "base64":
		dst := make([]byte, base64.StdEncoding.DecodedLen(len(data)))
		n, err := base64.StdEncoding.Decode(dst, bytes.TrimSpace(data))
		if err == nil {
			return dst[:n]
		}
	}
	return data
}

func decodeBodyText(data []byte, ctype string) string {
	mt, params, _ := mime.ParseMediaType(ctype)
	if !strings.Contains(strings.ToLower(mt), "text/") {
		return string(data)
	}
	cs := strings.ToLower(params["charset"])
	if cs == "windows-1255" {
		dec, err := charmap.Windows1255.NewDecoder().Bytes(data)
		if err == nil {
			return string(dec)
		}
	}
	return string(data)
}

func partFileName(p *multipart.Part) string {
	if p.FileName() != "" {
		return p.FileName()
	}
	_, params, _ := mime.ParseMediaType(p.Header.Get("Content-Disposition"))
	return params["filename"]
}

func writeTempAttachment(name string, data []byte) (string, error) {
	if strings.TrimSpace(name) == "" {
		name = "reply.xlsx"
	}
	path := filepath.Join(os.TempDir(), "fcs_reply_"+name)
	if err := os.WriteFile(path, data, 0644); err != nil {
		return "", err
	}
	return path, nil
}

func strconvI(n int) string {
	return fmt.Sprintf("%d", n)
}

func escIMAP(s string) string {
	return strings.ReplaceAll(s, `"`, `\"`)
}

func imapCmd(br *bufio.Reader, bw *bufio.Writer, format string, args ...interface{}) ([]string, error) {
	tag := "A" + strconvI(int(time.Now().UnixNano()%1000000))
	cmd := fmt.Sprintf(format, args...)
	if _, err := bw.WriteString(tag + " " + cmd + "\r\n"); err != nil {
		return nil, err
	}
	if err := bw.Flush(); err != nil {
		return nil, err
	}
	var lines []string
	for {
		line, err := br.ReadString('\n')
		if err != nil {
			return lines, err
		}
		line = strings.TrimRight(line, "\r\n")
		lines = append(lines, line)
		if strings.HasPrefix(line, tag+" ") {
			if strings.Contains(strings.ToUpper(line), "OK") {
				return lines, nil
			}
			return lines, fmt.Errorf("imap command failed: %s", line)
		}
	}
}

func parseSearchIDs(lines []string) []string {
	var out []string
	for _, ln := range lines {
		if strings.HasPrefix(strings.ToUpper(ln), "* SEARCH") {
			parts := strings.Fields(ln)
			if len(parts) > 2 {
				out = append(out, parts[2:]...)
			}
		}
	}
	return out
}

func imapFetchRFC822(br *bufio.Reader, bw *bufio.Writer, id string) ([]byte, error) {
	tag := "F" + strconvI(int(time.Now().UnixNano()%1000000))
	if _, err := bw.WriteString(fmt.Sprintf("%s FETCH %s (RFC822)\r\n", tag, id)); err != nil {
		return nil, err
	}
	if err := bw.Flush(); err != nil {
		return nil, err
	}
	var raw []byte
	litRe := regexp.MustCompile(`\{(\d+)\}$`)
	for {
		line, err := br.ReadString('\n')
		if err != nil {
			return nil, err
		}
		trimmed := strings.TrimRight(line, "\r\n")
		if m := litRe.FindStringSubmatch(trimmed); len(m) == 2 {
			n := 0
			fmt.Sscanf(m[1], "%d", &n)
			if n > 0 {
				buf := make([]byte, n)
				if _, err := io.ReadFull(br, buf); err != nil {
					return nil, err
				}
				raw = append(raw, buf...)
				_, _ = br.ReadString('\n') // closing line after literal
			}
		}
		if strings.HasPrefix(trimmed, tag+" ") {
			if strings.Contains(strings.ToUpper(trimmed), "OK") {
				return raw, nil
			}
			return nil, fmt.Errorf("fetch failed: %s", trimmed)
		}
	}
}
