package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
)

func main() {
	res, err := http.Head("http://140.210.198.229/v1/signature") // 无超时控制不完美
	if err != nil {
		fmt.Println("请求错误：", err)
		return
	}

	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		fmt.Println("数据错误：", err)
		return
	}

	fmt.Println(string(data))
}
