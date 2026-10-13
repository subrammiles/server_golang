// Package collections covers arrays, slices, maps, strings and the slices/maps packages.
package collections

import (
	"cmp"
	"iter"
	"maps"
	"slices"
	"strings"
	"unicode/utf8"
)

// WordCount counts words with a map. Reading a missing key returns the zero value (0), handy for counters.
func WordCount(text string) map[string]int {
	counts := make(map[string]int)
	for _, word := range strings.Fields(strings.ToLower(text)) {
		counts[word]++
	}
	return counts
}

// TopWords returns the n most frequent words. Maps are UNORDERED: sort the keys when order matters.
func TopWords(counts map[string]int, n int) []string {
	words := slices.Collect(maps.Keys(counts))
	slices.SortFunc(words, func(a, b string) int {
		if c := cmp.Compare(counts[b], counts[a]); c != 0 { // higher count first
			return c
		}
		return cmp.Compare(a, b) // then alphabetical, for a stable result
	})
	return words[:min(n, len(words))]
}

// Filter keeps the elements that match. Pre-sizing capacity avoids repeated reallocation.
func Filter(numbers []int, keep func(int) bool) []int {
	out := make([]int, 0, len(numbers))
	for _, n := range numbers {
		if keep(n) {
			out = append(out, n)
		}
	}
	return out
}

// Reverse reverses a string by RUNES (characters), not bytes: strings are UTF-8 bytes.
func Reverse(s string) string {
	runes := []rune(s)
	slices.Reverse(runes)
	return string(runes)
}

// CharCount returns bytes vs characters: they differ for non-ASCII text.
func CharCount(s string) (bytes, runes int) {
	return len(s), utf8.RuneCountInString(s)
}

// Evens is an ITERATOR (Go 1.23): a function you can range over, producing values lazily.
func Evens(limit int) iter.Seq[int] {
	return func(yield func(int) bool) {
		for n := 0; n < limit; n += 2 {
			if !yield(n) { // the loop body asked to stop (break)
				return
			}
		}
	}
}
