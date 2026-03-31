package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/atotto/clipboard"
)

var DEBUG = true

func main() {
	// Open log file
	logFile, err := os.OpenFile("mdify.log", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error opening log file:", err)
		os.Exit(1)
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

	inputText, err := fetchInput(*inputPath)
	if err != nil {
		log.Printf("error fetching input: %v", err)
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}

	if DEBUG {
		log.Printf("Input: %s", inputText)
	}

	md := processText(inputText)

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
			log.Printf("error writing output file %s: %v", *outputPath, err)
			fmt.Fprintln(os.Stderr, "error writing output file:", err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stdout, "wrote markdown to %s\n", *outputPath)
		log.Printf("successfully wrote markdown to file: %s", *outputPath)
	} else {
		if err := clipboard.WriteAll(md); err != nil {
			log.Printf("error writing to clipboard: %v", err)
			fmt.Fprintln(os.Stderr, "error writing clipboard:", err)
			os.Exit(1)
		}
		fmt.Fprintln(os.Stdout, "clipboard updated with markdown table")
		log.Printf("successfully wrote markdown to clipboard")
	}

	log.Printf("mdify completed successfully")
}

func fetchInput(path string) (string, error) {
	if path != "" {
		b, err := os.ReadFile(path)
		if err != nil {
			return "", err
		}
		return string(b), nil
	}

	text, err := clipboard.ReadAll()
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(text) == "" {
		return "", errors.New("clipboard is empty")
	}
	return text, nil
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
