package main

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"syscall"
	"unsafe"

	"github.com/wailsapp/wails/v2/pkg/runtime"
	"golang.org/x/net/html"
	"golang.org/x/sys/windows"
)

// Windows API 定義
var (
	kernel32                       = windows.NewLazySystemDLL("kernel32.dll")
	procGlobalLock                 = kernel32.NewProc("GlobalLock")
	procGlobalUnlock               = kernel32.NewProc("GlobalUnlock")
	procGlobalSize                 = kernel32.NewProc("GlobalSize")
	user32                         = windows.NewLazySystemDLL("user32.dll")
	procOpenClipboard              = user32.NewProc("OpenClipboard")
	procCloseClipboard             = user32.NewProc("CloseClipboard")
	procGetClipboardData           = user32.NewProc("GetClipboardData")
	procIsClipboardFormatAvailable = user32.NewProc("IsClipboardFormatAvailable")
	procRegisterClipboardFmt       = user32.NewProc("RegisterClipboardFormatW")
)

// CF_HTML / CF_RTF 定義
const (
	CF_UNICODETEXT uint = 13
	CF_HTML        uint = 49362 // HTML形式
	CF_RTF         uint = 49303 // LibreOfficeなどで登録されるRTF形式
)

var CF_LIST = []uint{CF_UNICODETEXT, CF_HTML, CF_RTF}

var DEBUG = false

func (a *App) ProcessClipboard(mode string) (*ClipboardResult, error) {
	input, format, err := getClipboardContent(mode)
	if err != nil {
		return nil, err
	}

	output, err := processByFormat(input, format)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(output) == "" {
		return nil, errors.New("変換結果が空です")
	}

	if err := runtime.ClipboardSetText(a.ctx, output); err != nil {
		return nil, fmt.Errorf("クリップボードへの書き込みに失敗しました: %w", err)
	}

	return &ClipboardResult{
		Input:       input,
		Output:      output,
		InputFormat: formatName(format),
	}, nil
}

func getClipboardContent(mode string) (string, uint, error) {
	if mode != "" && mode != "AUTO" {
		format, err := parseFormatMode(mode)
		if err != nil {
			return "", 0, err
		}
		text, err := readClipboardText(format)
		if err != nil {
			return "", 0, fmt.Errorf("%s をクリップボードから読み込めません: %w", formatName(format), err)
		}
		if strings.TrimSpace(text) == "" {
			return "", format, errors.New("クリップボードが空です")
		}
		return text, format, nil
	}

	var lastErr error
	for _, format := range CF_LIST {
		text, err := readClipboardText(format)
		if err == nil {
			if strings.TrimSpace(text) == "" {
				return "", format, errors.New("クリップボードが空です")
			}
			return text, format, nil
		}
		lastErr = err
	}
	if lastErr == nil {
		lastErr = errors.New("クリップボードからテキストが読み込めません")
	}
	return "", 0, lastErr
}

func parseFormatMode(mode string) (uint, error) {
	switch mode {
	case "CF_UNICODETEXT":
		return CF_UNICODETEXT, nil
	case "CF_HTML":
		return CF_HTML, nil
	case "CF_RTF":
		return CF_RTF, nil
	default:
		return 0, fmt.Errorf("未対応の変換形式です: %s", mode)
	}
}

func processByFormat(input string, format uint) (string, error) {
	switch format {
	case CF_UNICODETEXT:
		return processText(input), nil
	case CF_HTML:
		return processHTML(input)
	case CF_RTF:
		return processRtf(input), nil
	default:
		return "", fmt.Errorf("未対応のクリップボード形式です: %d", format)
	}
}

func formatName(format uint) string {
	switch format {
	case CF_UNICODETEXT:
		return "TEXT"
	case CF_HTML:
		return "HTML"
	case CF_RTF:
		return "RTF"
	default:
		return fmt.Sprintf("UNKNOWN(%d)", format)
	}
}

func convertNewlinesInQuotes(raw string) string {
	raw = strings.ReplaceAll(raw, "\r\n", "\n")
	var b strings.Builder
	inQuote := false
	for i, r := range raw {
		if r == '"' {
			inQuote = !inQuote
			b.WriteRune(r)
			continue
		}
		if r == '\n' && inQuote {
			b.WriteString("<br />")
			continue
		}
		b.WriteRune(r)
		_ = i
	}
	return b.String()
}

func splitRow(line string) []string {
	if strings.Contains(line, "\t") {
		cells := strings.Split(line, "\t")
		for i := range cells {
			cells[i] = strings.TrimSpace(cells[i])
		}
		return cells
	}

	var cells []string
	var b strings.Builder
	inQuote := false
	for _, r := range line {
		switch r {
		case '"':
			inQuote = !inQuote
		case ' ', '\t':
			if inQuote {
				b.WriteRune(r)
			} else if b.Len() > 0 {
				cells = append(cells, b.String())
				b.Reset()
			}
		default:
			b.WriteRune(r)
		}
	}
	if b.Len() > 0 {
		cells = append(cells, b.String())
	}
	return cells
}

func parseTable(raw string) [][]string {
	raw = convertNewlinesInQuotes(raw)
	raw = strings.ReplaceAll(raw, "\r\n", "\n")
	lines := strings.Split(raw, "\n")
	var table [][]string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		cells := splitRow(line)
		for i := range cells {
			cells[i] = sanitizeMarkdownCell(cells[i])
		}
		table = append(table, cells)
	}
	return table
}

func sanitizeMarkdownCell(cell string) string {
	cell = strings.TrimSpace(cell)
	cell = strings.ReplaceAll(cell, "|", "\\|")
	cell = strings.ReplaceAll(cell, "\t", " ")
	return cell
}

func processText(input string) string {
	input = convertNewlinesInQuotes(input)
	lines := strings.Split(strings.ReplaceAll(input, "\r\n", "\n"), "\n")
	var output strings.Builder
	i := 0
	for i < len(lines) {
		if strings.Contains(lines[i], "\t") {
			// 表開始
			var tableLines []string
			for i < len(lines) && strings.Contains(lines[i], "\t") {
				tableLines = append(tableLines, lines[i])
				i++
			}
			// tableLinesをparseTableで処理
			tableText := strings.Join(tableLines, "\n")
			table := parseTable(tableText)
			if len(table) > 0 {
				md := toMarkdown(table)
				output.WriteString(md)
			}
		} else {
			output.WriteString(lines[i] + "\n")
			i++
		}
	}
	return strings.TrimSuffix(output.String(), "\n")
}

func processHTML(input string) (string, error) {
	// Find the HTML fragment
	htmlStart := strings.Index(input, "<html>")
	if htmlStart == -1 {
		return "", errors.New("HTML 断片が見つかりません")
	}
	htmlEnd := strings.LastIndex(input, "</html>")
	if htmlEnd == -1 {
		htmlEnd = len(input)
	} else {
		htmlEnd += len("</html>")
	}
	htmlFragment := input[htmlStart:htmlEnd]
	doc, err := html.Parse(strings.NewReader(htmlFragment))
	if err != nil {
		return "", fmt.Errorf("HTML の解析に失敗しました: %w", err)
	}
	table := extractTable(doc)
	if table == nil {
		return "", errors.New("HTML 内に表が見つかりません")
	}
	return toMarkdown(table), nil
}

func processRtf(input string) string {
	plain := rtfToPlainText(input)
	return processText(plain)
}

func rtfToPlainText(rtf string) string {
	var b strings.Builder
	ignoreDepth := 0
	for i := 0; i < len(rtf); {
		switch rtf[i] {
		case '{':
			if ignoreDepth > 0 {
				ignoreDepth++
			}
			i++
		case '}':
			if ignoreDepth > 0 {
				ignoreDepth--
			}
			i++
		case '\\':
			i++
			if i >= len(rtf) {
				break
			}
			c := rtf[i]
			if c == '\\' || c == '{' || c == '}' {
				if ignoreDepth == 0 {
					b.WriteByte(c)
				}
				i++
				continue
			}
			if c == '*' {
				ignoreDepth++
				i++
				continue
			}

			if c == '~' {
				if ignoreDepth == 0 {
					b.WriteByte(' ')
				}
				i++
				continue
			}
			if c == '-' {
				if ignoreDepth == 0 {
					b.WriteByte('-')
				}
				i++
				continue
			}
			if c == '_' {
				if ignoreDepth == 0 {
					b.WriteByte('-')
				}
				i++
				continue
			}
			if c == '\'' {
				i++
				if i+1 <= len(rtf)-2 {
					hex := rtf[i : i+2]
					if v, err := strconv.ParseInt(hex, 16, 8); err == nil && ignoreDepth == 0 {
						b.WriteByte(byte(v))
					}
					i += 2
				}
				continue
			}

			if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') {
				start := i
				for i < len(rtf) && ((rtf[i] >= 'a' && rtf[i] <= 'z') || (rtf[i] >= 'A' && rtf[i] <= 'Z')) {
					i++
				}
				word := rtf[start:i]
				arg := 0
				hasArg := false
				if i < len(rtf) && (rtf[i] == '-' || rtf[i] == '+' || (rtf[i] >= '0' && rtf[i] <= '9')) {
					sign := 1
					if rtf[i] == '-' {
						sign = -1
						i++
					} else if rtf[i] == '+' {
						i++
					}
					valStart := i
					for i < len(rtf) && rtf[i] >= '0' && rtf[i] <= '9' {
						i++
					}
					if valStart < i {
						num, err := strconv.Atoi(rtf[valStart:i])
						if err == nil {
							arg = num * sign
							hasArg = true
						}
					}
				}
				if i < len(rtf) && rtf[i] == ' ' {
					i++
				}

				if ignoreDepth == 0 {
					switch word {
					case "par", "line", "row":
						b.WriteByte('\n')
					case "tab", "cell":
						b.WriteByte('\t')
					case "emdash":
						b.WriteRune('—')
					case "endash":
						b.WriteRune('–')
					case "u":
						if hasArg {
							u := arg
							if u < 0 {
								u += 65536
							}
							b.WriteRune(rune(u))
							if i < len(rtf) && rtf[i] != '\\' && rtf[i] != '{' && rtf[i] != '}' {
								i++
							}
						}
					}
				}
				continue
			}

			// unknown sequence, skip this char
			i++
		default:
			if ignoreDepth == 0 {
				if rtf[i] == '\r' || rtf[i] == '\n' {
					// ignore raw CR/LF from RTF structure
				} else {
					b.WriteByte(rtf[i])
				}
			}
			i++
		}
	}
	return b.String()
}

func extractTable(n *html.Node) [][]string {
	if n.Type == html.ElementNode && n.Data == "table" {
		return parseTableFromNode(n)
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if table := extractTable(c); table != nil {
			return table
		}
	}
	return nil
}

func parseTableFromNode(tableNode *html.Node) [][]string {
	var table [][]string
	var parseRows func(*html.Node)
	parseRows = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "tr" {
			var row []string
			for td := n.FirstChild; td != nil; td = td.NextSibling {
				if td.Type == html.ElementNode && (td.Data == "td" || td.Data == "th") {
					text := extractText(td)
					row = append(row, text)
				}
			}
			if len(row) > 0 {
				table = append(table, row)
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			parseRows(c)
		}
	}
	parseRows(tableNode)
	return table
}

func extractText(n *html.Node) string {
	if n.Type == html.TextNode {
		return n.Data
	}
	var text string
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		text += extractText(c)
	}
	return strings.TrimSpace(text)
}

func toMarkdown(table [][]string) string {
	maxCols := 0
	for _, row := range table {
		if len(row) > maxCols {
			maxCols = len(row)
		}
	}

	for i := range table {
		if len(table[i]) < maxCols {
			pad := make([]string, maxCols-len(table[i]))
			for j := range pad {
				pad[j] = ""
			}
			table[i] = append(table[i], pad...)
		}
	}

	var b strings.Builder
	for i, row := range table {
		b.WriteString("| ")
		b.WriteString(strings.Join(row, " | "))
		b.WriteString(" |\n")
		if i == 0 {
			b.WriteString("| ")
			sep := make([]string, maxCols)
			for i := 0; i < maxCols; i++ {
				sep[i] = "---"
			}
			b.WriteString(strings.Join(sep, " | "))
			b.WriteString(" |\n")
		}
	}
	return b.String()
}

func globalLock(h windows.Handle) uintptr {
	ret, _, _ := procGlobalLock.Call(uintptr(h))
	return ret
}

func globalUnlock(h windows.Handle) bool {
	ret, _, _ := procGlobalUnlock.Call(uintptr(h))
	return ret != 0
}

func globalSize(h windows.Handle) int {
	ret, _, _ := procGlobalSize.Call(uintptr(h))
	return int(ret)
}

func readClipboardText(format uint) (string, error) {
	ret, _, _ := procIsClipboardFormatAvailable.Call(uintptr(format))
	if ret == 0 {
		return "", fmt.Errorf("指定の形式は存在しません")
	}

	ret, _, _ = procOpenClipboard.Call(0)
	if ret == 0 {
		return "", fmt.Errorf("OpenClipboard 失敗")
	}
	defer procCloseClipboard.Call()

	h, _, _ := procGetClipboardData.Call(uintptr(format))
	if h == 0 {
		return "", fmt.Errorf("GetClipboardData 失敗")
	}

	ptr := globalLock(windows.Handle(h))
	if ptr == 0 {
		return "", fmt.Errorf("GlobalLock 失敗")
	}
	defer globalUnlock(windows.Handle(h))

	size := globalSize(windows.Handle(h))
	data := (*[1 << 30]byte)(unsafe.Pointer(ptr))[:size:size]

	if format == CF_UNICODETEXT {
		u16 := (*[1 << 30]uint16)(unsafe.Pointer(ptr))[:size/2]
		return syscall.UTF16ToString(u16), nil
	}
	return string(data), nil
}
