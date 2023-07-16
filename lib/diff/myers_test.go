package diff

import (
	"reflect"
	"strings"
	"testing"
)

func TestDiff(t *testing.T) {
	testCases := []struct {
		a, b     []string
		expected []string
	}{
		{
			a:        strings.Split("ABCABBA", ""),
			b:        strings.Split("CBABAC", ""),
			expected: []string{"-A", "-B", " C", "+B", " A", " B", "-B", " A", "+C"},
		},
		{
			a:        strings.Split("ABBBACA", ""),
			b:        strings.Split("CBABAC", ""),
			expected: []string{"+C", "+B", " A", " B", "-B", "-B", " A", " C", "-A"},
		},
		{
			a:        strings.Split("contents\n", "\n"),
			b:        strings.Split("changed\n", "\n"),
			expected: []string{"-contents", "+changed", " "},
		},
	}

	for _, tc := range testCases {
		result := Diff(tc.a, tc.b)
		if !reflect.DeepEqual(result, tc.expected) {
			t.Errorf("Diff(%v, %v) = %v, want %v", tc.a, tc.b, result, tc.expected)
		}
	}
}
