package merge

import (
	"building-git/lib/diff"
	"strings"
)

type Clean struct {
	lines []string
}

func NewClean(lines []string) *Clean {
	return &Clean{
		lines: lines,
	}
}

func (c *Clean) String(aName, bName string) string {
	return strings.Join(c.lines, "")
}

type Conflict struct {
	oLines []string
	aLines []string
	bLines []string
}

func NewConflict(oLines, aLines, bLines []string) *Conflict {
	return &Conflict{
		oLines: oLines,
		aLines: aLines,
		bLines: bLines,
	}
}

func (c *Conflict) String(aName, bName string) string {
	var builder strings.Builder

	c.separator(&builder, "<", aName)
	for _, line := range c.aLines {
		builder.WriteString(line)
	}
	c.separator(&builder, "=", "")
	for _, line := range c.bLines {
		builder.WriteString(line)
	}
	c.separator(&builder, ">", bName)

	return builder.String()
}

func (c *Conflict) separator(builder *strings.Builder, char, name string) {
	builder.WriteString(strings.Repeat(char, 7))
	if name != "" {
		builder.WriteString(" " + name)
	}
	builder.WriteString("\n")
}

type chunk interface {
	String(aName, bName string) string
}

type Result struct {
	chunks []chunk
}

func NewResult(chunks []chunk) *Result {
	return &Result{
		chunks: chunks,
	}
}

func (r *Result) isClean() bool {
	for _, c := range r.chunks {
		if _, ok := c.(*Conflict); ok {
			return false
		}
	}
	return true
}

func (r *Result) String(aName, bName string) string {
	chunks := []string{}
	for _, c := range r.chunks {
		chunks = append(chunks, c.String(aName, bName))
	}
	return strings.Join(chunks, "")
}

func Merge(o, a, b interface{}) *Result {
	oLines := convertToLines(o)
	aLines := convertToLines(a)
	bLines := convertToLines(b)

	diff3 := NewDiff3(oLines, aLines, bLines)
	return diff3.Merge()
}

func convertToLines(input interface{}) []string {
	switch v := input.(type) {
	case string:
		result := strings.SplitAfter(v, "\n")
		if result[len(result)-1] == "" {
			result = result[:len(result)-1]
		}
		return result
	case []string:
		return v
	default:
		return []string{}
	}
}

type Diff3 struct {
	O, A, B             []string
	chunks              []chunk
	matchA, matchB      map[int]int
	lineO, lineA, lineB int
}

func NewDiff3(o, a, b []string) *Diff3 {
	return &Diff3{O: o, A: a, B: b}
}

func (d *Diff3) Merge() *Result {
	d.setup()
	d.generateChunks()
	return NewResult(d.chunks)
}

func (d *Diff3) setup() {
	d.chunks = []chunk{}
	d.lineO = 0
	d.lineA = 0
	d.lineB = 0

	d.matchA = d.matchSet(d.A)
	d.matchB = d.matchSet(d.B)
}

func (d *Diff3) matchSet(file []string) map[int]int {
	matches := make(map[int]int)
	diffs := diff.Diff(strings.Join(d.O, "\n"), strings.Join(file, "\n"))
	for _, edit := range diffs {
		if edit.Type() == diff.EQL {
			matches[edit.ALine().Number] = edit.BLine().Number
		}
	}
	return matches
}

func (d *Diff3) generateChunks() Result {
	for {
		i, found := d.findNextMismatch()
		if !found {
			d.emitFinalChunk()
			return Result{chunks: d.chunks}
		}

		if i == 1 {
			o, a, b, matchFound := d.findNextMatch()
			if matchFound {
				d.emitChunk(o, a, b)
			} else {
				d.emitFinalChunk()
				return Result{chunks: d.chunks}
			}
		} else {
			d.emitChunk(d.lineO+i, d.lineA+i, d.lineB+i)
		}
	}
}

func (d *Diff3) findNextMismatch() (int, bool) {
	i := 1
	for d.inBounds(i) && d.match(d.matchA, d.lineA, i) && d.match(d.matchB, d.lineB, i) {
		i++
	}
	return i, d.inBounds(i)
}

func (d *Diff3) inBounds(i int) bool {
	return d.lineO+i <= len(d.O) || d.lineA+i <= len(d.A) || d.lineB+i <= len(d.B)
}

func (d *Diff3) match(matches map[int]int, offset, i int) bool {
	matchIndex, exists := matches[d.lineO+i]
	return exists && matchIndex == offset+i
}

func (d *Diff3) findNextMatch() (int, int, int, bool) {
	o := d.lineO + 1
	for o <= len(d.O) {
		_, aExists := d.matchA[o]
		_, bExists := d.matchB[o]
		if aExists && bExists {
			a := d.matchA[o]
			b := d.matchB[o]
			return o, a, b, true
		}
		o++
	}
	return 0, 0, 0, false
}

func (d *Diff3) emitChunk(o, a, b int) {
	oChunk := d.O[d.lineO : o-1]
	aChunk := d.A[d.lineA : a-1]
	bChunk := d.B[d.lineB : b-1]

	if areSlicesEqual(aChunk, oChunk) || areSlicesEqual(aChunk, bChunk) {
		d.chunks = append(d.chunks, &Clean{lines: bChunk})
	} else if areSlicesEqual(bChunk, oChunk) {
		d.chunks = append(d.chunks, &Clean{lines: aChunk})
	} else {
		d.chunks = append(d.chunks, &Conflict{oLines: oChunk, aLines: aChunk, bLines: bChunk})
	}

	d.lineO, d.lineA, d.lineB = o-1, a-1, b-1
}

func (d *Diff3) emitFinalChunk() {
	oChunk := d.O[d.lineO:]
	aChunk := d.A[d.lineA:]
	bChunk := d.B[d.lineB:]

	if areSlicesEqual(aChunk, oChunk) || areSlicesEqual(aChunk, bChunk) {
		d.chunks = append(d.chunks, &Clean{lines: bChunk})
	} else if areSlicesEqual(bChunk, oChunk) {
		d.chunks = append(d.chunks, &Clean{lines: aChunk})
	} else {
		d.chunks = append(d.chunks, &Conflict{oLines: oChunk, aLines: aChunk, bLines: bChunk})
	}
}

func areSlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
