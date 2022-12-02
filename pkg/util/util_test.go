package util

import (
	"strconv"
	"testing"
)

func TestStringSliceContains(t *testing.T) {
	tests := []struct {
		name  string
		slice []string
		str   string
		want  bool
	}{{
		name:  "empty slice and empty string",
		slice: []string{},
		str:   "",
		want:  false,
	}, {
		name:  "empty slice",
		slice: []string{},
		str:   "want",
		want:  false,
	}, {
		name:  "regular slice and string",
		slice: []string{"a", "b"},
		str:   "a",
		want:  true,
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := StringSliceContains(tt.slice, tt.str); got != tt.want {
				t.Errorf("StringSliceContains() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestJoinReversedStringSliceForK8s(t *testing.T) {
	var longSlice []string
	for i := 0; i <= 100; i++ {
		longSlice = append(longSlice, strconv.Itoa(i))
	}

	tests := []struct {
		name  string
		slice []string
		want  string
	}{{
		name:  "single entry",
		slice: []string{"a"},
		want:  "a",
	}, {
		name:  "two entries",
		slice: []string{"b", "a"},
		want:  "a,b",
	}, {
		name:  "way too long slice, should be capped at 63 characters",
		slice: longSlice,
		want:  "100,99,98,97,96,95,94,93,92,91,90,89,88,87,86,85,84,83,82,81,80",
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if len(tt.want) > 63 {
				t.Errorf("want is loo long (> 63): %v", tt.want)
			}
			if got := JoinReversedStringSliceForK8s(tt.slice); got != tt.want {
				t.Errorf("JoinReversedStringSliceForK8s() = %v, want %v, len(slice) %v",
					got, tt.want, len(tt.slice))
			}
		})
	}
}
