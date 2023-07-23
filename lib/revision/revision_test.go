package revision

import (
	"reflect"
	"testing"
)

func assertParse(t *testing.T, expression string, tree Revision) {
	result := Parse(expression)
	if !reflect.DeepEqual(tree, result) {
		t.Errorf("want %q, but got %q", tree, result)
	}
}

func TestParse(t *testing.T) {
	t.Run("parses HEAD", func(t *testing.T) {
		assertParse(t, "HEAD", Ref{"HEAD"})
	})

	t.Run("parses @", func(t *testing.T) {
		assertParse(t, "@", Ref{"HEAD"})
	})

	t.Run("parses a branch name", func(t *testing.T) {
		assertParse(t, "master", Ref{"master"})
	})

	t.Run("parses an object ID", func(t *testing.T) {
		assertParse(t, "3803cb6dc4ab0a852c6762394397dc44405b5ae4", Ref{"3803cb6dc4ab0a852c6762394397dc44405b5ae4"})
	})

	t.Run("parses a parent ref", func(t *testing.T) {
		assertParse(t, "HEAD^", Parent{Ref{"HEAD"}})
	})

	t.Run("parses a chain of parent refs", func(t *testing.T) {
		assertParse(t, "master^^^",
			Parent{Parent{Parent{Ref{"master"}}}})
	})

	t.Run("parses an ancestor ref", func(t *testing.T) {
		assertParse(t, "@~3", Ancestor{Ref{"HEAD"}, 3})
	})

	t.Run("parses a chain of parents and ancestors", func(t *testing.T) {
		assertParse(t, "@~2^^~3",
			Ancestor{
				Parent{
					Parent{
						Ancestor{
							Ref{"HEAD"},
							2,
						},
					},
				},
				3,
			},
		)
	})
}
