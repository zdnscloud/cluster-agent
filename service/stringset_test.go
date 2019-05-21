package service

import (
	"testing"

	ut "github.com/zdnscloud/cement/unittest"
)

func stringSetFromSlice(slice []string) StringSet {
	ss := NewStringSet()
	for _, s := range slice {
		ss.Add(s)
	}
	return ss
}

func isStringSetEqual(ss1, ss2 StringSet) bool {
	if len(ss1) != len(ss2) {
		return false
	}

	for s := range ss1 {
		if _, ok := ss2[s]; ok == false {
			return false
		}
	}

	return true
}

func TestStringSetDifference(t *testing.T) {
	cases := []struct {
		ss1        []string
		ss2        []string
		difference []string
	}{
		{
			[]string{"a", "b", "c"},
			[]string{"a", "b"},
			[]string{"c"},
		},
		{
			[]string{"a"},
			[]string{"a", "b"},
			nil,
		},
		{
			[]string{"a"},
			[]string{"a"},
			nil,
		},

		{
			[]string{"a", "c"},
			[]string{"a", "b"},
			[]string{"c"},
		},
	}

	for _, tc := range cases {
		ss1 := stringSetFromSlice(tc.ss1)
		ss2 := stringSetFromSlice(tc.ss2)
		difference := stringSetFromSlice(tc.difference)
		ut.Assert(t, isStringSetEqual(ss1.Difference(ss2), difference), "")
	}
}
