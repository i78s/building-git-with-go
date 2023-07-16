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
	number int
	text   string
}

func NewLine(number int, text string) *Line {
	return &Line{
		number: number,
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
	etype EditType
	aLine *Line
	bLine *Line
}

func NewEdit(etype EditType, aLine, bLine *Line) *Edit {
	return &Edit{
		etype: etype,
		aLine: aLine,
		bLine: bLine,
	}
}

var symbols = map[EditType]string{
	EQL: " ",
	INS: "+",
	DEL: "-",
}

func (e Edit) String() string {
	line := e.aLine
	if line == nil {
		line = e.bLine
	}
	return symbols[EditType(e.etype)] + line.text
}