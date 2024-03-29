package diff

import (
	"strings"
)

type Combined struct {
	diffs   [][]Diffable
	offsets []int
}

func NewCombined(diffs [][]Diffable) *Combined {
	return &Combined{diffs: diffs}
}

func (c *Combined) ToSlice() []Diffable {
	var rows []Diffable
	c.Each(func(row *Row) {
		rows = append(rows, row)
	})
	return rows
}

func (c *Combined) Each(f func(*Row)) {
	c.offsets = make([]int, len(c.diffs))

	for {
		for i, diff := range c.diffs {
			c.consumeDeletions(diff, i, f)
		}
		if c.isComplete() {
			break
		}

		edits := []Diffable{}
		for _, od := range c.offsetDiffs() {
			offset := od.offset
			diff := od.diff
			edits = append(edits, diff[offset])
		}

		for i := range c.offsets {
			if c.offsets[i] < len(c.diffs[i]) {
				c.offsets[i]++
			}
		}
		f(NewRow(edits))
	}
}

func (c *Combined) isComplete() bool {
	for i, diff := range c.diffs {
		if c.offsets[i] < len(diff) {
			return false
		}
	}
	return true
}

func (c *Combined) offsetDiffs() []struct {
	offset int
	diff   []Diffable
} {
	result := make([]struct {
		offset int
		diff   []Diffable
	}, len(c.diffs))
	for i, diff := range c.diffs {
		result[i] = struct {
			offset int
			diff   []Diffable
		}{c.offsets[i], diff}
	}
	return result
}

func (c *Combined) consumeDeletions(diff []Diffable, i int, f func(*Row)) {
	for c.offsets[i] < len(diff) && diff[c.offsets[i]].Type() == DEL {
		edits := make([]Diffable, len(c.diffs))
		edits[i] = diff[c.offsets[i]]
		c.offsets[i]++

		f(NewRow(edits))
	}
}

type Row struct {
	Edits []Diffable
}

func NewRow(edits []Diffable) *Row {
	return &Row{Edits: edits}
}

func (r Row) Type() EditType {
	eTypes := []EditType{}
	for _, e := range r.Edits {
		if e == nil {
			continue
		}
		eType := e.Type()
		if eType == INS {
			return INS
		}
		eTypes = append(eTypes, e.Type())
	}
	return eTypes[0]
}

func (r Row) ALines() []*Line {
	lines := []*Line{}
	for _, e := range r.Edits {
		if e == nil {
			lines = append(lines, nil)
			continue
		}
		lines = append(lines, e.ALine())
	}
	return lines
}

func (r Row) ALine() *Line {
	return r.Edits[0].ALine()
}

func (r Row) BLine() *Line {
	if r.Edits[0] == nil {
		return nil
	}
	return r.Edits[0].BLine()
}

func (r Row) String() string {
	symbols := make([]string, len(r.Edits))

	for i, edit := range r.Edits {
		if edit == nil {
			symbols[i] = " "
			continue
		}
		symbols[i] = SYMBOLS[edit.Type()]
	}

	var del Diffable
	for _, edit := range r.Edits {
		if edit != nil && edit.Type() == DEL {
			del = edit
			break
		}
	}
	var line *Line
	if del != nil {
		line = del.ALine()
	} else {
		line = r.Edits[0].BLine()
	}

	return strings.Join(symbols, "") + line.Text
}
