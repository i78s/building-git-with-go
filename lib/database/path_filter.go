package database

import (
	"path/filepath"
	"strings"
)

type PathFilter struct {
	routes *Trie
	Path   string
}

func NewPathFilter(routes *Trie, path string) *PathFilter {
	if routes == nil {
		routes = &Trie{
			matched:  true,
			children: make(map[string]*Trie),
		}
	}

	return &PathFilter{
		routes: routes,
		Path:   path,
	}
}

func PathFilterBuild(paths []string) *PathFilter {
	return NewPathFilter(
		TrieFromPaths(paths),
		"",
	)
}

func (pf *PathFilter) EachEntry(entries map[string]TreeObject, fn func(string, TreeObject)) {
	for name, entry := range entries {
		if pf.routes.matched || pf.routes.children[name] != nil {
			fn(name, entry)
		}
	}
}

func (pf *PathFilter) Join(name string) *PathFilter {
	nextRoutes := pf.routes
	if !pf.routes.matched {
		nextRoutes = pf.routes.children[name]
	}
	return NewPathFilter(nextRoutes, filepath.Join(pf.Path, name))
}

type Trie struct {
	matched  bool
	children map[string]*Trie
}

func TrieFromPaths(paths []string) *Trie {
	root := TrieNode()
	if len(paths) == 0 {
		root.matched = true
	}
	for _, path := range paths {
		trie := root
		elements := strings.Split(filepath.ToSlash(path), "/")
		for _, elem := range elements {
			if _, ok := trie.children[elem]; !ok {
				trie.children[elem] = TrieNode()
			}
			trie = trie.children[elem]
		}
		trie.matched = true
	}
	return root
}

func TrieNode() *Trie {
	return &Trie{
		matched:  false,
		children: make(map[string]*Trie),
	}
}
