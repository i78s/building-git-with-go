package index

import (
	"testing"
)

func TestParentDirectories(t *testing.T) {
	// Test data and expected results
	tests := []struct {
		path     string
		expected []string
	}{
		{
			path:     "dir1/dir2/dir3/file.txt",
			expected: []string{"dir1", "dir1/dir2", "dir1/dir2/dir3"},
		},
		{
			path:     "dir1/file.txt",
			expected: []string{"dir1"},
		},
		{
			path:     "file.txt",
			expected: []string{},
		},
	}

	for _, tt := range tests {
		// Create entry and call ParentDirectories
		e := &Entry{path: tt.path}
		result := e.ParentDirectories()

		// Compare results
		if len(result) != len(tt.expected) {
			t.Errorf("Expected %v but got %v", tt.expected, result)
			continue
		}

		for i := range result {
			if result[i] != tt.expected[i] {
				t.Errorf("Expected %v but got %v", tt.expected, result)
				break
			}
		}
	}
}
