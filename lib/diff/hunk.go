package diff

import "fmt"

const HUNK_CONTEXT = 3

type Hunk struct {
	aStart int
	bStart int
	Edits  []*Edit
}

func NewHunk(aStart, bStart int, edits []*Edit) *Hunk {
	return &Hunk{
		aStart: 0,
		bStart: 0,
		Edits:  []*Edit{},
	}
}

func HunkFilter(edits []*Edit) []*Hunk {
	hunks := []*Hunk{}
	offset := 0

	for {
		for offset < len(edits) && edits[offset].Type == EQL {
			offset++
		}
		if offset >= len(edits) {
			return hunks
		}

		offset -= HUNK_CONTEXT + 1

		aStart := 0
		if offset >= 0 {
			aStart = edits[offset].ALine.Number
		}

		bStart := 0
		if offset >= 0 {
			bStart = edits[offset].BLine.Number
		}

		hunks = append(hunks, NewHunk(aStart, bStart, []*Edit{}))
		offset = HunkBuild(hunks[len(hunks)-1], edits, offset)
	}
}

func HunkBuild(hunk *Hunk, edits []*Edit, offset int) int {
	counter := -1

	for counter != 0 {
		if offset >= 0 && counter > 0 {
			hunk.Edits = append(hunk.Edits, edits[offset])
		}

		offset++
		if offset >= len(edits) {
			break
		}

		idx := offset + HUNK_CONTEXT
		if idx >= len(edits) {
			counter--
			continue
		}

		switch edits[idx].Type {
		case INS, DEL:
			counter = 2*HUNK_CONTEXT + 1
		default:
			counter--
		}
	}

	return offset
}

func (h *Hunk) Header() string {
	aLine := []*Line{}
	bLine := []*Line{}
	for _, e := range h.Edits {
		if e.ALine != nil {
			aLine = append(aLine, e.ALine)
		}
		if e.BLine != nil {
			bLine = append(bLine, e.BLine)
		}
	}

	aStart := h.aStart
	if len(aLine) > 0 {
		aStart = aLine[0].Number
	}
	bStart := h.bStart
	if len(bLine) > 0 {
		bStart = bLine[0].Number
	}

	aOffset := fmt.Sprintf("%v,%v", aStart, len(aLine))
	bOffset := fmt.Sprintf("%v,%v", bStart, len(bLine))

	return fmt.Sprintf("@@ -%s +%s @@", aOffset, bOffset)
}
