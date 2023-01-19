package main

import (
	"fmt"
	"sync"
)

type Obj struct {
	Id   int
	Name string
}

var idx int = 0

var objPool = sync.Pool{
	New: func() interface{} {
		idx++
		return &Obj{Id: idx}
	},
}

func main() {
	for i := 0; i < 10; i++ {
		go func() {
			obj := objPool.Get().(*Obj)
			fmt.Printf("obj: %+v\n", obj)
			objPool.Put(obj)
		}()
	}
}
