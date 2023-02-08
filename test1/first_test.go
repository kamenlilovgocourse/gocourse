package main

import "testing"

func TestProbaSum_1(t *testing.T) {
	got := ProbaSum(1, 2)
	if got != 3 {
		t.Errorf("Bad!")
	}
}
