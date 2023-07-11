package lib

import (
	"sort"
)

type SortedMap[T any] struct {
	Keys   []string
	Values map[string]T
}

func NewSortedMap[T any]() *SortedMap[T] {
	return &SortedMap[T]{
		Keys:   make([]string, 0),
		Values: map[string]T{},
	}
}

func (sm *SortedMap[T]) Set(key string, value T) {
	if _, ok := sm.Values[key]; !ok {
		sm.Keys = append(sm.Keys, key)
		sort.Strings(sm.Keys)
	}
	sm.Values[key] = value
}

func (sm *SortedMap[T]) Get(key string) (T, bool) {
	val, ok := sm.Values[key]
	return val, ok
}

func (sm *SortedMap[T]) Iterate(f func(key string, value T)) {
	for _, key := range sm.Keys {
		f(key, sm.Values[key])
	}
}

func (sm *SortedMap[T]) Len() int {
	return len(sm.Values)
}
