package test1

import "testing"

func TestProbaSum_1(t *testing.T) {
	got := test1.ProbaSum(1, 2)
	if got != 2 {
		t.Errorf("Bad!")
	}
}
