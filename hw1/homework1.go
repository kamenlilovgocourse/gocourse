package main

import (
	"fmt"
)

func Repeater(s, sep string) func(int) string {
	f := func(count int) string {
		str := ""
		for i := 0; i < count; i++ {
			if i > 0 {
				str = str + sep
			}
			str = str + s
		}
		return str
	}
	return f
}

func Generator(gen func(int) int, initial int) func() int {
	next := initial
	return func() int {
		toReturn := next
		next = gen(next)
		return toReturn
	}
}

func MapReducer(mapper func(int) int, reducer func(a int, v int) int, initial int) func(...int) int {
	f := func(values ...int) int {
		accumulated := initial
		for _, v := range values {
			mappedV := mapper(v)
			accumulated = reducer(accumulated, mappedV)
		}
		return accumulated
	}
	return f
}

func main() {
	f := Repeater("Go", ",")
	fmt.Printf("Repeater: %s\n", f(3))

	part2Counter := Generator(
		func(v int) int { return v + 1 },
		1,
	)
	part2Square := Generator(
		func(v int) int { return v * v },
		3,
	)
	fmt.Printf("counter: %d\n", part2Counter())
	fmt.Printf("counter: %d\n", part2Counter())
	fmt.Printf("square: %d\n", part2Square())
	fmt.Printf("square: %d\n", part2Square())
	fmt.Printf("square: %d\n", part2Square())

	powerSum := MapReducer(
		func(v int) int { return v * v },
		func(a, v int) int { return a + v },
		0,
	)
	fmt.Printf("powersum 1-4: %d\n", powerSum(1, 2, 3, 4))
}
