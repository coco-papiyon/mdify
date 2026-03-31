package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"github.com/atotto/clipboard"
	"github.com/gen2brain/beeep"
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
)

// CF_HTML / CF_RTF 定義
const (
	CF_UNICODETEXT = 13
	CF_HTML        = 49362  // 通常登録されるHTML形式
	CF_RTF         = 0xC0A0 // LibreOfficeなどで登録されるRTF形式
)

var CF_LIST = []uint{CF_UNICODETEXT, CF_HTML, CF_RTF}

var DEBUG = true

func main() {
	// Change current working directory to the directory containing the executable.
	if exePath, err := os.Executable(); err != nil {
		fatalError("error getting executable path: %v", err)
	} else if exeDir := filepath.Dir(exePath); exeDir != "" {
		if err := os.Chdir(exeDir); err != nil {
			fatalError("error changing directory to executable dir %s: %v", exeDir, err)
		}
	}

	// Open log file
	logFile, err := os.OpenFile("mdify.log", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		fatalError("error opening log file: %v", err)
	}
	defer logFile.Close()
	log.SetOutput(logFile)
	log.SetFlags(log.LstdFlags)

	log.Printf("mdify started at %s", time.Now().Format("2006-01-02 15:04:05"))

	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: %s [options]\n", os.Args[0])
		flag.PrintDefaults()
	}

	inputPath := flag.String("input", "", "input file path (optional; clipboard used if omitted)")
	outputPath := flag.String("output", "", "output file path (optional; clipboard used if omitted)")
	dryRun := flag.Bool("dry-run", false, "print markdown to stdout without modifying clipboard/file")
	flag.Parse()

	inputText, format, err := fetchInput(*inputPath)
	if err != nil {
		fatalError("error fetching input: %v", err)
	}

	if DEBUG {
		log.Printf("Input (%d): %s", format, inputText)
	}

	var md string
	if format == CF_UNICODETEXT {
		md = processText(inputText)
	} else {
		md = processHtml(inputText)
	}

	if DEBUG {
		log.Printf("Output: %s", md)
	}

	if *dryRun {
		fmt.Println(md)
		log.Printf("dry-run completed, output to stdout")
		return
	}

	if *outputPath != "" {
		if err := os.WriteFile(*outputPath, []byte(md), 0644); err != nil {
			fatalError("error writing output file %s: %v", *outputPath, err)
		}
		fmt.Fprintf(os.Stdout, "wrote markdown to %s\n", *outputPath)
		log.Printf("successfully wrote markdown to file: %s", *outputPath)
	} else {
		if err := clipboard.WriteAll(md); err != nil {
			fatalError("error writing to clipboard: %v", err)
		}
		fmt.Fprintln(os.Stdout, "clipboard updated with markdown table")
		log.Printf("successfully wrote markdown to clipboard")
	}

	log.Printf("mdify completed successfully")
}

func fetchInput(path string) (string, uint, error) {
	if path != "" {
		b, err := os.ReadFile(path)
		if err != nil {
			log.Printf("error reading input file %s: %v", path, err)
			return "", 0, err
		}
		return string(b), 0, nil
	}
	var text string
	var err error
	for _, format := range CF_LIST {
		text, err = readClipboardText(format)
		if err == nil {
			if strings.TrimSpace(text) == "" {
				return "", format, errors.New("clipboard is empty")
			}
		}
	}
	log.Printf("error reading clipboard data: %v", err)
	return "", 0, err
}

func showErrorDialog(message string) {
	if err := beeep.Alert("mdify", message, ""); err != nil {
		log.Printf("failed to show alert: %v", err)
	}
}

func fatalError(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	log.Println(msg)
	fmt.Fprintln(os.Stderr, "error:", msg)
	showErrorDialog(msg)
	os.Exit(1)
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

func processHtml(input string) string {
	// Find the HTML fragment
	htmlStart := strings.Index(input, "<html>")
	if htmlStart == -1 {
		return ""
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
		fatalError("error parsing HTML: %v", err)
	}
	table := extractTable(doc)
	if table == nil {
		return ""
	}
	return toMarkdown(table)
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
