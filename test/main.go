package main

import (
	"fmt"
	"strings"
)

func main() {
	url := "/"
	result := strings.Split(url, "/")
	fmt.Printf("result(%d): %+v\n", len(result), result)
	before, after, found := strings.Cut(url, "/")
	fmt.Printf("before: %s, after: %s, found: %v\n", before, after, found)
}
