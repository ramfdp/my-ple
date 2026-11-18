package db

import (
	"slices"
	"testing"

	"my-ple-backend/model"
)

func TestAttributePageContent(t *testing.T) {
	a := model.PageAuthor{ID: "a", Name: "Ana"}
	b := model.PageAuthor{ID: "b", Name: "Ben"}
	c := model.PageAuthor{ID: "c", Name: "Cam"}
	old := []model.PageLine{
		{Kind: "text", Text: "removed"},
		{Kind: "text", Text: "first\nold\nkeep\nlast", Authors: []model.PageAuthor{a, b, b, a}},
		{Kind: "text", Text: "legacy"},
	}
	next := []model.PageLine{
		{Kind: "text", Origin: 2, Text: "first\nnew\nkeep\nadded\nlast", Authors: []model.PageAuthor{a, a, a, a, a}},
		{Kind: "text", Origin: 3, Text: "legacy"},
		{Kind: "text", Origin: -1, Text: "legacy"},
	}
	attributePageContent(old, next, c)
	want := [][]model.PageAuthor{{a, c, b, c, a}, {{}}, {c}}
	for i := range next {
		if !slices.Equal(next[i].Authors, want[i]) || next[i].Origin != 0 {
			t.Errorf("paragraph %d: %+v", i, next[i])
		}
	}
}
