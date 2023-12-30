package diff

import "strings"

type Line struct {
	Number int
	Text   string
}

func NewLine(number int, text string) *Line {
	return &Line{
		Number: number,
		Text:   text,
	}
}

type EditType string

const (
	EQL EditType = "eql"
	INS EditType = "ins"
	DEL EditType = "del"
)

var SYMBOLS = map[EditType]string{
	EQL: " ",
	INS: "+",
	DEL: "-",
}

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

func (e Edit) Type() EditType {
	return e.etype
}

func (e Edit) ALines() []*Line {
	return []*Line{e.aLine}
}

func (e Edit) ALine() *Line {
	return e.aLine
}

func (e Edit) BLine() *Line {
	return e.bLine
}

func (e Edit) String() string {
	line := e.aLine
	if line == nil {
		line = e.bLine
	}
	return SYMBOLS[EditType(e.etype)] + line.Text
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

func Diff(a, b string) []Diffable {
	m := &Myers{
		a: lines(a),
		b: lines(b),
	}
	return m.diff()
}

func DiffHunk(a, b string) []*Hunk {
	return HunkFilter(Diff(a, b))
}

func DiffCombined(as []string, b string) []Diffable {
	diffs := [][]Diffable{}
	for _, a := range as {
		diffs = append(diffs, Diff(a, b))
	}
	return NewCombined(diffs).ToSlice()
}

func DiffCombinedHunks(as []string, b string) []*Hunk {
	return HunkFilter(DiffCombined(as, b))
}
