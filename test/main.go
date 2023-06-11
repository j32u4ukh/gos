package main

import "fmt"

func testFunc(data []int) {
	for i, _ := range data {
		data[i] = i
	}
}

func main() {
	data := make([]int, 6)
	data[0] = 7
	data[1] = 8
	data[2] = 9
	testFunc(data)
	fmt.Printf("data: %+v\n", data)
}
