package diff

import (
	"fmt"
	"reflect"
	"testing"
)

func TestDiff(t *testing.T) {
	testCases := []struct {
		a, b     string
		expected []string
	}{
		{
			a:        "A\nB\nC\nA\nB\nB\nA\n",
			b:        "C\nB\nA\nB\nA\nC\n",
			expected: []string{"-A\n", "-B\n", " C\n", "+B\n", " A\n", " B\n", "-B\n", " A\n", "+C\n"},
		},
		{
			a:        "A\nB\nB\nB\nA\nC\nA\n",
			b:        "C\nB\nA\nB\nA\nC\n",
			expected: []string{"+C\n", "+B\n", " A\n", " B\n", "-B\n", "-B\n", " A\n", " C\n", "-A\n"},
		},
		{
			a:        "contents\n",
			b:        "changed\n",
			expected: []string{"-contents\n", "+changed\n"},
		},
		{
			a:        "",
			b:        "added\n",
			expected: []string{"+added\n"},
		},
	}

	for _, tc := range testCases {
		result := fmt.Sprintf("%v", Diff(tc.a, tc.b))
		expected := fmt.Sprintf("%v", tc.expected)

		if !reflect.DeepEqual(result, expected) {
			t.Errorf("Diff(%v, %v) = %v, want %v", tc.a, tc.b, result, tc.expected)
		}
	}
}

func TestDiffHunk(t *testing.T) {
	diffHunks := func(a, b string) []string {
		hunks := []string{}
		for _, hunk := range DiffHunk(a, b) {
			hunks = append(hunks, hunk.Header())
			for _, e := range hunk.Edits {
				hunks = append(hunks, e.String())
			}
		}
		return hunks
	}
	doc := "the\nquick\nbrown\nfox\njumps\nover\nthe\nlazy\ndog"

	t.Run("detects a deletion at the start", func(t *testing.T) {
		changed := "quick\nbrown\nfox\njumps\nover\nthe\nlazy\ndog"
		result := diffHunks(doc, changed)
		expected := []string{"@@ -1,4 +1,3 @@",
			"-the\n",
			" quick\n",
			" brown\n",
			" fox\n",
		}
		if !reflect.DeepEqual(result, expected) {
			t.Errorf("DiffHunk(%v, %v) = %v, want %v", doc, changed, result, expected)
		}
	})

	t.Run("detects an insertion at the start", func(t *testing.T) {
		changed := "so\nthe\nquick\nbrown\nfox\njumps\nover\nthe\nlazy\ndog"
		result := diffHunks(doc, changed)
		expected := []string{"@@ -1,3 +1,4 @@",
			"+so\n",
			" the\n",
			" quick\n",
			" brown\n",
		}
		if !reflect.DeepEqual(result, expected) {
			t.Errorf("DiffHunk(%v, %v) = %v, want %v", doc, changed, result, expected)
		}
	})

	t.Run("detects a change skipping the start and end", func(t *testing.T) {
		changed := "the\nquick\nbrown\nfox\nleaps\nright\nover\nthe\nlazy\ndog"
		result := diffHunks(doc, changed)
		expected := []string{"@@ -2,7 +2,8 @@",
			" quick\n",
			" brown\n",
			" fox\n",
			"-jumps\n",
			"+leaps\n",
			"+right\n",
			" over\n",
			" the\n",
			" lazy\n",
		}
		if !reflect.DeepEqual(result, expected) {
			t.Errorf("DiffHunk(%v, %v) = %v, want %v", doc, changed, result, expected)
		}
	})

	t.Run("puts nearby changes in the same hunk", func(t *testing.T) {
		changed := "the\nbrown\nfox\njumps\nover\nthe\nlazy\ncat"
		result := diffHunks(doc, changed)
		expected := []string{"@@ -1,9 +1,8 @@",
			" the\n",
			"-quick\n",
			" brown\n",
			" fox\n",
			" jumps\n",
			" over\n",
			" the\n",
			" lazy\n",
			"-dog",
			"+cat",
		}
		if !reflect.DeepEqual(result, expected) {
			t.Errorf("DiffHunk(%v, %v) = %v, want %v", doc, changed, result, expected)
		}
	})

	t.Run("puts distant changes in different hunks", func(t *testing.T) {
		changed := "a\nquick\nbrown\nfox\njumps\nover\nthe\nlazy\ncat"
		result := diffHunks(doc, changed)
		expected := []string{
			"@@ -1,4 +1,4 @@",
			"-the\n",
			"+a\n",
			" quick\n",
			" brown\n",
			" fox\n",
			"@@ -6,4 +6,4 @@",
			" over\n",
			" the\n",
			" lazy\n",
			"-dog",
			"+cat",
		}
		if !reflect.DeepEqual(result, expected) {
			t.Errorf("DiffHunk(%v, %v) = %v, want %v", doc, changed, result, expected)
		}
	})
}
