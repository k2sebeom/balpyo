package main

import (
	"reflect"
	"testing"
)

func TestParseSlideList(t *testing.T) {
	got, err := parseSlideList("1, 3-5,8")
	if err != nil {
		t.Fatal(err)
	}
	if want := []int{1, 3, 4, 5, 8}; !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
	for _, bad := range []string{"x", "5-3", "1-"} {
		if _, err := parseSlideList(bad); err == nil {
			t.Errorf("%q: want an error", bad)
		}
	}
}
