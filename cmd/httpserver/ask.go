package httpserver

import (
	"encoding/json"
	"fmt"

	"github.com/j32u4ukh/gos/server/ghttp"
)

func Get1Demo() {
	client := ghttp.NewHttpAsker()
	response, err := client.Get("localhost:5000", nil, nil)
	if err != nil {
		fmt.Printf("送出 Get 請求時發生錯誤, err: %+v\n", err)
		return
	}
	fmt.Printf("response:\n%s", response.String())
}

func Get2Demo() {
	client := ghttp.NewHttpAsker()
	response, err := client.Get("localhost:5000/abc/get/", nil, nil)
	if err != nil {
		fmt.Printf("送出 Get 請求時發生錯誤, err: %+v\n", err)
		return
	}
	fmt.Printf("response:\n%s", response.String())
}



func Post1Demo() {
	client := ghttp.NewHttpAsker()
	response, err := client.Post("localhost:5000", nil, nil)
	if err != nil {
		fmt.Printf("送出 Post 請求時發生錯誤, err: %+v\n", err)
		return
	}
	fmt.Printf("response:\n%s", response.String())
}




func Post2Demo() {
	client := ghttp.NewHttpAsker()
	body := map[string]any{
		"name": "Tom",
		"age": 32,
		"height": 187.3,
	}
	data, err := json.Marshal(body)
	if err != nil {
		fmt.Printf("Marshal data 時發生錯誤, err: %+v\n", err)
		return
	}
	response, err := client.Post("localhost:5000/abc/post/", data, nil)
	if err != nil {
		fmt.Printf("送出 Post 請求時發生錯誤, err: %+v\n", err)
		return
	}
	fmt.Printf("response:\n%s", response.String())
}
