package gompare

import (
	"fmt"
	"testing"
)

type benchItem struct {
	ID    int `cmp:"id,identifier"`
	Value int `cmp:"value"`
}

type benchPlain struct {
	A int
	B string
}

func BenchmarkUnorderedInts(b *testing.B) {
	const n = 2000
	left := make([]int, n)
	right := make([]int, n)
	for i := range left {
		left[i], right[i] = i, n-i
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := Compare(left, right); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkOrderedInts(b *testing.B) {
	const n = 2000
	left := make([]int, n)
	right := make([]int, n)
	for i := range left {
		left[i], right[i] = i, n-i
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := Compare(left, right, WithSliceOrdering()); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkUnorderedStructsMostlyEqual(b *testing.B) {
	const n = 300
	left := make([]benchPlain, n)
	right := make([]benchPlain, n)
	for i := range left {
		left[i] = benchPlain{A: i, B: fmt.Sprint(i)}
		right[i] = left[i]
	}
	right[n/2].A = -1
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := Compare(left, right); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkIdentifiableStructs(b *testing.B) {
	const n = 2000
	left := make([]benchItem, n)
	right := make([]benchItem, n)
	for i := range left {
		left[i] = benchItem{ID: i, Value: i}
		right[i] = benchItem{ID: i, Value: i + i%2}
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := Compare(left, right); err != nil {
			b.Fatal(err)
		}
	}
}
