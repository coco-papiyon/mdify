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

func TestProcessHtml(t *testing.T) {
	input := `Version:0.9
StartHTML:00000097
EndHTML:00000195
StartFragment:00000131
EndFragment:00000159
<html>
<body>
<!--StartFragment-->
<table border="1">
  <tr><th>a</th><th>b</th></tr>
  <tr><td>1</td><td>2</td></tr>
  <tr><td>3</td><td>4</td></tr>
</table>
<!--EndFragment-->
</body>
</html>`
	md := processHtml(input)
	expected := "| a | b |\n| --- | --- |\n| 1 | 2 |\n| 3 | 4 |\n"
	if md != expected {
		t.Fatalf("unexpected markdown:\nexpected:\n%s\nactual:\n%s", expected, md)
	}
}

func TestProcessRtf(t *testing.T) {
	input := `{\rtf\ansi
\trowd\trgaph30\trleft-30\trrh1013\cellx1598\cellx3196\pard\plain\intbl
\qj a\cell\qj b\cell\row
\trowd\trgaph30\trleft-30\trrh1013\cellx1598\cellx3196\pard\plain\intbl
\qj 1\cell\qj 2\cell\row
}`
	md := processRtf(input)
	expected := "| a | b |\n| --- | --- |\n| 1 | 2 |\n"
	if md != expected {
		t.Fatalf("unexpected markdown:\nexpected:\n%s\nactual:\n%s", expected, md)
	}
}

func TestFromFile(t *testing.T) {
	input := "test.txt"
	md := mdify(input)
	expected := "| a | b |\n| --- | --- |\n| 1 | 2 |\n"
	if md != expected {
		t.Fatalf("unexpected markdown:\nexpected:\n%s\nactual:\n%s", expected, md)
	}
}
