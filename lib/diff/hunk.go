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
			aStart = edits[offset].aLine.number
		}

		bStart := 0
		if offset >= 0 {
			bStart = edits[offset].bLine.number
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
		if e.aLine != nil {
			aLine = append(aLine, e.aLine)
		}
		if e.bLine != nil {
			bLine = append(bLine, e.bLine)
		}
	}

	aStart := h.aStart
	if len(aLine) > 0 {
		aStart = aLine[0].number
	}
	bStart := h.bStart
	if len(bLine) > 0 {
		bStart = bLine[0].number
	}

	aOffset := fmt.Sprintf("%v,%v", aStart, len(aLine))
	bOffset := fmt.Sprintf("%v,%v", bStart, len(bLine))

	return fmt.Sprintf("@@ -%s +%s @@", aOffset, bOffset)
}
