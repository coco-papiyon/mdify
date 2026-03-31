package main

import (
	"testing"
)

func TestParseTableAndMarkdown(t *testing.T) {
	raw := "a\tb\tc\n1\t2\t3\n4\t5\t6"
	table := parseTable(raw)
	if len(table) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(table))
	}

	md := toMarkdown(table)
	expected := "| a | b | c |\n| --- | --- | --- |\n| 1 | 2 | 3 |\n| 4 | 5 | 6 |\n"
	if md != expected {
		t.Fatalf("unexpected markdown:\nexpected:\n%s\nactual:\n%s", expected, md)
	}
}

func TestParseTableSpaceSeparated(t *testing.T) {
	raw := "x y z\n7 8 9"
	table := parseTable(raw)
	if len(table) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(table))
	}
	if table[0][0] != "x" || table[1][2] != "9" {
		t.Fatalf("unexpected parse results: %#v", table)
	}
}

func TestParseTableNewline(t *testing.T) {
	raw := "x y\n\"7\n8\" 9"
	table := parseTable(raw)
	//for _, row := range table {
	//	fmt.Println(strings.Join(row, ", "))
	//}
	if len(table) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(table))
	}
	if table[0][0] != "x" || table[1][0] != "7<br />8" {
		t.Fatalf("unexpected parse results: %#v", table)
	}
}
