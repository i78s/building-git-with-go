package diff

import (
	"fmt"
	"strings"
)

const HUNK_CONTEXT = 3

type Hunk struct {
	aStarts []int
	bStart  int
	Edits   []Diffable
}

func NewHunk(aStarts []int, bStart int, edits []*Edit) *Hunk {
	return &Hunk{
		aStarts: aStarts,
		bStart:  0,
		Edits:   []Diffable{},
	}
}

type Diffable interface {
	Type() EditType
	ALines() []*Line
	ALine() *Line
	BLine() *Line
	String() string
}

func HunkFilter(edits []Diffable) []*Hunk {
	hunks := []*Hunk{}
	offset := 0

	for {
		for offset < len(edits) && edits[offset].Type() == EQL {
			offset++
		}
		if offset >= len(edits) {
			return hunks
		}

		offset -= HUNK_CONTEXT + 1

		aStarts := []int{}
		if offset >= 0 {
			for _, aline := range edits[offset].ALines() {
				aStarts = append(aStarts, aline.Number)
			}
		}

		bStart := -1
		if offset >= 0 {
			bStart = edits[offset].BLine().Number
		}

		hunks = append(hunks, NewHunk(aStarts, bStart, []*Edit{}))
		offset = HunkBuild(hunks[len(hunks)-1], edits, offset)
	}
}

func HunkBuild(hunk *Hunk, edits []Diffable, offset int) int {
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

		switch edits[idx].Type() {
		case INS, DEL:
			counter = 2*HUNK_CONTEXT + 1
		default:
			counter--
		}
	}

	return offset
}

func (h *Hunk) Header() string {
	aLines := transposeALines(h.Edits)
	var offsets []string
	for i, lines := range aLines {
		start := 0
		if i < len(h.aStarts) {
			start = h.aStarts[i]
		}
		offsets = append(offsets, format("-", lines, start))
	}

	var bLines []*Line
	for _, edit := range h.Edits {
		bLines = append(bLines, edit.BLine())
	}
	offsets = append(offsets, format("+", bLines, h.bStart))

	sep := strings.Repeat("@", len(offsets))

	return strings.Join(append([]string{sep}, append(offsets, sep)...), " ")
}

func format(sign string, lines []*Line, start int) string {
	compactLines := []*Line{}
	for _, line := range lines {
		if line != nil {
			compactLines = append(compactLines, line)
		}
	}
	if len(compactLines) > 0 {
		start = compactLines[0].Number
	}
	return fmt.Sprintf("%s%d,%d", sign, start, len(compactLines))
}

func transposeALines(edits []Diffable) [][]*Line {
	maxLength := 0
	for _, edit := range edits {
		if len(edit.ALines()) > maxLength {
			maxLength = len(edit.ALines())
		}
	}

	transposed := make([][]*Line, maxLength)
	for i := range transposed {
		transposed[i] = make([]*Line, len(edits))
		for j, edit := range edits {
			if i < len(edit.ALines()) {
				transposed[i][j] = edit.ALines()[i]
			}
		}
	}

	return transposed
}
