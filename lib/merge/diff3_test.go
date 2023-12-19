package merge

import (
	"testing"
)

func TestMergeDiff3(t *testing.T) {
	t.Run("cleanly merges two lists", func(t *testing.T) {
		merge := Merge(
			[]string{"a", "b", "c"},
			[]string{"d", "b", "c"},
			[]string{"a", "b", "e"},
		)

		if got := merge.isClean(); !got {
			t.Errorf("want %v, but got %v", true, got)
		}

		expected := "dbe"
		if got := merge.String("", ""); got != expected {
			t.Errorf("want %q, but got %q", expected, got)
		}
	})

	t.Run("cleanly merges two lists with the same edit", func(t *testing.T) {
		merge := Merge(
			[]string{"a", "b", "c"},
			[]string{"d", "b", "c"},
			[]string{"d", "b", "e"},
		)

		if got := merge.isClean(); !got {
			t.Errorf("want %v, but got %v", true, got)
		}

		expected := "dbe"
		if got := merge.String("", ""); got != expected {
			t.Errorf("want %q, but got %q", expected, got)
		}
	})

	t.Run("uncleanly merges two lists", func(t *testing.T) {
		merge := Merge(
			[]string{"a", "b", "c"},
			[]string{"d", "b", "c"},
			[]string{"e", "b", "c"},
		)

		if got := merge.isClean(); got {
			t.Errorf("want %v, but got %v", true, got)
		}

		expected := `<<<<<<<
d=======
e>>>>>>>
bc`
		if got := merge.String("", ""); got != expected {
			t.Errorf("want %q, but got %q", expected, got)
		}
	})

	t.Run("uncleanly merges two lists against an empty list", func(t *testing.T) {
		merge := Merge(
			[]string{},
			[]string{"d", "b", "c"},
			[]string{"e", "b", "c"},
		)

		if got := merge.isClean(); got {
			t.Errorf("want %v, but got %v", true, got)
		}

		expected := `<<<<<<<
dbc=======
ebc>>>>>>>
`
		if got := merge.String("", ""); got != expected {
			t.Errorf("want %q, but got %q", expected, got)
		}
	})

	t.Run("uncleanly merges two lists with head names", func(t *testing.T) {
		merge := Merge(
			[]string{"a", "b", "c"},
			[]string{"d", "b", "c"},
			[]string{"e", "b", "c"},
		)

		if got := merge.isClean(); got {
			t.Errorf("want %v, but got %v", true, got)
		}

		expected := `<<<<<<< left
d=======
e>>>>>>> right
bc`
		if got := merge.String("left", "right"); got != expected {
			t.Errorf("want %q, but got %q", expected, got)
		}
	})
}
