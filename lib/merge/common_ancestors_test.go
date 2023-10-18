package merge

import (
	"building-git/lib/database"
	"os"
	"testing"
	"time"
)

var setUp = func(t *testing.T) (
	tmpDir string,
	commit func(parents []string, message string, now time.Time),
	chain func(names []string),
	ancestor func(left, right string) []string,
	mergeBase func(left, right string) []string,
) {
	tmpDir, _ = os.MkdirTemp("", "test-database")

	db := database.NewDatabase(tmpDir)
	commits := map[string]string{}

	commit = func(parentMsgs []string, message string, now time.Time) {
		parents := []string{}
		for _, msg := range parentMsgs {
			parents = append(parents, commits[msg])
		}

		author := database.NewAuthor("A. U. Thor", "author@example.com", now)
		c := database.NewCommit(parents, "'0' *  40", author, message)
		db.Store(c)
		commits[message] = c.Oid()
	}

	chain = func(names []string) {
		for i := 0; i < len(names)-1; i++ {
			parentMsgs := []string{}
			if names[i] != "" {
				parentMsgs = append(parentMsgs, names[i])
			}
			message := names[i+1]
			commit(parentMsgs, message, time.Now())
		}
	}

	ancestor = func(left, right string) []string {
		common := NewCommonAncestors(db, commits[left], []string{commits[right]})
		commitMessages := []string{}

		for _, oid := range common.Find() {
			obj, _ := db.Load(oid)
			commitMessages = append(commitMessages, obj.(*database.Commit).Message())
		}
		return commitMessages
	}

	mergeBase = func(left, right string) []string {
		bases := NewBases(db, commits[left], commits[right])
		commitMessages := []string{}

		for _, oid := range bases.Find() {
			obj, _ := db.Load(oid)
			commitMessages = append(commitMessages, obj.(*database.Commit).Message())
		}
		return commitMessages
	}

	return
}

func TestCommonAncestorWithLinearHistory(t *testing.T) {
	//   o---o---o---o
	//   A   B   C   D

	var before = func(chain func([]string)) {
		chain([]string{"", "A", "B", "C", "D"})
	}

	t.Run("finds the common ancestor of a commit with itself", func(t *testing.T) {
		tmpDir, _, chain, ancestor, _ := setUp(t)
		defer os.RemoveAll(tmpDir)

		before(chain)

		expected := "D"
		if got := ancestor("D", "D"); got[0] != expected {
			t.Errorf("want %q, but got %q", expected, got)
		}
	})

	t.Run("finds the commit that is an ancestor of the other", func(t *testing.T) {
		tmpDir, _, chain, ancestor, _ := setUp(t)
		defer os.RemoveAll(tmpDir)

		before(chain)

		expected := "B"
		if got := ancestor("B", "D"); got[0] != expected {
			t.Errorf("want %q, but got %q", expected, got)
		}
	})

	t.Run("find the same commit if the arguments are reversed", func(t *testing.T) {
		tmpDir, _, chain, ancestor, _ := setUp(t)
		defer os.RemoveAll(tmpDir)

		before(chain)

		expected := "B"
		if got := ancestor("D", "B"); got[0] != expected {
			t.Errorf("want %q, but got %q", expected, got)
		}
	})

	t.Run("finds a root commit", func(t *testing.T) {
		tmpDir, _, chain, ancestor, _ := setUp(t)
		defer os.RemoveAll(tmpDir)

		before(chain)

		expected := "A"
		if got := ancestor("A", "C"); got[0] != expected {
			t.Errorf("want %q, but got %q", expected, got)
		}
	})

	t.Run("finds the intersection of a root commit with itself", func(t *testing.T) {
		tmpDir, _, chain, ancestor, _ := setUp(t)
		defer os.RemoveAll(tmpDir)

		before(chain)

		expected := "A"
		if got := ancestor("A", "A"); got[0] != expected {
			t.Errorf("want %q, but got %q", expected, got)
		}
	})
}

func TestCommonAncestorWithForkingHistory(t *testing.T) {
	//          E   F   G   H
	//          o---o---o---o
	//         /         \
	//        /  C   D    \
	//   o---o---o---o     o---o
	//   A   B    \        J   K
	//             \
	//              o---o---o
	//              L   M   N

	var before = func(chain func([]string)) {
		chain([]string{"", "A", "B", "C", "D"})
		chain([]string{"B", "E", "F", "G", "H"})
		chain([]string{"G", "J", "K"})
		chain([]string{"C", "L", "M", "N"})
	}

	t.Run("finds the nearest fork point", func(t *testing.T) {
		tmpDir, _, chain, ancestor, _ := setUp(t)
		defer os.RemoveAll(tmpDir)

		before(chain)

		expected := "G"
		if got := ancestor("H", "K"); got[0] != expected {
			t.Errorf("want %q, but got %q", expected, got)
		}
	})

	t.Run("finds an ancestor multiple forks away", func(t *testing.T) {
		tmpDir, _, chain, ancestor, _ := setUp(t)
		defer os.RemoveAll(tmpDir)

		before(chain)

		expected := "B"
		if got := ancestor("D", "K"); got[0] != expected {
			t.Errorf("want %q, but got %q", expected, got)
		}
	})

	t.Run("finds the same fork point for any point on a branch", func(t *testing.T) {
		tmpDir, _, chain, ancestor, _ := setUp(t)
		defer os.RemoveAll(tmpDir)

		before(chain)

		expected := "C"
		if got := ancestor("D", "L"); got[0] != expected {
			t.Errorf("want %q, but got %q", expected, got)
		}
		if got := ancestor("M", "D"); got[0] != expected {
			t.Errorf("want %q, but got %q", expected, got)
		}
		if got := ancestor("D", "N"); got[0] != expected {
			t.Errorf("want %q, but got %q", expected, got)
		}
	})

	t.Run("finds the commit that is an ancestor of the other", func(t *testing.T) {
		tmpDir, _, chain, ancestor, _ := setUp(t)
		defer os.RemoveAll(tmpDir)

		before(chain)

		expected := "E"
		if got := ancestor("K", "E"); got[0] != expected {
			t.Errorf("want %q, but got %q", expected, got)
		}
	})

	t.Run("finds a root commit", func(t *testing.T) {
		tmpDir, _, chain, ancestor, _ := setUp(t)
		defer os.RemoveAll(tmpDir)

		before(chain)

		expected := "A"
		if got := ancestor("J", "A"); got[0] != expected {
			t.Errorf("want %q, but got %q", expected, got)
		}
	})
}

func TestCommonAncestorWithMerge(t *testing.T) {
	//   A   B   C   G   H
	//   o---o---o---o---o
	//        \     /
	//         o---o---o
	//         D   E   F

	var before = func(chain func([]string), commit func(parents []string, message string, now time.Time)) {
		chain([]string{"", "A", "B", "C"})
		chain([]string{"B", "D", "E", "F"})
		commit([]string{"C", "E"}, "G", time.Now())
		chain([]string{"G", "H"})
	}

	t.Run("finds the most recent common ancestor", func(t *testing.T) {
		tmpDir, commit, chain, ancestor, _ := setUp(t)
		defer os.RemoveAll(tmpDir)

		before(chain, commit)

		expected := "E"
		if got := ancestor("H", "F"); got[0] != expected {
			t.Errorf("want %q, but got %q", expected, got)
		}
	})

	t.Run("finds the common ancestor of a merge and its parents", func(t *testing.T) {
		tmpDir, commit, chain, ancestor, _ := setUp(t)
		defer os.RemoveAll(tmpDir)

		before(chain, commit)

		expected := "C"
		if got := ancestor("C", "G"); got[0] != expected {
			t.Errorf("want %q, but got %q", expected, got)
		}
		expected = "E"
		if got := ancestor("G", "E"); got[0] != expected {
			t.Errorf("want %q, but got %q", expected, got)
		}
	})
}

func TestCommonAncestorWithMergeFurtherFromOneParent(t *testing.T) {
	//   A   B   C   G   H   J
	//   o---o---o---o---o---o
	//        \     /
	//         o---o---o
	//         D   E   F

	var before = func(chain func([]string), commit func(parents []string, message string, now time.Time)) {
		chain([]string{"", "A", "B", "C"})
		chain([]string{"B", "D", "E", "F"})
		commit([]string{"C", "E"}, "G", time.Now())
		chain([]string{"G", "H", "J"})
	}

	t.Run("finds all the common ancestors", func(t *testing.T) {
		tmpDir, commit, chain, ancestor, _ := setUp(t)
		defer os.RemoveAll(tmpDir)

		before(chain, commit)

		expected := []string{"E", "B"}
		if got := ancestor("J", "F"); got[0] != expected[0] || got[1] != expected[1] {
			t.Errorf("want %q, but got %q", expected, got)
		}
	})

	t.Run("finds the best common ancestor", func(t *testing.T) {
		tmpDir, commit, chain, _, mergeBase := setUp(t)
		defer os.RemoveAll(tmpDir)

		before(chain, commit)

		expected := []string{"E"}
		if got := mergeBase("J", "F"); got[0] != expected[0] {
			t.Errorf("want %q, but got %q", expected, got)
		}
	})
}

func TestCommonAncestorWithCommitsBetweenCommonAncestorAndMerge(t *testing.T) {
	//   A   B   C       H   J
	//   o---o---o-------o---o
	//        \         /
	//         o---o---o G
	//         D  E \
	//               o F

	var before = func(chain func([]string), commit func(parents []string, message string, now time.Time)) {
		chain([]string{"", "A", "B", "C"})
		chain([]string{"B", "D", "E", "F"})
		chain([]string{"E", "G"})
		commit([]string{"C", "G"}, "H", time.Now())
		chain([]string{"H", "J"})
	}

	t.Run("finds all the common ancestors", func(t *testing.T) {
		tmpDir, commit, chain, ancestor, _ := setUp(t)
		defer os.RemoveAll(tmpDir)

		before(chain, commit)

		expected := []string{"B", "E"}
		if got := ancestor("J", "F"); got[0] != expected[0] || got[1] != expected[1] {
			t.Errorf("want %q, but got %q", expected, got)
		}
	})

	t.Run("finds the best common ancestor", func(t *testing.T) {
		tmpDir, commit, chain, _, mergeBase := setUp(t)
		defer os.RemoveAll(tmpDir)

		before(chain, commit)

		expected := []string{"E"}
		if got := mergeBase("J", "F"); got[0] != expected[0] {
			t.Errorf("want %q, but got %q", expected, got)
		}
	})
}

func TestCommonAncestorWithEnoughHistoryToFindAllStaleResults(t *testing.T) {
	//   A   B   C             H   J
	//   o---o---o-------------o---o
	//        \      E        /
	//         o-----o-------o
	//        D \     \     / G
	//           \     o   /
	//            \    F  /
	//             o-----o
	//             P     Q

	var before = func(chain func([]string), commit func(parents []string, message string, now time.Time)) {
		chain([]string{"", "A", "B", "C"})
		chain([]string{"B", "D", "E", "F"})
		chain([]string{"D", "P", "Q"})
		commit([]string{"E", "Q"}, "G", time.Now())
		commit([]string{"C", "G"}, "H", time.Now())
		chain([]string{"H", "J"})
	}

	t.Run("finds the best common ancestor", func(t *testing.T) {
		tmpDir, commit, chain, ancestor, _ := setUp(t)
		defer os.RemoveAll(tmpDir)

		before(chain, commit)

		expected := []string{"E"}
		if got := ancestor("J", "F"); got[0] != expected[0] {
			t.Errorf("want %q, but got %q", expected, got)
		}
		expected = []string{"E"}
		if got := ancestor("F", "J"); got[0] != expected[0] {
			t.Errorf("want %q, but got %q", expected, got)
		}
	})
}

func TestCommonAncestorWithManyCommonAncestors(t *testing.T) {
	//         L   M   N   P   Q   R   S   T
	//         o---o---o---o---o---o---o---o
	//        /       /       /       /
	//   o---o---o...o---o...o---o---o---o---o
	//   A   B  C \  D  E \  F  G \  H   J   K
	//             \       \       \
	//              o---o---o---o---o---o
	//              U   V   W   X   Y   Z

	var before = func(chain func([]string), commit func(parents []string, message string, now time.Time)) {
		chain([]string{"", "A", "B", "C", "pad-1-1", "pad-1-2", "pad-1-3", "pad-1-4",
			"D", "E", "pad-2-1", "pad-2-2", "pad-2-3", "pad-2-4",
			"F", "G",
			"H", "J", "K"})

		chain([]string{"B", "L", "M"})
		commit([]string{"M", "D"}, "N", time.Now())
		chain([]string{"N", "P"})
		commit([]string{"P", "F"}, "Q", time.Now())
		chain([]string{"Q", "R"})
		commit([]string{"R", "H"}, "S", time.Now())
		chain([]string{"S", "T"})

		chain([]string{"C", "U", "V"})
		commit([]string{"V", "E"}, "W", time.Now())
		chain([]string{"W", "X"})
		commit([]string{"X", "G"}, "Y", time.Now())
		chain([]string{"Y", "Z"})
	}

	t.Run("finds multiple candidate common ancestors", func(t *testing.T) {
		tmpDir, commit, chain, ancestor, _ := setUp(t)
		defer os.RemoveAll(tmpDir)

		before(chain, commit)

		expected := []string{"G", "D", "B"}
		if got := ancestor("T", "Z"); got[0] != expected[0] || got[1] != expected[1] || got[2] != expected[2] {
			t.Errorf("want %q, but got %q", expected, got)
		}
	})

	t.Run("finds the best common ancestor", func(t *testing.T) {
		tmpDir, commit, chain, _, mergeBase := setUp(t)
		defer os.RemoveAll(tmpDir)

		before(chain, commit)

		expected := []string{"G"}
		if got := mergeBase("T", "Z"); got[0] != expected[0] {
			t.Errorf("want %q, but got %q", expected, got)
		}
	})
}
