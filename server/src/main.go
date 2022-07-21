package main

import (
	"os"

	"./server"
)

func main() {
	var port string
	default_port := "8000"
	if len(os.Args) > 2 {
		panic("Illegal input")
	} else if len(os.Args) == 2 {
		port = os.Args[1]
	} else {
		port = default_port
	}

	server.Start(port)
}
