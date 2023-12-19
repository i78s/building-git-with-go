package diff

import "strings"

func Diff(a, b string) []*Edit {
	m := &Myers{
		a: lines(a),
		b: lines(b),
	}
	return m.diff()
}

func DiffHunk(a, b string) []*Hunk {
	return HunkFilter(Diff(a, b))
}

func lines(s string) []*Line {
	// Replicate Ruby's String#lines
	a := strings.SplitAfter(s, "\n")
	if a[len(a)-1] == "" { // Remove trailing empty string if present
		a = a[:len(a)-1]
	}

	lines := make([]*Line, len(a))
	for i, text := range a {
		lines[i] = NewLine(i+1, text)
	}
	return lines
}

type Line struct {
	Number int
	text   string
}

func NewLine(number int, text string) *Line {
	return &Line{
		Number: number,
		text:   text,
	}
}

type EditType string

const (
	EQL EditType = "eql"
	INS EditType = "ins"
	DEL EditType = "del"
)

type Edit struct {
	Type  EditType
	ALine *Line
	BLine *Line
}

func NewEdit(etype EditType, aLine, bLine *Line) *Edit {
	return &Edit{
		Type:  etype,
		ALine: aLine,
		BLine: bLine,
	}
}

var symbols = map[EditType]string{
	EQL: " ",
	INS: "+",
	DEL: "-",
}

func (e Edit) String() string {
	line := e.ALine
	if line == nil {
		line = e.BLine
	}
	return symbols[EditType(e.Type)] + line.text
}
