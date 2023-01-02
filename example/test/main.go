package main

import "fmt"

type ClassA struct {
	*ClassB
	A1 int
}

type ClassB struct {
	*ClassA
	B1 int
}

func main() {
	a := &ClassA{A1: 123}
	b := &ClassB{ClassA: a, B1: 456}
	a.ClassB = b
	fmt.Printf("A(%p): %+v\n", a, a)
	fmt.Printf("B(%p): %+v\n", b, b)
}
