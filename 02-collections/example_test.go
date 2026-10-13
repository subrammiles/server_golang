package collections

import (
	"fmt"
	"maps"
	"slices"
	"strings"
)

func Example_arraysAndSlices() {
	arr := [3]int{1, 2, 3} // ARRAY: fixed size, part of the type, rarely used directly
	s := []int{1, 2, 3}    // SLICE: a view (pointer, length, capacity) over an array; the everyday type
	s = append(s, 4)       // append may allocate a new, bigger array: always use the returned slice
	fmt.Println(arr, s, len(s), cap(s) >= 4)
	fmt.Println(s[1:3]) // slicing: from index 1 up to (not including) 3
	// Output:
	// [1 2 3] [1 2 3 4] 4 true
	// [2 3]
}

func Example_sliceSharing() {
	// GOTCHA: sub-slices SHARE the underlying array
	original := []int{1, 2, 3, 4}
	view := original[:2]
	view[0] = 99
	fmt.Println(original)

	independent := slices.Clone(original[:2]) // copy when you need independence
	independent[0] = -1
	fmt.Println(original[0], independent[0])
	// Output:
	// [99 2 3 4]
	// 99 -1
}

func Example_maps() {
	ages := map[string]int{"asha": 30, "ravi": 25}
	ages["meera"] = 28
	delete(ages, "ravi")
	age, ok := ages["ravi"] // "comma ok": distinguish "missing" from "zero"
	fmt.Println(len(ages), age, ok)
	fmt.Println(slices.Sorted(maps.Keys(ages))) // iterate keys in a stable order
	// Output:
	// 2 0 false
	// [asha meera]
}

func ExampleWordCount() {
	counts := WordCount("go is fun and go is fast and go")
	fmt.Println(counts["go"], counts["missing"])
	fmt.Println(TopWords(counts, 3))
	// Output:
	// 3 0
	// [go and is]
}

func ExampleFilter() {
	evens := Filter([]int{1, 2, 3, 4, 5, 6}, func(n int) bool { return n%2 == 0 })
	fmt.Println(evens)
	// Output: [2 4 6]
}

func Example_slicesPackage() {
	nums := []int{5, 2, 8, 1}
	slices.Sort(nums)
	i, found := slices.BinarySearch(nums, 5)
	fmt.Println(nums, slices.Contains(nums, 8), slices.Max(nums), i, found)
	// Output: [1 2 5 8] true 8 2 true
}

func ExampleReverse() {
	fmt.Println(Reverse("hello"), Reverse("నమస్తే") == "ేత్సమన")
	fmt.Println(CharCount("నమస్తే"))
	// Output:
	// olleh true
	// 18 6
}

func Example_strings() {
	var b strings.Builder // efficient string building (strings are immutable)
	for i := range 3 {
		fmt.Fprintf(&b, "%d,", i)
	}
	fmt.Println(strings.TrimSuffix(b.String(), ","), strings.Split("a-b-c", "-"), strings.Repeat("ab", 2))
	// Output: 0,1,2 [a b c] abab
}

func ExampleEvens() {
	for n := range Evens(100) {
		if n > 6 {
			break
		}
		fmt.Print(n, " ")
	}
	fmt.Println(slices.Collect(Evens(5)))
	// Output: 0 2 4 6 [0 2 4]
}
