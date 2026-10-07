package gompare_test

import (
	"encoding/json"
	"fmt"

	"github.com/chriss-de/gompare/v2"
)

type Item struct {
	SKU string `cmp:"sku,identifier"`
	Qty int    `cmp:"qty"`
}

type Order struct {
	ID    string            `cmp:"id"`
	Items []Item            `cmp:"items"`
	Tags  []string          `cmp:"tags"`
	Meta  map[string]string `cmp:"meta"`
	Note  string            `cmp:"-"`
}

func ExampleCompare() {
	left := Order{
		ID:    "1234",
		Items: []Item{{SKU: "apple", Qty: 1}, {SKU: "pear", Qty: 2}},
		Tags:  []string{"new"},
		Meta:  map[string]string{"channel": "web"},
		Note:  "ignored",
	}
	right := Order{
		ID:    "1234",
		Items: []Item{{SKU: "apple", Qty: 3}, {SKU: "plum", Qty: 1}},
		Tags:  []string{"new", "paid"},
		Meta:  map[string]string{"channel": "shop"},
		Note:  "also ignored",
	}

	diffs, err := gompare.Compare(left, right)
	if err != nil {
		panic(err)
	}

	for _, d := range diffs {
		out, _ := json.Marshal(d)
		fmt.Println(string(out))
	}
	// Output:
	// {"type":"changed","path":["items","apple","qty"],"left":1,"right":3}
	// {"type":"removed","path":["items","pear","sku"],"left":"pear","right":null}
	// {"type":"removed","path":["items","pear","qty"],"left":2,"right":null}
	// {"type":"added","path":["items","plum","sku"],"left":null,"right":"plum"}
	// {"type":"added","path":["items","plum","qty"],"left":null,"right":1}
	// {"type":"added","path":["tags","1"],"left":null,"right":"paid"}
	// {"type":"changed","path":["meta","channel"],"left":"web","right":"shop"}
}

func ExampleDifferences_GetDifferences() {
	diffs, err := gompare.Compare(
		Order{Items: []Item{{SKU: "apple", Qty: 1}}, Tags: []string{"a"}},
		Order{Items: []Item{{SKU: "apple", Qty: 2}}, Tags: []string{"b"}},
	)
	if err != nil {
		panic(err)
	}

	for d := range diffs.GetDifferences(gompare.WherePathAt("items", 0), gompare.WhereDiffType(gompare.CHANGED)) {
		fmt.Printf("%s: %v -> %v\n", d.Path, d.Left, d.Right)
	}
	fmt.Println(diffs.HasDifferences(gompare.WherePath("tags")))
	// Output:
	// [items apple qty]: 1 -> 2
	// true
}
