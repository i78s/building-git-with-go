package merge

import (
	"building-git/lib/database"
	"os"
	"testing"
	"time"
)

var setUp = func(t *testing.T) (tmpDir string, chain func(names []string), ancestor func(left, right string) string) {
	tmpDir, _ = os.MkdirTemp("", "test-database")

	db := database.NewDatabase(tmpDir)
	commits := map[string]string{}

	commit := func(parent, message string, now time.Time) {
		author := database.NewAuthor("A. U. Thor", "author@example.com", now)
		c := database.NewCommit(parent, "'0' *  40", author, message)
		db.Store(c)
		commits[message] = c.Oid()
	}

	chain = func(names []string) {
		for i := 0; i < len(names)-1; i++ {
			parent := ""
			if names[i] != "" {
				parent = commits[names[i]]
			}
			message := names[i+1]
			commit(parent, message, time.Now())
		}
	}

	ancestor = func(left, right string) string {
		common := NewCommonAncestors(db, commits[left], commits[right])
		obj, _ := db.Load(common.Find())
		return obj.(*database.Commit).Message()
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
		tmpDir, chain, ancestor := setUp(t)
		defer os.RemoveAll(tmpDir)

		before(chain)

		expected := "D"
		if got := ancestor("D", "D"); got != expected {
			t.Errorf("want %q, but got %q", expected, got)
		}
	})

	t.Run("finds the commit that is an ancestor of the other", func(t *testing.T) {
		tmpDir, chain, ancestor := setUp(t)
		defer os.RemoveAll(tmpDir)

		before(chain)

		expected := "B"
		if got := ancestor("B", "D"); got != expected {
			t.Errorf("want %q, but got %q", expected, got)
		}
	})

	t.Run("find the same commit if the arguments are reversed", func(t *testing.T) {
		tmpDir, chain, ancestor := setUp(t)
		defer os.RemoveAll(tmpDir)

		before(chain)

		expected := "B"
		if got := ancestor("D", "B"); got != expected {
			t.Errorf("want %q, but got %q", expected, got)
		}
	})

	t.Run("finds a root commit", func(t *testing.T) {
		tmpDir, chain, ancestor := setUp(t)
		defer os.RemoveAll(tmpDir)

		before(chain)

		expected := "A"
		if got := ancestor("A", "C"); got != expected {
			t.Errorf("want %q, but got %q", expected, got)
		}
	})

	t.Run("finds the intersection of a root commit with itself", func(t *testing.T) {
		tmpDir, chain, ancestor := setUp(t)
		defer os.RemoveAll(tmpDir)

		before(chain)

		expected := "A"
		if got := ancestor("A", "A"); got != expected {
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
		tmpDir, chain, ancestor := setUp(t)
		defer os.RemoveAll(tmpDir)

		before(chain)

		expected := "G"
		if got := ancestor("H", "K"); got != expected {
			t.Errorf("want %q, but got %q", expected, got)
		}
	})

	t.Run("finds an ancestor multiple forks away", func(t *testing.T) {
		tmpDir, chain, ancestor := setUp(t)
		defer os.RemoveAll(tmpDir)

		before(chain)

		expected := "B"
		if got := ancestor("D", "K"); got != expected {
			t.Errorf("want %q, but got %q", expected, got)
		}
	})

	t.Run("finds the same fork point for any point on a branch", func(t *testing.T) {
		tmpDir, chain, ancestor := setUp(t)
		defer os.RemoveAll(tmpDir)

		before(chain)

		expected := "C"
		if got := ancestor("D", "L"); got != expected {
			t.Errorf("want %q, but got %q", expected, got)
		}
		if got := ancestor("M", "D"); got != expected {
			t.Errorf("want %q, but got %q", expected, got)
		}
		if got := ancestor("D", "N"); got != expected {
			t.Errorf("want %q, but got %q", expected, got)
		}
	})

	t.Run("finds the commit that is an ancestor of the other", func(t *testing.T) {
		tmpDir, chain, ancestor := setUp(t)
		defer os.RemoveAll(tmpDir)

		before(chain)

		expected := "E"
		if got := ancestor("K", "E"); got != expected {
			t.Errorf("want %q, but got %q", expected, got)
		}
	})

	t.Run("finds a root commit", func(t *testing.T) {
		tmpDir, chain, ancestor := setUp(t)
		defer os.RemoveAll(tmpDir)

		before(chain)

		expected := "A"
		if got := ancestor("J", "A"); got != expected {
			t.Errorf("want %q, but got %q", expected, got)
		}
	})
}
